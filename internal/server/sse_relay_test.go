package server

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// fakeRemoteSSE serves /api/events: it records the Authorization header,
// sends one issue event, then holds the stream open until release closes
// (or ends it at once when release is nil).
func fakeRemoteSSE(t *testing.T, auth chan<- string, release <-chan struct{}) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/events" {
			http.NotFound(w, r)
			return
		}
		auth <- r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "event: issue_updated\ndata: {\"issue_id\":\"remote-1\"}\n\n")
		w.(http.Flusher).Flush()
		if release == nil {
			return
		}
		select {
		case <-release:
		case <-r.Context().Done():
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func newRelayServer(t *testing.T, remoteURL string) (*Server, *httptest.Server) {
	t.Helper()
	s := setupTestServer(t)
	s.SSEHub = NewSSEHub()
	s.ProxyURL = remoteURL
	s.ProxyToken = "tok"
	srv := httptest.NewServer(s.SetupRoutes())
	t.Cleanup(srv.Close)
	return s, srv
}

func getEvents(t *testing.T, url string) *http.Response {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	t.Cleanup(cancel)
	req, _ := http.NewRequestWithContext(ctx, "GET", url+"/api/events", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /api/events: %v", err)
	}
	t.Cleanup(func() { resp.Body.Close() })
	return resp
}

func TestSSERelay_MergesRemoteAndLocalEvents(t *testing.T) {
	auth := make(chan string, 1)
	release := make(chan struct{})
	defer close(release)
	remote := fakeRemoteSSE(t, auth, release)
	s, srv := newRelayServer(t, remote.URL)

	resp := getEvents(t, srv.URL)
	if resp.StatusCode != http.StatusOK || resp.Header.Get("Content-Type") != "text/event-stream" {
		t.Fatalf("status %d, content-type %q", resp.StatusCode, resp.Header.Get("Content-Type"))
	}
	if got := <-auth; got != "Bearer tok" {
		t.Errorf("remote Authorization = %q", got)
	}

	sc := bufio.NewScanner(resp.Body)
	events := map[string]string{} // id -> event name
	var name string
	sentLocal := false
	for sc.Scan() && len(events) < 2 {
		line := sc.Text()
		if n, ok := strings.CutPrefix(line, "event: "); ok {
			name = n
		}
		for _, id := range []string{"remote-1", "chat-1"} {
			if strings.Contains(line, id) {
				events[id] = name
			}
		}
		if events["remote-1"] != "" && !sentLocal {
			sentLocal = true
			s.ChatChanged("chat-1")
		}
	}
	if events["remote-1"] != "issue_updated" || events["chat-1"] != "chat_changed" {
		t.Fatalf("events = %v", events)
	}
}

func TestSSERelay_UnreachableRemoteIs502(t *testing.T) {
	remote := httptest.NewServer(http.NotFoundHandler())
	url := remote.URL
	remote.Close()
	_, srv := newRelayServer(t, url)

	resp := getEvents(t, srv.URL)
	if resp.StatusCode != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502", resp.StatusCode)
	}
}

func TestSSERelay_RemoteErrorIs502(t *testing.T) {
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusUnauthorized)
	}))
	defer remote.Close()
	_, srv := newRelayServer(t, remote.URL)

	resp := getEvents(t, srv.URL)
	if resp.StatusCode != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502", resp.StatusCode)
	}
}

func TestSSERelay_EndsWhenRemoteEnds(t *testing.T) {
	auth := make(chan string, 1)
	remote := fakeRemoteSSE(t, auth, nil)
	_, srv := newRelayServer(t, remote.URL)

	resp := getEvents(t, srv.URL)
	body, err := io.ReadAll(resp.Body) // returns only once the relay ends
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !strings.Contains(string(body), "remote-1") {
		t.Fatalf("body = %q", body)
	}
}
