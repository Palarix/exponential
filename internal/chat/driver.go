package chat

import (
	"context"
	"time"

	"github.com/palarix/exponential/internal/exponential"
)

// Driver runs one chat turn on one agent CLI. It maps the agent's native
// output to exponential.StreamEvents, so the runner and the UI stay the same
// for every agent.
type Driver interface {
	// Turn runs one turn and blocks until it ends. Cancelling ctx stops the
	// agent. It returns a nil result when the agent ended without one; the
	// error then says why.
	Turn(ctx context.Context, in TurnInput, emit exponential.StreamCallback) (*TurnResult, error)
}

// TurnInput is everything a driver needs for one turn.
type TurnInput struct {
	Prompt    string
	SessionID string // the agent's own session or thread ID; empty for a new chat
	Model     string
	Effort    string
	Dir       string // working directory: the project root
	MCP       MCPTarget

	// Approve asks the user about a tool call the agent wants to run. Nil
	// denies every request.
	Approve Approver
	// Grants are rules the user allowed for the whole chat. Claude Code gets
	// them as extra --allowedTools values.
	Grants []string
}

// TurnResult is the outcome of a turn the agent finished.
type TurnResult struct {
	SessionID string
	IsError   bool
	Output    string // the error text when IsError
	Duration  time.Duration
	CostUSD   float64 // 0 when the agent does not report a price
	TokensIn  int
	TokensOut int

	Model         string // the model the agent ran, as it reports it; empty when unknown
	ContextWindow int    // that model's context window in tokens; 0 when unknown
}
