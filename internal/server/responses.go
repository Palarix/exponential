package server

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/palarix/beats/internal/config"
	"github.com/palarix/beats/internal/model"
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

// --- Serialization Helpers ---

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

// isMeaningfulActivityEvent returns true for events that should appear in the
// project-wide activity feed. Filters out sort_order-only UPDATE events (drag-
// reorder noise) and DELETE events.
func isMeaningfulActivityEvent(evt model.Event) bool {
	switch evt.Type {
	case model.EventTypeCreate, model.EventTypeComment:
		return true
	case model.EventTypeUpdate:
		payload, ok := evt.Payload.(map[string]interface{})
		if !ok {
			return true
		}
		if len(payload) == 1 {
			if _, hasOnlySortOrder := payload["sort_order"]; hasOnlySortOrder {
				return false
			}
		}
		return true
	default:
		return false
	}
}
