package exponential

import (
	"encoding/json"
	"strings"
	"time"
)

// ClaudeTurnResult is the final `result` message of one `claude -p` turn.
type ClaudeTurnResult struct {
	AgentResult
	SessionID string
	IsError   bool

	Model         string        // the model the init line reported
	ContextWindow int           // that model's context window in tokens; 0 when unknown
	CacheTTL      time.Duration // TTL of the cache the turn wrote; 0 when none or unknown
}

// ClaudeStreamParser turns `claude -p --output-format stream-json --verbose
// --include-partial-messages` lines into StreamEvents. It keeps the state of
// the tool block in progress, so use one parser per turn. Drive and the web
// chat both use it.
type ClaudeStreamParser struct {
	currentTool   string
	currentToolID string
	toolArgBuf    string
	denied        map[string]bool // tool use IDs that Claude Code denied
	model         string          // from the init line
}

// Parse handles one output line, calling cb for each progress event. It
// returns the turn result when the line is the final `result` message, and
// nil otherwise. Lines that are not JSON are ignored.
func (p *ClaudeStreamParser) Parse(line []byte, cb StreamCallback) *ClaudeTurnResult {
	if len(line) == 0 {
		return nil
	}
	var msg streamMessage
	if err := json.Unmarshal(line, &msg); err != nil {
		return nil
	}

	switch msg.Type {
	case "system":
		switch msg.Subtype {
		case "init":
			p.model = msg.Model
			if msg.SessionID != "" {
				cb(StreamEvent{Kind: StreamSession, Text: msg.SessionID})
			}
		case "status":
			cb(StreamEvent{Kind: StreamStatus, Text: msg.Status})
		case "permission_denied":
			if p.denied == nil {
				p.denied = map[string]bool{}
			}
			p.denied[msg.ToolUseID] = true
		}
	case "stream_event":
		p.handleStreamEvent(msg.Event, cb)
	case "user":
		// Results of a subagent's own tools are not steps of this turn.
		if msg.ParentToolUseID == "" {
			p.handleToolResults(msg, cb)
		}
	case "result":
		return &ClaudeTurnResult{
			AgentResult: AgentResult{
				Output:    msg.Result,
				Duration:  time.Duration(msg.DurationMS) * time.Millisecond,
				TokensIn:  msg.Usage.TotalInputTokens(),
				TokensOut: msg.Usage.OutputTokens,
				CostUSD:   msg.TotalCostUSD,
			},
			SessionID:     msg.SessionID,
			IsError:       msg.IsError,
			Model:         p.model,
			ContextWindow: contextWindow(msg.ModelUsage, p.model),
			CacheTTL:      msg.Usage.CacheTTL(),
		}
	}
	return nil
}

func (p *ClaudeStreamParser) handleStreamEvent(event json.RawMessage, cb StreamCallback) {
	var ev streamEvent
	if err := json.Unmarshal(event, &ev); err != nil {
		return
	}

	switch ev.Type {
	case "content_block_start":
		var block struct {
			Type string `json:"type"`
			ID   string `json:"id"`
			Name string `json:"name"`
		}
		if err := json.Unmarshal(ev.ContentBlock, &block); err != nil {
			return
		}
		switch block.Type {
		case "thinking":
			cb(StreamEvent{Kind: StreamThinking})
		case "tool_use":
			p.currentTool = block.Name
			p.currentToolID = block.ID
			p.toolArgBuf = ""
			cb(StreamEvent{Kind: StreamToolStart, Text: block.Name, ToolID: block.ID})
		}
	case "content_block_delta":
		var delta struct {
			Type        string `json:"type"`
			Text        string `json:"text"`
			PartialJSON string `json:"partial_json"`
		}
		if err := json.Unmarshal(ev.Delta, &delta); err != nil {
			return
		}
		if delta.Type == "text_delta" && delta.Text != "" {
			cb(StreamEvent{Kind: StreamText, Text: delta.Text})
		}
		if delta.Type == "input_json_delta" {
			p.toolArgBuf += delta.PartialJSON
			if detail := p.extractToolDetail(); detail != "" {
				cb(StreamEvent{Kind: StreamToolStart, Text: p.currentTool, ToolArg: detail, ToolID: p.currentToolID})
			}
		}
	case "content_block_stop":
		done := StreamEvent{Kind: StreamToolDone}
		if p.currentTool != "" {
			done.Text = p.currentTool
			done.ToolID = p.currentToolID
			// The input is complete now: decode it properly.
			done.ToolArg = p.finalToolDetail()
		}
		p.currentTool = ""
		p.currentToolID = ""
		p.toolArgBuf = ""
		cb(done)
	}
}

// handleToolResults emits one StreamToolResult per tool_result block of a
// `user` message.
func (p *ClaudeStreamParser) handleToolResults(msg streamMessage, cb StreamCallback) {
	var body struct {
		Content []struct {
			Type      string          `json:"type"`
			ToolUseID string          `json:"tool_use_id"`
			IsError   bool            `json:"is_error"`
			Content   json.RawMessage `json:"content"`
		} `json:"content"`
	}
	if err := json.Unmarshal(msg.Message, &body); err != nil {
		return
	}
	rejected := map[string]bool{}
	for _, m := range msg.ToolResultMeta {
		if m.NonExecutionKind == "user-rejected" {
			rejected[m.ID] = true
		}
	}
	for _, block := range body.Content {
		if block.Type != "tool_result" {
			continue
		}
		ev := StreamEvent{Kind: StreamToolResult, ToolID: block.ToolUseID, ToolStatus: ToolStatusDone}
		switch {
		case p.denied[block.ToolUseID] || rejected[block.ToolUseID]:
			ev.ToolStatus = ToolStatusDenied
		case block.IsError:
			ev.ToolStatus = ToolStatusError
		}
		if ev.ToolStatus != ToolStatusDone {
			ev.ToolMessage = ToolResultText(block.Content)
		}
		cb(ev)
	}
}

// maxToolMessage caps the error text kept for a failed tool call, in runes.
const maxToolMessage = 500

// ToolResultText turns tool_result content (a string, or an array of content
// blocks) into a short message: the text blocks joined, without Claude Code's
// "Error: " prefix, cut to maxToolMessage runes.
func ToolResultText(content json.RawMessage) string {
	var text string
	if err := json.Unmarshal(content, &text); err != nil {
		var blocks []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		}
		if err := json.Unmarshal(content, &blocks); err != nil {
			return ""
		}
		var parts []string
		for _, b := range blocks {
			if b.Type == "text" && b.Text != "" {
				parts = append(parts, b.Text)
			}
		}
		text = strings.Join(parts, "\n")
	}
	text = strings.TrimPrefix(strings.TrimSpace(text), "Error: ")
	if r := []rune(text); len(r) > maxToolMessage {
		text = string(r[:maxToolMessage]) + "…"
	}
	return text
}

// toolDetailKeys are the argument keys shown next to a built-in tool's name,
// in order of preference. mcpToolDetailKeys are for MCP tools such as xpo's,
// where "description" is long body text, not a summary.
var (
	toolDetailKeys    = []string{"file_path", "description", "command", "url", "skill", "pattern", "id", "title", "query", "match"}
	mcpToolDetailKeys = []string{"id", "title", "query", "match", "pattern", "file_path", "url", "path", "sha"}
)

// MCPToolDetail picks the argument shown next to an MCP tool's name from its
// decoded input, as the Claude stream does for mcp__ tools.
func MCPToolDetail(input map[string]any) string {
	for _, key := range mcpToolDetailKeys {
		if v, ok := input[key].(string); ok && v != "" {
			return v
		}
	}
	return ""
}

func (p *ClaudeStreamParser) detailKeys() []string {
	if strings.HasPrefix(p.currentTool, "mcp__") {
		return mcpToolDetailKeys
	}
	return toolDetailKeys
}

// extractToolDetail extracts meaningful detail from the partially-accumulated
// tool argument JSON. Because input_json_delta arrives in chunks, the buffer
// is usually not valid JSON yet, so we extract values with string matching.
func (p *ClaudeStreamParser) extractToolDetail() string {
	// Some tools have noisy args that aren't useful to display
	switch p.currentTool {
	case "ToolSearch":
		return ""
	}
	for _, key := range p.detailKeys() {
		if v := extractJSONStringValue(p.toolArgBuf, key); v != "" {
			return v
		}
	}
	return ""
}

// finalToolDetail picks the detail from the complete tool input, decoded as
// JSON. It falls back to the partial-input scan if the input is not an object.
func (p *ClaudeStreamParser) finalToolDetail() string {
	var input map[string]any
	if err := json.Unmarshal([]byte(p.toolArgBuf), &input); err != nil {
		if p.currentTool == "ToolSearch" {
			return ""
		}
		return p.extractToolDetail()
	}
	return ToolDetail(p.currentTool, input)
}

// ToolDetail picks the argument shown next to a tool's name from its decoded
// input: the same detail the stream shows for a finished tool call.
func ToolDetail(name string, input map[string]any) string {
	if name == "ToolSearch" {
		return ""
	}
	keys := toolDetailKeys
	if strings.HasPrefix(name, "mcp__") {
		keys = mcpToolDetailKeys
	}
	for _, key := range keys {
		if v, ok := input[key].(string); ok && v != "" {
			return v
		}
	}
	return ""
}

// extractJSONStringValue pulls the value for a given key from a (possibly
// incomplete) JSON string using substring matching. The value ends at the
// first unescaped quote, and its JSON escapes are decoded.
func extractJSONStringValue(buf, key string) string {
	needle := `"` + key + `": "`
	idx := strings.Index(buf, needle)
	if idx < 0 {
		needle = `"` + key + `":"`
		idx = strings.Index(buf, needle)
	}
	if idx < 0 {
		return ""
	}
	start := idx + len(needle)
	if start >= len(buf) {
		return ""
	}
	raw := buf[start:] // value still streaming in: use what we have so far
	escaped := false
	for i := 0; i < len(raw); i++ {
		if escaped {
			escaped = false
			continue
		}
		if raw[i] == '\\' {
			escaped = true
			continue
		}
		if raw[i] == '"' {
			raw = raw[:i]
			escaped = false
			break
		}
	}
	if escaped {
		raw = raw[:len(raw)-1] // drop a dangling backslash
	}
	var decoded string
	if err := json.Unmarshal([]byte(`"`+raw+`"`), &decoded); err != nil {
		return raw // e.g. a half-received \u escape
	}
	return decoded
}

// streamMessage is the top-level envelope for stream-json output.
type streamMessage struct {
	Type         string                `json:"type"`
	Subtype      string                `json:"subtype,omitempty"`
	Status       string                `json:"status,omitempty"`
	SessionID    string                `json:"session_id,omitempty"`
	Event        json.RawMessage       `json:"event,omitempty"`
	Result       string                `json:"result,omitempty"`
	IsError      bool                  `json:"is_error,omitempty"`
	DurationMS   int                   `json:"duration_ms,omitempty"`
	TotalCostUSD float64               `json:"total_cost_usd,omitempty"`
	Usage        claudeUsage           `json:"usage,omitempty"`
	Model        string                `json:"model,omitempty"`      // system/init
	ModelUsage   map[string]modelUsage `json:"modelUsage,omitempty"` // result

	ToolUseID       string           `json:"tool_use_id,omitempty"`        // system/permission_denied
	ParentToolUseID string           `json:"parent_tool_use_id,omitempty"` // user: set inside a subagent
	Message         json.RawMessage  `json:"message,omitempty"`            // user: the tool results
	ToolResultMeta  []toolResultMeta `json:"tool_result_meta,omitempty"`   // user
}

// contextWindow returns the context window of model from a result's
// modelUsage. When model has no entry (the init line can name an alias), it
// returns the largest window: a subagent on a smaller model also shows up.
func contextWindow(usage map[string]modelUsage, model string) int {
	if u, ok := usage[model]; ok {
		return u.ContextWindow
	}
	largest := 0
	for _, u := range usage {
		largest = max(largest, u.ContextWindow)
	}
	return largest
}

// modelUsage is one model's entry in a result's modelUsage.
type modelUsage struct {
	ContextWindow int `json:"contextWindow"`
}

// toolResultMeta is Claude Code's note on how a tool result came about.
type toolResultMeta struct {
	ID               string `json:"id"`
	NonExecutionKind string `json:"non_execution_kind"`
}

// streamEvent represents the inner event of a stream_event message.
type streamEvent struct {
	Type         string          `json:"type"`
	Index        int             `json:"index"`
	ContentBlock json.RawMessage `json:"content_block,omitempty"`
	Delta        json.RawMessage `json:"delta,omitempty"`
}
