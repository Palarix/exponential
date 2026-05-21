package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/palarix/beats/internal/beats"
	"github.com/palarix/beats/internal/config"
	"github.com/palarix/beats/internal/model"
	"github.com/palarix/beats/internal/storage"
	"github.com/palarix/beats/internal/version"
)

// --- Response Structures ---

type IssueResponse struct {
	ID           string               `json:"id"`
	Title        string               `json:"title"`
	Description  string               `json:"description"`
	Status       string               `json:"status"`
	ParentID     string               `json:"parent_id,omitempty"`
	Estimate     int                  `json:"estimate"`
	Priority     int                  `json:"priority"`
	SortOrder    string               `json:"sort_order"`
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
	IssueID   string      `json:"issue_id"`
	Type      string      `json:"type"`
	Payload   interface{} `json:"payload,omitempty"`
	CreatedAt time.Time   `json:"created_at"`
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

	client := beats.NewClient(s.Config)

	switch model.EventType(req.Type) {
	case model.EventTypeCreate:
		var p model.CreatePayload
		json.Unmarshal(req.Payload, &p)
		issue, err := client.AddIssue(beats.AddOptions{
			Title:       p.Title,
			Description: p.Description,
			ParentID:    p.ParentID,
			Estimate:    p.Estimate,
			Assignee:    p.Assignee,
			Labels:      p.Labels,
		})
		if err != nil {
			respondError(w, http.StatusInternalServerError, fmt.Sprintf("Error saving: %v", err))
			return
		}
		respondJSON(w, http.StatusOK, map[string]string{"status": "ok", "issue_id": issue.ID})

	case model.EventTypeUpdate:
		var p model.UpdatePayload
		json.Unmarshal(req.Payload, &p)
		if _, err := client.UpdateIssue(req.IssueID, p, "update"); err != nil {
			respondError(w, http.StatusInternalServerError, fmt.Sprintf("Error saving: %v", err))
			return
		}
		respondJSON(w, http.StatusOK, map[string]string{"status": "ok", "issue_id": req.IssueID})

	case model.EventTypeComment:
		var p model.CommentPayload
		json.Unmarshal(req.Payload, &p)
		if err := client.AddComment(req.IssueID, p.Text); err != nil {
			respondError(w, http.StatusInternalServerError, fmt.Sprintf("Error saving: %v", err))
			return
		}
		respondJSON(w, http.StatusOK, map[string]string{"status": "ok", "issue_id": req.IssueID})

	case model.EventTypeDelete:
		var p model.DeletePayload
		json.Unmarshal(req.Payload, &p)
		if err := client.DeleteIssue(req.IssueID, p.Reason); err != nil {
			respondError(w, http.StatusInternalServerError, fmt.Sprintf("Error saving: %v", err))
			return
		}
		respondJSON(w, http.StatusOK, map[string]string{"status": "ok", "issue_id": req.IssueID})

	default:
		respondError(w, http.StatusBadRequest, fmt.Sprintf("Unknown event type: %s", req.Type))
	}
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
			Payload:   evt.Payload,
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
	var req struct {
		Message string `json:"message"`
	}
	json.NewDecoder(r.Body).Decode(&req)

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

	committed := false
	if s.Config.AutoCommit {
		msg := req.Message
		if msg == "" {
			msg = "beats: web UI batch save"
		}
		beats.GitCommit(msg)
		committed = true
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":    "saved",
		"count":     len(pending),
		"committed": committed,
	})
}

func (s *Server) handleDiscardPending(w http.ResponseWriter, r *http.Request) {
	s.DiscardPending()
	respondJSON(w, http.StatusOK, map[string]string{"status": "discarded"})
}

func (s *Server) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	prefix := "beats-"
	if s.Config.Prefix != "" {
		prefix = s.Config.Prefix
	}
	labels := s.Config.Labels
	if labels == nil {
		labels = map[string]string{}
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"auto_commit": s.Config.AutoCommit,
		"prefix":      prefix,
		"version":     version.CLIVersion,
		"labels":      labels,
	})
}

func (s *Server) handleAddLabel(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name  string `json:"name"`
		Color string `json:"color"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if req.Name == "" || req.Color == "" {
		respondError(w, http.StatusBadRequest, "name and color required")
		return
	}

	if err := config.AddLabel(req.Name, req.Color); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if s.Config.Labels == nil {
		s.Config.Labels = make(map[string]string)
	}
	s.Config.Labels[req.Name] = req.Color

	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleGetIssueHistory(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		respondError(w, http.StatusBadRequest, "issue ID required")
		return
	}

	events, err := storage.ReadEvents()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.mu.RLock()
	allEvents := append(events, s.pendingEvents...)
	s.mu.RUnlock()

	var history []map[string]interface{}
	for _, evt := range allEvents {
		if evt.ID != id {
			continue
		}
		history = append(history, map[string]interface{}{
			"type":       evt.Type,
			"payload":    evt.Payload,
			"created_at": evt.CreatedAt,
			"created_by": evt.CreatedBy,
		})
	}

	respondJSON(w, http.StatusOK, history)
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
		Priority:    issue.Priority,
		SortOrder:   issue.SortOrder,
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

	nameBytes, _ := exec.Command("git", "config", "user.name").Output()
	emailBytes, _ := exec.Command("git", "config", "user.email").Output()
	name := strings.TrimSpace(string(nameBytes))
	email := strings.TrimSpace(string(emailBytes))
	if name == "" {
		name = "Unknown"
	}
	if email == "" {
		email = "unknown@example.com"
	}
	return fmt.Sprintf("%s <%s>", name, email)
}
