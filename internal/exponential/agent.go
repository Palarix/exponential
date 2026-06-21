package exponential

import (
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

// AgentExecutor runs prompts against an AI agent.
type AgentExecutor interface {
	Run(ctx context.Context, prompt string) (*AgentResult, error)
	Name() string
	Stats() AgentStats
}

// NewAgentExecutor creates the appropriate executor for the given agent name.
func NewAgentExecutor(agent string) AgentExecutor {
	bin := agentBinary(agent)
	switch bin {
	case "claude":
		return &claudeExecutor{agent: agent}
	default:
		return &genericExecutor{agent: agent}
	}
}

// claudeExecutor uses `claude -p --output-format json` to get structured
// output with usage stats.
type claudeExecutor struct {
	agent string
	mu    sync.Mutex
	stats AgentStats
}

type claudeJSONResponse struct {
	Result       string      `json:"result"`
	IsError      bool        `json:"is_error"`
	DurationMS   int         `json:"duration_ms"`
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

func (e *claudeExecutor) Run(ctx context.Context, prompt string) (*AgentResult, error) {
	args := []string{"claude", "-p", "--dangerously-skip-permissions", "--output-format", "json", prompt}
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
