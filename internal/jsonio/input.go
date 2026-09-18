package jsonio

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/palarix/exponential/internal/model"
)

// --- Shared payload types (used by both CLI --json and MCP tools) ---

type AddInput struct {
	Title       string      `json:"title" jsonschema:"Issue title (required)"`
	Description string      `json:"description,omitempty" jsonschema:"Markdown description; multi-line and special characters supported"`
	Status      string      `json:"status,omitempty" jsonschema:"Initial status: BACKLOG, PLANNED, DOING, BLOCKED, DONE, CANCELED, or DUPLICATE (default BACKLOG)"`
	Parent      string      `json:"parent,omitempty" jsonschema:"Parent issue ID for sub-issue relationships"`
	StoryPoints int         `json:"story_points,omitempty" jsonschema:"Effort estimate in story points"`
	Priority    int         `json:"priority,omitempty"`
	Assignee    string      `json:"assignee,omitempty" jsonschema:"Assignee in 'Name <email>' format"`
	Labels      []string    `json:"labels,omitempty" jsonschema:"Labels such as feature, bug, epic"`
	CycleID     string      `json:"cycle_id,omitempty" jsonschema:"Cycle ID (YYYY-MM-DD start date) to assign this issue to"`
	Links       []LinkInput `json:"links,omitempty" jsonschema:"Dependencies/relationships to other issues, set at creation time"`
}

type UpdateInput struct {
	Title       *string     `json:"title,omitempty"`
	Description *string     `json:"description,omitempty" jsonschema:"Replace the issue description"`
	Status      *string     `json:"status,omitempty" jsonschema:"New status: BACKLOG, PLANNED, DOING, BLOCKED, DONE, CANCELED, or DUPLICATE"`
	Parent      *string     `json:"parent,omitempty"`
	StoryPoints *int        `json:"story_points,omitempty"`
	Priority    *int        `json:"priority,omitempty"`
	Assignee    *string     `json:"assignee,omitempty"`
	Labels      []string    `json:"labels,omitempty" jsonschema:"Replace the full label list"`
	CycleID     *string     `json:"cycle_id,omitempty" jsonschema:"Cycle ID (YYYY-MM-DD start date) to assign this issue to"`
	Links       []LinkInput `json:"links,omitempty" jsonschema:"Replace the full dependency list"`
}

type CommentInput struct {
	Body string `json:"body" jsonschema:"Comment text in markdown"`
}

type LinkInput struct {
	Target string `json:"target" jsonschema:"Target issue ID"`
	Type   string `json:"type" jsonschema:"Relationship type: blocks, blocked_by, depends_on, dependency_of, duplicates, duplicated_by, relates_to"`
}

// --- Validation and conversion helpers ---

func DecodeStrict(content string, dst interface{}) error {
	dec := json.NewDecoder(strings.NewReader(content))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return fmt.Errorf("invalid JSON payload: %w", err)
	}
	return nil
}

func ValidateStatus(s string) error {
	switch model.IssueStatus(s) {
	case model.StatusBacklog, model.StatusPlanned, model.StatusDoing, model.StatusBlocked,
		model.StatusDone, model.StatusCanceled, model.StatusDuplicate:
		return nil
	}
	return fmt.Errorf("invalid status %q: must be one of BACKLOG, PLANNED, DOING, BLOCKED, DONE, CANCELED, DUPLICATE", s)
}

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

func UpdatePayloadEmpty(p model.UpdatePayload) bool {
	return p.Title == nil && p.Description == nil && p.Status == nil &&
		p.ParentID == nil && p.Estimate == nil && p.Priority == nil &&
		p.Assignee == nil && p.CycleID == nil && p.Labels == nil && p.Dependencies == nil
}

// --- MCP tool parameter types (add ID or other tool-specific fields) ---

type ListToolInput struct {
	Status      FlexStrings `json:"status,omitempty" jsonschema:"Filter by status: BACKLOG, PLANNED, DOING, BLOCKED, DONE"`
	Label       string      `json:"label,omitempty" jsonschema:"Substring match on a label"`
	Assignee    string      `json:"assignee,omitempty" jsonschema:"Substring match on assignee"`
	Parent      string      `json:"parent,omitempty" jsonschema:"Only return sub-issues of this parent ID"`
	Match       string      `json:"match,omitempty" jsonschema:"Substring match across id, title, status, parent, assignee, labels"`
	IncludeDone bool        `json:"include_done,omitempty" jsonschema:"Include DONE issues (otherwise only recent DONE are shown)"`
}

// FlexStrings accepts either a single string or an array of strings in JSON.
type FlexStrings []string

func (f *FlexStrings) UnmarshalJSON(data []byte) error {
	var arr []string
	if err := json.Unmarshal(data, &arr); err == nil {
		*f = arr
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	*f = []string{s}
	return nil
}

type ShowToolInput struct {
	ID                 string `json:"id" jsonschema:"Issue ID (full or unique suffix)"`
	IncludeEvents      bool   `json:"include_events,omitempty" jsonschema:"Include the full event audit trail"`
	IncludeSpec        bool   `json:"include_spec,omitempty" jsonschema:"Include spec content"`
	IncludeWalkthrough bool   `json:"include_walkthrough,omitempty" jsonschema:"Include walkthrough content"`
}

type HistoryToolInput struct {
	ID string `json:"id" jsonschema:"Issue ID"`
}

type UpdateToolInput struct {
	ID string `json:"id" jsonschema:"Issue ID to update"`
	UpdateInput
}

type CommentToolInput struct {
	ID   string `json:"id" jsonschema:"Issue ID to comment on"`
	Body string `json:"body" jsonschema:"Comment text in markdown"`
}

type StartToolInput struct {
	ID    string `json:"id" jsonschema:"Issue ID to start working on"`
	Force bool   `json:"force,omitempty" jsonschema:"Force take-over if already in progress or branch exists"`
	Mode  string `json:"mode,omitempty" jsonschema:"Create a worktree or a branch, overriding the global config"`
}

type MergeToolInput struct {
	ID            string `json:"id" jsonschema:"Issue ID to merge"`
	Strategy      string `json:"strategy,omitempty" jsonschema:"Merge strategy: squash (default), merge, or ff"`
	CommitMessage string `json:"commit_message,omitempty" jsonschema:"Custom commit message (auto-generated if omitted)"`
	KeepBranch    bool   `json:"keep_branch,omitempty" jsonschema:"Keep the branch after merge (default: delete)"`
}

type LinkToolInput struct {
	Source string `json:"source" jsonschema:"Source issue ID"`
	Target string `json:"target" jsonschema:"Target issue ID"`
	Type   string `json:"type" jsonschema:"Relationship type: blocks, blocked_by, depends_on, dependency_of, duplicates, duplicated_by, relates_to"`
}

type SpecToolInput struct {
	Operation string `json:"operation" jsonschema:"Operation: write, read, or delete"`
	IssueID   string `json:"issue_id" jsonschema:"Issue ID"`
	Content   string `json:"content,omitempty" jsonschema:"Markdown content (required for write)"`
}

type WalkthroughToolInput struct {
	Operation string `json:"operation" jsonschema:"Operation: write, read, or delete"`
	IssueID   string `json:"issue_id" jsonschema:"Issue ID"`
	Content   string `json:"content,omitempty" jsonschema:"Markdown content (required for write)"`
}

type ArtifactToolInput struct {
	Operation string `json:"operation" jsonschema:"Operation: add, read, delete, or list"`
	IssueID   string `json:"issue_id" jsonschema:"Issue ID"`
	Filename  string `json:"filename,omitempty" jsonschema:"Artifact filename (required for add, read, delete)"`
	Content   string `json:"content,omitempty" jsonschema:"File content (required for add)"`
}

type RationaleToolInput struct {
	Query string `json:"query" jsonschema:"Free-text search query"`
	Limit int    `json:"limit,omitempty" jsonschema:"Max results to return (default 5, max 20)"`
}
