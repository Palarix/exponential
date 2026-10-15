package server

import (
	"sync"
	"time"
)

// throttle calls fire for a key at most once per window. The first trigger
// fires at once; triggers during the window collapse into one fire at its
// end, so the last change is never lost.
type throttle struct {
	window time.Duration
	fire   func(key string)

	mu      sync.Mutex
	pending map[string]bool // keys in a window; true when triggered again
}

func newThrottle(window time.Duration, fire func(key string)) *throttle {
	return &throttle{window: window, fire: fire, pending: map[string]bool{}}
}

// Trigger fires key now, or at the end of its current window.
func (t *throttle) Trigger(key string) {
	t.mu.Lock()
	if _, open := t.pending[key]; open {
		t.pending[key] = true
		t.mu.Unlock()
		return
	}
	t.pending[key] = false
	t.mu.Unlock()
	t.fire(key)
	time.AfterFunc(t.window, func() { t.endWindow(key) })
}

func (t *throttle) endWindow(key string) {
	t.mu.Lock()
	if !t.pending[key] {
		delete(t.pending, key)
		t.mu.Unlock()
		return
	}
	t.pending[key] = false
	t.mu.Unlock()
	t.fire(key)
	time.AfterFunc(t.window, func() { t.endWindow(key) })
}
