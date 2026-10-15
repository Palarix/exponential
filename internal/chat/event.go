// Package chat runs the web board's agent chat: chat sessions stored in
// .xpo/chats/, one agent CLI process per chat that lives across turns, and
// a canonical event stream that the UI renders independent of the agent
// that produced it.
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

	// The board started a turn with no user message: the agent's start was
	// approved and the chat continues in Work Mode on IssueID.
	EventTurnContinued = "turn_continued"

	// The agent started a turn on its own, for example after a background
	// task ended. Text, when set, says what woke it.
	EventAgentTurn = "agent_turn"
)

// Live event types. Subscribers see these while a turn runs; the store does
// not persist them because the settled events above supersede them.
const (
	EventTextDelta = "text_delta"
	EventThinking  = "thinking"
	EventToolStart = "tool_start"

	// The agent's live background tasks changed. Tasks is the new list;
	// empty means none run.
	EventBackgroundTasks = "background_tasks"

	// The board replaced the chat's placeholder title with a generated one.
	// Text is the new title.
	EventTitle = "title"
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

	ApprovalID string            `json:"approval_id,omitempty"` // approval_requested, approval_resolved
	Input      string            `json:"input,omitempty"`       // approval_requested: the full tool input as JSON
	Decision   string            `json:"decision,omitempty"`    // approval_resolved: a Decision value
	Answers    map[string]string `json:"answers,omitempty"`     // approval_resolved: the answers to a question card, by question text

	Mode    string `json:"mode,omitempty"`     // mode_changed: the new mode
	IssueID string `json:"issue_id,omitempty"` // mode_changed: the issue entered or left; turn_continued

	DurationMS int64   `json:"duration_ms,omitempty"` // turn_done
	CostUSD    float64 `json:"cost_usd,omitempty"`    // turn_done: as reported; Claude's is the session's running total
	TokensIn   int     `json:"tokens_in,omitempty"`   // turn_done
	TokensOut  int     `json:"tokens_out,omitempty"`  // turn_done
	CacheTTL   int     `json:"cache_ttl,omitempty"`   // turn_done: seconds the prompt cache lives after the turn

	Message string `json:"message,omitempty"` // error, mode_changed: the reason; tool_call: the error or denial text

	Tasks []BackgroundTask `json:"tasks,omitempty"` // background_tasks
}

// Chat states.
const (
	StateIdle        = "idle"
	StateRunning     = "running"
	StateInterrupted = "interrupted"
)

// usageRev is the revision of how Usage is computed. A saved Usage with an
// older Rev is recomputed from the chat's log.
//
// Rev 1: a turn's cost is the difference between running totals. Before,
// each running total was added, which overcounted long chats.
const usageRev = 1

// Usage is the token and cost totals of a chat's finished turns, as the
// agent reported them. CostKnown is false while no turn reported a cost:
// Codex reports none.
type Usage struct {
	Rev       int     `json:"rev,omitempty"`
	Turns     int     `json:"turns"`
	TokensIn  int     `json:"tokens_in"`
	TokensOut int     `json:"tokens_out"`
	CostUSD   float64 `json:"cost_usd"`
	CostKnown bool    `json:"cost_known"`

	// CostReported is the last running total the agent reported.
	CostReported float64 `json:"cost_reported,omitempty"`
}

// add counts one turn_done event. Its cost is the session's running total,
// so the turn costs the difference to the last one. A total below the last
// one means a new session, which starts from zero.
func (u *Usage) add(e Event) {
	u.Turns++
	u.TokensIn += e.TokensIn
	u.TokensOut += e.TokensOut
	if e.CostUSD <= 0 {
		return
	}
	if e.CostUSD >= u.CostReported {
		u.CostUSD += e.CostUSD - u.CostReported
	} else {
		u.CostUSD += e.CostUSD
	}
	u.CostReported = e.CostUSD
	u.CostKnown = true
}

// Meta is a chat's metadata, stored next to its event log.
type Meta struct {
	ID              string    `json:"id"`
	Title           string    `json:"title"`
	// AutoTitle is set while Title is a placeholder: the first message, or
	// the issue title of a work chat. The board replaces it once, with a
	// generated title, after the first finished turn. A rename clears it.
	AutoTitle bool `json:"auto_title,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	Agent           string    `json:"agent"` // provider ID; set by the first message
	Model           string    `json:"model,omitempty"`
	Effort          string    `json:"effort,omitempty"`
	SessionID       string    `json:"session_id,omitempty"`        // the agent's own session or thread ID
	ClaudeSessionID string    `json:"claude_session_id,omitempty"` // read only: chats saved before session_id
	State           string    `json:"state"`

	// LastError is the error message of the last turn when it failed. A
	// turn clears it when it starts.
	LastError string `json:"last_error,omitempty"`

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

	// CacheTTL is the prompt cache TTL in seconds the agent last reported
	// for this chat. CacheExpiresAt is when the cache of the last finished
	// turn expires; after it, the next message re-sends the conversation.
	CacheTTL       int        `json:"cache_ttl,omitempty"`
	CacheExpiresAt *time.Time `json:"cache_expires_at,omitempty"`

	// AgentPID and AgentStarted identify the chat's live agent process
	// while it runs. A board that starts finds the agents a crashed board
	// left behind by them, and stops them.
	AgentPID     int        `json:"agent_pid,omitempty"`
	AgentStarted *time.Time `json:"agent_started,omitempty"`

	// ArchivedAt is set while the chat is archived: it is out of the active
	// chat lists and takes no messages.
	ArchivedAt *time.Time `json:"archived_at,omitempty"`

	// Usage totals the chat's finished turns. Nil only in a meta saved
	// before totals existed; Store.Meta fills it from the event log.
	Usage *Usage `json:"usage,omitempty"`

	// Waiting is set in chat lists while an approval card waits for the
	// user. It is never stored.
	Waiting bool `json:"waiting,omitempty"`
}
