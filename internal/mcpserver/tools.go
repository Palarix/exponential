package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/palarix/exponential/internal/exponential"
	"github.com/palarix/exponential/internal/config"
	"github.com/palarix/exponential/internal/inputs"
	"github.com/palarix/exponential/internal/model"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// --- Tool input/output types ---
//
// Type names are kept private. JSONSchema is derived from the struct tags by
// the MCP SDK; jsonschema tags add human-readable descriptions that show up
// in tool catalogs.

type issueSummary struct {
	ID          string             `json:"id"`
	Title       string             `json:"title"`
	Status      string             `json:"status"`
	IsInferred  bool               `json:"is_inferred,omitempty"`
	Labels      []string           `json:"labels,omitempty"`
	ParentID    string             `json:"parent_id,omitempty"`
	StoryPoints int                `json:"story_points,omitempty"`
	Assignee    string             `json:"assignee,omitempty"`
	CycleID          string             `json:"cycle_id,omitempty"`
	EffectiveCycleID string             `json:"effective_cycle_id,omitempty"`
	BranchStats      *model.BranchStats `json:"branch_stats,omitempty"`
	UpdatedAt        string             `json:"updated_at"`
}

type commentSummary struct {
	ID        string `json:"id"`
	Text      string `json:"text"`
	CreatedBy string `json:"created_by"`
	CreatedAt string `json:"created_at"`
}

type eventSummary struct {
	Type      string `json:"type"`
	CreatedBy string `json:"created_by"`
	CreatedAt string `json:"created_at"`
}

type listIn struct {
	Status      flexStrings `json:"status,omitempty" jsonschema:"Filter by status: BACKLOG, PLANNED, DOING, BLOCKED, DONE"`
	Label       string      `json:"label,omitempty" jsonschema:"Substring match on a label"`
	Assignee    string      `json:"assignee,omitempty" jsonschema:"Substring match on assignee"`
	Parent      string      `json:"parent,omitempty" jsonschema:"Only return sub-issues of this parent ID"`
	Match       string      `json:"match,omitempty" jsonschema:"Substring match across id, title, status, parent, assignee, labels"`
	IncludeDone bool        `json:"include_done,omitempty" jsonschema:"Include DONE issues (otherwise only recent DONE are shown)"`
}

// flexStrings accepts either a single string or an array of strings in JSON.
type flexStrings []string

func (f *flexStrings) UnmarshalJSON(data []byte) error {
	// Try as array first
	var arr []string
	if err := json.Unmarshal(data, &arr); err == nil {
		*f = arr
		return nil
	}
	// Fall back to single string
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	*f = []string{s}
	return nil
}

type listOut struct {
	Issues []issueSummary `json:"issues"`
}

type showIn struct {
	ID            string `json:"id" jsonschema:"Issue ID (full or unique suffix)"`
	IncludeEvents bool   `json:"include_events,omitempty" jsonschema:"Include the full event audit trail"`
}

type showOut struct {
	ID           string             `json:"id"`
	Title        string             `json:"title"`
	Status       string             `json:"status"`
	IsInferred   bool               `json:"is_inferred,omitempty"`
	Description  string             `json:"description,omitempty"`
	Labels       []string           `json:"labels,omitempty"`
	ParentID     string             `json:"parent_id,omitempty"`
	StoryPoints  int                `json:"story_points,omitempty"`
	Assignee     string             `json:"assignee,omitempty"`
	CycleID          string             `json:"cycle_id,omitempty"`
	EffectiveCycleID string             `json:"effective_cycle_id,omitempty"`
	BranchStats      *model.BranchStats `json:"branch_stats,omitempty"`
	CreatedBy        string             `json:"created_by"`
	CreatedAt    string             `json:"created_at"`
	UpdatedAt    string             `json:"updated_at"`
	Dependencies []model.Dependency `json:"dependencies,omitempty"`
	Comments     []commentSummary   `json:"comments,omitempty"`
	Events       []eventSummary     `json:"events,omitempty"`
}

type historyIn struct {
	ID string `json:"id" jsonschema:"Issue ID"`
}

type historyOut struct {
	Events []eventSummary `json:"events"`
}

type addOut struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Status string `json:"status"`
}

type updateIn struct {
	ID string `json:"id" jsonschema:"Issue ID to update"`
	inputs.UpdateInput
}

type updateOut struct {
	ID       string   `json:"id"`
	Messages []string `json:"messages"`
}

type commentIn struct {
	ID   string `json:"id" jsonschema:"Issue ID to comment on"`
	Body string `json:"body" jsonschema:"Comment text in markdown"`
}

type commentOut struct {
	ID string `json:"id"`
}

type startIn struct {
	ID    string `json:"id" jsonschema:"Issue ID to start working on"`
	Force bool   `json:"force,omitempty" jsonschema:"Force take-over if already in progress or branch exists"`
}

type startOut struct {
	ID       string   `json:"id"`
	Branch   string   `json:"branch,omitempty"`
	Messages []string `json:"messages"`
}

type mergeIn struct {
	ID            string `json:"id" jsonschema:"Issue ID to merge"`
	Strategy      string `json:"strategy,omitempty" jsonschema:"Merge strategy: squash (default), merge, or ff"`
	CommitMessage string `json:"commit_message,omitempty" jsonschema:"Custom commit message (auto-generated if omitted)"`
	DeleteBranch  bool   `json:"delete_branch,omitempty" jsonschema:"Delete the branch after merge"`
}

type mergeOut struct {
	ID       string   `json:"id"`
	MergeSHA string   `json:"merge_sha"`
	Messages []string `json:"messages"`
}

type linkIn struct {
	Source string `json:"source" jsonschema:"Source issue ID"`
	Target string `json:"target" jsonschema:"Target issue ID"`
	Type   string `json:"type" jsonschema:"Relationship type: blocks, blocked_by, depends_on, dependency_of, duplicates, duplicated_by, relates_to"`
}

type linkOut struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Kind   string `json:"kind"`
}

// --- Registration ---

func (t *toolset) register(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "list",
		Description: "List issues with optional filters (status, label, assignee, parent, free-text match).",
	}, t.list)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "show",
		Description: "Show one issue with its description, dependencies, comments, and optionally event history.",
	}, t.show)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "history",
		Description: "Return the audit trail (events) for an issue.",
	}, t.history)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "add",
		Description: "Create a new issue. Set status, labels, parent, story_points, links etc. in one call.",
	}, t.add)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "update",
		Description: "Patch fields on an existing issue. Use this to transition status (BACKLOG/PLANNED/DOING/BLOCKED/DONE) instead of separate start/done tools.",
	}, t.update)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "comment",
		Description: "Add a markdown comment to an issue.",
	}, t.comment)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "link",
		Description: "Add a relationship (blocks, depends_on, relates_to, …) between two existing issues.",
	}, t.link)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "start",
		Description: "Start working on an issue: transitions status to DOING and creates a git branch named <issue-id>-<slug> off the default branch.",
	}, t.start)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "merge",
		Description: "Merge an issue's branch into the default branch, record a MERGE event, and close the issue. Requires a clean working tree.",
	}, t.merge)
}

// --- Handlers ---

func (t *toolset) list(ctx context.Context, req *mcp.CallToolRequest, in listIn) (*mcp.CallToolResult, listOut, error) {
	c := t.clientFor(req)
	opts := exponential.FilterOptions{
		Statuses: []string(in.Status),
		Label:    in.Label,
		Assignee: in.Assignee,
		ParentID: in.Parent,
		Match:    in.Match,
		All:      in.IncludeDone,
	}
	issues, err := c.ListIssues(opts)
	if err != nil {
		return nil, listOut{}, err
	}
	out := listOut{Issues: make([]issueSummary, 0, len(issues))}
	for _, i := range issues {
		out.Issues = append(out.Issues, toSummary(i))
	}
	return textResult(fmt.Sprintf("%d issue(s)", len(out.Issues))), out, nil
}

func (t *toolset) show(ctx context.Context, req *mcp.CallToolRequest, in showIn) (*mcp.CallToolResult, showOut, error) {
	if in.ID == "" {
		return nil, showOut{}, fmt.Errorf("'id' is required")
	}
	c := t.clientFor(req)
	issue, err := c.GetIssue(in.ID)
	if err != nil {
		return nil, showOut{}, err
	}
	out := showOut{
		ID:           issue.ID,
		Title:        issue.Title,
		Status:       string(issue.Status),
		IsInferred:   issue.InferredStatus,
		Description:  issue.Description,
		Labels:       issue.Labels,
		ParentID:     issue.ParentID,
		StoryPoints:  issue.Estimate,
		Assignee:     issue.Assignee,
		CycleID:          issue.CycleID,
		EffectiveCycleID: issue.EffectiveCycleID,
		BranchStats:      issue.BranchStats,
		CreatedBy:        issue.CreatedBy,
		CreatedAt:    issue.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    issue.UpdatedAt.Format(time.RFC3339),
		Dependencies: issue.Dependencies,
		Comments:     toCommentSummaries(issue.Comments),
	}
	if in.IncludeEvents {
		out.Events = toEventSummaries(issue.Events)
	}
	return textResult(fmt.Sprintf("%s — %s [%s]", issue.ID, issue.Title, issue.Status)), out, nil
}

func (t *toolset) history(ctx context.Context, req *mcp.CallToolRequest, in historyIn) (*mcp.CallToolResult, historyOut, error) {
	if in.ID == "" {
		return nil, historyOut{}, fmt.Errorf("'id' is required")
	}
	c := t.clientFor(req)
	issue, err := c.GetIssue(in.ID)
	if err != nil {
		return nil, historyOut{}, err
	}
	return textResult(fmt.Sprintf("%d event(s)", len(issue.Events))), historyOut{Events: toEventSummaries(issue.Events)}, nil
}

func (t *toolset) add(ctx context.Context, req *mcp.CallToolRequest, in inputs.AddInput) (*mcp.CallToolResult, addOut, error) {
	payload, err := in.ToCreatePayload()
	if err != nil {
		return nil, addOut{}, err
	}
	c := t.clientFor(req)
	if payload.Estimate > 0 {
		if err := config.ValidateEstimate(c.Config.EstimationSystem, payload.Estimate); err != nil {
			return nil, addOut{}, err
		}
	}
	issue, err := c.AddIssue(payload)
	if err != nil {
		return nil, addOut{}, err
	}
	return textResult(fmt.Sprintf("Created %s: %s", issue.ID, issue.Title)),
		addOut{ID: issue.ID, Title: issue.Title, Status: string(issue.Status)}, nil
}

func (t *toolset) update(ctx context.Context, req *mcp.CallToolRequest, in updateIn) (*mcp.CallToolResult, updateOut, error) {
	if in.ID == "" {
		return nil, updateOut{}, fmt.Errorf("'id' is required")
	}
	payload, err := in.UpdateInput.ToUpdatePayload()
	if err != nil {
		return nil, updateOut{}, err
	}
	if inputs.UpdatePayloadEmpty(payload) {
		return nil, updateOut{}, fmt.Errorf("no fields set: provide at least one field to update")
	}
	c := t.clientFor(req)
	msgs, err := c.UpdateIssue(in.ID, payload, "update")
	if err != nil {
		return nil, updateOut{}, err
	}
	return textResult(strings.Join(msgs, "\n")), updateOut{ID: in.ID, Messages: msgs}, nil
}

func (t *toolset) comment(ctx context.Context, req *mcp.CallToolRequest, in commentIn) (*mcp.CallToolResult, commentOut, error) {
	if in.ID == "" {
		return nil, commentOut{}, fmt.Errorf("'id' is required")
	}
	if in.Body == "" {
		return nil, commentOut{}, fmt.Errorf("'body' is required")
	}
	c := t.clientFor(req)
	if _, err := c.GetIssue(in.ID); err != nil {
		return nil, commentOut{}, err
	}
	if err := c.AddComment(in.ID, in.Body); err != nil {
		return nil, commentOut{}, err
	}
	return textResult(fmt.Sprintf("Comment added to %s", in.ID)), commentOut{ID: in.ID}, nil
}

func (t *toolset) link(ctx context.Context, req *mcp.CallToolRequest, in linkIn) (*mcp.CallToolResult, linkOut, error) {
	if in.Source == "" || in.Target == "" {
		return nil, linkOut{}, fmt.Errorf("'source' and 'target' are required")
	}
	kind := model.NormalizeDependencyKind(in.Type)
	if kind == "" {
		return nil, linkOut{}, fmt.Errorf("invalid link type %q", in.Type)
	}
	c := t.clientFor(req)
	src, err := c.GetIssue(in.Source)
	if err != nil {
		return nil, linkOut{}, fmt.Errorf("source: %w", err)
	}
	tgt, err := c.GetIssue(in.Target)
	if err != nil {
		return nil, linkOut{}, fmt.Errorf("target: %w", err)
	}
	newDeps := append(src.Dependencies, model.Dependency{
		SourceID: src.ID,
		TargetID: tgt.ID,
		Kind:     model.DependencyKind(kind),
	})
	if _, err := c.UpdateIssue(src.ID, model.UpdatePayload{Dependencies: newDeps}, "link"); err != nil {
		return nil, linkOut{}, err
	}
	return textResult(fmt.Sprintf("Linked %s %s %s", src.ID, kind, tgt.ID)),
		linkOut{Source: src.ID, Target: tgt.ID, Kind: kind}, nil
}

func (t *toolset) start(ctx context.Context, req *mcp.CallToolRequest, in startIn) (*mcp.CallToolResult, startOut, error) {
	if in.ID == "" {
		return nil, startOut{}, fmt.Errorf("'id' is required")
	}
	c := t.clientFor(req)
	branch, msgs, err := c.StartWork(in.ID, in.Force)
	if err != nil {
		return nil, startOut{}, err
	}
	text := strings.Join(msgs, "\n")
	return textResult(text), startOut{ID: in.ID, Branch: branch, Messages: msgs}, nil
}

func (t *toolset) merge(ctx context.Context, req *mcp.CallToolRequest, in mergeIn) (*mcp.CallToolResult, mergeOut, error) {
	if in.ID == "" {
		return nil, mergeOut{}, fmt.Errorf("'id' is required")
	}

	if !exponential.IsWorkingTreeClean() {
		return nil, mergeOut{}, fmt.Errorf("working tree is not clean — commit or stash your changes first")
	}

	strategy := exponential.MergeStrategySquash
	switch in.Strategy {
	case "merge":
		strategy = exponential.MergeStrategyMerge
	case "ff":
		strategy = exponential.MergeStrategyFF
	}

	c := t.clientFor(req)
	result, err := c.MergeIssue(in.ID, exponential.MergeOptions{
		Strategy:      strategy,
		CommitMessage: in.CommitMessage,
		DeleteBranch:  in.DeleteBranch,
	})
	if err != nil {
		return nil, mergeOut{}, err
	}

	text := strings.Join(result.Messages, "\n")
	return textResult(text), mergeOut{ID: in.ID, MergeSHA: result.MergeSHA, Messages: result.Messages}, nil
}

// --- Helpers ---

func toSummary(i *model.Issue) issueSummary {
	return issueSummary{
		ID:          i.ID,
		Title:       i.Title,
		Status:      string(i.Status),
		IsInferred:  i.InferredStatus,
		Labels:      i.Labels,
		ParentID:    i.ParentID,
		StoryPoints: i.Estimate,
		Assignee:    i.Assignee,
		CycleID:          i.CycleID,
		EffectiveCycleID: i.EffectiveCycleID,
		BranchStats:      i.BranchStats,
		UpdatedAt:        i.UpdatedAt.Format(time.RFC3339),
	}
}

func toCommentSummaries(cs []model.Comment) []commentSummary {
	if len(cs) == 0 {
		return nil
	}
	out := make([]commentSummary, len(cs))
	for i, c := range cs {
		out[i] = commentSummary{
			ID:        c.ID,
			Text:      c.Text,
			CreatedBy: c.CreatedBy,
			CreatedAt: c.CreatedAt.Format(time.RFC3339),
		}
	}
	return out
}

func toEventSummaries(es []model.Event) []eventSummary {
	if len(es) == 0 {
		return nil
	}
	out := make([]eventSummary, len(es))
	for i, e := range es {
		out[i] = eventSummary{
			Type:      string(e.Type),
			CreatedBy: e.CreatedBy,
			CreatedAt: e.CreatedAt.Format(time.RFC3339),
		}
	}
	return out
}

func textResult(s string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: s}},
	}
}
