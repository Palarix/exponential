package chat

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/palarix/exponential/internal/exponential"
)

// ErrBusy is returned when a chat already has a turn running.
var ErrBusy = errors.New("a reply is already running")

// ErrProviderLocked is returned when a message asks for another provider
// than the one that holds the chat's session.
var ErrProviderLocked = errors.New("this chat belongs to another agent; start a new chat to switch")

// RunnerConfig configures how turns are run.
type RunnerConfig struct {
	Store           *Store
	Drivers         map[string]Driver // by provider ID; Claude and Codex when nil
	Dir             string            // working directory of the agent: the project root
	MCP             MCPTarget
	DefaultProvider string  // used when a message names none; "claude" when empty
	DefaultModel    string  // chat.model from config; used for the default provider when a chat has no model
	Learner         Learner // told which model each finished turn ran; optional

	// CanWork allows Work Mode. False in remote mode, where start and merge
	// run on the server (xpo-a60065).
	CanWork bool
	// ResolveWork returns an issue's full ID and worktree, once the agent's
	// start call on it succeeded. The ID may be a unique suffix.
	ResolveWork func(issueID string) (WorkBinding, error)
}

// Learner learns which model a picked model ID resolved to. The Catalog is
// one.
type Learner interface {
	Learn(provider, picked, resolved string, window int)
}

// SendRequest is one user message.
type SendRequest struct {
	Text     string
	Context  *Context
	Provider string // empty means the default provider
	Model    string
	Effort   string
}

// Runner runs chat turns, one agent process per turn, and fans their events
// out to subscribers.
type Runner struct {
	cfg RunnerConfig

	mu        sync.Mutex
	turns     map[string]*turn
	subs      map[string]map[chan Event]struct{}
	approvals map[string]map[string]chan string // open approval cards: chat ID → approval ID → decision
	closing   bool                              // Shutdown began: no turn continues on its own
	wg        sync.WaitGroup
}

type turn struct {
	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{} // closed when the turn has finished

	// denied holds the tool use IDs whose approval the user denied. The
	// agents report such a call as an error; the work list shows it denied.
	denied sync.Map
}

// NewRunner returns a runner for cfg.
func NewRunner(cfg RunnerConfig) *Runner {
	if cfg.Drivers == nil {
		cfg.Drivers = map[string]Driver{
			ProviderClaude: &ClaudeDriver{},
			ProviderCodex:  &CodexDriver{},
		}
	}
	if cfg.DefaultProvider == "" {
		cfg.DefaultProvider = ProviderClaude
	}
	return &Runner{cfg: cfg, turns: map[string]*turn{}, subs: map[string]map[chan Event]struct{}{}, approvals: map[string]map[string]chan string{}}
}

// Send records the user message and starts a turn in the background.
func (r *Runner) Send(id string, req SendRequest) error {
	meta, err := r.cfg.Store.Meta(id)
	if err != nil {
		return err
	}
	provider := req.Provider
	if provider == "" {
		provider = r.cfg.DefaultProvider
	}
	driver, ok := r.cfg.Drivers[provider]
	if !ok {
		return fmt.Errorf("unknown provider %q", provider)
	}
	if meta.SessionID != "" && meta.Agent != "" && meta.Agent != provider {
		return ErrProviderLocked
	}

	r.mu.Lock()
	if _, busy := r.turns[id]; busy {
		r.mu.Unlock()
		return ErrBusy
	}
	ctx, cancel := context.WithCancel(context.Background())
	t := &turn{ctx: ctx, cancel: cancel, done: make(chan struct{})}
	r.turns[id] = t
	r.wg.Add(1)
	r.mu.Unlock()

	// Read the meta again: BindWork may have changed it before the turn was
	// registered, and no mode change can happen from here on.
	if meta, err = r.cfg.Store.Meta(id); err != nil {
		r.finish(id, t)
		return err
	}
	notice := checkWorktree(&meta)
	if meta.Title == "" {
		meta.Title = TitleFrom(req.Text)
	}
	if meta.Agent != provider {
		meta.SessionID = "" // a session belongs to the agent that made it
	}
	meta.Agent = provider
	meta.Model = req.Model
	meta.Effort = req.Effort
	meta.State = StateRunning
	meta.UpdatedAt = time.Now().UTC()
	if err := r.cfg.Store.SaveMeta(meta); err != nil {
		r.finish(id, t)
		return err
	}
	if notice != nil {
		r.settle(id, *notice)
	}
	r.settle(id, Event{Type: EventUserMessage, Text: req.Text, Context: req.Context})

	profile := r.profileFor(meta)
	go r.run(id, meta, driver, profile, buildPrompt(req.Text, req.Context, profile), t)
	return nil
}

// continuePrompt is the message of the turn the board starts after an
// approved start put the chat in Work Mode.
func continuePrompt(issueID string) string {
	return "Work Mode is on for " + issueID + ". Continue with what the user asked for."
}

// handOverLocked registers the next turn of a chat in place of t, so no
// message can start a turn in between. r.mu must be held.
func (r *Runner) handOverLocked(id string) *turn {
	ctx, cancel := context.WithCancel(context.Background())
	next := &turn{ctx: ctx, cancel: cancel, done: make(chan struct{})}
	r.turns[id] = next
	r.wg.Add(1)
	return next
}

// Running reports whether a chat has a turn in progress.
func (r *Runner) Running(id string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.turns[id]
	return ok
}

// Cancel stops a chat's running turn. It returns false when none is running.
func (r *Runner) Cancel(id string) bool {
	r.mu.Lock()
	t, ok := r.turns[id]
	r.mu.Unlock()
	if !ok {
		return false
	}
	t.cancel()
	return true
}

// CancelAndWait stops a chat's running turn, if any, and waits until it has
// finished writing its events.
func (r *Runner) CancelAndWait(id string) {
	r.mu.Lock()
	t, ok := r.turns[id]
	r.mu.Unlock()
	if !ok {
		return
	}
	r.Cancel(id)
	<-t.done
}

// Wait blocks until no turn is running.
func (r *Runner) Wait() { r.wg.Wait() }

// Shutdown stops every running turn and waits for them to end.
func (r *Runner) Shutdown() {
	r.mu.Lock()
	r.closing = true
	ids := make([]string, 0, len(r.turns))
	for id := range r.turns {
		ids = append(ids, id)
	}
	r.mu.Unlock()
	for _, id := range ids {
		r.Cancel(id)
	}
	r.Wait()
}

// Subscribe returns a channel of every event published for a chat, settled
// and live. A slow subscriber misses events rather than blocking the turn.
func (r *Runner) Subscribe(id string) (<-chan Event, func()) {
	ch := make(chan Event, 256)
	r.mu.Lock()
	if r.subs[id] == nil {
		r.subs[id] = map[chan Event]struct{}{}
	}
	r.subs[id][ch] = struct{}{}
	r.mu.Unlock()
	return ch, func() {
		r.mu.Lock()
		delete(r.subs[id], ch)
		if len(r.subs[id]) == 0 {
			delete(r.subs, id)
		}
		r.mu.Unlock()
	}
}

func (r *Runner) publish(id string, e Event) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for ch := range r.subs[id] {
		select {
		case ch <- e:
		default:
		}
	}
}

// settle persists events and publishes them as stored (with Seq).
func (r *Runner) settle(id string, events ...Event) {
	stored, err := r.cfg.Store.Append(id, events...)
	if err != nil {
		stored = events
	}
	for _, e := range stored {
		r.publish(id, e)
	}
}

func (r *Runner) live(id string, e Event) {
	e.Time = time.Now().UTC()
	r.publish(id, e)
}

// finish ends turn t. It leaves the chat registered when t handed it over
// to a next turn.
func (r *Runner) finish(id string, t *turn) {
	r.mu.Lock()
	t.cancel()
	close(t.done)
	if r.turns[id] == t {
		delete(r.turns, id)
	}
	r.mu.Unlock()
	r.wg.Done()
}

func (r *Runner) run(id string, meta Meta, driver Driver, profile Profile, prompt string, t *turn) {
	defer r.finish(id, t)

	sessionID := meta.SessionID
	var resolved *TurnResult   // set when the turn finished
	var cacheTTL time.Duration // the prompt cache TTL of a finished turn
	var cacheEnd time.Time     // when that turn finished
	modes := &modeTools{profile: profile}
	defer func() {
		r.mu.Lock()
		m, err := r.cfg.Store.Meta(id)
		var notice *Event
		var next *turn
		if err == nil {
			m.State = StateIdle
			m.SessionID = sessionID
			if resolved != nil && resolved.Model != "" {
				m.ResolvedModel = resolved.Model
				m.ContextWindow = resolved.ContextWindow
			}
			if cacheTTL > 0 {
				expires := cacheEnd.Add(cacheTTL).UTC()
				m.CacheTTL = int(cacheTTL / time.Second)
				m.CacheExpiresAt = &expires
			}
			notice = modes.apply(r, &m)
			// An approved start put the chat in Work Mode: continue there,
			// unless the turn was stopped or failed.
			if notice != nil && notice.Mode == ModeWork && resolved != nil && t.ctx.Err() == nil && !r.closing {
				next = r.handOverLocked(id)
				m.State = StateRunning
			}
			m.UpdatedAt = time.Now().UTC()
			_ = r.cfg.Store.SaveMeta(m)
		}
		r.mu.Unlock()
		if notice != nil {
			r.settle(id, *notice)
		}
		if next != nil {
			r.settle(id, Event{Type: EventTurnContinued, IssueID: m.Work.IssueID})
			p := r.profileFor(m)
			go r.run(id, m, driver, p, buildPrompt(continuePrompt(m.Work.IssueID), nil, p), next)
		}
	}()

	model := meta.Model
	if model == "" && meta.Agent == r.cfg.DefaultProvider {
		model = r.cfg.DefaultModel
	}

	var text strings.Builder
	flushText := func() {
		if text.Len() > 0 {
			r.settle(id, Event{Type: EventAssistantText, Text: text.String()})
			text.Reset()
		}
	}

	// A tool call settles when its result arrives. pending holds the calls
	// whose input is complete but whose result has not arrived yet.
	pending := map[string]Event{}
	var pendingOrder []string
	settlePending := func() {
		for _, toolID := range pendingOrder {
			if call, ok := pending[toolID]; ok {
				r.settle(id, call)
			}
		}
		clear(pending)
		pendingOrder = nil
	}

	if t.ctx.Err() != nil {
		r.settle(id, Event{Type: EventTurnStopped})
		return
	}
	result, err := driver.Turn(t.ctx, TurnInput{
		Prompt:    prompt,
		SessionID: sessionID,
		Model:     model,
		Effort:    meta.Effort,
		Dir:       profile.Dir,
		MCP:       r.cfg.MCP,
		Profile:   profile,
		Approve:   r.approver(id, t),
		Grants:    meta.Grants,
	}, func(ev exponential.StreamEvent) {
		switch ev.Kind {
		case exponential.StreamSession:
			sessionID = ev.Text
		case exponential.StreamText:
			text.WriteString(ev.Text)
			r.live(id, Event{Type: EventTextDelta, Text: ev.Text})
		case exponential.StreamThinking:
			r.live(id, Event{Type: EventThinking})
		case exponential.StreamToolStart:
			flushText()
			r.live(id, Event{Type: EventToolStart, Name: ev.Text, Arg: ev.ToolArg, ToolID: ev.ToolID})
		case exponential.StreamToolDone:
			if ev.Text == "" {
				flushText() // end of a text or thinking block
				return
			}
			call := Event{Type: EventToolCall, Name: ev.Text, Arg: ev.ToolArg, ToolID: ev.ToolID, Status: exponential.ToolStatusDone}
			if ev.ToolID == "" {
				r.settle(id, call) // no ID to match a result to
				return
			}
			pending[ev.ToolID] = call
			pendingOrder = append(pendingOrder, ev.ToolID)
		case exponential.StreamToolResult:
			call, ok := pending[ev.ToolID]
			if !ok {
				return
			}
			delete(pending, ev.ToolID)
			call.Status = ev.ToolStatus
			if _, denied := t.denied.Load(ev.ToolID); denied && call.Status == exponential.ToolStatusError {
				call.Status = exponential.ToolStatusDenied
			}
			call.Message = ev.ToolMessage
			r.settle(id, call)
			modes.observe(call)
		}
	})
	flushText()
	settlePending()
	if result != nil && result.SessionID != "" {
		sessionID = result.SessionID
	}

	switch {
	case t.ctx.Err() != nil:
		r.settle(id, Event{Type: EventTurnStopped})
	case result == nil:
		msg := "the agent exited without a reply"
		if err != nil {
			msg = err.Error()
		}
		r.settle(id, Event{Type: EventError, Message: msg})
	case result.IsError:
		r.settle(id, Event{Type: EventError, Message: result.Output})
	default:
		resolved = result
		cacheTTL = turnCacheTTL(result.CacheTTL, meta.CacheTTL)
		cacheEnd = time.Now()
		if r.cfg.Learner != nil {
			r.cfg.Learner.Learn(meta.Agent, meta.Model, result.Model, result.ContextWindow)
		}
		r.settle(id, Event{
			Type:       EventTurnDone,
			DurationMS: result.Duration.Milliseconds(),
			CostUSD:    result.CostUSD,
			TokensIn:   result.TokensIn,
			TokensOut:  result.TokensOut,
			CacheTTL:   int(cacheTTL / time.Second),
		})
	}
}

// Decide answers an open approval card of a chat.
func (r *Runner) Decide(chatID, approvalID, decision string) error {
	switch decision {
	case DecisionAllow, DecisionAllowSession, DecisionDeny:
	default:
		return ErrBadDecision
	}
	r.mu.Lock()
	ch, ok := r.approvals[chatID][approvalID]
	if ok {
		delete(r.approvals[chatID], approvalID)
	}
	r.mu.Unlock()
	if !ok {
		return ErrApprovalNotFound
	}
	ch <- decision
	return nil
}

// Waiting reports whether a chat has an approval card open.
func (r *Runner) Waiting(chatID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.approvals[chatID]) > 0
}

// approver returns the Approver for a chat's turn. A request that matches a
// grant is allowed at once. Any other request opens a card and waits for
// Decide, or for the turn to end.
func (r *Runner) approver(chatID string, t *turn) Approver {
	return func(ctx context.Context, req ApprovalRequest) string {
		if r.granted(chatID, req.Rule) {
			return DecisionAllow
		}
		approvalID := uuid.NewString()
		ch := make(chan string, 1)
		r.mu.Lock()
		if r.approvals[chatID] == nil {
			r.approvals[chatID] = map[string]chan string{}
		}
		r.approvals[chatID][approvalID] = ch
		r.mu.Unlock()
		r.settle(chatID, Event{Type: EventApprovalRequested, ApprovalID: approvalID, Name: req.Tool, Arg: req.Detail, Input: req.Input, ToolID: req.ToolID})

		var decision string
		select {
		case decision = <-ch:
		case <-ctx.Done():
			decision = DecisionCancelled
		}
		r.mu.Lock()
		delete(r.approvals[chatID], approvalID)
		if len(r.approvals[chatID]) == 0 {
			delete(r.approvals, chatID)
		}
		r.mu.Unlock()

		switch {
		case decision == DecisionAllowSession:
			r.grant(chatID, req.Rule)
		case (decision == DecisionDeny || decision == DecisionCancelled) && req.ToolID != "":
			t.denied.Store(req.ToolID, true)
		}
		r.settle(chatID, Event{Type: EventApprovalResolved, ApprovalID: approvalID, Decision: decision})
		return decision
	}
}

func (r *Runner) granted(chatID, rule string) bool {
	if rule == "" {
		return false
	}
	m, err := r.cfg.Store.Meta(chatID)
	return err == nil && slices.Contains(m.Grants, rule)
}

// grant adds a rule to the chat's grants, for the rest of the chat.
func (r *Runner) grant(chatID, rule string) {
	if rule == "" {
		return
	}
	m, err := r.cfg.Store.Meta(chatID)
	if err != nil || slices.Contains(m.Grants, rule) {
		return
	}
	m.Grants = append(m.Grants, rule)
	_ = r.cfg.Store.SaveMeta(m)
}

// tailBuffer keeps the last max bytes written to it.
type tailBuffer struct {
	mu  sync.Mutex
	max int
	buf []byte
}

func (b *tailBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.buf = append(b.buf, p...)
	if len(b.buf) > b.max {
		b.buf = b.buf[len(b.buf)-b.max:]
	}
	return len(p), nil
}

func (b *tailBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return string(b.buf)
}

// defaultCacheTTL is the prompt cache TTL when no agent reported one. It is
// the shorter of the two TTLs Claude writes.
const defaultCacheTTL = 5 * time.Minute

// turnCacheTTL returns the prompt cache TTL after a finished turn: the TTL
// the turn wrote, else the TTL the chat last wrote (a turn that only reads
// the cache refreshes it), else defaultCacheTTL.
func turnCacheTTL(wrote time.Duration, lastSeconds int) time.Duration {
	switch {
	case wrote > 0:
		return wrote
	case lastSeconds > 0:
		return time.Duration(lastSeconds) * time.Second
	}
	return defaultCacheTTL
}
