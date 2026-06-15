package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/palarix/beats/internal/model"
)

func TestRenderReview_NoBranch(t *testing.T) {
	data := ReviewData{
		Issue: &model.Issue{
			ID: "test-1", Title: "No branch", Status: model.StatusDoing,
			CreatedBy: "A <a@b.com>", CreatedAt: time.Now(), UpdatedAt: time.Now(),
		},
	}
	got := RenderReview(data, 80)
	if !strings.Contains(got, "No branch detected") {
		t.Errorf("expected no-branch message, got %q", got)
	}
}

func TestRenderReview_Full(t *testing.T) {
	data := ReviewData{
		Issue: &model.Issue{
			ID: "test-1", Title: "With branch", Status: model.StatusDoing,
			Description: "A description",
			BranchStats: &model.BranchStats{Branch: "feat/test", HeadSHA: "abc1234", Commits: 2, FilesChanged: 3, Insertions: 50, Deletions: 10},
			CreatedBy:   "A <a@b.com>", CreatedAt: time.Now(), UpdatedAt: time.Now(),
		},
		Base: "main",
		Commits: []ReviewCommit{
			{SHA: "abc1234", Message: "First commit"},
			{SHA: "def5678", Message: "Second commit"},
		},
		Files: []ReviewFile{
			{Status: "A", Path: "new.go", Insertions: 30, Deletions: 0},
			{Status: "M", Path: "mod.go", Insertions: 15, Deletions: 5},
			{Status: "D", Path: "old.go", Insertions: 0, Deletions: 5},
		},
	}
	got := RenderReview(data, 120)
	if got == "" {
		t.Fatal("should render non-empty")
	}
	if !strings.Contains(got, "feat/test") {
		t.Error("should contain branch name")
	}
	if !strings.Contains(got, "new.go") {
		t.Error("should contain file names")
	}
}

func TestRenderReview_SingleCommit(t *testing.T) {
	data := ReviewData{
		Issue: &model.Issue{
			ID: "test-1", Title: "One commit", Status: model.StatusDoing,
			BranchStats: &model.BranchStats{Branch: "feat/x", Commits: 1, FilesChanged: 1},
			CreatedBy:   "A <a@b.com>", CreatedAt: time.Now(), UpdatedAt: time.Now(),
		},
		Base:    "main",
		Commits: []ReviewCommit{{SHA: "aaa", Message: "Only commit"}},
		Files:   []ReviewFile{{Status: "M", Path: "f.go"}},
	}
	got := RenderReview(data, 80)
	if got == "" {
		t.Fatal("should render")
	}
}
