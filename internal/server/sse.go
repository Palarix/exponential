package server

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

// SSEEvent is a lightweight notification sent to connected clients.
type SSEEvent struct {
	Type    string `json:"type"`
	IssueID string `json:"issue_id"`
}

// SSEHub manages connected SSE clients and broadcasts events.
type SSEHub struct {
	mu      sync.Mutex
	clients map[chan SSEEvent]struct{}
}

// NewSSEHub creates a new hub.
func NewSSEHub() *SSEHub {
	return &SSEHub{
		clients: make(map[chan SSEEvent]struct{}),
	}
}

func (h *SSEHub) subscribe() chan SSEEvent {
	ch := make(chan SSEEvent, 16)
	h.mu.Lock()
	h.clients[ch] = struct{}{}
	h.mu.Unlock()
	return ch
}

func (h *SSEHub) unsubscribe(ch chan SSEEvent) {
	h.mu.Lock()
	delete(h.clients, ch)
	h.mu.Unlock()
	close(ch)
}

// Broadcast sends an event to all connected clients.
func (h *SSEHub) Broadcast(evt SSEEvent) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.clients {
		select {
		case ch <- evt:
		default:
			// Drop if client is slow
		}
	}
}

// ClientCount returns the number of connected clients.
func (h *SSEHub) ClientCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.clients)
}

func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	if s.SSEHub == nil {
		http.Error(w, "SSE not enabled", http.StatusServiceUnavailable)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	flusher.Flush()

	ch := s.SSEHub.subscribe()
	defer s.SSEHub.unsubscribe(ch)

	keepalive := time.NewTicker(30 * time.Second)
	defer keepalive.Stop()

	for {
		select {
		case evt, ok := <-ch:
			if !ok {
				return
			}
			eventName := "issue_updated"
			switch evt.Type {
			case "CREATE":
				eventName = "issue_created"
			case "DELETE":
				eventName = "issue_deleted"
			case "MERGE":
				eventName = "issue_merged"
			case "COMMENT":
				eventName = "issue_commented"
			}
			fmt.Fprintf(w, "event: %s\ndata: {\"issue_id\":%q,\"type\":%q,\"timestamp\":%q}\n\n",
				eventName, evt.IssueID, evt.Type, time.Now().UTC().Format(time.RFC3339))
			flusher.Flush()

		case <-keepalive.C:
			fmt.Fprintf(w, ": keepalive\n\n")
			flusher.Flush()

		case <-r.Context().Done():
			return
		}
	}
}
