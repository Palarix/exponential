package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kuyio/beats/internal/model"
)

// addJSONInput is the agent-facing JSON shape for `beats add --json`. Field
// names favour ergonomic short forms (parent, story_points, links) over the
// internal storage names (parent_id, estimate, dependencies).
type addJSONInput struct {
	Title       string      `json:"title"`
	Description string      `json:"description,omitempty"`
	Status      string      `json:"status,omitempty"`
	Parent      string      `json:"parent,omitempty"`
	StoryPoints int         `json:"story_points,omitempty"`
	Priority    int         `json:"priority,omitempty"`
	Assignee    string      `json:"assignee,omitempty"`
	Labels      []string    `json:"labels,omitempty"`
	Links       []linkInput `json:"links,omitempty"`
}

// updateJSONInput uses pointers so omitted fields stay nil (no change).
type updateJSONInput struct {
	Title       *string     `json:"title,omitempty"`
	Description *string     `json:"description,omitempty"`
	Status      *string     `json:"status,omitempty"`
	Parent      *string     `json:"parent,omitempty"`
	StoryPoints *int        `json:"story_points,omitempty"`
	Priority    *int        `json:"priority,omitempty"`
	Assignee    *string     `json:"assignee,omitempty"`
	Labels      []string    `json:"labels,omitempty"`
	Links       []linkInput `json:"links,omitempty"`
}

type commentJSONInput struct {
	Body string `json:"body"`
}

type linkInput struct {
	Target string `json:"target"`
	Type   string `json:"type"`
}

// decodeStrict unmarshals JSON, rejecting unknown fields so typos surface
// instead of silently dropping data.
func decodeStrict(content string, dst interface{}) error {
	dec := json.NewDecoder(strings.NewReader(content))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return fmt.Errorf("invalid JSON payload: %w", err)
	}
	return nil
}

func validateStatus(s string) error {
	switch model.IssueStatus(s) {
	case model.StatusBacklog, model.StatusPlanned, model.StatusDoing, model.StatusBlocked, model.StatusDone:
		return nil
	}
	return fmt.Errorf("invalid status %q: must be one of BACKLOG, PLANNED, DOING, BLOCKED, DONE", s)
}

func linksToDependencies(links []linkInput) ([]model.Dependency, error) {
	if len(links) == 0 {
		return nil, nil
	}
	deps := make([]model.Dependency, 0, len(links))
	for i, l := range links {
		if l.Target == "" {
			return nil, fmt.Errorf("links[%d]: 'target' is required", i)
		}
		kind := normalizeDependencyKind(l.Type)
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

func (in addJSONInput) toCreatePayload() (model.CreatePayload, error) {
	if in.Title == "" {
		return model.CreatePayload{}, fmt.Errorf("'title' is required")
	}
	if in.Status != "" {
		if err := validateStatus(in.Status); err != nil {
			return model.CreatePayload{}, err
		}
	}
	deps, err := linksToDependencies(in.Links)
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
		Labels:       in.Labels,
		Dependencies: deps,
	}, nil
}

func (in updateJSONInput) toUpdatePayload() (model.UpdatePayload, error) {
	if in.Status != nil {
		if err := validateStatus(*in.Status); err != nil {
			return model.UpdatePayload{}, err
		}
	}
	deps, err := linksToDependencies(in.Links)
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
		Labels:       in.Labels,
		Dependencies: deps,
	}, nil
}

func updatePayloadEmpty(p model.UpdatePayload) bool {
	return p.Title == nil && p.Description == nil && p.Status == nil &&
		p.ParentID == nil && p.Estimate == nil && p.Priority == nil &&
		p.Assignee == nil && p.Labels == nil && p.Dependencies == nil
}
