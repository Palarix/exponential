package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/palarix/exponential/internal/exponential"
	"github.com/palarix/exponential/internal/inputs"
	"github.com/palarix/exponential/internal/model"
	"github.com/palarix/exponential/internal/storage"
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
	Artifacts    []artifactOutEntry `json:"artifacts,omitempty"`
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
	ID           string   `json:"id"`
	Branch       string   `json:"branch,omitempty"`
	WorktreePath string   `json:"worktree_path,omitempty"`
	Messages     []string `json:"messages"`
}

type mergeIn struct {
	ID            string `json:"id" jsonschema:"Issue ID to merge"`
	Strategy      string `json:"strategy,omitempty" jsonschema:"Merge strategy: squash (default), merge, or ff"`
	CommitMessage string `json:"commit_message,omitempty" jsonschema:"Custom commit message (auto-generated if omitted)"`
	KeepBranch    bool   `json:"keep_branch,omitempty" jsonschema:"Keep the branch after merge (default: delete)"`
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

// --- Artifact tool types ---

type specIn struct {
	Operation string `json:"operation" jsonschema:"Operation: write, read, or delete"`
	IssueID   string `json:"issue_id" jsonschema:"Issue ID"`
	Content   string `json:"content,omitempty" jsonschema:"Markdown content (required for write)"`
}

type specOut struct {
	OK      bool   `json:"ok"`
	IssueID string `json:"issue_id"`
	Path    string `json:"path,omitempty"`
	Content string `json:"content,omitempty"`
	Error   string `json:"error,omitempty"`
}

type walkthroughIn struct {
	Operation string `json:"operation" jsonschema:"Operation: write, read, or delete"`
	IssueID   string `json:"issue_id" jsonschema:"Issue ID"`
	Content   string `json:"content,omitempty" jsonschema:"Markdown content (required for write)"`
}

type walkthroughOut struct {
	OK      bool   `json:"ok"`
	IssueID string `json:"issue_id"`
	Path    string `json:"path,omitempty"`
	Content string `json:"content,omitempty"`
	Error   string `json:"error,omitempty"`
}

type artifactIn struct {
	Operation string `json:"operation" jsonschema:"Operation: add, read, delete, or list"`
	IssueID   string `json:"issue_id" jsonschema:"Issue ID"`
	Filename  string `json:"filename,omitempty" jsonschema:"Artifact filename (required for add, read, delete)"`
	Content   string `json:"content,omitempty" jsonschema:"File content (required for add)"`
}

type artifactOutEntry struct {
	ArtifactType string `json:"type"`
	Filename     string `json:"filename"`
	UpdatedAt    string `json:"updated_at"`
	UpdatedBy    string `json:"updated_by"`
}

type artifactOut struct {
	OK        bool               `json:"ok"`
	IssueID   string             `json:"issue_id"`
	Path      string             `json:"path,omitempty"`
	Content   string             `json:"content,omitempty"`
	Artifacts []artifactOutEntry `json:"artifacts,omitempty"`
	Error     string             `json:"error,omitempty"`
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
		Description: "Start working on an issue: transitions status to DOING and creates a git worktree (default) or branch for <issue-id>-<slug> off the default branch. Returns the worktree path when worktrees are enabled.",
	}, t.start)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "merge",
		Description: "Merge an issue's branch into the default branch, record a MERGE event, and close the issue. When worktrees are enabled, the merge runs from the hub (primary checkout on main) and the worktree is cleaned up automatically.",
	}, t.merge)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "spec",
		Description: "Read, write, or delete the design spec (spec.md) for an issue. Specs are written before implementation to capture requirements, acceptance criteria, and design decisions. Use `write` to create/update, `read` to retrieve, `delete` to remove.",
	}, t.spec)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "walkthrough",
		Description: "Read, write, or delete the implementation walkthrough (walkthrough.md) for an issue. Walkthroughs are written after implementation to document what changed and why. Use `write` to create/update, `read` to retrieve, `delete` to remove.",
	}, t.walkthrough)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "artifact",
		Description: "Manage generic file artifacts on an issue. Use `add` to attach a file, `read` to retrieve it, `delete` to remove it, `list` to see all artifacts. Cannot write to spec.md or walkthrough.md — use the dedicated spec/walkthrough tools for those.",
	}, t.artifact)
}

// --- Handlers ---

func (t *toolset) list(ctx context.Context, req *mcp.CallToolRequest, in listIn) (*mcp.CallToolResult, listOut, error) {
	for _, s := range in.Status {
		switch model.IssueStatus(s) {
		case model.StatusBacklog, model.StatusPlanned, model.StatusDoing, model.StatusBlocked,
			model.StatusDone, model.StatusCanceled, model.StatusDuplicate:
		default:
			return nil, listOut{}, fmt.Errorf("invalid status filter %q: must be one of BACKLOG, PLANNED, DOING, BLOCKED, DONE, CANCELED, DUPLICATE", s)
		}
	}
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
		Artifacts:    toArtifactEntries(issue.Artifacts),
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
	if err := c.ValidateCreatePayload(&payload); err != nil {
		return nil, addOut{}, err
	}
	issue, err := c.AddIssue(payload)
	if err != nil {
		return nil, addOut{}, err
	}
	t.broadcast("CREATE", issue.ID)
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
	issue, err := c.GetIssue(in.ID)
	if err != nil {
		return nil, updateOut{}, err
	}
	if err := c.ValidateUpdatePayload(&payload); err != nil {
		return nil, updateOut{}, err
	}
	msgs, err := c.UpdateIssue(issue.ID, payload, "update")
	if err != nil {
		return nil, updateOut{}, err
	}
	t.broadcast("UPDATE", issue.ID)
	return textResult(strings.Join(msgs, "\n")), updateOut{ID: issue.ID, Messages: msgs}, nil
}

func (t *toolset) comment(ctx context.Context, req *mcp.CallToolRequest, in commentIn) (*mcp.CallToolResult, commentOut, error) {
	if in.ID == "" {
		return nil, commentOut{}, fmt.Errorf("'id' is required")
	}
	if in.Body == "" {
		return nil, commentOut{}, fmt.Errorf("'body' is required")
	}
	c := t.clientFor(req)
	issue, err := c.GetIssue(in.ID)
	if err != nil {
		return nil, commentOut{}, err
	}
	if err := c.AddComment(issue.ID, in.Body); err != nil {
		return nil, commentOut{}, err
	}
	t.broadcast("COMMENT", issue.ID)
	return textResult(fmt.Sprintf("Comment added to %s", issue.ID)), commentOut{ID: issue.ID}, nil
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
	if src.ID == tgt.ID {
		return nil, linkOut{}, fmt.Errorf("cannot link an issue to itself")
	}
	for _, dep := range src.Dependencies {
		if dep.TargetID == tgt.ID && string(dep.Kind) == kind {
			return nil, linkOut{}, fmt.Errorf("link %s %s already exists on %s", kind, tgt.ID, src.ID)
		}
	}
	newDeps := append(src.Dependencies, model.Dependency{
		SourceID: src.ID,
		TargetID: tgt.ID,
		Kind:     model.DependencyKind(kind),
	})
	if _, err := c.UpdateIssue(src.ID, model.UpdatePayload{Dependencies: newDeps}, "link"); err != nil {
		return nil, linkOut{}, err
	}
	t.broadcast("UPDATE", src.ID)
	return textResult(fmt.Sprintf("Linked %s %s %s", src.ID, kind, tgt.ID)),
		linkOut{Source: src.ID, Target: tgt.ID, Kind: kind}, nil
}

func (t *toolset) start(ctx context.Context, req *mcp.CallToolRequest, in startIn) (*mcp.CallToolResult, startOut, error) {
	if in.ID == "" {
		return nil, startOut{}, fmt.Errorf("'id' is required")
	}
	c := t.clientFor(req)
	issue, err := c.GetIssue(in.ID)
	if err != nil {
		return nil, startOut{}, err
	}
	branch, wtPath, msgs, err := c.StartWork(issue.ID, in.Force)
	if err != nil {
		return nil, startOut{}, err
	}
	t.broadcast("UPDATE", issue.ID)
	text := strings.Join(msgs, "\n")
	return textResult(text), startOut{ID: issue.ID, Branch: branch, WorktreePath: wtPath, Messages: msgs}, nil
}

func (t *toolset) merge(ctx context.Context, req *mcp.CallToolRequest, in mergeIn) (*mcp.CallToolResult, mergeOut, error) {
	if in.ID == "" {
		return nil, mergeOut{}, fmt.Errorf("'id' is required")
	}

	c := t.clientFor(req)

	strategy := exponential.MergeStrategySquash
	switch in.Strategy {
	case "", "squash":
		// default
	case "merge":
		strategy = exponential.MergeStrategyMerge
	case "ff":
		strategy = exponential.MergeStrategyFF
	default:
		return nil, mergeOut{}, fmt.Errorf("invalid merge strategy %q: must be one of squash, merge, ff", in.Strategy)
	}

	issue, err := c.ResolveReviewIssue(in.ID)
	if err != nil {
		return nil, mergeOut{}, err
	}
	if issue.BranchStats == nil {
		return nil, mergeOut{}, fmt.Errorf("no branch found for %s", issue.ID)
	}

	if err := exponential.HubCleanForMerge(issue.BranchStats.Branch); err != nil {
		return nil, mergeOut{}, err
	}

	result, err := c.MergeIssue(issue.ID, exponential.MergeOptions{
		Strategy:      strategy,
		CommitMessage: in.CommitMessage,
		DeleteBranch:  !in.KeepBranch,
	})
	if err != nil {
		return nil, mergeOut{}, err
	}
	t.broadcast("MERGE", issue.ID)

	text := strings.Join(result.Messages, "\n")
	return textResult(text), mergeOut{ID: issue.ID, MergeSHA: result.MergeSHA, Messages: result.Messages}, nil
}

// --- Artifact handlers ---

func (t *toolset) spec(ctx context.Context, req *mcp.CallToolRequest, in specIn) (*mcp.CallToolResult, specOut, error) {
	if in.IssueID == "" {
		return nil, specOut{OK: false, Error: "'issue_id' is required"}, fmt.Errorf("'issue_id' is required")
	}
	c := t.clientFor(req)
	issue, err := c.GetIssue(in.IssueID)
	if err != nil {
		return nil, specOut{OK: false, Error: err.Error()}, err
	}
	issueID := issue.ID
	path := filepath.Join(storage.XpoDir(), "artifacts", issueID, "spec.md")

	switch in.Operation {
	case "write":
		if in.Content == "" {
			return nil, specOut{OK: false, Error: "'content' is required for write"}, fmt.Errorf("'content' is required for write")
		}
		if err := c.WriteSpec(issueID, in.Content); err != nil {
			return nil, specOut{OK: false, Error: err.Error()}, err
		}
		t.broadcast("ARTIFACT", issueID)
		return textResult(fmt.Sprintf("Spec written for %s", issueID)),
			specOut{OK: true, IssueID: issueID, Path: path}, nil

	case "read":
		content, err := c.ReadSpec(issueID)
		if err != nil {
			return nil, specOut{OK: false, Error: err.Error()}, err
		}
		return textResult(content),
			specOut{OK: true, IssueID: issueID, Path: path, Content: content}, nil

	case "delete":
		if err := c.DeleteSpec(issueID); err != nil {
			return nil, specOut{OK: false, Error: err.Error()}, err
		}
		t.broadcast("ARTIFACT", issueID)
		return textResult(fmt.Sprintf("Spec deleted from %s", issueID)),
			specOut{OK: true, IssueID: issueID, Path: path}, nil

	default:
		return nil, specOut{OK: false, Error: "invalid operation: must be write, read, or delete"},
			fmt.Errorf("invalid operation %q: must be write, read, or delete", in.Operation)
	}
}

func (t *toolset) walkthrough(ctx context.Context, req *mcp.CallToolRequest, in walkthroughIn) (*mcp.CallToolResult, walkthroughOut, error) {
	if in.IssueID == "" {
		return nil, walkthroughOut{OK: false, Error: "'issue_id' is required"}, fmt.Errorf("'issue_id' is required")
	}
	c := t.clientFor(req)
	issue, err := c.GetIssue(in.IssueID)
	if err != nil {
		return nil, walkthroughOut{OK: false, Error: err.Error()}, err
	}
	issueID := issue.ID
	path := filepath.Join(storage.XpoDir(), "artifacts", issueID, "walkthrough.md")

	switch in.Operation {
	case "write":
		if in.Content == "" {
			return nil, walkthroughOut{OK: false, Error: "'content' is required for write"}, fmt.Errorf("'content' is required for write")
		}
		if err := c.WriteWalkthrough(issueID, in.Content); err != nil {
			return nil, walkthroughOut{OK: false, Error: err.Error()}, err
		}
		t.broadcast("ARTIFACT", issueID)
		return textResult(fmt.Sprintf("Walkthrough written for %s", issueID)),
			walkthroughOut{OK: true, IssueID: issueID, Path: path}, nil

	case "read":
		content, err := c.ReadWalkthrough(issueID)
		if err != nil {
			return nil, walkthroughOut{OK: false, Error: err.Error()}, err
		}
		return textResult(content),
			walkthroughOut{OK: true, IssueID: issueID, Path: path, Content: content}, nil

	case "delete":
		if err := c.DeleteWalkthrough(issueID); err != nil {
			return nil, walkthroughOut{OK: false, Error: err.Error()}, err
		}
		t.broadcast("ARTIFACT", issueID)
		return textResult(fmt.Sprintf("Walkthrough deleted from %s", issueID)),
			walkthroughOut{OK: true, IssueID: issueID, Path: path}, nil

	default:
		return nil, walkthroughOut{OK: false, Error: "invalid operation: must be write, read, or delete"},
			fmt.Errorf("invalid operation %q: must be write, read, or delete", in.Operation)
	}
}

func (t *toolset) artifact(ctx context.Context, req *mcp.CallToolRequest, in artifactIn) (*mcp.CallToolResult, artifactOut, error) {
	if in.IssueID == "" {
		return nil, artifactOut{OK: false, Error: "'issue_id' is required"}, fmt.Errorf("'issue_id' is required")
	}
	c := t.clientFor(req)
	issue, err := c.GetIssue(in.IssueID)
	if err != nil {
		return nil, artifactOut{OK: false, Error: err.Error()}, err
	}
	issueID := issue.ID

	switch in.Operation {
	case "add":
		if in.Filename == "" {
			return nil, artifactOut{OK: false, Error: "'filename' is required for add"}, fmt.Errorf("'filename' is required for add")
		}
		if in.Content == "" {
			return nil, artifactOut{OK: false, Error: "'content' is required for add"}, fmt.Errorf("'content' is required for add")
		}
		if err := c.AddArtifact(issueID, "generic", in.Filename, in.Content); err != nil {
			return nil, artifactOut{OK: false, Error: err.Error()}, err
		}
		path := filepath.Join(storage.XpoDir(), "artifacts", issueID, in.Filename)
		t.broadcast("ARTIFACT", issueID)
		return textResult(fmt.Sprintf("Artifact %s added to %s", in.Filename, issueID)),
			artifactOut{OK: true, IssueID: issueID, Path: path}, nil

	case "read":
		if in.Filename == "" {
			return nil, artifactOut{OK: false, Error: "'filename' is required for read"}, fmt.Errorf("'filename' is required for read")
		}
		content, err := c.ReadArtifact(issueID, in.Filename)
		if err != nil {
			return nil, artifactOut{OK: false, Error: err.Error()}, err
		}
		path := filepath.Join(storage.XpoDir(), "artifacts", issueID, in.Filename)
		return textResult(content),
			artifactOut{OK: true, IssueID: issueID, Path: path, Content: content}, nil

	case "delete":
		if in.Filename == "" {
			return nil, artifactOut{OK: false, Error: "'filename' is required for delete"}, fmt.Errorf("'filename' is required for delete")
		}
		if err := c.DeleteArtifact(issueID, in.Filename); err != nil {
			return nil, artifactOut{OK: false, Error: err.Error()}, err
		}
		path := filepath.Join(storage.XpoDir(), "artifacts", issueID, in.Filename)
		t.broadcast("ARTIFACT", issueID)
		return textResult(fmt.Sprintf("Artifact %s deleted from %s", in.Filename, issueID)),
			artifactOut{OK: true, IssueID: issueID, Path: path}, nil

	case "list":
		artifacts, err := c.ListArtifacts(issueID)
		if err != nil {
			return nil, artifactOut{OK: false, Error: err.Error()}, err
		}
		entries := toArtifactEntries(artifacts)
		return textResult(fmt.Sprintf("%d artifact(s)", len(entries))),
			artifactOut{OK: true, IssueID: issueID, Artifacts: entries}, nil

	default:
		return nil, artifactOut{OK: false, Error: "invalid operation: must be add, read, delete, or list"},
			fmt.Errorf("invalid operation %q: must be add, read, delete, or list", in.Operation)
	}
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

func toArtifactEntries(as []model.ArtifactSummary) []artifactOutEntry {
	if len(as) == 0 {
		return nil
	}
	out := make([]artifactOutEntry, len(as))
	for i, a := range as {
		out[i] = artifactOutEntry{
			ArtifactType: a.ArtifactType,
			Filename:     a.Filename,
			UpdatedAt:    a.UpdatedAt.Format(time.RFC3339),
			UpdatedBy:    a.UpdatedBy,
		}
	}
	return out
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
