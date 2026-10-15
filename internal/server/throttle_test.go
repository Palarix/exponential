package server

import (
	"sync"
	"testing"
	"time"
)

type fireLog struct {
	mu  sync.Mutex
	ids []string
}

func (l *fireLog) add(id string) {
	l.mu.Lock()
	l.ids = append(l.ids, id)
	l.mu.Unlock()
}

func (l *fireLog) count(id string) int {
	l.mu.Lock()
	defer l.mu.Unlock()
	n := 0
	for _, x := range l.ids {
		if x == id {
			n++
		}
	}
	return n
}

func TestThrottle_FirstFiresAtOnce(t *testing.T) {
	var log fireLog
	th := newThrottle(time.Hour, log.add)
	th.Trigger("a")
	if log.count("a") != 1 {
		t.Fatalf("fired %d times, want 1", log.count("a"))
	}
}

func TestThrottle_BurstCollapsesIntoOneTrailingFire(t *testing.T) {
	var log fireLog
	th := newThrottle(50*time.Millisecond, log.add)
	for range 10 {
		th.Trigger("a")
	}
	if log.count("a") != 1 {
		t.Fatalf("during window: fired %d times, want 1", log.count("a"))
	}
	time.Sleep(200 * time.Millisecond)
	if log.count("a") != 2 {
		t.Fatalf("after window: fired %d times, want 2", log.count("a"))
	}
}

func TestThrottle_NoTrailingFireWithoutChange(t *testing.T) {
	var log fireLog
	th := newThrottle(20*time.Millisecond, log.add)
	th.Trigger("a")
	time.Sleep(100 * time.Millisecond)
	if log.count("a") != 1 {
		t.Fatalf("fired %d times, want 1", log.count("a"))
	}
	th.Trigger("a") // window over: fires at once again
	if log.count("a") != 2 {
		t.Fatalf("fired %d times, want 2", log.count("a"))
	}
}

func TestThrottle_KeysAreIndependent(t *testing.T) {
	var log fireLog
	th := newThrottle(time.Hour, log.add)
	th.Trigger("a")
	th.Trigger("b")
	if log.count("a") != 1 || log.count("b") != 1 {
		t.Fatalf("fired a=%d b=%d, want 1 each", log.count("a"), log.count("b"))
	}
}
