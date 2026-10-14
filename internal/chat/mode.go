package chat

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/palarix/exponential/internal/exponential"
)

// ErrWorkUnavailable is returned for Work Mode in remote mode, where start
// and merge run on the server (xpo-a60065).
var ErrWorkUnavailable = errors.New("work mode needs a local board")

// ErrIssueHeld is returned when another chat is in Work Mode on the issue.
var ErrIssueHeld = errors.New("another chat is in work mode on this issue")

// ErrNotInWork is returned by LeaveWork for a chat in Chat Mode.
var ErrNotInWork = errors.New("this chat is not in work mode")

// ErrWorkingElsewhere refuses to bind a chat that is in Work Mode on
// another issue.
var ErrWorkingElsewhere = errors.New("this chat is in work mode on another issue")

// CanWork reports whether chats may enter Work Mode.
func (r *Runner) CanWork() bool { return r.cfg.CanWork }

// profileFor builds the turn's profile from the chat's mode.
func (r *Runner) profileFor(m Meta) Profile {
	if m.Mode == ModeWork && m.Work != nil {
		return WorkProfile(m.Work.IssueID, m.Work.Worktree, r.cfg.Dir)
	}
	return ChatProfile(r.cfg.Dir, r.cfg.CanWork)
}

// BindWork puts a chat in Work Mode on an issue. It refuses while a turn
// runs, in remote mode, and when another chat holds the issue.
func (r *Runner) BindWork(chatID string, b WorkBinding) (Meta, error) {
	if !r.cfg.CanWork {
		return Meta{}, ErrWorkUnavailable
	}
	r.mu.Lock()
	if _, busy := r.turns[chatID]; busy {
		r.mu.Unlock()
		return Meta{}, ErrBusy
	}
	m, err := r.cfg.Store.Meta(chatID)
	if err != nil {
		r.mu.Unlock()
		return Meta{}, err
	}
	notice, err := r.bindLocked(&m, b)
	if err == nil {
		err = r.cfg.Store.SaveMeta(m)
	}
	r.mu.Unlock()
	if err != nil {
		return Meta{}, err
	}
	if notice != nil {
		r.settle(chatID, *notice)
	}
	return m, nil
}

// bindLocked sets the Work Mode fields of m and returns the mode_changed
// event to settle, or nil when m was already bound to the issue. r.mu must
// be held, so two chats cannot take the same issue at once.
func (r *Runner) bindLocked(m *Meta, b WorkBinding) (*Event, error) {
	if m.Mode == ModeWork && m.Work != nil {
		if m.Work.IssueID == b.IssueID {
			return nil, nil
		}
		return nil, ErrWorkingElsewhere
	}
	if holder, ok := r.WorkChat(b.IssueID); ok && holder.ID != m.ID {
		return nil, ErrIssueHeld
	}
	m.Mode = ModeWork
	m.Work = &b
	if !slices.Contains(m.WorkedIssues, b.IssueID) {
		m.WorkedIssues = append(m.WorkedIssues, b.IssueID)
	}
	m.UpdatedAt = time.Now().UTC()
	return &Event{Type: EventModeChanged, Mode: ModeWork, IssueID: b.IssueID}, nil
}

// LeaveWork puts a chat back in Chat Mode without a merge. The issue stays
// DOING and its worktree stays.
func (r *Runner) LeaveWork(chatID string) (Meta, error) {
	r.mu.Lock()
	if _, busy := r.turns[chatID]; busy {
		r.mu.Unlock()
		return Meta{}, ErrBusy
	}
	m, err := r.cfg.Store.Meta(chatID)
	if err == nil && m.Mode != ModeWork {
		err = ErrNotInWork
	}
	var notice Event
	if err == nil {
		notice = leaveWork(&m, "")
		err = r.cfg.Store.SaveMeta(m)
	}
	r.mu.Unlock()
	if err != nil {
		return Meta{}, err
	}
	r.settle(chatID, notice)
	return m, nil
}

// leaveWork clears the Work Mode fields of m and returns the mode_changed
// event, with reason as its message.
func leaveWork(m *Meta, reason string) Event {
	issue := ""
	if m.Work != nil {
		issue = m.Work.IssueID
	}
	m.Mode = ModeChat
	m.Work = nil
	m.UpdatedAt = time.Now().UTC()
	return Event{Type: EventModeChanged, Mode: ModeChat, IssueID: issue, Message: reason}
}

// WorkChat returns the chat in Work Mode on an issue.
func (r *Runner) WorkChat(issueID string) (Meta, bool) {
	list, err := r.cfg.Store.List()
	if err != nil {
		return Meta{}, false
	}
	for _, m := range list {
		if m.Mode == ModeWork && m.Work != nil && m.Work.IssueID == issueID {
			return m, true
		}
	}
	return Meta{}, false
}

// checkWorktree returns m to Chat Mode when its worktree is gone, for
// example after a merge from the CLI. It returns the mode_changed event to
// settle, or nil when nothing changed.
func checkWorktree(m *Meta) *Event {
	if m.Mode != ModeWork {
		return nil
	}
	if m.Work != nil {
		if info, err := os.Stat(m.Work.Worktree); err == nil && info.IsDir() {
			return nil
		}
	}
	e := leaveWork(m, "")
	e.Message = "The worktree of " + e.IssueID + " is gone, so the chat is back in Chat Mode."
	return &e
}

// modeTools watches a turn's tool calls for the ones that change the mode
// after the turn: a successful start in Chat Mode, a successful merge of the
// bound issue in Work Mode.
type modeTools struct {
	profile Profile
	started string // the issue ID start was called with
	merged  bool
}

func (w *modeTools) observe(call Event) {
	if call.Status != exponential.ToolStatusDone {
		return
	}
	switch {
	case call.Name == "mcp__xpo__start" && !w.profile.work() && w.profile.CanWork:
		w.started = call.Arg
	case call.Name == "mcp__xpo__merge" && w.profile.work() && sameIssue(w.profile.IssueID, call.Arg):
		w.merged = true
	}
}

// sameIssue reports whether ref names the issue id, in full or as a unique
// suffix as the xpo tools accept it.
func sameIssue(id, ref string) bool {
	return ref != "" && strings.HasSuffix(id, ref)
}

// apply changes m's mode after the turn and returns the mode_changed event
// to settle, or nil. r.mu must be held.
func (w *modeTools) apply(r *Runner, m *Meta) *Event {
	switch {
	case w.merged:
		e := leaveWork(m, "")
		return &e
	case w.started != "":
		if r.cfg.ResolveWork == nil {
			return nil
		}
		b, err := r.cfg.ResolveWork(w.started)
		if err == nil {
			var notice *Event
			if notice, err = r.bindLocked(m, b); err == nil {
				return notice
			}
		}
		return &Event{Type: EventModeChanged, Mode: ModeChat, IssueID: w.started, Message: fmt.Sprintf("Could not enter Work Mode on %s: %v.", w.started, err)}
	}
	return nil
}
