package chat

import (
	"errors"
	"log"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ErrQueuedNotFound is returned for a queued message that is no longer in
// the queue: it was sent, steered or removed.
var ErrQueuedNotFound = errors.New("the message is no longer queued")

// QueuedMessage is a message sent while a turn ran. The chat's queue goes
// out as one message when the turn ends with turn_done. The queue lives in
// the runner's memory: every client of the chat sees it, and a board
// restart drops it.
type QueuedMessage struct {
	ID          string       `json:"id"`
	Text        string       `json:"text,omitempty"`
	Context     *Context     `json:"context,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"` // saved when queued
	Model       string       `json:"model,omitempty"`
	Effort      string       `json:"effort,omitempty"`
	Time        time.Time    `json:"time"`
}

// enqueue saves req's attachments and adds it to the chat's queue. When the
// turn has ended meanwhile, it sends req instead.
func (r *Runner) enqueue(id string, req SendRequest, provider string, driver Driver) (bool, error) {
	saved, err := r.cfg.Store.SaveAttachments(id, req.Attachments)
	if err != nil {
		return false, err
	}
	req.Attachments = nil
	r.mu.Lock()
	if _, busy := r.turns[id]; !busy {
		t := r.newTurnLocked(id)
		r.mu.Unlock()
		return false, r.begin(id, t, req, provider, driver, saved)
	}
	r.queues[id] = append(r.queues[id], QueuedMessage{
		ID:          uuid.NewString(),
		Text:        req.Text,
		Context:     req.Context,
		Attachments: saved,
		Model:       req.Model,
		Effort:      req.Effort,
		Time:        time.Now().UTC(),
	})
	list := slices.Clone(r.queues[id])
	r.mu.Unlock()
	r.live(id, Event{Type: EventQueue, Queue: list})
	return true, nil
}

// Queued returns the chat's queued messages, oldest first.
func (r *Runner) Queued(id string) []QueuedMessage {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.queues[id])
}

// RemoveQueued takes a message out of the chat's queue and returns it, for
// example to edit it in a composer.
func (r *Runner) RemoveQueued(id, qid string) (QueuedMessage, error) {
	r.mu.Lock()
	m, _, ok := r.takeQueuedLocked(id, qid)
	list := slices.Clone(r.queues[id])
	r.mu.Unlock()
	if !ok {
		return QueuedMessage{}, ErrQueuedNotFound
	}
	r.live(id, Event{Type: EventQueue, Queue: list})
	return m, nil
}

// ClearQueue drops the chat's queued messages.
func (r *Runner) ClearQueue(id string) {
	r.mu.Lock()
	_, had := r.queues[id]
	delete(r.queues, id)
	r.mu.Unlock()
	if had {
		r.live(id, Event{Type: EventQueue})
	}
}

// takeQueuedLocked removes a message from the queue and returns it with its
// position. r.mu must be held.
func (r *Runner) takeQueuedLocked(id, qid string) (QueuedMessage, int, bool) {
	q := r.queues[id]
	i := slices.IndexFunc(q, func(m QueuedMessage) bool { return m.ID == qid })
	if i < 0 {
		return QueuedMessage{}, -1, false
	}
	m := q[i]
	q = slices.Delete(slices.Clone(q), i, i+1)
	if len(q) == 0 {
		delete(r.queues, id)
	} else {
		r.queues[id] = q
	}
	return m, i, true
}

// SteerQueued sends a queued message into the chat's running turn, which
// goes on with it. It returns ErrNotRunning when no turn takes it; the
// message then stays queued.
func (r *Runner) SteerQueued(id, qid string) error {
	r.mu.Lock()
	m, at, ok := r.takeQueuedLocked(id, qid)
	t := r.turns[id]
	var ls *liveSession
	if t != nil {
		ls = t.ls
	}
	r.mu.Unlock()
	if !ok {
		return ErrQueuedNotFound
	}
	err := ErrNotRunning
	if t != nil && ls != nil && t.ctx.Err() == nil {
		err = r.steer(id, ls, m)
	}
	r.mu.Lock()
	if err != nil {
		// Back in its place, for the turn's end or another try.
		q := r.queues[id]
		r.queues[id] = slices.Insert(slices.Clone(q), min(at, len(q)), m)
	}
	list := slices.Clone(r.queues[id])
	r.mu.Unlock()
	if err != nil {
		return err
	}
	r.live(id, Event{Type: EventQueue, Queue: list})
	r.settle(id, Event{Type: EventSteer, Text: m.Text, Attachments: m.Attachments})
	return nil
}

func (r *Runner) steer(id string, ls *liveSession, m QueuedMessage) error {
	prompt, err := r.promptFiles(id, m.Attachments)
	if err != nil {
		return err
	}
	prompt.Text = withAttachments(m.Text, prompt.files)
	return ls.s.Steer(prompt.Prompt)
}

// combineQueued is the queue as one message: the texts with a blank line
// between, every attachment, the latest context, and the latest model and
// effort.
func combineQueued(provider string, list []QueuedMessage) (SendRequest, []Attachment) {
	req := SendRequest{Provider: provider}
	var texts []string
	var saved []Attachment
	for _, m := range list {
		if t := strings.TrimSpace(m.Text); t != "" {
			texts = append(texts, t)
		}
		saved = append(saved, m.Attachments...)
		if m.Context != nil {
			req.Context = m.Context
		}
		req.Model, req.Effort = m.Model, m.Effort
	}
	req.Text = strings.Join(texts, "\n\n")
	return req, saved
}

// sendQueued runs the registered turn t with the chat's drained queue.
func (r *Runner) sendQueued(id string, t *turn, provider string, list []QueuedMessage) {
	r.live(id, Event{Type: EventQueue})
	req, saved := combineQueued(provider, list)
	driver, ok := r.cfg.Drivers[provider]
	if !ok {
		r.finish(id, t)
		log.Printf("chat %s: send queued messages: unknown provider %q", id, provider)
		return
	}
	if err := r.begin(id, t, req, provider, driver, saved); err != nil {
		log.Printf("chat %s: send queued messages: %v", id, err)
	}
}
