package server

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSSEHub_BroadcastToClients(t *testing.T) {
	hub := NewSSEHub()

	ch1 := hub.subscribe()
	ch2 := hub.subscribe()
	defer hub.unsubscribe(ch1)
	defer hub.unsubscribe(ch2)

	hub.Broadcast(SSEEvent{Type: "CREATE", IssueID: "test-123"})

	select {
	case evt := <-ch1:
		if evt.IssueID != "test-123" {
			t.Errorf("ch1: got %q", evt.IssueID)
		}
	case <-time.After(time.Second):
		t.Fatal("ch1: timeout")
	}

	select {
	case evt := <-ch2:
		if evt.Type != "CREATE" {
			t.Errorf("ch2: got %q", evt.Type)
		}
	case <-time.After(time.Second):
		t.Fatal("ch2: timeout")
	}
}

func TestSSEHub_UnsubscribeRemovesClient(t *testing.T) {
	hub := NewSSEHub()
	ch := hub.subscribe()

	if hub.ClientCount() != 1 {
		t.Fatalf("expected 1 client, got %d", hub.ClientCount())
	}

	hub.unsubscribe(ch)

	if hub.ClientCount() != 0 {
		t.Fatalf("expected 0 clients, got %d", hub.ClientCount())
	}
}

// A client that falls behind (a frozen background tab) is closed rather than
// left open with events dropped, so its browser reconnects and refetches.
func TestSSEHub_BroadcastClosesSlowClient(t *testing.T) {
	hub := NewSSEHub()
	ch := hub.subscribe()
	keep := hub.subscribe()
	defer hub.unsubscribe(keep)

	// Overflow ch's buffer; keep drains as it goes. Must not block.
	for i := range cap(ch) + 1 {
		hub.Broadcast(SSEEvent{Type: "UPDATE", IssueID: fmt.Sprint(i)})
		<-keep
	}
	n := 0
drain:
	for {
		select {
		case _, ok := <-ch:
			if !ok {
				break drain
			}
			n++
		case <-time.After(time.Second):
			t.Fatalf("client still open after %d events", n)
		}
	}
	if n != cap(ch) {
		t.Fatalf("got %d events before close, want %d", n, cap(ch))
	}
	if hub.ClientCount() != 1 {
		t.Fatalf("expected 1 client, got %d", hub.ClientCount())
	}

	hub.Broadcast(SSEEvent{Type: "UPDATE", IssueID: "after"})
	if evt := <-keep; evt.IssueID != "after" {
		t.Fatalf("kept client got %q", evt.IssueID)
	}
	// The handler's deferred unsubscribe must not close it again.
	hub.unsubscribe(ch)
}

func TestHandleSSE_StreamsEvents(t *testing.T) {
	s := setupTestServer(t)
	s.SSEHub = NewSSEHub()
	mux := s.SetupRoutes()

	srv := httptest.NewServer(mux)
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", srv.URL+"/api/events", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /api/events: %v", err)
	}
	defer resp.Body.Close()

	if resp.Header.Get("Content-Type") != "text/event-stream" {
		t.Errorf("expected text/event-stream, got %q", resp.Header.Get("Content-Type"))
	}

	// Broadcast an event
	go func() {
		time.Sleep(50 * time.Millisecond)
		s.SSEHub.Broadcast(SSEEvent{Type: "CREATE", IssueID: "test-abc"})
	}()

	scanner := bufio.NewScanner(resp.Body)
	var gotEvent bool
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "event: issue_created") {
			gotEvent = true
		}
		if strings.Contains(line, "test-abc") {
			break
		}
	}

	if !gotEvent {
		t.Error("expected to receive issue_created event")
	}
}

func TestHandleSSE_NoHub_Returns503(t *testing.T) {
	s := setupTestServer(t)
	s.SSEHub = nil

	req := httptest.NewRequest("GET", "/api/events", nil)
	w := httptest.NewRecorder()
	s.handleSSE(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}

func TestBroadcastEvent_NilHub(t *testing.T) {
	s := setupTestServer(t)
	s.SSEHub = nil
	// Should not panic
	s.broadcastEvent("UPDATE", "test-123")
}

func TestHandleSSE_StreamsDraftChanged(t *testing.T) {
	s := setupTestServer(t)
	s.SSEHub = NewSSEHub()
	srv := httptest.NewServer(s.SetupRoutes())
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", srv.URL+"/api/events", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /api/events: %v", err)
	}
	defer resp.Body.Close()

	go func() {
		time.Sleep(50 * time.Millisecond)
		s.SSEHub.Broadcast(SSEEvent{Type: "draft_changed", IssueID: "draft-0123abcd"})
	}()

	scanner := bufio.NewScanner(resp.Body)
	var event string
	for scanner.Scan() {
		line := scanner.Text()
		if name, ok := strings.CutPrefix(line, "event: "); ok {
			event = name
		}
		if strings.Contains(line, "draft-0123abcd") {
			break
		}
	}
	if event != "draft_changed" {
		t.Errorf("expected draft_changed event, got %q", event)
	}
}

func TestChatChanged_StreamsChatChanged(t *testing.T) {
	s := setupTestServer(t)
	s.SSEHub = NewSSEHub()
	srv := httptest.NewServer(s.SetupRoutes())
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", srv.URL+"/api/events", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /api/events: %v", err)
	}
	defer resp.Body.Close()

	go func() {
		time.Sleep(50 * time.Millisecond)
		s.ChatChanged("chat-0123abcd")
	}()

	scanner := bufio.NewScanner(resp.Body)
	var event string
	for scanner.Scan() {
		line := scanner.Text()
		if name, ok := strings.CutPrefix(line, "event: "); ok {
			event = name
		}
		if strings.Contains(line, "chat-0123abcd") {
			break
		}
	}
	if event != "chat_changed" {
		t.Errorf("expected chat_changed event, got %q", event)
	}
}

func TestChatChanged_NilHub(t *testing.T) {
	s := setupTestServer(t)
	s.SSEHub = nil
	s.ChatChanged("chat-1") // must not panic
}
