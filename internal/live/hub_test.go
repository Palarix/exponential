package live

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// fakeLog is a log topic's store: the settled frames of each key.
type fakeLog struct {
	mu     sync.Mutex
	frames map[string][]Frame
	// during runs inside replay, after the store is read, to model an event
	// settling while the store is read.
	during func()
}

func (l *fakeLog) add(key string, seq int) Frame {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.frames == nil {
		l.frames = map[string][]Frame{}
	}
	f := Frame{Seq: seq, Data: seq}
	l.frames[key] = append(l.frames[key], f)
	return f
}

func (l *fakeLog) replay(key string, after int) ([]Frame, error) {
	l.mu.Lock()
	all, ok := l.frames[key]
	var out []Frame
	for _, f := range all {
		if f.Seq > after {
			out = append(out, f)
		}
	}
	during := l.during
	l.during = nil
	l.mu.Unlock()
	if !ok {
		return nil, ErrGone
	}
	if during != nil {
		during()
	}
	return out, nil
}

func newTestHub(log *fakeLog) *Hub {
	h := NewHub()
	h.Signal("issues")
	h.Signal("chats")
	h.Log("chat", log.replay)
	return h
}

// take waits for work and returns it, failing the test after a second.
func take(t *testing.T, s *Sub) []Out {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if !s.Wait(ctx) {
		t.Fatal("no work for the subscriber")
	}
	return s.Take()
}

// idle reports whether s has no work right now.
func idle(s *Sub) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	return !s.Wait(ctx) || len(s.Take()) == 0
}

func seqs(outs []Out, topic string) []int {
	var got []int
	for _, o := range outs {
		if o.Event == EventLog && o.Topic == topic {
			got = append(got, o.Data.(int))
		}
	}
	return got
}

func equalInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestNotify_MergesRepeatsIntoOneSignal(t *testing.T) {
	h := newTestHub(&fakeLog{})
	s, _ := h.Subscribe()
	for range 1000 {
		h.Notify("issues")
	}
	h.Notify("chats")

	outs := take(t, s)
	if len(outs) != 2 {
		t.Fatalf("outs = %+v, want one signal per topic", outs)
	}
	got := map[string]bool{}
	for _, o := range outs {
		if o.Event != EventSignal {
			t.Errorf("event = %q, want signal", o.Event)
		}
		got[o.Topic] = true
	}
	if !got["issues"] || !got["chats"] {
		t.Errorf("topics = %v", got)
	}
	if !idle(s) {
		t.Error("work left after the signals were taken")
	}
}

func TestNotify_UnknownTopicIsIgnored(t *testing.T) {
	h := newTestHub(&fakeLog{})
	s, _ := h.Subscribe()
	h.Notify("nope")
	if !idle(s) {
		t.Error("an unknown topic woke the subscriber")
	}
}

func TestAdd_ReplaysFromTheStoreThenGoesLive(t *testing.T) {
	log := &fakeLog{}
	for seq := 1; seq <= 4; seq++ {
		log.add("a", seq)
	}
	h := newTestHub(log)
	s, _ := h.Subscribe()
	if err := s.Add("chat:a", 2); err != nil {
		t.Fatal(err)
	}
	if got := seqs(take(t, s), "chat:a"); !equalInts(got, []int{3, 4}) {
		t.Fatalf("replay = %v, want [3 4]", got)
	}

	h.Append("chat", "a", log.add("a", 5))
	h.Append("chat", "a", Frame{Data: 0}) // a live frame
	if got := seqs(take(t, s), "chat:a"); !equalInts(got, []int{5, 0}) {
		t.Errorf("live = %v, want [5 0]", got)
	}
}

func TestAdd_NoEventFallsBetweenReplayAndLive(t *testing.T) {
	log := &fakeLog{}
	log.add("a", 1)
	h := newTestHub(log)
	s, _ := h.Subscribe()
	// Seq 2 settles after the store was read but before the replay is
	// written: it must come once, from the queue.
	log.during = func() { h.Append("chat", "a", log.add("a", 2)) }
	if err := s.Add("chat:a", 0); err != nil {
		t.Fatal(err)
	}
	var got []int
	for len(got) < 2 {
		got = append(got, seqs(take(t, s), "chat:a")...)
	}
	if !equalInts(got, []int{1, 2}) {
		t.Errorf("seqs = %v, want [1 2]", got)
	}
	if !idle(s) {
		t.Error("a duplicate was left over")
	}
}

func TestAdd_SkipsQueuedFramesTheReplayAlreadySent(t *testing.T) {
	log := &fakeLog{}
	log.add("a", 1)
	h := newTestHub(log)
	s, _ := h.Subscribe()
	if err := s.Add("chat:a", 0); err != nil {
		t.Fatal(err)
	}
	// Seq 2 lands in the store and in the queue before the writer runs.
	h.Append("chat", "a", log.add("a", 2))
	if got := seqs(take(t, s), "chat:a"); !equalInts(got, []int{1, 2}) {
		t.Errorf("seqs = %v, want [1 2]", got)
	}
}

func TestAppend_OverflowReplaysThatTopicOnly(t *testing.T) {
	log := &fakeLog{frames: map[string][]Frame{"a": nil}}
	log.add("b", 1)
	h := newTestHub(log)
	h.QueueCap = 4
	slow, _ := h.Subscribe()
	fast, _ := h.Subscribe()
	for _, s := range []*Sub{slow, fast} {
		if err := s.Add("chat:a", 0); err != nil {
			t.Fatal(err)
		}
		if err := s.Add("chat:b", 1); err != nil {
			t.Fatal(err)
		}
	}
	take(t, fast)
	take(t, slow)

	// fast keeps up; slow doesn't read until the end.
	var fastGot []int
	for seq := 1; seq <= 20; seq++ {
		h.Append("chat", "a", log.add("a", seq))
		if seq%2 == 0 {
			fastGot = append(fastGot, seqs(take(t, fast), "chat:a")...)
		}
	}
	h.Append("chat", "b", log.add("b", 2))
	h.Notify("issues")

	want := make([]int, 20)
	for i := range want {
		want[i] = i + 1
	}
	if !equalInts(fastGot, want) {
		t.Errorf("fast subscriber got %v", fastGot)
	}

	outs := take(t, slow)
	if got := seqs(outs, "chat:a"); !equalInts(got, want) {
		t.Errorf("slow subscriber got %v, want 1..20 once, in order", got)
	}
	if got := seqs(outs, "chat:b"); !equalInts(got, []int{2}) {
		t.Errorf("other topic got %v, want [2]", got)
	}
	signals := 0
	for _, o := range outs {
		if o.Event == EventSignal {
			signals++
		}
	}
	if signals != 1 {
		t.Errorf("signals = %d, want 1", signals)
	}
}

func TestAppend_LiveFramesAreDroppedOnOverflow(t *testing.T) {
	log := &fakeLog{}
	log.add("a", 1)
	h := newTestHub(log)
	h.QueueCap = 2
	s, _ := h.Subscribe()
	if err := s.Add("chat:a", 1); err != nil {
		t.Fatal(err)
	}
	for range 5 {
		h.Append("chat", "a", Frame{Data: 0})
	}
	h.Append("chat", "a", log.add("a", 2))
	if got := seqs(take(t, s), "chat:a"); !equalInts(got, []int{2}) {
		t.Errorf("seqs = %v, want [2] from the replay", got)
	}
}

func TestReplay_GoneDropsTheTopic(t *testing.T) {
	h := newTestHub(&fakeLog{})
	s, _ := h.Subscribe()
	if err := s.Add("chat:x", 0); err != nil {
		t.Fatal(err)
	}
	outs := take(t, s)
	if len(outs) != 1 || outs[0].Event != EventGone || outs[0].Topic != "chat:x" {
		t.Fatalf("outs = %+v, want one gone frame", outs)
	}
	h.Append("chat", "x", Frame{Seq: 1, Data: 1})
	if !idle(s) {
		t.Error("a dropped topic still gets frames")
	}
}

func TestReplay_ErrorDropsTheTopic(t *testing.T) {
	h := NewHub()
	h.Log("chat", func(string, int) ([]Frame, error) { return nil, errors.New("disk on fire") })
	s, _ := h.Subscribe()
	if err := s.Add("chat:x", 0); err != nil {
		t.Fatal(err)
	}
	if outs := take(t, s); len(outs) != 1 || outs[0].Event != EventGone {
		t.Errorf("outs = %+v, want gone, so the browser reloads the chat", outs)
	}
}

func TestAdd_RejectsUnknownTopics(t *testing.T) {
	h := newTestHub(&fakeLog{})
	s, _ := h.Subscribe()
	for _, topic := range []string{"nope:1", "chat", "chat:", "issues", ""} {
		if err := s.Add(topic, 0); !errors.Is(err, ErrUnknownTopic) {
			t.Errorf("Add(%q) = %v, want ErrUnknownTopic", topic, err)
		}
	}
}

func TestRemove_StopsFrames(t *testing.T) {
	log := &fakeLog{}
	log.add("a", 1)
	h := newTestHub(log)
	s, _ := h.Subscribe()
	if err := s.Add("chat:a", 1); err != nil {
		t.Fatal(err)
	}
	s.Remove("chat:a")
	s.Remove("chat:a") // a no-op
	h.Append("chat", "a", log.add("a", 2))
	if !idle(s) {
		t.Error("a removed topic still gets frames")
	}
}

func TestLookup_FindsOpenStreamsOnly(t *testing.T) {
	h := newTestHub(&fakeLog{})
	s, _ := h.Subscribe()
	if h.Lookup(s.ID) != s {
		t.Fatal("Lookup did not find the stream")
	}
	s.Close()
	s.Close() // a no-op
	if h.Lookup(s.ID) != nil {
		t.Error("Lookup found a closed stream")
	}
	if h.Count() != 0 {
		t.Errorf("Count = %d after Close", h.Count())
	}
}

func TestClose_EndsEveryStream(t *testing.T) {
	h := newTestHub(&fakeLog{})
	s, _ := h.Subscribe()
	h.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if s.Wait(ctx) {
		t.Error("Wait = true after the hub closed")
	}
	if ctx.Err() != nil {
		t.Error("Wait blocked after the hub closed")
	}
	if _, err := h.Subscribe(); !errors.Is(err, ErrClosed) {
		t.Errorf("Subscribe after Close = %v, want ErrClosed", err)
	}
	h.Notify("issues") // no panic
}
