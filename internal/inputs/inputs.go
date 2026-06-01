// Package inputs defines the agent-facing input types shared by the beats
// CLI (--json mode) and the MCP server. Both transports decode user input
// into these structs and then call into the internal/beats service layer,
// so there is exactly one validation/conversion path regardless of transport.
package inputs

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kuyio/beats/internal/model"
)

// AddInput is the agent-facing payload for creating an issue. JSON field
// names favour ergonomic short forms (parent, story_points, links) over the
// internal storage names (parent_id, estimate, dependencies). The
// jsonschema tags are consumed by the MCP SDK to generate tool input
// schemas.
type AddInput struct {
	Title       string      `json:"title" jsonschema:"Issue title (required)"`
	Description string      `json:"description,omitempty" jsonschema:"Markdown description; multi-line and special characters supported"`
	Status      string      `json:"status,omitempty" jsonschema:"Initial status: BACKLOG, PLANNED, DOING, BLOCKED, or DONE (default BACKLOG)"`
	Parent      string      `json:"parent,omitempty" jsonschema:"Parent issue ID for sub-issue relationships"`
	StoryPoints int         `json:"story_points,omitempty" jsonschema:"Effort estimate in story points"`
	Priority    int         `json:"priority,omitempty"`
	Assignee    string      `json:"assignee,omitempty" jsonschema:"Assignee in 'Name <email>' format"`
	Labels      []string    `json:"labels,omitempty" jsonschema:"Labels such as feature, bug, epic"`
	CycleID     string      `json:"cycle_id,omitempty" jsonschema:"Cycle ID (YYYY-MM-DD start date) to assign this issue to"`
	Links       []LinkInput `json:"links,omitempty" jsonschema:"Dependencies/relationships to other issues, set at creation time"`
}

// UpdateInput is a partial patch. Pointer fields distinguish 'not provided'
// (nil — no change) from 'cleared' (pointer to empty value).
type UpdateInput struct {
	Title       *string     `json:"title,omitempty"`
	Description *string     `json:"description,omitempty" jsonschema:"Replace the issue description"`
	Status      *string     `json:"status,omitempty" jsonschema:"New status: BACKLOG, PLANNED, DOING, BLOCKED, or DONE"`
	Parent      *string     `json:"parent,omitempty"`
	StoryPoints *int        `json:"story_points,omitempty"`
	Priority    *int        `json:"priority,omitempty"`
	Assignee    *string     `json:"assignee,omitempty"`
	Labels      []string    `json:"labels,omitempty" jsonschema:"Replace the full label list"`
	CycleID     *string     `json:"cycle_id,omitempty" jsonschema:"Cycle ID (YYYY-MM-DD start date) to assign this issue to"`
	Links       []LinkInput `json:"links,omitempty" jsonschema:"Replace the full dependency list"`
}

// CommentInput is the agent-facing payload for adding a comment.
type CommentInput struct {
	Body string `json:"body" jsonschema:"Comment text in markdown"`
}

// LinkInput describes one dependency/relationship target. The source side is
// implicit (the issue being created or updated).
type LinkInput struct {
	Target string `json:"target" jsonschema:"Target issue ID"`
	Type   string `json:"type" jsonschema:"Relationship type: blocks, blocked_by, depends_on, dependency_of, duplicates, duplicated_by, relates_to"`
}

// DecodeStrict unmarshals JSON, rejecting unknown fields so typos surface
// instead of silently dropping data.
func DecodeStrict(content string, dst interface{}) error {
	dec := json.NewDecoder(strings.NewReader(content))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return fmt.Errorf("invalid JSON payload: %w", err)
	}
	return nil
}

// ValidateStatus returns nil if s is one of the five known issue statuses.
func ValidateStatus(s string) error {
	switch model.IssueStatus(s) {
	case model.StatusBacklog, model.StatusPlanned, model.StatusDoing, model.StatusBlocked, model.StatusDone:
		return nil
	}
	return fmt.Errorf("invalid status %q: must be one of BACKLOG, PLANNED, DOING, BLOCKED, DONE", s)
}

// LinksToDependencies converts agent-facing LinkInputs to internal
// model.Dependency values, normalizing the type string.
func LinksToDependencies(links []LinkInput) ([]model.Dependency, error) {
	if len(links) == 0 {
		return nil, nil
	}
	deps := make([]model.Dependency, 0, len(links))
	for i, l := range links {
		if l.Target == "" {
			return nil, fmt.Errorf("links[%d]: 'target' is required", i)
		}
		kind := model.NormalizeDependencyKind(l.Type)
		if kind == "" {
			return nil, fmt.Errorf("links[%d]: invalid type %q", i, l.Type)
		}
		deps = append(deps, model.Dependency{
			TargetID: l.Target,
			Kind:     model.DependencyKind(kind),
		})
	}
	return deps, nil
}

// ToCreatePayload validates and converts to the internal CreatePayload type.
func (in AddInput) ToCreatePayload() (model.CreatePayload, error) {
	if in.Title == "" {
		return model.CreatePayload{}, fmt.Errorf("'title' is required")
	}
	if in.Status != "" {
		if err := ValidateStatus(in.Status); err != nil {
			return model.CreatePayload{}, err
		}
	}
	deps, err := LinksToDependencies(in.Links)
	if err != nil {
		return model.CreatePayload{}, err
	}
	return model.CreatePayload{
		Title:        in.Title,
		Description:  in.Description,
		Status:       in.Status,
		ParentID:     in.Parent,
		Estimate:     in.StoryPoints,
		Priority:     in.Priority,
		Assignee:     in.Assignee,
		CycleID:      in.CycleID,
		Labels:       in.Labels,
		Dependencies: deps,
	}, nil
}

// ToUpdatePayload validates and converts to the internal UpdatePayload type.
func (in UpdateInput) ToUpdatePayload() (model.UpdatePayload, error) {
	if in.Status != nil {
		if err := ValidateStatus(*in.Status); err != nil {
			return model.UpdatePayload{}, err
		}
	}
	deps, err := LinksToDependencies(in.Links)
	if err != nil {
		return model.UpdatePayload{}, err
	}
	return model.UpdatePayload{
		Title:        in.Title,
		Description:  in.Description,
		Status:       in.Status,
		ParentID:     in.Parent,
		Estimate:     in.StoryPoints,
		Priority:     in.Priority,
		Assignee:     in.Assignee,
		CycleID:      in.CycleID,
		Labels:       in.Labels,
		Dependencies: deps,
	}, nil
}

// UpdatePayloadEmpty reports whether every field of p is unset (a no-op
// patch). Used by both CLI and MCP write paths to reject empty updates
// rather than wasting an event.
func UpdatePayloadEmpty(p model.UpdatePayload) bool {
	return p.Title == nil && p.Description == nil && p.Status == nil &&
		p.ParentID == nil && p.Estimate == nil && p.Priority == nil &&
		p.Assignee == nil && p.CycleID == nil && p.Labels == nil && p.Dependencies == nil
}
