package chat

import (
	"errors"
	"strings"
	"time"

	"github.com/palarix/exponential/internal/exponential"
)

// Driver starts an agent CLI for a chat. It maps the agent's native output
// to exponential.StreamEvents, so the runner and the UI stay the same for
// every agent.
type Driver interface {
	// Open starts the agent. The agent lives across turns, until Close or
	// until it exits, so background tasks it starts outlive the turn that
	// started them. h receives the agent's output from one goroutine.
	Open(in SessionInput, h SessionHandler) (Session, error)
}

// SessionInput is everything a driver needs to start an agent.
type SessionInput struct {
	SessionID string // the agent's own session or thread ID; empty for a new chat
	Model     string // the model and effort the agent starts with
	Effort    string
	Dir       string // working directory: the project root, or the worktree in Work Mode
	MCP       MCPTarget
	Profile   Profile // the session's tools and rules

	// Approve asks the user about a tool call the agent wants to run. Nil
	// denies every request.
	Approve Approver
	// Grants are rules the user allowed for the whole chat. Claude Code gets
	// them as extra --allowedTools values.
	Grants []string
}

// errAgentExited is returned by Send when the agent has already exited.
var errAgentExited = errors.New("the agent exited")

// agentExited is errAgentExited with the reason the agent gave.
type agentExited struct{ reason error }

func (e agentExited) Error() string        { return e.reason.Error() }
func (e agentExited) Is(target error) bool { return target == errAgentExited }

// Prompt is one turn's input.
type Prompt struct {
	Text   string
	Images []PromptImage
}

// PromptImage is an image the agent gets as image input, not only as a path.
type PromptImage struct {
	Path string // absolute
	MIME string // one of imageTypes
}

// Session is a running agent.
type Session interface {
	// Send starts a turn with prompt. A model or effort that differs from
	// the session's current one is switched in place first.
	Send(prompt Prompt, model, effort string) error
	// Interrupt asks the agent to stop the running turn. Background tasks
	// keep running.
	Interrupt()
	// Close stops the agent and its background tasks, and blocks until it
	// has exited.
	Close()
	// PID is the agent's process ID.
	PID() int
}

// SessionHandler receives a session's output. Every call comes from the
// session's reader goroutine, in order.
type SessionHandler interface {
	// AgentTurn reports a turn the agent started on its own, for example
	// after a background task ended. reason says what woke it, when known.
	AgentTurn(reason string)
	// Stream reports progress of the running turn. StreamSession may also
	// arrive between turns.
	Stream(ev exponential.StreamEvent)
	// TurnEnded ends the running turn. A nil result with an error means the
	// agent ended the turn without one; the error says why.
	TurnEnded(res *TurnResult, err error)
	// Tasks reports the agent's live background tasks, after each change.
	Tasks(tasks []BackgroundTask)
	// Exited reports that the agent exited. A running turn has ended first.
	Exited(err error)
}

// Background task kinds.
const (
	TaskAgent   = "agent"   // subagents and workflows
	TaskMonitor = "monitor" // background shells and monitors
)

// BackgroundTask is work the agent keeps running between turns.
type BackgroundTask struct {
	ID          string `json:"id"`
	Kind        string `json:"kind"` // TaskAgent or TaskMonitor
	Description string `json:"description,omitempty"`
}

// monitorTaskTypes are the Claude Code task types that watch something,
// rather than do agent work. Elysium uses the same list.
var monitorTaskTypes = map[string]bool{"monitor": true, "monitor_mcp": true, "local_bash": true, "shell": true}

// inertTaskTypes are Claude Code bookkeeping, not work.
var inertTaskTypes = map[string]bool{"plan": true, "dream": true}

// launchKey identifies what a session was started with that it cannot
// change while it runs: the agent, its directory and its tools. A session
// whose key no longer matches the chat is restarted. Model, effort and
// grants are not part of it: they change in place.
func launchKey(provider string, p Profile) string {
	return strings.Join([]string{provider, p.Mode, p.Dir, p.Root, p.IssueID, boolString(p.CanWork)}, "\x00")
}

func boolString(b bool) string {
	if b {
		return "1"
	}
	return "0"
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

	Model         string        // the model the agent ran, as it reports it; empty when unknown
	ContextWindow int           // that model's context window in tokens; 0 when unknown
	CacheTTL      time.Duration // TTL of the prompt cache the turn wrote; 0 when none or unknown
}
