package live

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// signalsOf returns the signal topics in outs.
func signalsOf(outs []Out) map[string]bool {
	got := map[string]bool{}
	for _, o := range outs {
		if o.Event == EventSignal {
			got[o.Topic] = true
		}
	}
	return got
}

func TestMirror_ForwardsRemoteSignalsAndResyncsOnEachConnect(t *testing.T) {
	var conns atomic.Int32
	release := make(chan struct{})
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tok" {
			http.Error(w, "no token", http.StatusUnauthorized)
			return
		}
		n := conns.Add(1)
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "event: hello\ndata: {\"stream\":\"r\"}\n\n")
		w.(http.Flusher).Flush()
		if n == 1 {
			<-release
			fmt.Fprint(w, "event: signal\ndata: {\"topic\":\"drafts\"}\n\n")
			fmt.Fprint(w, "event: signal\ndata: {\"topic\":\"chats\"}\n\n")
			w.(http.Flusher).Flush()
			return // the remote stream ends: the mirror reconnects
		}
		<-r.Context().Done()
	}))
	defer remote.Close()

	h := NewHub()
	h.Signal("issues")
	h.Signal("drafts")
	h.Signal("chats")
	s, _ := h.Subscribe()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m := &Mirror{
		Hub:        h,
		URL:        remote.URL,
		Header:     http.Header{"Authorization": {"Bearer tok"}},
		Topics:     []string{"issues", "drafts"},
		MinBackoff: time.Millisecond,
	}
	go m.Run(ctx)

	if got := signalsOf(take(t, s)); !got["issues"] || !got["drafts"] || got["chats"] {
		t.Fatalf("on connect: %v, want issues and drafts", got)
	}
	close(release)
	// The remote's drafts signal, then the reconnect's resync. The remote's
	// chats signal is not the remote's to send: chats live on this board.
	seen := map[string]bool{}
	deadline := time.Now().Add(2 * time.Second)
	for conns.Load() < 2 || !seen["issues"] {
		if time.Now().After(deadline) {
			t.Fatalf("conns = %d, seen = %v", conns.Load(), seen)
		}
		for k := range signalsOf(take(t, s)) {
			seen[k] = true
		}
	}
	if seen["chats"] {
		t.Error("the remote's chats signal was forwarded")
	}
}

func TestMirror_RetriesWithBackoffWhileTheRemoteIsDown(t *testing.T) {
	var tries atomic.Int32
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tries.Add(1)
		http.Error(w, "down", http.StatusBadGateway)
	}))
	defer remote.Close()

	h := NewHub()
	h.Signal("issues")
	s, _ := h.Subscribe()
	ctx, cancel := context.WithCancel(context.Background())
	m := &Mirror{Hub: h, URL: remote.URL, Topics: []string{"issues"}, MinBackoff: 5 * time.Millisecond, MaxBackoff: 20 * time.Millisecond}
	done := make(chan struct{})
	go func() { m.Run(ctx); close(done) }()

	waitUntil(t, func() bool { return tries.Load() >= 3 })
	if !idle(s) {
		t.Error("a failed connect notified the hub")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run did not return after the context ended")
	}
}

func TestBackoff_DoublesUpToTheMax(t *testing.T) {
	b := backoff{min: time.Second, max: 30 * time.Second}
	var got []time.Duration
	for range 7 {
		got = append(got, b.next())
	}
	want := []time.Duration{1, 2, 4, 8, 16, 30, 30}
	for i := range want {
		if got[i] != want[i]*time.Second {
			t.Fatalf("delays = %v", got)
		}
	}
	b.reset()
	if d := b.next(); d != time.Second {
		t.Errorf("after reset = %v", d)
	}
}
