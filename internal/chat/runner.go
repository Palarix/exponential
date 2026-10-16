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

	// Titlers name chats after their first finished turn, by provider ID. A
	// provider with no titler keeps the placeholder title.
	Titlers map[string]Titler
	// TitleModel returns the model a provider names chats with; "" means
	// the chat's own model. Optional.
	TitleModel func(provider string) string
	// IssueTitle returns an issue's title, for the title hint of a work
	// chat; "" when unknown. Optional.
	IssueTitle func(issueID string) string

	// CanWork allows Work Mode. False in remote mode, where start and merge
	// run on the server (xpo-a60065).
	CanWork bool
	// ResolveWork returns an issue's full ID and worktree, once the agent's
	// start call on it succeeded. The ID may be a unique suffix.
	ResolveWork func(issueID string) (WorkBinding, error)

	// InterruptGrace is how long a stopped turn may take to end after the
	// agent was asked to stop. Then the runner closes the agent. 3s when
	// zero.
	InterruptGrace time.Duration
	// IdleTimeout closes an agent that has no turn and no background task
	// after this long without activity. 30 minutes when zero.
	IdleTimeout time.Duration
	// ReapEvery is how often the runner looks for idle agents. 5 minutes
	// when zero.
	ReapEvery time.Duration

	// OnEvent gets every event published for a chat, settled (with Seq) and
	// live, in order. It is called under the runner's lock, so it must not
	// block or call back into the runner. Optional.
	OnEvent func(chatID string, e Event)
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

	// Attachments are files the user pasted or dropped. A message may have
	// attachments and no text.
	Attachments []AttachmentData
}

// Runner runs chat turns and hands their events to RunnerConfig.OnEvent. Each
// chat keeps one agent process across turns (a live session), so background
// tasks the agent starts outlive the turn that started them.
type Runner struct {
	cfg RunnerConfig

	mu        sync.Mutex
	turns     map[string]*turn
	sessions  map[string]*liveSession
	approvals map[string]map[string]openCard // open approval cards: chat ID → approval ID → card
	closing   bool                           // Shutdown began: no turn continues on its own
	wg        sync.WaitGroup                 // running turns
	closers   []chan struct{}                // sessions being closed; each closes when done
	titles    sync.WaitGroup                 // running title calls
	titleCtx  context.Context                // ends title calls on Shutdown
	endTitles context.CancelFunc
	stopReap  chan struct{}
	stopOnce  sync.Once
}

// turn is one running turn. Its output fields are used by one goroutine at
// a time: the session's reader, or the goroutine that ends the turn.
type turn struct {
	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{} // closed when the turn has finished

	byAgent bool         // the agent started the turn on its own
	ls      *liveSession // the session that runs the turn; set under r.mu
	ended   bool         // endTurn ran; under r.mu

	// denied holds the tool use IDs whose approval the user denied. The
	// agents report such a call as an error; the work list shows it denied.
	denied sync.Map

	meta      Meta // the chat when the turn began
	sessionID string
	modes     *modeTools
	text      strings.Builder
	// A tool call settles when its result arrives. pending holds the calls
	// whose input is complete but whose result has not arrived yet.
	pending      map[string]Event
	pendingOrder []string
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
	if cfg.InterruptGrace == 0 {
		cfg.InterruptGrace = 3 * time.Second
	}
	if cfg.IdleTimeout == 0 {
		cfg.IdleTimeout = 30 * time.Minute
	}
	if cfg.ReapEvery == 0 {
		cfg.ReapEvery = 5 * time.Minute
	}
	r := &Runner{
		cfg:       cfg,
		turns:     map[string]*turn{},
		sessions:  map[string]*liveSession{},
		approvals: map[string]map[string]openCard{},
		stopReap:  make(chan struct{}),
	}
	r.titleCtx, r.endTitles = context.WithCancel(context.Background())
	go r.reap()
	return r
}

// Send records the user message and starts a turn in the background.
func (r *Runner) Send(id string, req SendRequest) error {
	meta, err := r.cfg.Store.Meta(id)
	if err != nil {
		return err
	}
	if meta.ArchivedAt != nil {
		return ErrArchived
	}
	if err := ValidateAttachments(req.Attachments); err != nil {
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
	t := r.newTurnLocked(id)
	r.mu.Unlock()

	// Read the meta again: BindWork may have changed it before the turn was
	// registered, and no mode change can happen from here on. Archive may
	// have too: it refuses only once the turn is registered.
	if meta, err = r.cfg.Store.Meta(id); err != nil {
		r.finish(id, t)
		return err
	}
	if meta.ArchivedAt != nil {
		r.finish(id, t)
		return ErrArchived
	}
	notice := checkWorktree(&meta)
	if meta.Title == "" {
		meta.Title = TitleFrom(req.Text)
		if meta.Title == "" && len(req.Attachments) > 0 {
			meta.Title = TitleFrom(req.Attachments[0].Name)
		}
		meta.AutoTitle = true
	}
	attachments, err := r.cfg.Store.SaveAttachments(id, req.Attachments)
	if err != nil {
		r.finish(id, t)
		return err
	}
	prompt, err := r.promptFiles(id, attachments)
	if err != nil {
		r.finish(id, t)
		return err
	}
	if meta.Agent != provider {
		meta.SessionID = "" // a session belongs to the agent that made it
	}
	meta.Agent = provider
	meta.Model = req.Model
	meta.Effort = req.Effort
	meta.State = StateRunning
	meta.LastError = ""
	meta.UpdatedAt = time.Now().UTC()
	if err := r.cfg.Store.SaveMeta(meta); err != nil {
		r.finish(id, t)
		return err
	}
	if notice != nil {
		r.mu.Lock()
		old := r.detachLocked(id, notice)
		r.mu.Unlock()
		r.closeAsync(old)
		r.settle(id, *notice)
	}
	r.settle(id, Event{Type: EventUserMessage, Text: req.Text, Context: req.Context, Attachments: attachments})

	profile := r.profileFor(meta)
	t.meta = meta
	t.sessionID = meta.SessionID
	t.modes = &modeTools{profile: profile}
	prompt.Text = buildPrompt(req.Text, req.Context, profile, prompt.files)
	go r.start(id, driver, profile, prompt.Prompt, t)
	return nil
}

// turnPrompt is a Prompt with the files its text lists.
type turnPrompt struct {
	Prompt
	files []promptFile
}

// promptFiles resolves a message's saved attachments: every one is listed
// in the prompt, and images also go in as image input.
func (r *Runner) promptFiles(id string, list []Attachment) (turnPrompt, error) {
	var p turnPrompt
	paths, err := r.cfg.Store.attachmentPaths(id, list)
	if err != nil {
		return p, err
	}
	for i, a := range list {
		name := a.Name
		if name == "" {
			name = "attachment"
		}
		p.files = append(p.files, promptFile{Name: name, Path: paths[i], Image: a.Image()})
		if a.Image() {
			p.Images = append(p.Images, PromptImage{Path: paths[i], MIME: a.MIME})
		}
	}
	return p, nil
}

// newTurnLocked registers a turn for a chat. r.mu must be held.
func (r *Runner) newTurnLocked(id string) *turn {
	ctx, cancel := context.WithCancel(context.Background())
	t := &turn{ctx: ctx, cancel: cancel, done: make(chan struct{}), pending: map[string]Event{}}
	r.turns[id] = t
	r.wg.Add(1)
	return t
}

// model is the model a turn runs: the chat's pick, else chat.model for the
// default provider.
func (r *Runner) model(m Meta) string {
	if m.Model == "" && m.Agent == r.cfg.DefaultProvider {
		return r.cfg.DefaultModel
	}
	return m.Model
}

// start sends turn t's prompt to the chat's live session, and opens one
// first when there is none.
func (r *Runner) start(id string, driver Driver, profile Profile, prompt Prompt, t *turn) {
	for attempt := 0; ; attempt++ {
		if t.ctx.Err() != nil {
			r.endTurn(id, t, nil, nil)
			return
		}
		ls, err := r.session(id, t.meta, driver, profile)
		if err != nil {
			r.endTurn(id, t, nil, err)
			return
		}
		r.mu.Lock()
		t.ls = ls
		if ls.sessionID != "" {
			t.sessionID = ls.sessionID
		}
		ls.lastActive = time.Now()
		r.mu.Unlock()
		if t.ctx.Err() != nil {
			r.endTurn(id, t, nil, nil)
			return
		}
		err = ls.s.Send(prompt, r.model(t.meta), t.meta.Effort)
		if errors.Is(err, errAgentExited) && attempt == 0 {
			// The agent exited since the last turn: start it once more.
			r.dropSession(id, ls)
			continue
		}
		if err != nil {
			r.endTurn(id, t, nil, err)
		}
		return
	}
}

// continuePrompt is the message of the turn the board starts after an
// approved start put the chat in Work Mode.
func continuePrompt(issueID string) string {
	return "Work Mode is on for " + issueID + ". Continue with what the user asked for."
}

// Running reports whether a chat has a turn in progress.
func (r *Runner) Running(id string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.turns[id]
	return ok
}

// Cancel stops a chat's running turn. It returns false when none is running.
// The agent is asked to stop the turn; its background tasks keep running.
// An agent that does not stop within InterruptGrace is closed.
func (r *Runner) Cancel(id string) bool {
	r.mu.Lock()
	t, ok := r.turns[id]
	var ls *liveSession
	if ok {
		ls = t.ls
	}
	r.mu.Unlock()
	if !ok {
		return false
	}
	t.cancel()
	if ls != nil {
		ls.s.Interrupt()
		go func() {
			select {
			case <-t.done:
			case <-time.After(r.cfg.InterruptGrace):
				r.dropSession(id, ls)
			}
		}()
	}
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

// Wait blocks until no turn and no title call is running.
func (r *Runner) Wait() {
	r.wg.Wait()
	r.titles.Wait()
}

// Shutdown stops every running turn and every agent, and waits for them to
// end.
func (r *Runner) Shutdown() {
	r.stopOnce.Do(func() { close(r.stopReap) })
	r.endTitles()
	r.mu.Lock()
	r.closing = true
	var turns []*turn
	for _, t := range r.turns {
		turns = append(turns, t)
	}
	var sessions []*liveSession
	for id, ls := range r.sessions {
		sessions = append(sessions, ls)
		delete(r.sessions, id)
	}
	r.mu.Unlock()
	for _, t := range turns {
		t.cancel()
	}
	for _, ls := range sessions {
		r.closeAsync(ls)
	}
	// Closing a session can end a turn that detaches another one, so wait
	// until no closer is left.
	for {
		r.mu.Lock()
		closers := r.closers
		r.closers = nil
		r.mu.Unlock()
		if len(closers) == 0 {
			break
		}
		for _, done := range closers {
			<-done
		}
	}
	r.Wait()
}

// publish hands e to OnEvent. The lock keeps the events of one chat in the
// order they were published, across the goroutines that publish them.
func (r *Runner) publish(id string, e Event) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cfg.OnEvent != nil {
		r.cfg.OnEvent(id, e)
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

// stream applies one progress event of the agent to turn t.
func (r *Runner) stream(id string, t *turn, ev exponential.StreamEvent) {
	switch ev.Kind {
	case exponential.StreamSession:
		t.sessionID = ev.Text
	case exponential.StreamText:
		t.text.WriteString(ev.Text)
		r.live(id, Event{Type: EventTextDelta, Text: ev.Text})
	case exponential.StreamThinking:
		r.live(id, Event{Type: EventThinking})
	case exponential.StreamToolStart:
		r.flushText(id, t)
		r.live(id, Event{Type: EventToolStart, Name: ev.Text, Arg: ev.ToolArg, ToolID: ev.ToolID})
	case exponential.StreamToolDone:
		if ev.Text == "" {
			r.flushText(id, t) // end of a text or thinking block
			return
		}
		call := Event{Type: EventToolCall, Name: ev.Text, Arg: ev.ToolArg, ToolID: ev.ToolID, Status: exponential.ToolStatusDone}
		if ev.ToolID == "" {
			r.settle(id, call) // no ID to match a result to
			return
		}
		t.pending[ev.ToolID] = call
		t.pendingOrder = append(t.pendingOrder, ev.ToolID)
	case exponential.StreamToolResult:
		call, ok := t.pending[ev.ToolID]
		if !ok {
			return
		}
		delete(t.pending, ev.ToolID)
		call.Status = ev.ToolStatus
		if _, denied := t.denied.Load(ev.ToolID); denied && call.Status == exponential.ToolStatusError {
			call.Status = exponential.ToolStatusDenied
		}
		call.Message = ev.ToolMessage
		r.settle(id, call)
		t.modes.observe(call)
	}
}

func (r *Runner) flushText(id string, t *turn) {
	if t.text.Len() > 0 {
		r.settle(id, Event{Type: EventAssistantText, Text: t.text.String()})
		t.text.Reset()
	}
}

// settlePending settles the tool calls that got no result before the turn
// ended.
func (r *Runner) settlePending(id string, t *turn) {
	for _, toolID := range t.pendingOrder {
		if call, ok := t.pending[toolID]; ok {
			r.settle(id, call)
		}
	}
	clear(t.pending)
	t.pendingOrder = nil
}

// endTurn settles how turn t ended and updates the chat. A nil res means
// the agent ended without a result; err says why. It runs once per turn.
func (r *Runner) endTurn(id string, t *turn, res *TurnResult, err error) {
	r.mu.Lock()
	if t.ended {
		r.mu.Unlock()
		return
	}
	t.ended = true
	r.mu.Unlock()
	defer r.finish(id, t)

	r.flushText(id, t)
	r.settlePending(id, t)
	if res != nil && res.SessionID != "" {
		t.sessionID = res.SessionID
	}

	var resolved *TurnResult   // set when the turn finished
	var cacheTTL time.Duration // the prompt cache TTL of a finished turn
	var cacheEnd time.Time     // when that turn finished
	var failure string         // the error message of a failed turn
	var done *Event            // the turn_done event of a finished turn
	switch {
	case t.ctx.Err() != nil:
		r.settle(id, Event{Type: EventTurnStopped})
	case res == nil:
		msg := "the agent exited without a reply"
		if err != nil {
			msg = err.Error()
		}
		failure = msg
		r.settle(id, Event{Type: EventError, Message: msg})
	case res.IsError:
		failure = res.Output
		r.settle(id, Event{Type: EventError, Message: res.Output})
	default:
		resolved = res
		cacheTTL = turnCacheTTL(res.CacheTTL, t.meta.CacheTTL)
		cacheEnd = time.Now()
		if r.cfg.Learner != nil {
			r.cfg.Learner.Learn(t.meta.Agent, t.meta.Model, res.Model, res.ContextWindow)
		}
		done = &Event{
			Type:       EventTurnDone,
			DurationMS: res.Duration.Milliseconds(),
			CostUSD:    res.CostUSD,
			TokensIn:   res.TokensIn,
			TokensOut:  res.TokensOut,
			CacheTTL:   int(cacheTTL / time.Second),
		}
	}
	if done != nil {
		// Backfill the totals before turn_done is logged, else the
		// backfill counts this turn and so does the update below.
		_, _ = r.cfg.Store.Meta(id)
		r.settle(id, *done)
	}

	r.mu.Lock()
	if t.ls != nil {
		t.ls.lastActive = time.Now()
	}
	m, merr := r.cfg.Store.Meta(id)
	var notice *Event
	var next *turn
	var old *liveSession
	var titled *Meta // the chat to name, after its first finished turn
	if merr == nil {
		m.State = StateIdle
		m.SessionID = t.sessionID
		m.LastError = failure
		if done != nil && m.Usage != nil {
			m.Usage.add(*done)
		}
		if resolved != nil && resolved.Model != "" {
			m.ResolvedModel = resolved.Model
			m.ContextWindow = resolved.ContextWindow
		}
		if cacheTTL > 0 {
			expires := cacheEnd.Add(cacheTTL).UTC()
			m.CacheTTL = int(cacheTTL / time.Second)
			m.CacheExpiresAt = &expires
		}
		notice = t.modes.apply(r, &m)
		if notice != nil {
			old = r.detachLocked(id, notice)
		}
		// An approved start put the chat in Work Mode: continue there,
		// unless the turn was stopped or failed.
		if notice != nil && notice.Mode == ModeWork && resolved != nil && t.ctx.Err() == nil && !r.closing {
			next = r.handOverLocked(id)
			m.State = StateRunning
		}
		if resolved != nil && m.AutoTitle && r.cfg.Titlers[m.Agent] != nil && !r.closing {
			m.AutoTitle = false // one attempt per chat
			titled = &m
			r.titles.Add(1)
		}
		m.UpdatedAt = time.Now().UTC()
		_ = r.cfg.Store.SaveMeta(m)
	}
	r.mu.Unlock()
	if titled != nil {
		go r.generateTitle(id, *titled, titled.Title)
	}
	r.closeAsync(old)
	if notice != nil {
		r.settle(id, *notice)
	}
	if next != nil {
		r.settle(id, Event{Type: EventTurnContinued, IssueID: m.Work.IssueID})
		p := r.profileFor(m)
		next.meta = m
		next.sessionID = m.SessionID
		next.modes = &modeTools{profile: p}
		go r.start(id, r.cfg.Drivers[m.Agent], p, Prompt{Text: buildPrompt(continuePrompt(m.Work.IssueID), nil, p, nil)}, next)
	}
}

// handOverLocked registers the next turn of a chat in place of t, so no
// message can start a turn in between. r.mu must be held.
func (r *Runner) handOverLocked(id string) *turn {
	ctx, cancel := context.WithCancel(context.Background())
	next := &turn{ctx: ctx, cancel: cancel, done: make(chan struct{}), pending: map[string]Event{}}
	r.turns[id] = next
	r.wg.Add(1)
	return next
}

// openCard is an approval or question card that waits for the user.
type openCard struct {
	reply    chan Verdict
	question bool // an AskUserQuestion card
}

// Decide answers an open card of a chat: allow, allow_session or deny on an
// approval card, deny (skip) on a question card.
func (r *Runner) Decide(chatID, approvalID, decision string) error {
	switch decision {
	case DecisionAllow, DecisionAllowSession, DecisionDeny:
	default:
		return ErrBadDecision
	}
	return r.reply(chatID, approvalID, Verdict{Decision: decision})
}

// Answer answers an open question card with an answer to each question,
// keyed by question text.
func (r *Runner) Answer(chatID, approvalID string, answers map[string]string) error {
	if len(answers) == 0 {
		return ErrBadDecision
	}
	for _, a := range answers {
		if strings.TrimSpace(a) == "" {
			return ErrBadDecision
		}
	}
	return r.reply(chatID, approvalID, Verdict{Decision: DecisionAnswer, Answers: answers})
}

// reply sends v to an open card, if the card takes it. A question card
// takes an answer or a skip: allowing it would answer nothing.
func (r *Runner) reply(chatID, approvalID string, v Verdict) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	card, ok := r.approvals[chatID][approvalID]
	if !ok {
		return ErrApprovalNotFound
	}
	takes := v.Decision == DecisionDeny || card.question == (v.Decision == DecisionAnswer)
	if !takes {
		return ErrBadDecision
	}
	delete(r.approvals[chatID], approvalID)
	card.reply <- v // buffered: never blocks
	return nil
}

// Waiting reports whether a chat has an approval or question card open.
func (r *Runner) Waiting(chatID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.approvals[chatID]) > 0
}

// approver returns the Approver for a chat's session. A request that
// matches a grant is allowed at once; a question is never granted. Any other request opens a card and
// waits for Decide, for the turn to be stopped, or for the agent to end
// the request.
func (r *Runner) approver(chatID string) Approver {
	return func(ctx context.Context, req ApprovalRequest) Verdict {
		question := req.Tool == ToolAskUserQuestion
		if !question && r.granted(chatID, req.Rule) {
			return Verdict{Decision: DecisionAllow}
		}
		approvalID := uuid.NewString()
		ch := make(chan Verdict, 1)
		r.mu.Lock()
		t := r.turns[chatID]
		if r.approvals[chatID] == nil {
			r.approvals[chatID] = map[string]openCard{}
		}
		r.approvals[chatID][approvalID] = openCard{reply: ch, question: question}
		r.mu.Unlock()
		var stopped <-chan struct{}
		if t != nil {
			stopped = t.ctx.Done()
		}
		r.settle(chatID, Event{Type: EventApprovalRequested, ApprovalID: approvalID, Name: req.Tool, Arg: req.Detail, Input: req.Input, ToolID: req.ToolID})

		var v Verdict
		select {
		case v = <-ch:
		case <-ctx.Done():
			v.Decision = DecisionCancelled
		case <-stopped:
			v.Decision = DecisionCancelled
		}
		decision := v.Decision
		r.mu.Lock()
		delete(r.approvals[chatID], approvalID)
		if len(r.approvals[chatID]) == 0 {
			delete(r.approvals, chatID)
		}
		r.mu.Unlock()

		switch {
		case decision == DecisionAllowSession:
			r.grant(chatID, req.Rule)
		case (decision == DecisionDeny || decision == DecisionCancelled) && req.ToolID != "" && t != nil:
			t.denied.Store(req.ToolID, true)
		}
		r.settle(chatID, Event{Type: EventApprovalResolved, ApprovalID: approvalID, Decision: decision, Answers: v.Answers})
		return v
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
