package chat

import (
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/palarix/exponential/internal/exponential"
)

var errShuttingDown = errors.New("the board is shutting down")

// liveSession is a chat's running agent. It outlives turns: it ends on
// Stop in the background banner, on a mode change, when the chat is
// deleted, when it is idle for IdleTimeout, or on Shutdown.
type liveSession struct {
	s   Session
	key string // launchKey of the profile it started with

	// Under r.mu.
	tasks      []BackgroundTask
	lastActive time.Time
	sessionID  string // reported by the agent, maybe before the first turn

	closeOnce sync.Once
}

// session returns the chat's live session, and opens one when there is
// none or the running one was started for another mode or worktree.
func (r *Runner) session(id string, meta Meta, driver Driver, profile Profile) (*liveSession, error) {
	key := launchKey(meta.Agent, profile)
	r.mu.Lock()
	cur := r.sessions[id]
	if cur != nil && cur.key == key {
		r.mu.Unlock()
		return cur, nil
	}
	if cur != nil {
		delete(r.sessions, id)
	}
	r.mu.Unlock()
	if cur != nil {
		r.closeSession(cur)
	}

	ls := &liveSession{key: key, lastActive: time.Now()}
	s, err := driver.Open(SessionInput{
		SessionID: meta.SessionID,
		Model:     r.model(meta),
		Effort:    meta.Effort,
		Dir:       profile.Dir,
		MCP:       r.cfg.MCP,
		Profile:   profile,
		Approve:   r.approver(id),
		Grants:    meta.Grants,
	}, &sessionHandler{r: r, id: id, ls: ls})
	if err != nil {
		return nil, err
	}
	started := time.Now().UTC()

	r.mu.Lock()
	ls.s = s
	if r.closing {
		r.mu.Unlock()
		r.closeSession(ls)
		return nil, errShuttingDown
	}
	r.sessions[id] = ls
	if m, err := r.cfg.Store.Meta(id); err == nil {
		m.AgentPID = s.PID()
		m.AgentStarted = &started
		_ = r.cfg.Store.SaveMeta(m)
	}
	r.mu.Unlock()
	return ls, nil
}

// detachLocked takes the chat's live session out of the runner after a
// mode change, and returns it for the caller to close. When it had
// background tasks, the notice says they stop. r.mu must be held.
func (r *Runner) detachLocked(id string, notice *Event) *liveSession {
	ls := r.sessions[id]
	if ls == nil {
		return nil
	}
	delete(r.sessions, id)
	if len(ls.tasks) > 0 {
		notice.Message = strings.TrimSuffix(modeChangeText(*notice), ".") + ". Background tasks stopped."
	}
	return ls
}

// modeChangeText is what the chat shows for a mode_changed event.
func modeChangeText(e Event) string {
	switch {
	case e.Message != "":
		return e.Message
	case e.Mode == ModeWork:
		return "Entered Work Mode on " + e.IssueID
	}
	return "Entered Chat Mode"
}

// closeAsync closes ls in the background. Shutdown waits for it.
func (r *Runner) closeAsync(ls *liveSession) {
	if ls == nil {
		return
	}
	done := make(chan struct{})
	r.mu.Lock()
	open := r.closers[:0]
	for _, c := range r.closers {
		select {
		case <-c: // done; forget it
		default:
			open = append(open, c)
		}
	}
	r.closers = append(open, done)
	r.mu.Unlock()
	go func() {
		defer close(done)
		r.closeSession(ls)
	}()
}

// closeSession stops the agent of ls and waits until it has exited. Its
// handler's Exited does the cleanup.
func (r *Runner) closeSession(ls *liveSession) {
	ls.closeOnce.Do(func() {
		if ls.s != nil {
			ls.s.Close()
		}
	})
}

// dropSession closes ls if it is still the chat's live session.
func (r *Runner) dropSession(id string, ls *liveSession) {
	r.mu.Lock()
	if r.sessions[id] == ls {
		delete(r.sessions, id)
	}
	r.mu.Unlock()
	r.closeSession(ls)
}

// StopBackground closes a chat's agent, with its background tasks. A
// running turn ends as stopped. It returns false when the chat has no live
// agent. The next message resumes the session from disk.
func (r *Runner) StopBackground(id string) bool {
	r.mu.Lock()
	ls := r.sessions[id]
	if ls == nil {
		r.mu.Unlock()
		return false
	}
	delete(r.sessions, id)
	t := r.turns[id]
	r.mu.Unlock()
	if t != nil && t.ls == ls {
		t.cancel()
	}
	r.closeSession(ls)
	return true
}

// Background returns a chat's live background tasks.
func (r *Runner) Background(id string) []BackgroundTask {
	r.mu.Lock()
	defer r.mu.Unlock()
	if ls := r.sessions[id]; ls != nil {
		return append([]BackgroundTask(nil), ls.tasks...)
	}
	return nil
}

// reap closes agents that have no turn and no background task, and saw no
// activity for IdleTimeout.
func (r *Runner) reap() {
	tick := time.NewTicker(r.cfg.ReapEvery)
	defer tick.Stop()
	for {
		select {
		case <-r.stopReap:
			return
		case now := <-tick.C:
			r.mu.Lock()
			var idle []*liveSession
			for id, ls := range r.sessions {
				if r.turns[id] == nil && len(ls.tasks) == 0 && now.Sub(ls.lastActive) >= r.cfg.IdleTimeout {
					delete(r.sessions, id)
					idle = append(idle, ls)
				}
			}
			r.mu.Unlock()
			for _, ls := range idle {
				r.closeAsync(ls)
			}
		}
	}
}

// sessionHandler routes a live session's output to the chat.
type sessionHandler struct {
	r  *Runner
	id string
	ls *liveSession
}

// current returns the turn this session runs, or nil. r.mu must be held.
func (h *sessionHandler) current() *turn {
	t := h.r.turns[h.id]
	if t == nil || t.ls != h.ls {
		return nil
	}
	return t
}

func (h *sessionHandler) AgentTurn(reason string) {
	r := h.r
	r.mu.Lock()
	if r.closing || r.sessions[h.id] != h.ls {
		r.mu.Unlock()
		return
	}
	if t := r.turns[h.id]; t != nil {
		// A user turn is registered: the turn is that one.
		if t.ls == nil {
			t.ls = h.ls
		}
		r.mu.Unlock()
		return
	}
	m, err := r.cfg.Store.Meta(h.id)
	if err != nil {
		r.mu.Unlock()
		return
	}
	t := r.newTurnLocked(h.id)
	t.byAgent = true
	t.ls = h.ls
	t.meta = m
	t.sessionID = m.SessionID
	t.modes = &modeTools{profile: r.profileFor(m)}
	h.ls.lastActive = time.Now()
	m.State = StateRunning
	m.LastError = ""
	m.UpdatedAt = time.Now().UTC()
	_ = r.cfg.Store.SaveMeta(m)
	r.mu.Unlock()
	r.settle(h.id, Event{Type: EventAgentTurn, Text: reason})
}

func (h *sessionHandler) Stream(ev exponential.StreamEvent) {
	r := h.r
	r.mu.Lock()
	if ev.Kind == exponential.StreamSession {
		h.ls.sessionID = ev.Text
	}
	t := h.current()
	r.mu.Unlock()
	if t != nil {
		r.stream(h.id, t, ev)
	}
}

func (h *sessionHandler) TurnEnded(res *TurnResult, err error) {
	r := h.r
	r.mu.Lock()
	t := h.current()
	r.mu.Unlock()
	if t != nil {
		r.endTurn(h.id, t, res, err)
	}
}

func (h *sessionHandler) Tasks(tasks []BackgroundTask) {
	r := h.r
	r.mu.Lock()
	h.ls.tasks = tasks
	h.ls.lastActive = time.Now()
	// A closed session's last report still reaches the chat, unless a
	// new session took its place.
	show := r.sessions[h.id] == h.ls || r.sessions[h.id] == nil
	r.mu.Unlock()
	if show {
		r.live(h.id, Event{Type: EventBackgroundTasks, Tasks: tasks})
	}
}

func (h *sessionHandler) Exited(err error) {
	r := h.r
	r.mu.Lock()
	if r.sessions[h.id] == h.ls {
		delete(r.sessions, h.id)
	}
	show := r.sessions[h.id] == nil
	t := h.current()
	hadTasks := len(h.ls.tasks) > 0
	h.ls.tasks = nil
	if m, merr := r.cfg.Store.Meta(h.id); merr == nil && h.ls.s != nil && m.AgentPID == h.ls.s.PID() {
		m.AgentPID = 0
		m.AgentStarted = nil
		_ = r.cfg.Store.SaveMeta(m)
	}
	r.mu.Unlock()
	if t != nil {
		r.endTurn(h.id, t, nil, err)
	}
	if hadTasks && show {
		r.live(h.id, Event{Type: EventBackgroundTasks})
	}
}
