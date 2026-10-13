package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/palarix/exponential/internal/chat"
)

// ChatService is the board's agent chat. The board runs the agent on the
// viewer's machine, so the chat routes live under /api/local/ and are
// answered by this process in local and proxy mode alike.
type ChatService struct {
	Store   *chat.Store
	Runner  *chat.Runner
	Catalog *chat.Catalog
}

func (s *Server) registerChatRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/local/agent", s.handleAgentStatus)
	if s.Chat == nil {
		return
	}
	mux.HandleFunc("GET /api/local/chats", s.handleListChats)
	mux.HandleFunc("POST /api/local/chats", s.handleCreateChat)
	mux.HandleFunc("GET /api/local/chats/{id}", s.handleGetChat)
	mux.HandleFunc("PATCH /api/local/chats/{id}", s.handleRenameChat)
	mux.HandleFunc("DELETE /api/local/chats/{id}", s.handleDeleteChat)
	mux.HandleFunc("POST /api/local/chats/{id}/messages", s.handleSendChatMessage)
	mux.HandleFunc("POST /api/local/chats/{id}/cancel", s.handleCancelChat)
	mux.HandleFunc("POST /api/local/chats/{id}/approvals/{approval}", s.handleDecideApproval)
	mux.HandleFunc("GET /api/local/chats/{id}/stream", s.handleChatStream)
}

func (s *Server) handleAgentStatus(w http.ResponseWriter, r *http.Request) {
	if s.Chat == nil {
		respondJSON(w, http.StatusOK, chat.AgentStatus{Reason: "Chat is not enabled on this server."})
		return
	}
	respondJSON(w, http.StatusOK, s.Chat.Catalog.Status())
}

func (s *Server) handleListChats(w http.ResponseWriter, r *http.Request) {
	list, err := s.Chat.Store.List()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	for i := range list {
		list[i].Waiting = s.Chat.Runner.Waiting(list[i].ID)
	}
	respondJSON(w, http.StatusOK, list)
}

func (s *Server) handleCreateChat(w http.ResponseWriter, r *http.Request) {
	m, err := s.Chat.Store.Create()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusCreated, m)
}

func (s *Server) handleGetChat(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	meta, events, err := s.Chat.Store.Get(id)
	if s.respondChatError(w, err) {
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"meta":    meta,
		"events":  events,
		"running": s.Chat.Runner.Running(id),
	})
}

func (s *Server) handleRenameChat(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title string `json:"title"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	meta, err := s.Chat.Store.Rename(r.PathValue("id"), req.Title)
	if s.respondChatError(w, err) {
		return
	}
	respondJSON(w, http.StatusOK, meta)
}

// handleDeleteChat stops the chat's running turn first, so no turn writes to
// the log after it is gone.
func (s *Server) handleDeleteChat(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := s.Chat.Store.Meta(id); s.respondChatError(w, err) {
		return
	}
	s.Chat.Runner.CancelAndWait(id)
	if err := s.Chat.Store.Delete(id); s.respondChatError(w, err) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type sendChatRequest struct {
	Text     string        `json:"text"`
	Context  *chat.Context `json:"context"`
	Provider string        `json:"provider"` // empty means the default provider
	Model    string        `json:"model"`
	Effort   string        `json:"effort"`
}

func (s *Server) handleSendChatMessage(w http.ResponseWriter, r *http.Request) {
	if st := s.Chat.Catalog.Status(); !st.Available {
		respondError(w, http.StatusServiceUnavailable, st.Reason)
		return
	}
	var req sendChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if strings.TrimSpace(req.Text) == "" {
		respondError(w, http.StatusBadRequest, "text is required")
		return
	}
	provider, err := s.Chat.Catalog.Resolve(req.Provider, req.Model, req.Effort)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	err = s.Chat.Runner.Send(r.PathValue("id"), chat.SendRequest{
		Text:     req.Text,
		Context:  req.Context,
		Provider: provider,
		Model:    req.Model,
		Effort:   req.Effort,
	})
	if s.respondChatError(w, err) {
		return
	}
	respondJSON(w, http.StatusAccepted, map[string]bool{"ok": true})
}

func (s *Server) handleCancelChat(w http.ResponseWriter, r *http.Request) {
	if !s.Chat.Runner.Cancel(r.PathValue("id")) {
		respondError(w, http.StatusConflict, "no reply is running")
		return
	}
	respondJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// handleDecideApproval answers an open approval card.
func (s *Server) handleDecideApproval(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Decision string `json:"decision"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	err := s.Chat.Runner.Decide(r.PathValue("id"), r.PathValue("approval"), req.Decision)
	if s.respondChatError(w, err) {
		return
	}
	respondJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// handleChatStream sends a chat's events as SSE: first the settled events
// after Last-Event-ID (or ?after=), then live events until the client goes.
// Settled events carry their Seq as the SSE id, so a reconnecting
// EventSource resumes without duplicates.
func (s *Server) handleChatStream(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := s.Chat.Store.Meta(id); s.respondChatError(w, err) {
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	after := r.Header.Get("Last-Event-ID")
	if after == "" {
		after = r.URL.Query().Get("after")
	}
	last, _ := strconv.Atoi(after)

	// Subscribe before reading the log, so no event falls between the two.
	ch, unsubscribe := s.Chat.Runner.Subscribe(id)
	defer unsubscribe()
	_, events, err := s.Chat.Store.Get(id)
	if s.respondChatError(w, err) {
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	send := func(e chat.Event) {
		if e.Seq > 0 {
			if e.Seq <= last {
				return
			}
			last = e.Seq
			fmt.Fprintf(w, "id: %d\n", e.Seq)
		}
		data, _ := json.Marshal(e)
		fmt.Fprintf(w, "data: %s\n\n", data)
	}
	for _, e := range events {
		send(e)
	}
	flusher.Flush()

	keepalive := time.NewTicker(30 * time.Second)
	defer keepalive.Stop()
	for {
		select {
		case e := <-ch:
			send(e)
			flusher.Flush()
		case <-keepalive.C:
			fmt.Fprint(w, ": keepalive\n\n")
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

// respondChatError writes the response for a chat error and reports whether
// there was one.
func (s *Server) respondChatError(w http.ResponseWriter, err error) bool {
	switch {
	case err == nil:
		return false
	case errors.Is(err, chat.ErrNotFound):
		respondError(w, http.StatusNotFound, "chat not found")
	case errors.Is(err, chat.ErrApprovalNotFound):
		respondError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, chat.ErrEmptyTitle), errors.Is(err, chat.ErrBadDecision):
		respondError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, chat.ErrBusy), errors.Is(err, chat.ErrProviderLocked):
		respondError(w, http.StatusConflict, err.Error())
	default:
		respondError(w, http.StatusInternalServerError, err.Error())
	}
	return true
}
