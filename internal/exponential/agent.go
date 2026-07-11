package exponential

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// AgentResult holds the output and usage from a single agent call.
type AgentResult struct {
	Output    string
	Duration  time.Duration
	TokensIn  int
	TokensOut int
	CostUSD   float64
}

// AgentStats holds accumulated usage across multiple calls.
type AgentStats struct {
	Calls     int
	Duration  time.Duration
	TokensIn  int
	TokensOut int
	CostUSD   float64
}

// StreamEventKind identifies the type of streaming progress event.
type StreamEventKind int

const (
	StreamText      StreamEventKind = iota // Partial text output
	StreamThinking                         // Agent is reasoning
	StreamToolStart                        // Agent started using a tool
	StreamToolDone                         // Tool execution completed
	StreamStatus                           // Status update (e.g. "requesting")
)

// StreamEvent is a progress update emitted during streaming execution.
type StreamEvent struct {
	Kind    StreamEventKind
	Text    string // text delta, tool name, or status message
	ToolArg string // tool description or command (for StreamToolStart)
}

// StreamCallback receives progress events during streaming execution.
type StreamCallback func(StreamEvent)

// AgentExecutor runs prompts against an AI agent.
type AgentExecutor interface {
	Run(ctx context.Context, prompt string) (*AgentResult, error)
	RunStreaming(ctx context.Context, prompt string, cb StreamCallback) (*AgentResult, error)
	Name() string
	Stats() AgentStats
}

// NewAgentExecutor creates the appropriate executor for the given agent name.
// An optional model override can be passed (e.g. "sonnet") to use a specific
// model instead of the CLI default.
func NewAgentExecutor(agent string, model ...string) AgentExecutor {
	bin := agentBinary(agent)
	m := ""
	if len(model) > 0 {
		m = model[0]
	}
	switch bin {
	case "claude":
		return &claudeExecutor{agent: agent, model: m}
	default:
		return &genericExecutor{agent: agent}
	}
}

// claudeExecutor uses `claude -p --output-format json` to get structured
// output with usage stats. It captures the session ID from the first call
// and reuses it via --resume for subsequent calls, enabling prompt caching.
type claudeExecutor struct {
	agent       string
	model       string
	mu          sync.Mutex
	stats       AgentStats
	sessionID   string
	currentTool string
	toolArgBuf  string
}

type claudeJSONResponse struct {
	Result       string      `json:"result"`
	IsError      bool        `json:"is_error"`
	DurationMS   int         `json:"duration_ms"`
	SessionID    string      `json:"session_id"`
	CostUSD      *float64    `json:"cost_usd"`
	TotalCostUSD float64     `json:"total_cost_usd"`
	Usage        claudeUsage `json:"usage"`
}

type claudeUsage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
}

func (u claudeUsage) TotalInputTokens() int {
	return u.InputTokens + u.CacheCreationInputTokens + u.CacheReadInputTokens
}

func (e *claudeExecutor) buildArgs(extra ...string) []string {
	args := []string{"claude", "-p", "--dangerously-skip-permissions"}
	if e.model != "" {
		args = append(args, "--model", e.model)
	}
	e.mu.Lock()
	sid := e.sessionID
	e.mu.Unlock()
	if sid != "" {
		args = append(args, "--resume", sid)
	}
	args = append(args, extra...)
	return args
}

func (e *claudeExecutor) Run(ctx context.Context, prompt string) (*AgentResult, error) {
	args := e.buildArgs("--output-format", "json", prompt)
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	start := time.Now()
	out, err := cmd.Output()
	elapsed := time.Since(start)

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("%s timed out", e.agent)
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("%s exited %d: %s", e.agent, exitErr.ExitCode(), string(exitErr.Stderr))
		}
		return nil, err
	}

	var resp claudeJSONResponse
	if err := json.Unmarshal(out, &resp); err != nil {
		// Fall back to raw output if JSON envelope parsing fails
		result := &AgentResult{Output: string(out), Duration: elapsed}
		e.record(result)
		return result, nil
	}

	if resp.SessionID != "" {
		e.mu.Lock()
		e.sessionID = resp.SessionID
		e.mu.Unlock()
	}

	if resp.IsError {
		return nil, fmt.Errorf("%s returned error: %s", e.agent, resp.Result)
	}

	cost := resp.TotalCostUSD
	if cost == 0 && resp.CostUSD != nil {
		cost = *resp.CostUSD
	}
	result := &AgentResult{
		Output:    resp.Result,
		Duration:  time.Duration(resp.DurationMS) * time.Millisecond,
		TokensIn:  resp.Usage.TotalInputTokens(),
		TokensOut: resp.Usage.OutputTokens,
		CostUSD:   cost,
	}
	if result.Duration == 0 {
		result.Duration = elapsed
	}
	e.record(result)
	return result, nil
}

func (e *claudeExecutor) RunStreaming(ctx context.Context, prompt string, cb StreamCallback) (*AgentResult, error) {
	args := e.buildArgs("--output-format", "stream-json", "--verbose", "--include-partial-messages", prompt)
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	start := time.Now()
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	var result *AgentResult
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var msg streamMessage
		if err := json.Unmarshal(line, &msg); err != nil {
			continue
		}

		switch msg.Type {
		case "system":
			if msg.Subtype == "status" {
				cb(StreamEvent{Kind: StreamStatus, Text: msg.Status})
			}
		case "stream_event":
			e.handleStreamEvent(msg.Event, cb)
		case "result":
			if msg.SessionID != "" {
				e.mu.Lock()
				e.sessionID = msg.SessionID
				e.mu.Unlock()
			}
			elapsed := time.Since(start)
			cost := msg.TotalCostUSD
			result = &AgentResult{
				Output:    msg.Result,
				Duration:  time.Duration(msg.DurationMS) * time.Millisecond,
				TokensIn:  msg.Usage.TotalInputTokens(),
				TokensOut: msg.Usage.OutputTokens,
				CostUSD:   cost,
			}
			if result.Duration == 0 {
				result.Duration = elapsed
			}
		}
	}

	if err := cmd.Wait(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("%s timed out", e.agent)
		}
		if result == nil {
			return nil, fmt.Errorf("%s exited with error: %w", e.agent, err)
		}
	}

	if result == nil {
		return nil, fmt.Errorf("%s produced no result", e.agent)
	}

	e.record(result)
	return result, nil
}

func (e *claudeExecutor) handleStreamEvent(event json.RawMessage, cb StreamCallback) {
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
			e.currentTool = block.Name
			e.toolArgBuf = ""
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
			e.toolArgBuf += delta.PartialJSON
			if detail := e.extractToolDetail(); detail != "" {
				cb(StreamEvent{Kind: StreamToolStart, Text: e.currentTool, ToolArg: detail})
			}
		}
	case "content_block_stop":
		e.currentTool = ""
		e.toolArgBuf = ""
		cb(StreamEvent{Kind: StreamToolDone})
	}
}

// extractToolDetail extracts meaningful detail from the partially-accumulated
// tool argument JSON. Because input_json_delta arrives in chunks, the buffer
// is usually not valid JSON yet, so we extract values with string matching.
func (e *claudeExecutor) extractToolDetail() string {
	// Some tools have noisy args that aren't useful to display
	switch e.currentTool {
	case "ToolSearch":
		return ""
	}
	for _, key := range []string{"file_path", "description", "command", "url", "skill"} {
		if v := extractJSONStringValue(e.toolArgBuf, key); v != "" {
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

func (e *claudeExecutor) Name() string { return e.agent }

func (e *claudeExecutor) Stats() AgentStats {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.stats
}

func (e *claudeExecutor) record(r *AgentResult) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.stats.Calls++
	e.stats.Duration += r.Duration
	e.stats.TokensIn += r.TokensIn
	e.stats.TokensOut += r.TokensOut
	e.stats.CostUSD += r.CostUSD
}

// genericExecutor handles any agent as a plain subprocess.
type genericExecutor struct {
	agent string
	mu    sync.Mutex
	stats AgentStats
}

func (e *genericExecutor) RunStreaming(ctx context.Context, prompt string, cb StreamCallback) (*AgentResult, error) {
	return e.Run(ctx, prompt)
}

func (e *genericExecutor) Run(ctx context.Context, prompt string) (*AgentResult, error) {
	bin := agentBinary(e.agent)
	args := []string{bin, prompt}
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	start := time.Now()
	out, err := cmd.Output()
	elapsed := time.Since(start)

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("%s timed out", e.agent)
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("%s exited %d: %s", e.agent, exitErr.ExitCode(), string(exitErr.Stderr))
		}
		return nil, err
	}

	result := &AgentResult{Output: string(out), Duration: elapsed}
	e.mu.Lock()
	e.stats.Calls++
	e.stats.Duration += elapsed
	e.mu.Unlock()
	return result, nil
}

func (e *genericExecutor) Name() string { return e.agent }

func (e *genericExecutor) Stats() AgentStats {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.stats
}

// agentBinary returns the executable name for an agent string.
func agentBinary(agent string) string {
	if agent == "claude" || strings.HasPrefix(agent, "claude ") {
		return "claude"
	}
	fields := strings.Fields(agent)
	if len(fields) > 0 {
		return fields[0]
	}
	return agent
}

// mergeStats combines supervisor and coder stats for display.
func mergeStats(a, b AgentStats) AgentStats {
	return AgentStats{
		Calls:     a.Calls + b.Calls,
		Duration:  a.Duration + b.Duration,
		TokensIn:  a.TokensIn + b.TokensIn,
		TokensOut: a.TokensOut + b.TokensOut,
		CostUSD:   a.CostUSD + b.CostUSD,
	}
}

func formatStats(s AgentStats) string {
	parts := []string{}
	if s.TokensIn > 0 || s.TokensOut > 0 {
		parts = append(parts, fmt.Sprintf("%dk in / %dk out", s.TokensIn/1000, s.TokensOut/1000))
	}
	if s.CostUSD > 0 {
		parts = append(parts, fmt.Sprintf("$%.2f", s.CostUSD))
	}
	if s.Calls > 0 {
		parts = append(parts, fmt.Sprintf("%d calls", s.Calls))
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, " · ")
}
