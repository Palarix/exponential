package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/kuyio/beats/internal/beats"
	"github.com/kuyio/beats/internal/model"
)

// IssueResponse represents an issue in API responses.
type IssueResponse struct {
	ID           string                `json:"id"`
	Kind         string                `json:"kind"`
	Title        string                `json:"title"`
	Description  string                `json:"description"`
	Status       string                `json:"status"`
	ParentID     string                `json:"parent_id,omitempty"`
	Estimate     int                   `json:"estimate"`
	LoggedEffort int                   `json:"logged_effort"`
	Labels       []string              `json:"labels"`
	Checklist    []model.ChecklistItem `json:"checklist,omitempty"`
	Dependencies []model.Dependency    `json:"dependencies,omitempty"`
	Comments     []model.Comment       `json:"comments,omitempty"`
	CreatedAt    time.Time             `json:"created_at"`
	UpdatedAt    time.Time             `json:"updated_at"`
	IsPending    bool                  `json:"is_pending"` // True if issue has pending changes
}

// PendingResponse represents the pending state.
type PendingResponse struct {
	Count  int           `json:"count"`
	Events []model.Event `json:"events,omitempty"`
}

// SaveRequest represents a save/sync request.
type SaveRequest struct {
	CommitMessage string `json:"commit_message"`
}

// DraftRequest represents a draft event request.
type DraftRequest struct {
	IssueID string          `json:"issue_id"`
	Type    model.EventType `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// handleGetIssues returns all issues.
func (s *Server) handleGetIssues(w http.ResponseWriter, r *http.Request) {
	issues, err := s.GetProjectedIssues()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Convert to list and sort
	sortedIssues := beats.SortIssues(issues)

	// Convert to response format
	response := make([]IssueResponse, 0, len(sortedIssues))
	for _, issue := range sortedIssues {
		response = append(response, issueToResponse(issue))
	}

	respondJSON(w, http.StatusOK, response)
}

// handleGetIssue returns a single issue by ID.
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

// handleDraft adds an event to the pending buffer.
func (s *Server) handleDraft(w http.ResponseWriter, r *http.Request) {
	var req DraftRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Parse payload based on event type
	var payload interface{}
	switch req.Type {
	case model.EventTypeCreate:
		var p model.CreatePayload
		if err := json.Unmarshal(req.Payload, &p); err != nil {
			respondError(w, http.StatusBadRequest, "invalid create payload")
			return
		}
		payload = p
	case model.EventTypeUpdate:
		var p model.UpdatePayload
		if err := json.Unmarshal(req.Payload, &p); err != nil {
			respondError(w, http.StatusBadRequest, "invalid update payload")
			return
		}
		payload = p
	case model.EventTypeComment:
		var p model.CommentPayload
		if err := json.Unmarshal(req.Payload, &p); err != nil {
			respondError(w, http.StatusBadRequest, "invalid comment payload")
			return
		}
		payload = p
	case model.EventTypeDelete:
		var p model.DeletePayload
		if err := json.Unmarshal(req.Payload, &p); err != nil {
			respondError(w, http.StatusBadRequest, "invalid delete payload")
			return
		}
		payload = p
	default:
		respondError(w, http.StatusBadRequest, "unsupported event type")
		return
	}

	// Generate ID for create events
	eventID := req.IssueID
	if req.Type == model.EventTypeCreate {
		eventID = "beats-" + uuid.New().String()[:6]
	}

	// Get user from config
	client := beats.NewClient(s.Config)
	user := client.GetUser()

	evt := model.Event{
		ID:        eventID,
		Type:      req.Type,
		Payload:   payload,
		CreatedAt: time.Now().UTC(),
		CreatedBy: user,
	}

	s.AddPendingEvent(evt)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success":  true,
		"event_id": eventID,
		"pending":  s.GetPendingCount(),
	})
}

// handleGetPending returns the pending events state.
func (s *Server) handleGetPending(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	respondJSON(w, http.StatusOK, PendingResponse{
		Count:  len(s.pendingEvents),
		Events: s.pendingEvents,
	})
}

// handleSave persists pending events and optionally commits to git.
func (s *Server) handleSave(w http.ResponseWriter, r *http.Request) {
	var req SaveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Default commit message if none provided
		req.CommitMessage = time.Now().Format("Update 2006-01-02 15:04")
	}

	if err := s.SaveAndSync(req.CommitMessage); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Changes saved",
	})
}

// handleDiscardPending clears all pending events.
func (s *Server) handleDiscardPending(w http.ResponseWriter, r *http.Request) {
	s.DiscardPending()

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Pending changes discarded",
	})
}

// issueToResponse converts a model.Issue to an IssueResponse.
func issueToResponse(issue *model.Issue) IssueResponse {
	labels := issue.Labels
	if labels == nil {
		labels = []string{}
	}

	return IssueResponse{
		ID:           issue.ID,
		Kind:         issue.Kind,
		Title:        issue.Title,
		Description:  issue.Description,
		Status:       string(issue.Status),
		ParentID:     issue.ParentID,
		Estimate:     issue.Estimate,
		LoggedEffort: issue.LoggedEffort,
		Labels:       labels,
		Checklist:    issue.Checklist,
		Dependencies: issue.Dependencies,
		Comments:     issue.Comments,
		CreatedAt:    issue.CreatedAt,
		UpdatedAt:    issue.UpdatedAt,
	}
}
