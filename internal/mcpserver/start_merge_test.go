package mcpserver

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/palarix/exponential/internal/config"
	"github.com/palarix/exponential/internal/jsonio"
	"github.com/palarix/exponential/internal/model"
	"github.com/palarix/exponential/internal/storage"
)

func mcpRunGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	fullArgs := append([]string{"-c", "safe.bareRepository=all"}, args...)
	cmd := exec.Command("git", fullArgs...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v in %s failed: %v\n%s", args, dir, err, out)
	}
}

func setupGitToolset(t *testing.T, cfgOverride *config.Config) (*toolset, string) {
	t.Helper()
	dir := t.TempDir()
	dir, _ = filepath.EvalSymlinks(dir)
	xpoDir := filepath.Join(dir, ".xpo")
	os.MkdirAll(xpoDir, 0755)
	os.WriteFile(filepath.Join(xpoDir, "issues.db"), []byte{}, 0644)

	mcpRunGit(t, dir, "init", "-b", "main")
	mcpRunGit(t, dir, "config", "user.email", "test@test.com")
	mcpRunGit(t, dir, "config", "user.name", "Test")
	os.WriteFile(filepath.Join(dir, "base.txt"), []byte("base\n"), 0644)
	mcpRunGit(t, dir, "add", ".")
	mcpRunGit(t, dir, "commit", "-m", "init")

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	storage.ResetHubRoot()
	t.Cleanup(func() {
		os.Chdir(origDir)
		storage.ResetHubRoot()
	})

	cfg := &config.Config{
		Prefix:           "test-",
		User:             "Test User <test@test.com>",
		EstimationSystem: "fibonacci",
		CountUnestimated: true,
		Version:          2,
	}
	if cfgOverride != nil {
		*cfg = *cfgOverride
		if cfg.Prefix == "" {
			cfg.Prefix = "test-"
		}
		if cfg.User == "" {
			cfg.User = "Test User <test@test.com>"
		}
	}
	return newToolset(cfg), dir
}

func seedMCPIssue(t *testing.T, id, title, status string) {
	t.Helper()
	evt := model.Event{
		ID:   id,
		Type: model.EventTypeCreate,
		Payload: model.CreatePayload{
			Title:  title,
			Status: status,
		},
		CreatedBy: "Test <test@test.com>",
	}
	if err := storage.AppendEvent(evt); err != nil {
		t.Fatalf("failed to seed issue: %v", err)
	}
}

// --- start tool tests ---

func TestMCPStart_CreatesWorktree(t *testing.T) {
	ts, _ := setupGitToolset(t, nil)
	seedMCPIssue(t, "test-wt01", "Worktree Mode", "PLANNED")

	_, out, err := ts.start(context.Background(), nil, jsonio.StartToolInput{ID: "test-wt01"})
	if err != nil {
		t.Fatalf("start failed: %v", err)
	}
	if out.Branch == "" {
		t.Error("expected non-empty branch")
	}
	if out.WorktreePath == "" {
		t.Error("expected a worktree path: start always creates a worktree")
	}
}

func TestMCPStart_MissingID(t *testing.T) {
	ts, _ := setupGitToolset(t, nil)

	_, _, err := ts.start(context.Background(), nil, jsonio.StartToolInput{})
	if err == nil {
		t.Fatal("expected error for missing ID")
	}
}

// --- merge tool tests ---

func TestMCPMerge_Squash(t *testing.T) {
	ts, _ := setupGitToolset(t, nil)
	seedMCPIssue(t, "test-sq01", "Squash Merge", "PLANNED")

	// Start work to create the worktree
	_, started, err := ts.start(context.Background(), nil, jsonio.StartToolInput{ID: "test-sq01"})
	if err != nil {
		t.Fatalf("start failed: %v", err)
	}
	wt := started.WorktreePath

	// Add a commit on the branch
	os.WriteFile(filepath.Join(wt, "feature.txt"), []byte("feature\n"), 0644)
	mcpRunGit(t, wt, "add", ".")
	mcpRunGit(t, wt, "commit", "-m", "add feature")

	_, out, err := ts.merge(context.Background(), nil, jsonio.MergeToolInput{
		ID:         "test-sq01",
		Strategy:   "squash",
		KeepBranch: true,
	})
	if err != nil {
		t.Fatalf("merge failed: %v", err)
	}
	if out.MergeSHA == "" {
		t.Error("expected non-empty merge SHA")
	}
}

func TestMCPMerge_DefaultStrategy(t *testing.T) {
	ts, _ := setupGitToolset(t, nil)
	seedMCPIssue(t, "test-def01", "Default Strategy", "PLANNED")

	_, started, err := ts.start(context.Background(), nil, jsonio.StartToolInput{ID: "test-def01"})
	if err != nil {
		t.Fatalf("start failed: %v", err)
	}
	wt := started.WorktreePath

	os.WriteFile(filepath.Join(wt, "feature.txt"), []byte("feature\n"), 0644)
	mcpRunGit(t, wt, "add", ".")
	mcpRunGit(t, wt, "commit", "-m", "add feature")

	// Empty strategy should default to squash
	_, out, err := ts.merge(context.Background(), nil, jsonio.MergeToolInput{
		ID:         "test-def01",
		Strategy:   "",
		KeepBranch: true,
	})
	if err != nil {
		t.Fatalf("merge failed: %v", err)
	}
	if out.MergeSHA == "" {
		t.Fatal("expected non-empty merge SHA")
	}

	// Verify squash semantics: HEAD should be a single-parent commit
	headOut, err := exec.Command("git", "cat-file", "-p", "HEAD").Output()
	if err != nil {
		t.Fatalf("git cat-file failed: %v", err)
	}
	parentCount := strings.Count(string(headOut), "\nparent ")
	if parentCount != 1 {
		t.Errorf("expected single-parent commit (squash default), got %d parents", parentCount)
	}
}

func TestMCPMerge_InvalidStrategy(t *testing.T) {
	ts, _ := setupGitToolset(t, nil)
	seedMCPIssue(t, "test-invstr01", "Invalid Strategy", "DOING")

	_, _, err := ts.merge(context.Background(), nil, jsonio.MergeToolInput{
		ID:       "test-invstr01",
		Strategy: "rebase",
	})
	if err == nil {
		t.Fatal("expected error for invalid strategy")
	}
	if !strings.Contains(err.Error(), "invalid merge strategy") {
		t.Errorf("expected 'invalid merge strategy' in error, got: %v", err)
	}
}

func TestMCPMerge_MissingID(t *testing.T) {
	ts, _ := setupGitToolset(t, nil)

	_, _, err := ts.merge(context.Background(), nil, jsonio.MergeToolInput{})
	if err == nil {
		t.Fatal("expected error for missing ID")
	}
}
