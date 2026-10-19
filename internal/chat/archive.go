package chat

import (
	"errors"
	"time"
)

// ErrArchived is returned for a message to an archived chat, which is read
// only until it is unarchived.
var ErrArchived = errors.New("this chat is archived; unarchive it to continue")

// ErrArchiveInWork refuses to archive a chat in Work Mode.
var ErrArchiveInWork = errors.New("this chat is in work mode; leave work mode before archiving")

// Archive takes a chat out of the active chat lists and makes it read only.
// It refuses while a turn runs and while the chat is in Work Mode, and
// stops the chat's agent, so no background task starts a turn.
func (r *Runner) Archive(id string) (Meta, error) {
	r.mu.Lock()
	if _, busy := r.turns[id]; busy {
		r.mu.Unlock()
		return Meta{}, ErrBusy
	}
	m, err := r.cfg.Store.Meta(id)
	switch {
	case err != nil:
	case m.Mode == ModeWork:
		err = ErrArchiveInWork
	case m.ArchivedAt == nil:
		now := time.Now().UTC()
		m.ArchivedAt = &now
		err = r.cfg.Store.SaveMeta(m)
	}
	r.mu.Unlock()
	if err != nil {
		return Meta{}, err
	}
	r.StopBackground(id)
	r.ClearQueue(id)
	return m, nil
}

// Unarchive puts an archived chat back in the active chat lists.
func (r *Runner) Unarchive(id string) (Meta, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	m, err := r.cfg.Store.Meta(id)
	if err != nil {
		return Meta{}, err
	}
	if m.ArchivedAt == nil {
		return m, nil
	}
	m.ArchivedAt = nil
	return m, r.cfg.Store.SaveMeta(m)
}
