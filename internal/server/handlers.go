package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/palarix/beats/internal/beats"
	"github.com/palarix/beats/internal/config"
	"github.com/palarix/beats/internal/model"
	"github.com/palarix/beats/internal/storage"
)

// --- Response Structures ---

type IssueResponse struct {
	ID           string               `json:"id"`
	Title        string               `json:"title"`
	Description  string               `json:"description"`
	Status       string               `json:"status"`
	ParentID     string               `json:"parent_id,omitempty"`
	Estimate     int                  `json:"estimate"`
	Assignee     string               `json:"assignee,omitempty"`
	Labels       []string             `json:"labels,omitempty"`
	Dependencies []DependencyResponse `json:"dependencies,omitempty"`
	Comments     []CommentResponse    `json:"comments,omitempty"`
	CreatedAt    time.Time            `json:"created_at"`
	CreatedBy    string               `json:"created_by"`
	UpdatedAt    time.Time            `json:"updated_at"`
	IsPending    bool                 `json:"is_pending"`
}

type DependencyResponse struct {
	SourceID string `json:"source_id"`
	TargetID string `json:"target_id"`
	Kind     string `json:"kind"`
}

type CommentResponse struct {
	ID        string    `json:"id"`
	Text      string    `json:"text"`
	CreatedBy string    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

type PendingResponse struct {
	HasPending bool                `json:"has_pending"`
	Events     []PendingEventEntry `json:"events"`
	IssueIDs   []string            `json:"issue_ids"`
}

type PendingEventEntry struct {
	IssueID   string    `json:"issue_id"`
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"created_at"`
}

// --- Handler Methods ---

func (s *Server) handleGetIssues(w http.ResponseWriter, r *http.Request) {
	issues, err := s.GetProjectedIssues()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	sorted := beats.SortIssues(issues)

	// Get pending issue IDs
	s.mu.RLock()
	pendingIDs := make(map[string]bool)
	for _, evt := range s.pendingEvents {
		pendingIDs[evt.ID] = true
	}
	s.mu.RUnlock()

	var response []IssueResponse
	for _, issue := range sorted {
		resp := issueToResponse(issue)
		if pendingIDs[issue.ID] {
			resp.IsPending = true
		}
		response = append(response, resp)
	}

	respondJSON(w, http.StatusOK, response)
}

func (s *Server) handleGetIssue(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		respondError(w, http.StatusBadRequest, "issue ID required")
		return
	}

	issues, err := s.GetProjectedIssues()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	issue, exists := issues[id]
	if !exists {
		respondError(w, http.StatusNotFound, "issue not found")
		return
	}

	respondJSON(w, http.StatusOK, issueToResponse(issue))
}

func (s *Server) handleDraft(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IssueID string          `json:"issue_id"`
		Type    string          `json:"type"`
		Payload json.RawMessage `json:"payload"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	user := getUser(s.Config)

	var payload interface{}
	switch model.EventType(req.Type) {
	case model.EventTypeUpdate:
		var p model.UpdatePayload
		json.Unmarshal(req.Payload, &p)
		payload = p
	case model.EventTypeComment:
		var p model.CommentPayload
		json.Unmarshal(req.Payload, &p)
		payload = p
	case model.EventTypeDelete:
		var p model.DeletePayload
		json.Unmarshal(req.Payload, &p)
		payload = p
	default:
		respondError(w, http.StatusBadRequest, fmt.Sprintf("Unknown event type: %s", req.Type))
		return
	}

	evt := model.Event{
		ID:        req.IssueID,
		Type:      model.EventType(req.Type),
		Payload:   payload,
		CreatedAt: time.Now().UTC(),
		CreatedBy: user,
	}

	s.AddPendingEvent(evt)
	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleGetPending(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	pending := s.pendingEvents
	s.mu.RUnlock()

	resp := PendingResponse{
		HasPending: len(pending) > 0,
	}

	issueIDSet := make(map[string]bool)
	for _, evt := range pending {
		resp.Events = append(resp.Events, PendingEventEntry{
			IssueID:   evt.ID,
			Type:      string(evt.Type),
			CreatedAt: evt.CreatedAt,
		})
		issueIDSet[evt.ID] = true
	}
	for id := range issueIDSet {
		resp.IssueIDs = append(resp.IssueIDs, id)
	}

	respondJSON(w, http.StatusOK, resp)
}

func (s *Server) handleSave(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	pending := s.pendingEvents
	s.mu.Unlock()

	if len(pending) == 0 {
		respondJSON(w, http.StatusOK, map[string]string{"status": "no_changes"})
		return
	}

	for _, evt := range pending {
		if err := storage.AppendEvent(evt); err != nil {
			respondError(w, http.StatusInternalServerError, fmt.Sprintf("Error saving: %v", err))
			return
		}
	}

	s.DiscardPending()

	if s.Config.AutoCommit {
		beats.GitCommit("beats: web UI batch save")
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "saved", "count": fmt.Sprintf("%d", len(pending))})
}

func (s *Server) handleDiscardPending(w http.ResponseWriter, r *http.Request) {
	s.DiscardPending()
	respondJSON(w, http.StatusOK, map[string]string{"status": "discarded"})
}

// --- Helpers ---

func issueToResponse(issue *model.Issue) IssueResponse {
	resp := IssueResponse{
		ID:          issue.ID,
		Title:       issue.Title,
		Description: issue.Description,
		Status:      string(issue.Status),
		ParentID:    issue.ParentID,
		Estimate:    issue.Estimate,
		Assignee:    issue.Assignee,
		Labels:      issue.Labels,
		CreatedAt:   issue.CreatedAt,
		CreatedBy:   issue.CreatedBy,
		UpdatedAt:   issue.UpdatedAt,
	}

	for _, dep := range issue.Dependencies {
		resp.Dependencies = append(resp.Dependencies, DependencyResponse{
			SourceID: dep.SourceID,
			TargetID: dep.TargetID,
			Kind:     string(dep.Kind),
		})
	}

	for _, c := range issue.Comments {
		resp.Comments = append(resp.Comments, CommentResponse{
			ID:        c.ID,
			Text:      c.Text,
			CreatedBy: c.CreatedBy,
			CreatedAt: c.CreatedAt,
		})
	}

	return resp
}

func getUser(cfg *config.Config) string {
	if cfg != nil && cfg.User != "" {
		return cfg.User
	}
	return "Web User <web@beats>"
}
