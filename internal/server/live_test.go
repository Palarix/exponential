package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/palarix/exponential/internal/chat"
	"github.com/palarix/exponential/internal/live"
)

// subscribeLive opens a hub subscriber on s, creating the hub if needed.
func subscribeLive(t *testing.T, s *Server) *live.Sub {
	t.Helper()
	if s.Live == nil {
		s.Live = NewLiveHub()
	}
	sub, err := s.Live.Subscribe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(sub.Close)
	return sub
}

// signals collects the signal topics sub gets within d.
func signals(sub *live.Sub, d time.Duration) map[string]int {
	got := map[string]int{}
	ctx, cancel := context.WithTimeout(context.Background(), d)
	defer cancel()
	for sub.Wait(ctx) {
		for _, o := range sub.Take() {
			if o.Event == live.EventSignal {
				got[o.Topic]++
			}
		}
	}
	return got
}

// waitSignal waits up to a second for the signal topic on sub.
func waitSignal(t *testing.T, sub *live.Sub, topic string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	for sub.Wait(ctx) {
		for _, o := range sub.Take() {
			if o.Event == live.EventSignal && o.Topic == topic {
				return
			}
		}
	}
	t.Fatalf("no %s signal", topic)
}

func TestNotify_NilHub(t *testing.T) {
	s := setupTestServer(t)
	s.notify(TopicIssues) // must not panic
	s.ChatChanged("chat-1")
}

func TestChatChanged_NotifiesChats(t *testing.T) {
	s := setupTestServer(t)
	sub := subscribeLive(t, s)
	s.ChatChanged("chat-0123abcd")
	waitSignal(t, sub, TopicChats)
}

// sseClient reads a board's /api/events.
type sseClient struct {
	t      *testing.T
	base   string
	stream string
	events chan sseFrame
}

type sseFrame struct {
	event string
	data  map[string]any
}

func openEvents(t *testing.T, base string) *sseClient {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	req, _ := http.NewRequestWithContext(ctx, "GET", base+"/api/events", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /api/events: %v", err)
	}
	if resp.StatusCode != http.StatusOK || resp.Header.Get("Content-Type") != "text/event-stream" {
		t.Fatalf("GET /api/events: %d %q", resp.StatusCode, resp.Header.Get("Content-Type"))
	}
	c := &sseClient{t: t, base: base, events: make(chan sseFrame)}
	go func() {
		defer close(c.events)
		defer resp.Body.Close()
		_ = live.ReadEvents(resp.Body, func(event, data string) error {
			var m map[string]any
			_ = json.Unmarshal([]byte(data), &m)
			c.events <- sseFrame{event, m}
			return nil
		})
	}()
	hello := c.next()
	if hello.event != live.EventHello {
		t.Fatalf("first event = %+v, want hello", hello)
	}
	c.stream = hello.data["stream"].(string)
	return c
}

func (c *sseClient) next() sseFrame {
	c.t.Helper()
	select {
	case f, ok := <-c.events:
		if !ok {
			c.t.Fatal("the stream ended")
		}
		return f
	case <-time.After(3 * time.Second):
		c.t.Fatal("no event")
	}
	return sseFrame{}
}

func (c *sseClient) topics(body string) int {
	c.t.Helper()
	resp, err := http.Post(c.base+"/api/events/"+c.stream+"/topics", "application/json", strings.NewReader(body))
	if err != nil {
		c.t.Fatal(err)
	}
	resp.Body.Close()
	return resp.StatusCode
}

func TestEvents_StreamsIssueSignals(t *testing.T) {
	s := setupTestServer(t)
	s.Live = NewLiveHub()
	srv := httptest.NewServer(s.SetupRoutes())
	t.Cleanup(srv.Close)

	c := openEvents(t, srv.URL)
	s.notify(TopicIssues)
	if f := c.next(); f.event != live.EventSignal || f.data["topic"] != TopicIssues {
		t.Errorf("event = %+v, want the issues signal", f)
	}
}

func TestEvents_NoHubIsNotRouted(t *testing.T) {
	s := setupTestServer(t)
	w := httptest.NewRecorder()
	s.SetupRoutes().ServeHTTP(w, httptest.NewRequest("GET", "/api/events", nil))
	if w.Code == http.StatusOK {
		t.Errorf("GET /api/events without a hub = %d", w.Code)
	}
}

// A chat's events reach the stream through Runner's OnEvent, and adding the
// chat's topic replays its log from the store first.
func TestEvents_ChatTopicReplaysThenStreamsRunnerEvents(t *testing.T) {
	s, srv := newChatServer(t)
	m, err := s.Chat.Store.Create()
	if err != nil {
		t.Fatal(err)
	}
	stored, err := s.Chat.Store.Append(m.ID, chat.Event{Type: chat.EventUserMessage, Text: "one"}, chat.Event{Type: chat.EventUserMessage, Text: "two"})
	if err != nil {
		t.Fatal(err)
	}

	c := openEvents(t, srv.URL)
	if code := c.topics(fmt.Sprintf(`{"add":[{"topic":"chat:%s","after":%d}]}`, m.ID, stored[0].Seq)); code != http.StatusNoContent {
		t.Fatalf("topics = %d", code)
	}
	f := c.next()
	data, _ := f.data["data"].(map[string]any)
	if f.event != live.EventLog || f.data["topic"] != "chat:"+m.ID || data["text"] != "two" {
		t.Fatalf("event = %+v, want the replay of the second message", f)
	}

	ChatEventsTo(s.Live)(m.ID, chat.Event{Type: chat.EventTextDelta, Text: "live"})
	f = c.next()
	if data, _ := f.data["data"].(map[string]any); f.event != live.EventLog || data["text"] != "live" {
		t.Errorf("event = %+v, want the live delta", f)
	}
}

func TestEvents_DeletedChatTopicIsGone(t *testing.T) {
	_, srv := newChatServer(t)
	c := openEvents(t, srv.URL)
	if code := c.topics(`{"add":[{"topic":"chat:00000000-0000-4000-8000-000000000000"}]}`); code != http.StatusNoContent {
		t.Fatalf("topics = %d", code)
	}
	if f := c.next(); f.event != live.EventGone {
		t.Errorf("event = %+v, want gone", f)
	}
}

func TestEvents_ChatStreamEndpointIsGone(t *testing.T) {
	s, srv := newChatServer(t)
	m, _ := s.Chat.Store.Create()
	resp, err := http.Get(srv.URL + "/api/local/chats/" + m.ID + "/stream")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound && resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("GET .../stream = %d, want it gone", resp.StatusCode)
	}
}

// In proxy mode the browser's stream is this board's: a remote that is down
// doesn't keep it from opening, and chat topics still work.
func TestEvents_ProxyModeStreamWorksWhileTheRemoteIsDown(t *testing.T) {
	s := setupTestServer(t)
	s.Live = NewLiveHub()
	s.ProxyURL = "http://127.0.0.1:1"
	srv := httptest.NewServer(s.SetupRoutes())
	t.Cleanup(srv.Close)

	c := openEvents(t, srv.URL)
	s.ChatChanged("chat-1")
	if f := c.next(); f.event != live.EventSignal || f.data["topic"] != TopicChats {
		t.Errorf("event = %+v, want the chats signal", f)
	}
	if code := c.topics(`{"add":[{"topic":"bogus:1"}]}`); code != http.StatusBadRequest {
		t.Errorf("topics in proxy mode = %d, want the local board's 400", code)
	}
}

func TestStartMirror_NotifiesRemoteTopics(t *testing.T) {
	auth := make(chan string, 4)
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/events" {
			http.NotFound(w, r)
			return
		}
		auth <- r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "event: hello\ndata: {\"stream\":\"r\"}\n\n")
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	}))
	t.Cleanup(remote.Close)

	s := setupTestServer(t)
	s.ProxyURL = remote.URL
	s.ProxyToken = "tok"
	sub := subscribeLive(t, s)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.startMirror(ctx)

	if got := <-auth; got != "Bearer tok" {
		t.Errorf("remote Authorization = %q", got)
	}
	got := signals(sub, 200*time.Millisecond)
	if got[TopicIssues] == 0 || got[TopicDrafts] == 0 || got[TopicChats] != 0 {
		t.Errorf("signals = %v, want issues and drafts", got)
	}
}

func TestIssueWrites_NotifyIssuesOnly(t *testing.T) {
	s := setupTestServer(t)
	sub := subscribeLive(t, s)
	w := httptest.NewRecorder()
	body := `{"type":"CREATE","payload":{"title":"New one","labels":["bug"]}}`
	s.SetupRoutes().ServeHTTP(w, httptest.NewRequest("POST", "/api/draft", strings.NewReader(body)))
	if w.Code != http.StatusOK {
		t.Fatalf("create: %d %s", w.Code, w.Body.String())
	}
	if got := signals(sub, 50*time.Millisecond); got[TopicIssues] != 1 || len(got) != 1 {
		t.Errorf("signals = %v, want issues only", got)
	}
}
