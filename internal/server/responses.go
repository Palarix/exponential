package server

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/palarix/exponential/internal/config"
	"github.com/palarix/exponential/internal/exponential"
	"github.com/palarix/exponential/internal/model"
)

// --- Response Structures ---

type IssueResponse struct {
	ID           string               `json:"id"`
	Title        string               `json:"title"`
	Description  string               `json:"description"`
	Status       string               `json:"status"`
	IsInferred   bool                 `json:"is_inferred,omitempty"`
	ParentID     string               `json:"parent_id,omitempty"`
	Estimate     int                  `json:"estimate"`
	Priority     int                  `json:"priority"`
	SortOrder    string               `json:"sort_order"`
	Assignee     string               `json:"assignee,omitempty"`
	CycleID          string               `json:"cycle_id,omitempty"`
	EffectiveCycleID string               `json:"effective_cycle_id,omitempty"`
	BranchStats      *model.BranchStats   `json:"branch_stats,omitempty"`
	Labels           []string             `json:"labels,omitempty"`
	Dependencies []DependencyResponse `json:"dependencies,omitempty"`
	Artifacts    []ArtifactResponse   `json:"artifacts,omitempty"`
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

type ArtifactResponse struct {
	ArtifactType string    `json:"artifact_type"`
	Filename     string    `json:"filename"`
	UpdatedAt    time.Time `json:"updated_at"`
	UpdatedBy    string    `json:"updated_by"`
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
		IsInferred:  issue.InferredStatus,
		ParentID:    issue.ParentID,
		Estimate:    issue.Estimate,
		Priority:    issue.Priority,
		SortOrder:   issue.SortOrder,
		Assignee:    issue.Assignee,
		CycleID:          issue.CycleID,
		EffectiveCycleID: issue.EffectiveCycleID,
		BranchStats:      issue.BranchStats,
		Labels:           issue.Labels,
		CreatedAt:   issue.CreatedAt,
		CreatedBy:   issue.CreatedBy,
		UpdatedAt:   issue.UpdatedAt,
	}

	for _, a := range issue.Artifacts {
		resp.Artifacts = append(resp.Artifacts, ArtifactResponse{
			ArtifactType: a.ArtifactType,
			Filename:     a.Filename,
			UpdatedAt:    a.UpdatedAt,
			UpdatedBy:    a.UpdatedBy,
		})
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

func getUserNameEmail(cfg *config.Config) (string, string) {
	raw := getUser(cfg)
	if idx := strings.Index(raw, " <"); idx != -1 {
		name := raw[:idx]
		email := strings.TrimSuffix(raw[idx+2:], ">")
		return name, email
	}
	return raw, ""
}

// isMeaningfulActivityEvent delegates to the exported version in package
// exponential so the same filter is shared with BuildTimeline.
func isMeaningfulActivityEvent(evt model.Event) bool {
	return exponential.IsMeaningfulActivityEvent(evt)
}
