package server

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/palarix/exponential/internal/chat"
	"github.com/palarix/exponential/internal/live"
)

// The topics of the board's event stream. See internal/live.
const (
	TopicIssues = "issues" // issues, git refs, the DB watch
	TopicDrafts = "drafts" // issue drafts
	TopicChats  = "chats"  // the chat list
	logChat     = "chat"   // chat:<id>, one chat's events
)

// remoteTopics are the topics a remote server owns in proxy mode.
var remoteTopics = []string{TopicIssues, TopicDrafts}

// NewLiveHub returns the hub of the event stream with its signal topics.
// The chat log topic is registered with the chat routes.
func NewLiveHub() *live.Hub {
	h := live.NewHub()
	h.Signal(TopicIssues)
	h.Signal(TopicDrafts)
	h.Signal(TopicChats)
	return h
}

// ChatEventsTo returns a chat.RunnerConfig.OnEvent that appends each chat
// event to its chat:<id> topic on h.
func ChatEventsTo(h *live.Hub) func(chatID string, e chat.Event) {
	return func(chatID string, e chat.Event) {
		h.Append(logChat, chatID, live.Frame{Seq: e.Seq, Data: e})
	}
}

// notify tells open streams that a signal topic changed, if the stream is
// enabled.
func (s *Server) notify(topic string) {
	if s.Live != nil {
		s.Live.Notify(topic)
	}
}

// Notify is notify for callers outside the package, such as MCP tool
// handlers.
func (s *Server) Notify(topic string) { s.notify(topic) }

// registerLiveRoutes serves the event stream: GET opens it, POST changes
// its topics. In proxy mode the stream is still this board's; the remote's
// topics arrive through the mirror.
func (s *Server) registerLiveRoutes(mux *http.ServeMux, handle func(string, http.HandlerFunc)) {
	if s.Live == nil {
		return
	}
	if s.Chat != nil && !s.Headless {
		s.Live.Log(logChat, s.replayChat)
	}
	st := &live.Stream{Hub: s.Live}
	if s.ProxyURL == "" {
		handle("GET /api/events", st.ServeEvents)
		handle("POST /api/events/{stream}/topics", st.ServeTopics)
		return
	}
	if !s.Headless {
		// More specific than the proxy's catch-all.
		mux.HandleFunc("GET /api/events", st.ServeEvents)
		mux.HandleFunc("POST /api/events/{stream}/topics", st.ServeTopics)
	}
}

// replayChat returns a chat's stored events after seq after.
func (s *Server) replayChat(id string, after int) ([]live.Frame, error) {
	_, events, err := s.Chat.Store.Get(id)
	if errors.Is(err, chat.ErrNotFound) {
		return nil, live.ErrGone
	}
	if err != nil {
		return nil, err
	}
	var frames []live.Frame
	for _, e := range events {
		if e.Seq > after {
			frames = append(frames, live.Frame{Seq: e.Seq, Data: e})
		}
	}
	return frames, nil
}

// startMirror follows the remote's event stream in proxy mode, until ctx
// ends, and notifies its topics here.
func (s *Server) startMirror(ctx context.Context) {
	if s.Live == nil || s.ProxyURL == "" {
		return
	}
	m := &live.Mirror{
		Hub:    s.Live,
		URL:    strings.TrimSuffix(s.ProxyURL, "/") + "/api/events",
		Header: http.Header{},
		Topics: remoteTopics,
	}
	if s.ProxyToken != "" {
		m.Header.Set("Authorization", "Bearer "+s.ProxyToken)
	}
	go m.Run(ctx)
}
