// Package chat runs the web board's agent chat: chat sessions stored in
// .xpo/chats/, one agent CLI process per turn, and a canonical event stream
// that the UI renders independent of the agent that produced it.
package chat

import "time"

// Settled event types. The store persists these; replaying them rebuilds a
// chat thread.
const (
	EventUserMessage   = "user_message"
	EventAssistantText = "assistant_text"
	EventToolCall      = "tool_call"
	EventTurnDone      = "turn_done"
	EventTurnStopped   = "turn_stopped"
	EventError         = "error"

	// An approval card opened, and its decision.
	EventApprovalRequested = "approval_requested"
	EventApprovalResolved  = "approval_resolved"

	// The chat entered or left Work Mode, between turns. The UI draws a
	// divider. Message, when set, says why.
	EventModeChanged = "mode_changed"
)

// Live event types. Subscribers see these while a turn runs; the store does
// not persist them because the settled events above supersede them.
const (
	EventTextDelta = "text_delta"
	EventThinking  = "thinking"
	EventToolStart = "tool_start"
)

// Context describes what the user had open when they sent a message.
type Context struct {
	View       string `json:"view,omitempty"`
	IssueID    string `json:"issue_id,omitempty"`
	IssueTitle string `json:"issue_title,omitempty"`
}

// Event is one entry in a chat's stream. Seq is set on settled events only
// and doubles as the SSE event ID.
type Event struct {
	Seq  int       `json:"seq,omitempty"`
	Type string    `json:"type"`
	Time time.Time `json:"time"`

	Text    string   `json:"text,omitempty"`    // user_message, assistant_text, text_delta
	Context *Context `json:"context,omitempty"` // user_message

	Name   string `json:"name,omitempty"`    // tool_call, tool_start, approval_requested
	Arg    string `json:"arg,omitempty"`     // tool_call, tool_start, approval_requested
	ToolID string `json:"tool_id,omitempty"` // tool_call, tool_start, approval_requested; empty in chats saved before tool results
	Status string `json:"status,omitempty"`  // tool_call: "done", "error" or "denied"

	ApprovalID string `json:"approval_id,omitempty"` // approval_requested, approval_resolved
	Input      string `json:"input,omitempty"`       // approval_requested: the full tool input as JSON
	Decision   string `json:"decision,omitempty"`    // approval_resolved: a Decision value

	Mode    string `json:"mode,omitempty"`     // mode_changed: the new mode
	IssueID string `json:"issue_id,omitempty"` // mode_changed: the issue entered or left

	DurationMS int64   `json:"duration_ms,omitempty"` // turn_done
	CostUSD    float64 `json:"cost_usd,omitempty"`    // turn_done
	TokensIn   int     `json:"tokens_in,omitempty"`   // turn_done
	TokensOut  int     `json:"tokens_out,omitempty"`  // turn_done

	Message string `json:"message,omitempty"` // error, mode_changed: the reason; tool_call: the error or denial text
}

// Chat states.
const (
	StateIdle        = "idle"
	StateRunning     = "running"
	StateInterrupted = "interrupted"
)

// Meta is a chat's metadata, stored next to its event log.
type Meta struct {
	ID              string    `json:"id"`
	Title           string    `json:"title"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	Agent           string    `json:"agent"` // provider ID; set by the first message
	Model           string    `json:"model,omitempty"`
	Effort          string    `json:"effort,omitempty"`
	SessionID       string    `json:"session_id,omitempty"`        // the agent's own session or thread ID
	ClaudeSessionID string    `json:"claude_session_id,omitempty"` // read only: chats saved before session_id
	State           string    `json:"state"`

	// The model the agent ran in the last finished turn, and its context
	// window in tokens.
	ResolvedModel string `json:"resolved_model,omitempty"`
	ContextWindow int    `json:"context_window,omitempty"`

	// Grants are the rules the user allowed with "Allow for this chat".
	// Every later turn gets them, because the agents forget session
	// permissions when a turn's process exits.
	Grants []string `json:"grants,omitempty"`

	// Mode is ModeChat or ModeWork; empty is Chat Mode. Work is the bound
	// issue in Work Mode. WorkedIssues lists every issue the chat entered
	// Work Mode on, oldest first.
	Mode         string       `json:"mode,omitempty"`
	Work         *WorkBinding `json:"work,omitempty"`
	WorkedIssues []string     `json:"worked_issues,omitempty"`

	// Waiting is set in chat lists while an approval card waits for the
	// user. It is never stored.
	Waiting bool `json:"waiting,omitempty"`
}
