package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/palarix/exponential/internal/model"
)

func TestRenderIssueDetails_Minimal(t *testing.T) {
	issue := &model.Issue{
		ID:        "test-abc123",
		Title:     "Test Issue",
		Status:    model.StatusBacklog,
		CreatedBy: "Alice <alice@test.com>",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	got := RenderIssueDetails(issue, nil, false, 80)
	if got == "" {
		t.Fatal("should render non-empty")
	}
	if !strings.Contains(got, "Test Issue") {
		t.Error("should contain title")
	}
	if !strings.Contains(got, "test-abc123") {
		t.Error("should contain issue ID")
	}
}

func TestRenderIssueDetails_AllFields(t *testing.T) {
	issue := &model.Issue{
		ID:          "test-abc123",
		Title:       "Full Issue",
		Status:      model.StatusDoing,
		Description: "A detailed description",
		Assignee:    "Bob <bob@test.com>",
		ParentID:    "test-parent",
		Estimate:    5,
		Labels:      []string{"bug", "feature"},
		CreatedBy:   "Alice <alice@test.com>",
		CreatedAt:   time.Now().Add(-48 * time.Hour),
		UpdatedAt:   time.Now(),
		BranchStats: &model.BranchStats{Branch: "feat/test", Commits: 3},
		Dependencies: []model.Dependency{
			{SourceID: "test-abc123", TargetID: "test-other", Kind: "depends_on"},
		},
		Comments: []model.Comment{
			{ID: "c1", Text: "A comment", CreatedBy: "Bob <bob@test.com>", CreatedAt: time.Now()},
		},
	}
	children := []*model.Issue{
		{ID: "test-child1", Title: "Child 1", Status: model.StatusDone},
		{ID: "test-child2", Title: "Child 2", Status: model.StatusPlanned},
	}
	got := RenderIssueDetails(issue, children, false, 120)
	if !strings.Contains(got, "Bob") {
		t.Error("should contain assignee")
	}
	if !strings.Contains(got, "5") {
		t.Error("should contain estimate")
	}
}

func TestRenderIssueDetails_Archived(t *testing.T) {
	issue := &model.Issue{
		ID:        "test-abc123",
		Title:     "Archived Issue",
		Status:    model.StatusDone,
		CreatedBy: "Alice <a@b.com>",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	got := RenderIssueDetails(issue, nil, true, 80)
	if !strings.Contains(got, "ARCHIVED") {
		t.Error("should indicate archived status")
	}
}

func TestRenderIssueDetails_NoDescription(t *testing.T) {
	issue := &model.Issue{
		ID:        "test-abc123",
		Title:     "No Desc",
		Status:    model.StatusBacklog,
		CreatedBy: "A <a@b.com>",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	got := RenderIssueDetails(issue, nil, false, 80)
	if got == "" {
		t.Fatal("should render even without description")
	}
}

func TestRenderIssueDetails_AssigneeVia(t *testing.T) {
	issue := &model.Issue{
		ID:          "test-abc123",
		Title:       "Agent work",
		Status:      model.StatusDoing,
		Assignee:    "Nicolas <nic@example.com>",
		AssigneeVia: "claude-code/2.1.263 <agent@mcp>",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	got := RenderIssueDetails(issue, nil, false, 120)
	if !strings.Contains(got, "Nicolas <nic@example.com>") || strings.Contains(got, "via") {
		t.Errorf("assignee should be shown as the person only, got %q", got)
	}
}

func TestRenderIssueDetails_AgentCommentIsPrincipalFirst(t *testing.T) {
	issue := &model.Issue{
		ID:     "test-abc123",
		Title:  "T",
		Status: model.StatusDoing,
		Comments: []model.Comment{
			{ID: "c1", Text: "done", CreatedBy: "claude-code/2.1.263 <agent@mcp>", OnBehalfOf: "Nicolas <nic@x.com>", CreatedAt: time.Now()},
		},
	}
	got := RenderIssueDetails(issue, nil, false, 120)
	if !strings.Contains(got, "Nicolas (via Claude Code)") {
		t.Errorf("comment author should be principal-first:\n%s", got)
	}
	if strings.Contains(got, "claude-code/2.1.263") {
		t.Errorf("raw agent identity should not be shown:\n%s", got)
	}
}
