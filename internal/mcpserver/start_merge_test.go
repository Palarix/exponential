package mcpserver

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/palarix/exponential/internal/config"
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

	mcpRunGit(t, dir, "init", "-b", "main")
	mcpRunGit(t, dir, "config", "user.email", "test@test.com")
	mcpRunGit(t, dir, "config", "user.name", "Test")
	os.WriteFile(filepath.Join(dir, "base.txt"), []byte("base\n"), 0644)
	mcpRunGit(t, dir, "add", ".")
	mcpRunGit(t, dir, "commit", "-m", "init")

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	storage.ResetHubRoot()
	storage.ResetRefStore()
	if err := storage.InitRefStore(); err != nil {
		t.Fatalf("InitRefStore: %v", err)
	}
	t.Cleanup(func() {
		os.Chdir(origDir)
		storage.ResetHubRoot()
		storage.ResetRefStore()
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

func TestMCPStart_BranchMode(t *testing.T) {
	ts, _ := setupGitToolset(t, nil)
	seedMCPIssue(t, "test-br01", "Branch Mode", "PLANNED")

	_, out, err := ts.start(context.Background(), nil, startIn{
		ID:   "test-br01",
		Mode: "branch",
	})
	if err != nil {
		t.Fatalf("start failed: %v", err)
	}
	if out.Branch == "" {
		t.Error("expected non-empty branch")
	}
	if out.WorktreePath != "" {
		t.Errorf("expected empty worktree path in branch mode, got %q", out.WorktreePath)
	}
}

func TestMCPStart_WorktreeMode(t *testing.T) {
	ts, _ := setupGitToolset(t, nil)
	seedMCPIssue(t, "test-wt01", "Worktree Mode", "PLANNED")

	_, out, err := ts.start(context.Background(), nil, startIn{
		ID:   "test-wt01",
		Mode: "worktree",
	})
	if err != nil {
		t.Fatalf("start failed: %v", err)
	}
	if out.Branch == "" {
		t.Error("expected non-empty branch")
	}
	if out.WorktreePath == "" {
		t.Error("expected non-empty worktree path in worktree mode")
	}
}

func TestMCPStart_InvalidMode(t *testing.T) {
	ts, _ := setupGitToolset(t, nil)
	seedMCPIssue(t, "test-inv01", "Invalid Mode", "PLANNED")

	_, _, err := ts.start(context.Background(), nil, startIn{
		ID:   "test-inv01",
		Mode: "invalid",
	})
	if err == nil {
		t.Fatal("expected error for invalid mode")
	}
	if !strings.Contains(err.Error(), "invalid mode") {
		t.Errorf("expected 'invalid mode' in error, got: %v", err)
	}
}

func TestMCPStart_MissingID(t *testing.T) {
	ts, _ := setupGitToolset(t, nil)

	_, _, err := ts.start(context.Background(), nil, startIn{})
	if err == nil {
		t.Fatal("expected error for missing ID")
	}
}

// --- merge tool tests ---

func TestMCPMerge_Squash(t *testing.T) {
	ts, dir := setupGitToolset(t, nil)
	seedMCPIssue(t, "test-sq01", "Squash Merge", "PLANNED")

	// Start work to create the branch
	_, _, err := ts.start(context.Background(), nil, startIn{
		ID:   "test-sq01",
		Mode: "branch",
	})
	if err != nil {
		t.Fatalf("start failed: %v", err)
	}

	// Add a commit on the branch
	os.WriteFile(filepath.Join(dir, "feature.txt"), []byte("feature\n"), 0644)
	mcpRunGit(t, dir, "add", ".")
	mcpRunGit(t, dir, "commit", "-m", "add feature")

	_, out, err := ts.merge(context.Background(), nil, mergeIn{
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
	ts, dir := setupGitToolset(t, nil)
	seedMCPIssue(t, "test-def01", "Default Strategy", "PLANNED")

	_, _, err := ts.start(context.Background(), nil, startIn{
		ID:   "test-def01",
		Mode: "branch",
	})
	if err != nil {
		t.Fatalf("start failed: %v", err)
	}

	os.WriteFile(filepath.Join(dir, "feature.txt"), []byte("feature\n"), 0644)
	mcpRunGit(t, dir, "add", ".")
	mcpRunGit(t, dir, "commit", "-m", "add feature")

	// Empty strategy should default to squash
	_, out, err := ts.merge(context.Background(), nil, mergeIn{
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

	_, _, err := ts.merge(context.Background(), nil, mergeIn{
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

	_, _, err := ts.merge(context.Background(), nil, mergeIn{})
	if err == nil {
		t.Fatal("expected error for missing ID")
	}
}
