package live

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type event struct {
	name string
	data map[string]any
}

// openStream starts a stream server for h and opens one stream. Events
// arrive on the returned channel, which closes when the stream ends.
func openStream(t *testing.T, st *Stream) (*httptest.Server, <-chan event) {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /events", st.ServeEvents)
	mux.HandleFunc("POST /events/{stream}/topics", st.ServeTopics)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	req, _ := http.NewRequestWithContext(ctx, "GET", srv.URL+"/events", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("Content-Type = %q", ct)
	}
	ch := make(chan event, 64)
	go func() {
		defer close(ch)
		defer resp.Body.Close()
		_ = ReadEvents(resp.Body, func(name, data string) error {
			var m map[string]any
			_ = json.Unmarshal([]byte(data), &m)
			ch <- event{name, m}
			return nil
		})
	}()
	return srv, ch
}

func next(t *testing.T, ch <-chan event) event {
	t.Helper()
	select {
	case e, ok := <-ch:
		if !ok {
			t.Fatal("the stream ended")
		}
		return e
	case <-time.After(2 * time.Second):
		t.Fatal("no event")
	}
	return event{}
}

func postTopics(t *testing.T, url, body string) int {
	t.Helper()
	resp, err := http.Post(url, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	return resp.StatusCode
}

func TestStream_HelloComesFirstThenSignals(t *testing.T) {
	h := newTestHub(&fakeLog{})
	_, ch := openStream(t, &Stream{Hub: h})

	hello := next(t, ch)
	if hello.name != EventHello || hello.data["stream"] == "" {
		t.Fatalf("first event = %+v, want hello with the stream id", hello)
	}
	h.Notify("issues")
	if e := next(t, ch); e.name != EventSignal || e.data["topic"] != "issues" {
		t.Errorf("event = %+v, want the issues signal", e)
	}
}

func TestStream_TopicsPostAddsAndRemovesLogTopics(t *testing.T) {
	log := &fakeLog{}
	log.add("a", 1)
	log.add("a", 2)
	h := newTestHub(log)
	srv, ch := openStream(t, &Stream{Hub: h})
	id := next(t, ch).data["stream"].(string)
	url := srv.URL + "/events/" + id + "/topics"

	if code := postTopics(t, url, `{"add":[{"topic":"chat:a","after":1}]}`); code != http.StatusNoContent {
		t.Fatalf("add = %d", code)
	}
	e := next(t, ch)
	if e.name != EventLog || e.data["topic"] != "chat:a" || e.data["data"] != float64(2) {
		t.Fatalf("event = %+v, want the replay of seq 2", e)
	}

	if code := postTopics(t, url, `{"remove":["chat:a"]}`); code != http.StatusNoContent {
		t.Fatalf("remove = %d", code)
	}
	h.Append("chat", "a", log.add("a", 3))
	h.Notify("chats")
	if e := next(t, ch); e.name != EventSignal {
		t.Errorf("event = %+v, want only the signal after the remove", e)
	}
}

func TestStream_TopicsPostRejectsUnknownStreamsAndTopics(t *testing.T) {
	h := newTestHub(&fakeLog{})
	srv, ch := openStream(t, &Stream{Hub: h})
	id := next(t, ch).data["stream"].(string)

	if code := postTopics(t, srv.URL+"/events/nope/topics", `{}`); code != http.StatusNotFound {
		t.Errorf("unknown stream = %d, want 404", code)
	}
	body := `{"add":[{"topic":"chat:a"},{"topic":"bogus:1"}]}`
	if code := postTopics(t, srv.URL+"/events/"+id+"/topics", body); code != http.StatusBadRequest {
		t.Errorf("unknown topic = %d, want 400", code)
	}
	if code := postTopics(t, srv.URL+"/events/"+id+"/topics", `not json`); code != http.StatusBadRequest {
		t.Errorf("bad JSON = %d, want 400", code)
	}
	// The rejected body added nothing, not even its valid topic.
	s := h.Lookup(id)
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.logs) != 0 {
		t.Errorf("logs = %v after a rejected add", s.logs)
	}
}

func TestStream_EndsWhenTheHubCloses(t *testing.T) {
	h := newTestHub(&fakeLog{})
	_, ch := openStream(t, &Stream{Hub: h})
	next(t, ch)
	h.Close()
	select {
	case _, ok := <-ch:
		if ok {
			t.Error("an event after Close")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the stream is still open after Close")
	}
}

func TestStream_SendsKeepalives(t *testing.T) {
	h := newTestHub(&fakeLog{})
	srv := httptest.NewServer(http.HandlerFunc((&Stream{Hub: h, Keepalive: 10 * time.Millisecond}).ServeEvents))
	defer srv.Close()
	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	br := bufio.NewReader(resp.Body)
	for range 10 {
		line, err := br.ReadString('\n')
		if err != nil {
			t.Fatal(err)
		}
		if line == ": keepalive\n" {
			return
		}
	}
	t.Error("no keepalive")
}

// A client that stops reading must not hold its handler forever: the write
// deadline ends the stream and frees the subscriber.
func TestStream_WriteDeadlineEndsAFrozenClient(t *testing.T) {
	big := strings.Repeat("x", 64<<10)
	h := NewHub()
	h.Log("chat", func(string, int) ([]Frame, error) {
		frames := make([]Frame, 200)
		for i := range frames {
			frames[i] = Frame{Seq: i + 1, Data: big}
		}
		return frames, nil
	})
	st := &Stream{Hub: h, WriteTimeout: 100 * time.Millisecond}
	srv := httptest.NewServer(http.HandlerFunc(st.ServeEvents))
	defer srv.Close()

	conn, err := net.Dial("tcp", srv.Listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if _, err := io.WriteString(conn, "GET / HTTP/1.1\r\nHost: x\r\n\r\n"); err != nil {
		t.Fatal(err)
	}
	waitUntil(t, func() bool { return h.Count() == 1 })
	var id string
	h.mu.Lock()
	for id = range h.subs {
	}
	h.mu.Unlock()
	if err := h.Lookup(id).Add("chat:a", 0); err != nil {
		t.Fatal(err)
	}
	// 12 MiB of replay, never read: the handler must give up.
	waitUntil(t, func() bool { return h.Count() == 0 })
}

func waitUntil(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal("condition not met")
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestReadEvents_ParsesNamesDataAndComments(t *testing.T) {
	in := ": hi\n\nevent: a\ndata: 1\n\ndata: x\ndata: y\nid: 3\n\nevent: b\r\ndata:2\r\n\r\n"
	var got []string
	if err := ReadEvents(strings.NewReader(in), func(e, d string) error {
		got = append(got, e+"="+d)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	want := []string{"a=1", "message=x\ny", "b=2"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("got %q, want %q", got, want)
	}
	stop := errors.New("stop")
	if err := ReadEvents(strings.NewReader(in), func(string, string) error { return stop }); !errors.Is(err, stop) {
		t.Errorf("err = %v, want fn's error", err)
	}
}
