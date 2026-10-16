// Package live is the board's one broadcaster of live events. Every event
// belongs to a topic, and a topic is one of two kinds:
//
//   - A signal topic says "something changed, refetch". It has no payload,
//     so repeats merge into one dirty flag per subscriber and can't overflow.
//   - A log topic is an ordered log with seq ids and a store to replay from.
//     Its instances are keyed: "chat:<id>". A subscriber that falls behind
//     loses its queue and replays that one topic from the store.
//
// Publishing never blocks. Each subscriber is drained by its own writer
// (the SSE handler), which calls Wait and Take.
package live

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"sync"
)

// Frame is one event of a log topic. Seq is its position in the log; 0 is
// a live frame that is never stored (a streaming delta, say).
type Frame struct {
	Seq  int
	Data any
}

// ReplayFunc returns the stored frames of a log topic's key with Seq >
// after. It returns ErrGone when the key no longer exists.
type ReplayFunc func(key string, after int) ([]Frame, error)

// The events a writer sends.
const (
	EventHello  = "hello"
	EventSignal = "signal"
	EventLog    = "log"
	EventGone   = "gone"
)

// Out is one event for a writer to send.
type Out struct {
	Event string
	Topic string
	Data  any // a log frame's data; nil otherwise
}

var (
	// ErrGone is returned by a ReplayFunc whose key no longer exists.
	ErrGone = errors.New("live: topic is gone")
	// ErrUnknownTopic is returned by Add for a topic of no registered log kind.
	ErrUnknownTopic = errors.New("live: unknown topic")
	// ErrClosed is returned by Subscribe after Close.
	ErrClosed = errors.New("live: hub is closed")
)

// DefaultQueueCap is how many frames of one log topic a subscriber holds
// before it falls back to a replay.
const DefaultQueueCap = 256

// Hub holds the registered topics and the open subscribers.
type Hub struct {
	// QueueCap overrides DefaultQueueCap; set it before the first Subscribe.
	QueueCap int

	mu      sync.Mutex
	signals map[string]bool
	logs    map[string]ReplayFunc
	subs    map[string]*Sub
	closed  bool
}

// NewHub returns a hub with no topics.
func NewHub() *Hub {
	return &Hub{signals: map[string]bool{}, logs: map[string]ReplayFunc{}, subs: map[string]*Sub{}}
}

// Signal registers a signal topic. Every subscriber gets it.
func (h *Hub) Signal(name string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.signals[name] = true
}

// Log registers a log topic kind: subscribers add "<kind>:<key>" topics.
func (h *Hub) Log(kind string, replay ReplayFunc) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.logs[kind] = replay
}

// Notify marks a signal topic dirty for every subscriber. It never blocks.
func (h *Hub) Notify(topic string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if !h.signals[topic] {
		return
	}
	for _, s := range h.subs {
		s.mu.Lock()
		s.dirty[topic] = true
		s.mu.Unlock()
		s.signal()
	}
}

// Append queues a frame of the log topic "<kind>:<key>" for every subscriber
// of it. A subscriber whose queue is full drops it and replays the topic
// from the store instead. It never blocks.
func (h *Hub) Append(kind, key string, f Frame) {
	topic := kind + ":" + key
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, s := range h.subs {
		s.mu.Lock()
		l, ok := s.logs[topic]
		if ok {
			if len(l.queue) >= h.queueCap() {
				l.queue = nil
				l.again = true
			} else {
				l.queue = append(l.queue, f)
			}
		}
		s.mu.Unlock()
		if ok {
			s.signal()
		}
	}
}

// Subscribe opens a subscriber to every signal topic and no log topic.
func (h *Hub) Subscribe() (*Sub, error) {
	var b [16]byte
	_, _ = rand.Read(b[:])
	s := &Sub{
		ID:    hex.EncodeToString(b[:]),
		hub:   h,
		dirty: map[string]bool{},
		logs:  map[string]*logState{},
		wake:  make(chan struct{}, 1),
		done:  make(chan struct{}),
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return nil, ErrClosed
	}
	h.subs[s.ID] = s
	return s, nil
}

// Lookup returns the open subscriber with the given ID, or nil.
func (h *Hub) Lookup(id string) *Sub {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.subs[id]
}

// Count returns the number of open subscribers.
func (h *Hub) Count() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.subs)
}

// Close ends every subscriber; their Wait returns false. Later Subscribe
// calls fail.
func (h *Hub) Close() {
	h.mu.Lock()
	h.closed = true
	subs := h.subs
	h.subs = map[string]*Sub{}
	h.mu.Unlock()
	for _, s := range subs {
		s.end()
	}
}

func (h *Hub) queueCap() int {
	if h.QueueCap > 0 {
		return h.QueueCap
	}
	return DefaultQueueCap
}

// replayFor returns the replay func of a log topic and its key.
func (h *Hub) replayFor(topic string) (ReplayFunc, string, bool) {
	kind, key, ok := strings.Cut(topic, ":")
	if !ok || key == "" {
		return nil, "", false
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	replay, ok := h.logs[kind]
	return replay, key, ok
}

// Sub is one subscriber: one SSE stream.
type Sub struct {
	ID string

	hub      *Hub
	mu       sync.Mutex
	dirty    map[string]bool
	logs     map[string]*logState
	wake     chan struct{}
	done     chan struct{}
	doneOnce sync.Once
}

type logState struct {
	replay ReplayFunc
	key    string
	last   int     // the last settled seq written
	queue  []Frame // frames waiting for the writer
	gone   bool    // the topic was dropped; send gone
	again  bool    // replay from the store before the queue
}

// Add subscribes to the log topic "<kind>:<key>" from seq after: the writer
// replays the stored frames after it, then goes live. Adding a topic again
// replays it again from the new position.
func (s *Sub) Add(topic string, after int) error {
	replay, key, ok := s.hub.replayFor(topic)
	if !ok {
		return ErrUnknownTopic
	}
	s.mu.Lock()
	s.logs[topic] = &logState{replay: replay, key: key, last: after, again: true}
	s.mu.Unlock()
	s.signal()
	return nil
}

// Remove unsubscribes from a log topic. Removing one it doesn't have is a
// no-op.
func (s *Sub) Remove(topic string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.logs, topic)
}

// Wait blocks until there is work to take. It returns false when the
// subscriber or its hub closed, or ctx ended.
func (s *Sub) Wait(ctx context.Context) bool {
	select {
	case <-s.done:
		return false
	default:
	}
	select {
	case <-s.wake:
		return true
	case <-s.done:
		return false
	case <-ctx.Done():
		return false
	}
}

// Take returns the pending events in the order to write them: dirty
// signals, then per log topic its replay and its queue, without the settled
// frames already written. It reads the store for a topic that must replay.
func (s *Sub) Take() []Out {
	s.mu.Lock()
	var outs []Out
	for topic := range s.dirty {
		outs = append(outs, Out{Event: EventSignal, Topic: topic})
	}
	clear(s.dirty)
	type job struct {
		l     *logState
		after int
	}
	var jobs []job
	for _, l := range s.logs {
		if l.again {
			l.again = false
			jobs = append(jobs, job{l, l.last})
		}
	}
	s.mu.Unlock()

	// Read the stores without the lock: frames that settle meanwhile queue
	// up, and the seq check below drops those the replay already has.
	replayed := map[*logState][]Frame{}
	for _, j := range jobs {
		frames, err := j.l.replay(j.l.key, j.after)
		if err != nil {
			s.mu.Lock()
			j.l.gone = true
			s.mu.Unlock()
			continue
		}
		replayed[j.l] = frames
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for topic, l := range s.logs {
		if l.gone {
			delete(s.logs, topic)
			outs = append(outs, Out{Event: EventGone, Topic: topic})
			continue
		}
		frames := append(replayed[l], l.queue...)
		l.queue = nil
		for _, f := range frames {
			if f.Seq > 0 {
				if f.Seq <= l.last {
					continue
				}
				l.last = f.Seq
			}
			outs = append(outs, Out{Event: EventLog, Topic: topic, Data: f.Data})
		}
	}
	return outs
}

// Close unsubscribes s from the hub. Closing twice is a no-op.
func (s *Sub) Close() {
	s.hub.mu.Lock()
	if s.hub.subs[s.ID] == s {
		delete(s.hub.subs, s.ID)
	}
	s.hub.mu.Unlock()
	s.end()
}

func (s *Sub) end() { s.doneOnce.Do(func() { close(s.done) }) }

// signal wakes the writer, unless a wake is already pending.
func (s *Sub) signal() {
	select {
	case s.wake <- struct{}{}:
	default:
	}
}
