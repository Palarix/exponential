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
}

// ClaudeStreamParser turns `claude -p --output-format stream-json --verbose
// --include-partial-messages` lines into StreamEvents. It keeps the state of
// the tool block in progress, so use one parser per turn. Drive and the web
// chat both use it.
type ClaudeStreamParser struct {
	currentTool string
	toolArgBuf  string
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
			if msg.SessionID != "" {
				cb(StreamEvent{Kind: StreamSession, Text: msg.SessionID})
			}
		case "status":
			cb(StreamEvent{Kind: StreamStatus, Text: msg.Status})
		}
	case "stream_event":
		p.handleStreamEvent(msg.Event, cb)
	case "result":
		return &ClaudeTurnResult{
			AgentResult: AgentResult{
				Output:    msg.Result,
				Duration:  time.Duration(msg.DurationMS) * time.Millisecond,
				TokensIn:  msg.Usage.TotalInputTokens(),
				TokensOut: msg.Usage.OutputTokens,
				CostUSD:   msg.TotalCostUSD,
			},
			SessionID: msg.SessionID,
			IsError:   msg.IsError,
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
			p.toolArgBuf = ""
			cb(StreamEvent{Kind: StreamToolStart, Text: block.Name})
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
				cb(StreamEvent{Kind: StreamToolStart, Text: p.currentTool, ToolArg: detail})
			}
		}
	case "content_block_stop":
		done := StreamEvent{Kind: StreamToolDone}
		if p.currentTool != "" {
			done.Text = p.currentTool
			done.ToolArg = p.extractToolDetail()
		}
		p.currentTool = ""
		p.toolArgBuf = ""
		cb(done)
	}
}

// toolDetailKeys are the argument keys shown next to a tool name, in order of
// preference. The last group covers Grep/Glob and the xpo MCP tools.
var toolDetailKeys = []string{"file_path", "description", "command", "url", "skill", "pattern", "title", "id", "query", "match"}

// extractToolDetail extracts meaningful detail from the partially-accumulated
// tool argument JSON. Because input_json_delta arrives in chunks, the buffer
// is usually not valid JSON yet, so we extract values with string matching.
func (p *ClaudeStreamParser) extractToolDetail() string {
	// Some tools have noisy args that aren't useful to display
	switch p.currentTool {
	case "ToolSearch":
		return ""
	}
	for _, key := range toolDetailKeys {
		if v := extractJSONStringValue(p.toolArgBuf, key); v != "" {
			return v
		}
	}
	return ""
}

// extractJSONStringValue pulls the value for a given key from a (possibly
// incomplete) JSON string using simple substring matching.
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
	end := strings.Index(buf[start:], `"`)
	if end < 0 {
		// Value still streaming in — return what we have so far
		return buf[start:]
	}
	return buf[start : start+end]
}

// streamMessage is the top-level envelope for stream-json output.
type streamMessage struct {
	Type         string          `json:"type"`
	Subtype      string          `json:"subtype,omitempty"`
	Status       string          `json:"status,omitempty"`
	SessionID    string          `json:"session_id,omitempty"`
	Event        json.RawMessage `json:"event,omitempty"`
	Result       string          `json:"result,omitempty"`
	IsError      bool            `json:"is_error,omitempty"`
	DurationMS   int             `json:"duration_ms,omitempty"`
	TotalCostUSD float64         `json:"total_cost_usd,omitempty"`
	Usage        claudeUsage     `json:"usage,omitempty"`
}

// streamEvent represents the inner event of a stream_event message.
type streamEvent struct {
	Type         string          `json:"type"`
	Index        int             `json:"index"`
	ContentBlock json.RawMessage `json:"content_block,omitempty"`
	Delta        json.RawMessage `json:"delta,omitempty"`
}
