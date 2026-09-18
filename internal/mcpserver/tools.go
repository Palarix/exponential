package mcpserver

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/palarix/exponential/internal/exponential"
	"github.com/palarix/exponential/internal/jsonio"
	"github.com/palarix/exponential/internal/model"
	"github.com/palarix/exponential/internal/storage"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

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

	mcp.AddTool(s, &mcp.Tool{
		Name:        "rationale",
		Description: "Search across specs and walkthroughs for design rationale. Returns matching fragments ranked by relevance (BM25 scoring with title/label boost and proximity bonus). Use this to find prior design decisions before modifying code or writing a new spec.",
	}, t.rationale)
}

// --- Handlers ---

func (t *toolset) list(ctx context.Context, req *mcp.CallToolRequest, in jsonio.ListToolInput) (*mcp.CallToolResult, jsonio.ListOutput, error) {
	for _, s := range in.Status {
		switch model.IssueStatus(s) {
		case model.StatusBacklog, model.StatusPlanned, model.StatusDoing, model.StatusBlocked,
			model.StatusDone, model.StatusCanceled, model.StatusDuplicate:
		default:
			return nil, jsonio.ListOutput{}, fmt.Errorf("invalid status filter %q: must be one of BACKLOG, PLANNED, DOING, BLOCKED, DONE, CANCELED, DUPLICATE", s)
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
		return nil, jsonio.ListOutput{}, err
	}
	out := jsonio.ListOutput{Issues: make([]jsonio.IssueSummary, 0, len(issues))}
	for _, i := range issues {
		out.Issues = append(out.Issues, jsonio.ToIssueSummary(i))
	}
	return textResult(fmt.Sprintf("%d issue(s)", len(out.Issues))), out, nil
}

func (t *toolset) show(ctx context.Context, req *mcp.CallToolRequest, in jsonio.ShowToolInput) (*mcp.CallToolResult, jsonio.ShowOutput, error) {
	if in.ID == "" {
		return nil, jsonio.ShowOutput{}, fmt.Errorf("'id' is required")
	}
	c := t.clientFor(req)
	issue, err := c.GetIssue(in.ID)
	if err != nil {
		return nil, jsonio.ShowOutput{}, err
	}
	out := jsonio.ShowOutput{
		ID:               issue.ID,
		Title:            issue.Title,
		Status:           string(issue.Status),
		IsInferred:       issue.InferredStatus,
		Description:      issue.Description,
		Labels:           issue.Labels,
		ParentID:         issue.ParentID,
		StoryPoints:      issue.Estimate,
		Assignee:         issue.Assignee,
		CycleID:          issue.CycleID,
		EffectiveCycleID: issue.EffectiveCycleID,
		BranchStats:      issue.BranchStats,
		CreatedBy:        issue.CreatedBy,
		CreatedAt:        issue.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        issue.UpdatedAt.Format(time.RFC3339),
		Dependencies:     issue.Dependencies,
		Artifacts:        jsonio.ToArtifactEntries(issue.Artifacts),
		Comments:         jsonio.ToCommentSummaries(issue.Comments),
	}
	if in.IncludeEvents {
		out.Events = jsonio.ToEventSummaries(issue.Events)
	}
	if in.IncludeSpec {
		if content, err := c.ReadSpec(issue.ID); err == nil {
			out.Spec = content
		}
	}
	if in.IncludeWalkthrough {
		if content, err := c.ReadWalkthrough(issue.ID); err == nil {
			out.Walkthrough = content
		}
	}
	return textResult(fmt.Sprintf("%s — %s [%s]", issue.ID, issue.Title, issue.Status)), out, nil
}

func (t *toolset) history(ctx context.Context, req *mcp.CallToolRequest, in jsonio.HistoryToolInput) (*mcp.CallToolResult, jsonio.HistoryOutput, error) {
	if in.ID == "" {
		return nil, jsonio.HistoryOutput{}, fmt.Errorf("'id' is required")
	}
	c := t.clientFor(req)
	issue, err := c.GetIssue(in.ID)
	if err != nil {
		return nil, jsonio.HistoryOutput{}, err
	}
	return textResult(fmt.Sprintf("%d event(s)", len(issue.Events))), jsonio.HistoryOutput{Timeline: jsonio.EventsToTimelineEntries(issue.Events)}, nil
}

func (t *toolset) add(ctx context.Context, req *mcp.CallToolRequest, in jsonio.AddInput) (*mcp.CallToolResult, jsonio.AddOutput, error) {
	payload, err := in.ToCreatePayload()
	if err != nil {
		return nil, jsonio.AddOutput{}, err
	}
	c := t.clientFor(req)
	if err := c.ValidateCreatePayload(&payload); err != nil {
		return nil, jsonio.AddOutput{}, err
	}
	issue, err := c.AddIssue(payload)
	if err != nil {
		return nil, jsonio.AddOutput{}, err
	}
	t.broadcast("CREATE", issue.ID)
	return textResult(fmt.Sprintf("Created %s: %s", issue.ID, issue.Title)),
		jsonio.AddOutput{ID: issue.ID, Title: issue.Title, Status: string(issue.Status)}, nil
}

func (t *toolset) update(ctx context.Context, req *mcp.CallToolRequest, in jsonio.UpdateToolInput) (*mcp.CallToolResult, jsonio.UpdateOutput, error) {
	if in.ID == "" {
		return nil, jsonio.UpdateOutput{}, fmt.Errorf("'id' is required")
	}
	payload, err := in.UpdateInput.ToUpdatePayload()
	if err != nil {
		return nil, jsonio.UpdateOutput{}, err
	}
	if jsonio.UpdatePayloadEmpty(payload) {
		return nil, jsonio.UpdateOutput{}, fmt.Errorf("no fields set: provide at least one field to update")
	}
	c := t.clientFor(req)
	issue, err := c.GetIssue(in.ID)
	if err != nil {
		return nil, jsonio.UpdateOutput{}, err
	}
	if err := c.ValidateUpdatePayload(&payload); err != nil {
		return nil, jsonio.UpdateOutput{}, err
	}
	msgs, err := c.UpdateIssue(issue.ID, payload, "update")
	if err != nil {
		return nil, jsonio.UpdateOutput{}, err
	}
	t.broadcast("UPDATE", issue.ID)
	return textResult(strings.Join(msgs, "\n")), jsonio.UpdateOutput{ID: issue.ID, Messages: msgs}, nil
}

func (t *toolset) comment(ctx context.Context, req *mcp.CallToolRequest, in jsonio.CommentToolInput) (*mcp.CallToolResult, jsonio.CommentOutput, error) {
	if in.ID == "" {
		return nil, jsonio.CommentOutput{}, fmt.Errorf("'id' is required")
	}
	if in.Body == "" {
		return nil, jsonio.CommentOutput{}, fmt.Errorf("'body' is required")
	}
	c := t.clientFor(req)
	issue, err := c.GetIssue(in.ID)
	if err != nil {
		return nil, jsonio.CommentOutput{}, err
	}
	if err := c.AddComment(issue.ID, in.Body); err != nil {
		return nil, jsonio.CommentOutput{}, err
	}
	t.broadcast("COMMENT", issue.ID)
	return textResult(fmt.Sprintf("Comment added to %s", issue.ID)), jsonio.CommentOutput{ID: issue.ID}, nil
}

func (t *toolset) link(ctx context.Context, req *mcp.CallToolRequest, in jsonio.LinkToolInput) (*mcp.CallToolResult, jsonio.LinkOutput, error) {
	if in.Source == "" || in.Target == "" {
		return nil, jsonio.LinkOutput{}, fmt.Errorf("'source' and 'target' are required")
	}
	kind := model.NormalizeDependencyKind(in.Type)
	if kind == "" {
		return nil, jsonio.LinkOutput{}, fmt.Errorf("invalid link type %q", in.Type)
	}
	c := t.clientFor(req)
	src, err := c.GetIssue(in.Source)
	if err != nil {
		return nil, jsonio.LinkOutput{}, fmt.Errorf("source: %w", err)
	}
	tgt, err := c.GetIssue(in.Target)
	if err != nil {
		return nil, jsonio.LinkOutput{}, fmt.Errorf("target: %w", err)
	}
	if src.ID == tgt.ID {
		return nil, jsonio.LinkOutput{}, fmt.Errorf("cannot link an issue to itself")
	}
	for _, dep := range src.Dependencies {
		if dep.TargetID == tgt.ID && string(dep.Kind) == kind {
			return nil, jsonio.LinkOutput{}, fmt.Errorf("link %s %s already exists on %s", kind, tgt.ID, src.ID)
		}
	}
	newDeps := append(src.Dependencies, model.Dependency{
		SourceID: src.ID,
		TargetID: tgt.ID,
		Kind:     model.DependencyKind(kind),
	})
	if _, err := c.UpdateIssue(src.ID, model.UpdatePayload{Dependencies: newDeps}, "link"); err != nil {
		return nil, jsonio.LinkOutput{}, err
	}
	t.broadcast("UPDATE", src.ID)
	return textResult(fmt.Sprintf("Linked %s %s %s", src.ID, kind, tgt.ID)),
		jsonio.LinkOutput{Source: src.ID, Target: tgt.ID, Kind: kind}, nil
}

func (t *toolset) start(ctx context.Context, req *mcp.CallToolRequest, in jsonio.StartToolInput) (*mcp.CallToolResult, jsonio.StartOutput, error) {
	if in.ID == "" {
		return nil, jsonio.StartOutput{}, fmt.Errorf("'id' is required")
	}
	c := t.clientFor(req)
	switch in.Mode {
	case "":
		// no override — use global config
	case "worktree", "branch":
		cfgCopy := *c.Config
		cfgCopy.Worktrees = in.Mode == "worktree"
		c.Config = &cfgCopy
	default:
		return nil, jsonio.StartOutput{}, fmt.Errorf("invalid mode %q: must be \"worktree\" or \"branch\"", in.Mode)
	}
	issue, err := c.GetIssue(in.ID)
	if err != nil {
		return nil, jsonio.StartOutput{}, err
	}
	branch, wtPath, msgs, err := c.StartWork(issue.ID, in.Force)
	if err != nil {
		return nil, jsonio.StartOutput{}, err
	}
	t.broadcast("UPDATE", issue.ID)
	text := strings.Join(msgs, "\n")
	return textResult(text), jsonio.StartOutput{ID: issue.ID, Branch: branch, WorktreePath: wtPath, Messages: msgs}, nil
}

func (t *toolset) merge(ctx context.Context, req *mcp.CallToolRequest, in jsonio.MergeToolInput) (*mcp.CallToolResult, jsonio.MergeOutput, error) {
	if in.ID == "" {
		return nil, jsonio.MergeOutput{}, fmt.Errorf("'id' is required")
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
		return nil, jsonio.MergeOutput{}, fmt.Errorf("invalid merge strategy %q: must be one of squash, merge, ff", in.Strategy)
	}

	issue, err := c.ResolveReviewIssue(in.ID)
	if err != nil {
		return nil, jsonio.MergeOutput{}, err
	}
	if issue.BranchStats == nil {
		return nil, jsonio.MergeOutput{}, fmt.Errorf("no branch found for %s", issue.ID)
	}

	if err := exponential.HubCleanForMerge(issue.BranchStats.Branch); err != nil {
		return nil, jsonio.MergeOutput{}, err
	}

	result, err := c.MergeIssue(issue.ID, exponential.MergeOptions{
		Strategy:      strategy,
		CommitMessage: in.CommitMessage,
		DeleteBranch:  !in.KeepBranch,
	})
	if err != nil {
		return nil, jsonio.MergeOutput{}, err
	}
	t.broadcast("MERGE", issue.ID)

	text := strings.Join(result.Messages, "\n")
	return textResult(text), jsonio.MergeOutput{ID: issue.ID, MergeSHA: result.MergeSHA, Messages: result.Messages}, nil
}

// --- Artifact handlers ---

func (t *toolset) spec(ctx context.Context, req *mcp.CallToolRequest, in jsonio.SpecToolInput) (*mcp.CallToolResult, jsonio.SpecOutput, error) {
	if in.IssueID == "" {
		return nil, jsonio.SpecOutput{OK: false, Error: "'issue_id' is required"}, fmt.Errorf("'issue_id' is required")
	}
	c := t.clientFor(req)
	issue, err := c.GetIssue(in.IssueID)
	if err != nil {
		return nil, jsonio.SpecOutput{OK: false, Error: err.Error()}, err
	}
	issueID := issue.ID
	path := filepath.Join(storage.XpoDir(), "artifacts", issueID, "spec.md")

	switch in.Operation {
	case "write":
		if in.Content == "" {
			return nil, jsonio.SpecOutput{OK: false, Error: "'content' is required for write"}, fmt.Errorf("'content' is required for write")
		}
		if err := c.WriteSpec(issueID, in.Content); err != nil {
			return nil, jsonio.SpecOutput{OK: false, Error: err.Error()}, err
		}
		t.broadcast("ARTIFACT", issueID)
		return textResult(fmt.Sprintf("Spec written for %s", issueID)),
			jsonio.SpecOutput{OK: true, IssueID: issueID, Path: path}, nil

	case "read":
		content, err := c.ReadSpec(issueID)
		if err != nil {
			return nil, jsonio.SpecOutput{OK: false, Error: err.Error()}, err
		}
		return textResult(content),
			jsonio.SpecOutput{OK: true, IssueID: issueID, Path: path, Content: content}, nil

	case "delete":
		if err := c.DeleteSpec(issueID); err != nil {
			return nil, jsonio.SpecOutput{OK: false, Error: err.Error()}, err
		}
		t.broadcast("ARTIFACT", issueID)
		return textResult(fmt.Sprintf("Spec deleted from %s", issueID)),
			jsonio.SpecOutput{OK: true, IssueID: issueID, Path: path}, nil

	default:
		return nil, jsonio.SpecOutput{OK: false, Error: "invalid operation: must be write, read, or delete"},
			fmt.Errorf("invalid operation %q: must be write, read, or delete", in.Operation)
	}
}

func (t *toolset) walkthrough(ctx context.Context, req *mcp.CallToolRequest, in jsonio.WalkthroughToolInput) (*mcp.CallToolResult, jsonio.WalkthroughOutput, error) {
	if in.IssueID == "" {
		return nil, jsonio.WalkthroughOutput{OK: false, Error: "'issue_id' is required"}, fmt.Errorf("'issue_id' is required")
	}
	c := t.clientFor(req)
	issue, err := c.GetIssue(in.IssueID)
	if err != nil {
		return nil, jsonio.WalkthroughOutput{OK: false, Error: err.Error()}, err
	}
	issueID := issue.ID
	path := filepath.Join(storage.XpoDir(), "artifacts", issueID, "walkthrough.md")

	switch in.Operation {
	case "write":
		if in.Content == "" {
			return nil, jsonio.WalkthroughOutput{OK: false, Error: "'content' is required for write"}, fmt.Errorf("'content' is required for write")
		}
		if err := c.WriteWalkthrough(issueID, in.Content); err != nil {
			return nil, jsonio.WalkthroughOutput{OK: false, Error: err.Error()}, err
		}
		t.broadcast("ARTIFACT", issueID)
		return textResult(fmt.Sprintf("Walkthrough written for %s", issueID)),
			jsonio.WalkthroughOutput{OK: true, IssueID: issueID, Path: path}, nil

	case "read":
		content, err := c.ReadWalkthrough(issueID)
		if err != nil {
			return nil, jsonio.WalkthroughOutput{OK: false, Error: err.Error()}, err
		}
		return textResult(content),
			jsonio.WalkthroughOutput{OK: true, IssueID: issueID, Path: path, Content: content}, nil

	case "delete":
		if err := c.DeleteWalkthrough(issueID); err != nil {
			return nil, jsonio.WalkthroughOutput{OK: false, Error: err.Error()}, err
		}
		t.broadcast("ARTIFACT", issueID)
		return textResult(fmt.Sprintf("Walkthrough deleted from %s", issueID)),
			jsonio.WalkthroughOutput{OK: true, IssueID: issueID, Path: path}, nil

	default:
		return nil, jsonio.WalkthroughOutput{OK: false, Error: "invalid operation: must be write, read, or delete"},
			fmt.Errorf("invalid operation %q: must be write, read, or delete", in.Operation)
	}
}

func (t *toolset) artifact(ctx context.Context, req *mcp.CallToolRequest, in jsonio.ArtifactToolInput) (*mcp.CallToolResult, jsonio.ArtifactOutput, error) {
	if in.IssueID == "" {
		return nil, jsonio.ArtifactOutput{OK: false, Error: "'issue_id' is required"}, fmt.Errorf("'issue_id' is required")
	}
	c := t.clientFor(req)
	issue, err := c.GetIssue(in.IssueID)
	if err != nil {
		return nil, jsonio.ArtifactOutput{OK: false, Error: err.Error()}, err
	}
	issueID := issue.ID

	switch in.Operation {
	case "add":
		if in.Filename == "" {
			return nil, jsonio.ArtifactOutput{OK: false, Error: "'filename' is required for add"}, fmt.Errorf("'filename' is required for add")
		}
		if in.Content == "" {
			return nil, jsonio.ArtifactOutput{OK: false, Error: "'content' is required for add"}, fmt.Errorf("'content' is required for add")
		}
		if err := c.AddArtifact(issueID, "generic", in.Filename, in.Content); err != nil {
			return nil, jsonio.ArtifactOutput{OK: false, Error: err.Error()}, err
		}
		path := filepath.Join(storage.XpoDir(), "artifacts", issueID, in.Filename)
		t.broadcast("ARTIFACT", issueID)
		return textResult(fmt.Sprintf("Artifact %s added to %s", in.Filename, issueID)),
			jsonio.ArtifactOutput{OK: true, IssueID: issueID, Path: path}, nil

	case "read":
		if in.Filename == "" {
			return nil, jsonio.ArtifactOutput{OK: false, Error: "'filename' is required for read"}, fmt.Errorf("'filename' is required for read")
		}
		content, err := c.ReadArtifact(issueID, in.Filename)
		if err != nil {
			return nil, jsonio.ArtifactOutput{OK: false, Error: err.Error()}, err
		}
		path := filepath.Join(storage.XpoDir(), "artifacts", issueID, in.Filename)
		return textResult(content),
			jsonio.ArtifactOutput{OK: true, IssueID: issueID, Path: path, Content: content}, nil

	case "delete":
		if in.Filename == "" {
			return nil, jsonio.ArtifactOutput{OK: false, Error: "'filename' is required for delete"}, fmt.Errorf("'filename' is required for delete")
		}
		if err := c.DeleteArtifact(issueID, in.Filename); err != nil {
			return nil, jsonio.ArtifactOutput{OK: false, Error: err.Error()}, err
		}
		path := filepath.Join(storage.XpoDir(), "artifacts", issueID, in.Filename)
		t.broadcast("ARTIFACT", issueID)
		return textResult(fmt.Sprintf("Artifact %s deleted from %s", in.Filename, issueID)),
			jsonio.ArtifactOutput{OK: true, IssueID: issueID, Path: path}, nil

	case "list":
		artifacts, err := c.ListArtifacts(issueID)
		if err != nil {
			return nil, jsonio.ArtifactOutput{OK: false, Error: err.Error()}, err
		}
		entries := jsonio.ToArtifactEntries(artifacts)
		return textResult(fmt.Sprintf("%d artifact(s)", len(entries))),
			jsonio.ArtifactOutput{OK: true, IssueID: issueID, Artifacts: entries}, nil

	default:
		return nil, jsonio.ArtifactOutput{OK: false, Error: "invalid operation: must be add, read, delete, or list"},
			fmt.Errorf("invalid operation %q: must be add, read, delete, or list", in.Operation)
	}
}

func (t *toolset) rationale(ctx context.Context, req *mcp.CallToolRequest, in jsonio.RationaleToolInput) (*mcp.CallToolResult, jsonio.RationaleOutput, error) {
	if in.Query == "" {
		return nil, jsonio.RationaleOutput{}, fmt.Errorf("'query' is required")
	}
	c := t.clientFor(req)
	result, err := c.SearchRationale(in.Query, in.Limit)
	if err != nil {
		return nil, jsonio.RationaleOutput{}, err
	}
	out := jsonio.RationaleOutput{
		Query:        result.Query,
		TotalMatches: result.TotalMatches,
		Results:      make([]jsonio.RationaleHit, len(result.Results)),
	}
	for i, r := range result.Results {
		out.Results[i] = jsonio.RationaleHit{
			IssueID:   r.IssueID,
			Title:     r.Title,
			Status:    r.Status,
			Labels:    r.Labels,
			Document:  r.Document,
			Fragment:  r.Fragment,
			Score:     r.Score,
			UpdatedAt: r.UpdatedAt,
		}
	}
	return textResult(fmt.Sprintf("%d result(s) for %q", len(out.Results), in.Query)), out, nil
}

// --- Helpers ---

func textResult(s string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: s}},
	}
}
