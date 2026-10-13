package chat

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

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

	mu    sync.Mutex
	turns map[string]*turn
	subs  map[string]map[chan Event]struct{}
	wg    sync.WaitGroup
}

type turn struct {
	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{} // closed when the turn has finished
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
	return &Runner{cfg: cfg, turns: map[string]*turn{}, subs: map[string]map[chan Event]struct{}{}}
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
		r.finish(id)
		return err
	}
	r.settle(id, Event{Type: EventUserMessage, Text: req.Text, Context: req.Context})

	go r.run(id, meta, driver, buildPrompt(req.Text, req.Context), t)
	return nil
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

func (r *Runner) finish(id string) {
	r.mu.Lock()
	if t, ok := r.turns[id]; ok {
		t.cancel()
		close(t.done)
		delete(r.turns, id)
	}
	r.mu.Unlock()
	r.wg.Done()
}

func (r *Runner) run(id string, meta Meta, driver Driver, prompt string, t *turn) {
	defer r.finish(id)

	sessionID := meta.SessionID
	var resolved *TurnResult // set when the turn finished
	defer func() {
		if m, err := r.cfg.Store.Meta(id); err == nil {
			m.State = StateIdle
			m.SessionID = sessionID
			if resolved != nil && resolved.Model != "" {
				m.ResolvedModel = resolved.Model
				m.ContextWindow = resolved.ContextWindow
			}
			m.UpdatedAt = time.Now().UTC()
			_ = r.cfg.Store.SaveMeta(m)
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
		Dir:       r.cfg.Dir,
		MCP:       r.cfg.MCP,
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
			call.Message = ev.ToolMessage
			r.settle(id, call)
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
		if r.cfg.Learner != nil {
			r.cfg.Learner.Learn(meta.Agent, meta.Model, result.Model, result.ContextWindow)
		}
		r.settle(id, Event{
			Type:       EventTurnDone,
			DurationMS: result.Duration.Milliseconds(),
			CostUSD:    result.CostUSD,
			TokensIn:   result.TokensIn,
			TokensOut:  result.TokensOut,
		})
	}
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
