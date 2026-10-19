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
	Release     string      `json:"release,omitempty" jsonschema:"Release to put the issue in: #7 (number), a label, @current, @next, or none. Only open releases accept issues. If omitted, a new child copies its parent's open release"`
	Links       []LinkInput `json:"links,omitempty" jsonschema:"Dependencies/relationships to other issues, set at creation time. At most one link per target issue. Links are bidirectional (A blocks B also means B blocked_by A); only blocked_by gates start, other kinds are informational"`
	Before      string      `json:"before,omitempty" jsonschema:"Place the new issue directly above this issue ID in the manual order. Mutually exclusive with after"`
	After       string      `json:"after,omitempty" jsonschema:"Place the new issue directly below this issue ID in the manual order. Mutually exclusive with before"`
}

type UpdateInput struct {
	Title        *string     `json:"title,omitempty"`
	Description  *string     `json:"description,omitempty" jsonschema:"Replace the issue description"`
	Status       *string     `json:"status,omitempty" jsonschema:"New status: BACKLOG, PLANNED, DOING, BLOCKED, DONE, CANCELED, or DUPLICATE"`
	Parent       *string     `json:"parent,omitempty" jsonschema:"Parent issue ID. Pass null or \"\" to remove the parent and make the issue top-level"`
	StoryPoints  *int        `json:"story_points,omitempty" jsonschema:"Effort estimate in story points. Pass null or 0 to clear"`
	Priority     *int        `json:"priority,omitempty" jsonschema:"Priority: 0=None, 1=Urgent, 2=High, 3=Medium, 4=Low. Pass null or 0 to clear"`
	Assignee     *string     `json:"assignee,omitempty" jsonschema:"Assignee in 'Name <email>' format. Pass null or \"\" to unassign"`
	Labels       []string    `json:"labels,omitempty" jsonschema:"Replace the full label list"`
	CycleID      *string     `json:"cycle_id,omitempty" jsonschema:"Cycle ID (YYYY-MM-DD start date) to assign this issue to. Pass null or \"\" to remove the issue from its cycle"`
	Release      *string     `json:"release,omitempty" jsonschema:"Move the issue to a release: #7 (number), a label, @current or @next. Pass none, null or \"\" to remove it from its release. Only open releases accept issues"`
	WithChildren bool        `json:"with_children,omitempty" jsonschema:"With release: also move the issue's open children (done children never move). Without it, only this issue changes"`
	Links        []LinkInput `json:"links,omitempty" jsonschema:"Replace the issue's full relationship set, including derived links stored on the other issue (as returned by show); a link left out is removed from whichever side stores it; pass [] to remove all links. At most one link per target issue: to change a link's kind, send the set with the new kind in place of the old. Only blocked_by gates start; other kinds are informational"`
	Before       string      `json:"before,omitempty" jsonschema:"Move the issue directly above this issue ID in the manual order. Mutually exclusive with after"`
	After        string      `json:"after,omitempty" jsonschema:"Move the issue directly below this issue ID in the manual order. Mutually exclusive with before"`
}

type CommentInput struct {
	Body string `json:"body" jsonschema:"Comment text in markdown"`
}

type LinkInput struct {
	Target string `json:"target" jsonschema:"Target issue ID"`
	Type   string `json:"type" jsonschema:"Relationship type: blocks, blocked_by, depends_on, dependency_of, duplicates, duplicated_by, relates_to"`
}

// --- Validation and conversion helpers ---

func validatePosition(before, after string) error {
	if before != "" && after != "" {
		return fmt.Errorf("'before' and 'after' are mutually exclusive")
	}
	return nil
}

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

// LinksToDependencies converts link inputs to dependencies. A nil slice
// (field omitted) stays nil; an empty slice becomes an empty, non-nil list
// so that an update with `links: []` clears all links.
func LinksToDependencies(links []LinkInput) ([]model.Dependency, error) {
	if links == nil {
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
	if err := validatePosition(in.Before, in.After); err != nil {
		return model.CreatePayload{}, err
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
		ReleaseID:    strings.TrimSpace(in.Release),
		Labels:       in.Labels,
		Dependencies: deps,
	}, nil
}

func (in UpdateInput) ToUpdatePayload() (model.UpdatePayload, error) {
	if err := validatePosition(in.Before, in.After); err != nil {
		return model.UpdatePayload{}, err
	}
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
		ParentID:     normalizeParent(in.Parent),
		Estimate:     in.StoryPoints,
		Priority:     in.Priority,
		Assignee:     in.Assignee,
		CycleID:      in.CycleID,
		ReleaseID:    in.Release,
		Labels:       in.Labels,
		Dependencies: deps,
	}, nil
}

// nullClears maps each scalar field where an explicit null means "clear" to a
// setter for its cleared value. description, labels and links are left out on
// purpose: null keeps meaning "unchanged" there, so a client that sends null
// for every unused field cannot erase content. They clear with "" or [].
var nullClears = map[string]func(*UpdateInput){
	"parent":       func(in *UpdateInput) { in.Parent = ptr("") },
	"assignee":     func(in *UpdateInput) { in.Assignee = ptr("") },
	"cycle_id":     func(in *UpdateInput) { in.CycleID = ptr("") },
	"release":      func(in *UpdateInput) { in.Release = ptr("") },
	"priority":     func(in *UpdateInput) { in.Priority = ptr(0) },
	"story_points": func(in *UpdateInput) { in.StoryPoints = ptr(0) },
}

func ptr[T any](v T) *T { return &v }

// ApplyExplicitNulls marks fields the raw JSON object sets to null as
// "clear". encoding/json decodes both an omitted field and an explicit null to
// a nil pointer, so the decoded UpdateInput alone cannot tell them apart.
// Only the fields in nullClears are affected.
func ApplyExplicitNulls(raw []byte, in *UpdateInput) error {
	if len(raw) == 0 {
		return nil
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return fmt.Errorf("invalid JSON payload: %w", err)
	}
	for name, v := range fields {
		if clear, ok := nullClears[name]; ok && strings.TrimSpace(string(v)) == "null" {
			clear(in)
		}
	}
	return nil
}

// normalizeParent trims the parent ID and treats a quoted empty string, in
// double or single quotes, as empty. Agents send the literal quotes when they
// mean "no parent"; no issue ID contains quote characters.
func normalizeParent(p *string) *string {
	if p == nil {
		return nil
	}
	v := strings.TrimSpace(*p)
	if v == `""` || v == "''" {
		v = ""
	}
	return &v
}

func UpdatePayloadEmpty(p model.UpdatePayload) bool {
	return p.Title == nil && p.Description == nil && p.Status == nil &&
		p.ParentID == nil && p.Estimate == nil && p.Priority == nil &&
		p.Assignee == nil && p.CycleID == nil && p.ReleaseID == nil && p.Labels == nil && p.Dependencies == nil && p.SortOrder == nil
}

// --- MCP tool parameter types (add ID or other tool-specific fields) ---

type ListToolInput struct {
	Status      FlexStrings `json:"status,omitempty" jsonschema:"Filter by status: BACKLOG, PLANNED, DOING, BLOCKED, DONE"`
	Label       string      `json:"label,omitempty" jsonschema:"Substring match on a label"`
	Assignee    string      `json:"assignee,omitempty" jsonschema:"Substring match on assignee"`
	Parent      string      `json:"parent,omitempty" jsonschema:"Only return sub-issues of this parent ID"`
	Match       string      `json:"match,omitempty" jsonschema:"Substring match across id, title, status, parent, assignee, labels"`
	Release     string      `json:"release,omitempty" jsonschema:"Only issues in this release: #7 (number), a label, @current, @next, or none for issues in no release"`
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
}

type MergeToolInput struct {
	ID            string `json:"id" jsonschema:"Issue ID to merge"`
	Strategy      string `json:"strategy,omitempty" jsonschema:"Merge strategy: squash (default), merge, or ff. Omit unless the user asks for a specific strategy"`
	CommitMessage string `json:"commit_message,omitempty" jsonschema:"Custom commit message (auto-generated if omitted)"`
	KeepBranch    bool   `json:"keep_branch,omitempty" jsonschema:"Keep the branch after merge (default: delete)"`
}

// Read-only git tools. The server builds every git command; these fields are
// the only caller input.

type GitStatusToolInput struct {
	ID string `json:"id" jsonschema:"Issue ID (full or unique suffix)"`
}

type GitDiffToolInput struct {
	ID   string `json:"id" jsonschema:"Issue ID (full or unique suffix)"`
	Path string `json:"path,omitempty" jsonschema:"Only this file or directory, relative to the repository root"`
	Stat bool   `json:"stat,omitempty" jsonschema:"Return per-file insertions and deletions instead of the patch"`
}

type GitLogToolInput struct {
	ID    string `json:"id,omitempty" jsonschema:"Issue ID: list only the commits on that issue's branch. Omit to list the default branch"`
	Path  string `json:"path,omitempty" jsonschema:"Only commits that touch this file or directory, relative to the repository root"`
	Limit int    `json:"limit,omitempty" jsonschema:"Maximum number of commits (default 20, max 100)"`
}

type GitShowToolInput struct {
	SHA  string `json:"sha" jsonschema:"Commit SHA, 4 to 40 hex characters"`
	Path string `json:"path,omitempty" jsonschema:"Only this file or directory, relative to the repository root"`
}

type CodeHistoryToolInput struct {
	Path      string `json:"path" jsonschema:"File, relative to the repository root"`
	StartLine int    `json:"start_line,omitempty" jsonschema:"First line of the range (1-based). Set with end_line to blame a line range"`
	EndLine   int    `json:"end_line,omitempty" jsonschema:"Last line of the range (inclusive)"`
	Limit     int    `json:"limit,omitempty" jsonschema:"Maximum number of commits (default 20, max 100)"`
}

type LinkToolInput struct {
	Source    string `json:"source" jsonschema:"Source issue ID"`
	Target    string `json:"target" jsonschema:"Target issue ID"`
	Type      string `json:"type" jsonschema:"Relationship type: blocks, blocked_by, depends_on, dependency_of, duplicates, duplicated_by, relates_to"`
	Overwrite bool   `json:"overwrite,omitempty" jsonschema:"Replace an existing link of a different kind between the two issues (whichever side stores it). Without it, linking an already-linked pair fails"`
}

type UnlinkToolInput struct {
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

type ReleaseToolInput struct {
	Operation  string  `json:"operation" jsonschema:"Operation: list, create, update, reorder, complete, or delete"`
	ID         string  `json:"id,omitempty" jsonschema:"Release for update, reorder, complete and delete: #7 (number), a label, @current or @next. Updating an unsaved Current or Next saves it"`
	Label      *string `json:"label,omitempty" jsonschema:"Optional label, such as a version (1.5.0) or a name (Team Mode). Unique among open releases. Pass \"\" to clear"`
	Goal       *string `json:"goal,omitempty" jsonschema:"One-line goal of the release"`
	TargetDate *string `json:"target_date,omitempty" jsonschema:"Optional target date (YYYY-MM-DD), for display only. Pass \"\" to clear"`
	Before     string  `json:"before,omitempty" jsonschema:"reorder: move the release directly above this open release. Mutually exclusive with after"`
	After      string  `json:"after,omitempty" jsonschema:"reorder: move the release directly below this open release. Mutually exclusive with before"`
	Issues     string  `json:"issues,omitempty" jsonschema:"complete and delete: what happens to the release's issues. move_to_next (default) moves them to Next; remove takes them out of the release. complete moves only open issues: done issues stay as the record. delete moves all issues"`
}
