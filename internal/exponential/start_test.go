package exponential

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/palarix/exponential/internal/config"
	"github.com/palarix/exponential/internal/model"
	"github.com/palarix/exponential/internal/storage"
)

func setupStartTestEnv(t *testing.T) (*Client, func()) {
	t.Helper()
	tmpDir := t.TempDir()
	xpoDir := filepath.Join(tmpDir, ".xpo")
	os.MkdirAll(xpoDir, 0755)
	os.WriteFile(filepath.Join(xpoDir, "issues.db"), []byte{}, 0644)

	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	storage.ResetHubRoot()

	cfg := &config.Config{
		Prefix:           "test-",
		User:             "Test User <test@test.com>",
		EstimationSystem: "fibonacci",
		CountUnestimated: true,
		Version:          2,
		Worktrees:        false,
	}

	client := NewClient(cfg)
	cleanup := func() {
		os.Chdir(origDir)
		storage.ResetHubRoot()
	}
	return client, cleanup
}

func createTestIssue(t *testing.T, id, title, status string) {
	t.Helper()
	evt := model.Event{
		ID:   id,
		Type: model.EventTypeCreate,
		Payload: model.CreatePayload{
			Title:  title,
			Status: status,
		},
	}
	if err := storage.AppendEvent(evt); err != nil {
		t.Fatalf("failed to create test issue: %v", err)
	}
}

func TestStartWork_NonGitRepo(t *testing.T) {
	client, cleanup := setupStartTestEnv(t)
	defer cleanup()

	createTestIssue(t, "test-abc123", "Fix login", "PLANNED")

	branch, _, msgs, err := client.StartWork("test-abc123", false)
	if err != nil {
		t.Fatalf("StartWork() unexpected error: %v", err)
	}
	if branch != "" {
		t.Errorf("expected empty branch in non-git repo, got %q", branch)
	}
	if len(msgs) == 0 {
		t.Error("expected update messages")
	}

	issue, err := client.GetIssue("test-abc123")
	if err != nil {
		t.Fatalf("GetIssue() error: %v", err)
	}
	if issue.Status != model.StatusDoing {
		t.Errorf("expected DOING, got %s", issue.Status)
	}
}

func TestStartWork_DoneIssue(t *testing.T) {
	client, cleanup := setupStartTestEnv(t)
	defer cleanup()

	createTestIssue(t, "test-abc123", "Fix login", "DONE")

	_, _, _, err := client.StartWork("test-abc123", false)
	if err == nil {
		t.Fatal("expected error for DONE issue")
	}
}

func TestStartWork_BlockedIssue(t *testing.T) {
	client, cleanup := setupStartTestEnv(t)
	defer cleanup()

	createTestIssue(t, "test-abc123", "Fix login", "BLOCKED")

	_, _, _, err := client.StartWork("test-abc123", false)
	if err == nil {
		t.Fatal("expected error for BLOCKED issue")
	}
}

func TestStartWork_GitRepo_CreatesBranch(t *testing.T) {
	client, cleanup := setupStartTestEnv(t)
	defer cleanup()

	cwd, _ := os.Getwd()
	runGit(t, cwd, "init", "-b", "main")
	runGit(t, cwd, "config", "user.email", "test@test.com")
	runGit(t, cwd, "config", "user.name", "Test")
	os.WriteFile("dummy.txt", []byte("init"), 0644)
	runGit(t, cwd, "add", ".")
	runGit(t, cwd, "commit", "-m", "init")

	createTestIssue(t, "test-abc123", "Fix Login Flow", "PLANNED")

	branch, _, msgs, err := client.StartWork("test-abc123", false)
	if err != nil {
		t.Fatalf("StartWork() unexpected error: %v", err)
	}
	if branch != "test-abc123-fix-login-flow" {
		t.Errorf("expected branch 'test-abc123-fix-login-flow', got %q", branch)
	}

	foundBranchMsg := false
	for _, msg := range msgs {
		if msg == "Created and switched to branch 'test-abc123-fix-login-flow'" {
			foundBranchMsg = true
		}
	}
	if !foundBranchMsg {
		t.Errorf("expected branch creation message in %v", msgs)
	}

	out, _ := exec.Command("git", "branch", "--show-current").Output()
	currentBranch := string(out)
	if currentBranch[:len(currentBranch)-1] != "test-abc123-fix-login-flow" {
		t.Errorf("expected to be on branch test-abc123-fix-login-flow, got %q", currentBranch)
	}
}

func TestStartWork_GitRepo_ExistingBranch_Rejected(t *testing.T) {
	client, cleanup := setupStartTestEnv(t)
	defer cleanup()

	cwd, _ := os.Getwd()
	runGit(t, cwd, "init", "-b", "main")
	runGit(t, cwd, "config", "user.email", "test@test.com")
	runGit(t, cwd, "config", "user.name", "Test")
	os.WriteFile("dummy.txt", []byte("init"), 0644)
	runGit(t, cwd, "add", ".")
	runGit(t, cwd, "commit", "-m", "init")

	runGit(t, cwd, "branch", "test-abc123-fix-login-flow")
	createTestIssue(t, "test-abc123", "Fix Login Flow", "PLANNED")

	_, _, _, err := client.StartWork("test-abc123", false)
	if err == nil {
		t.Fatal("expected error for existing branch")
	}
	if !strings.Contains(err.Error(), "branch already exists") {
		t.Errorf("expected 'branch already exists' error, got: %v", err)
	}
}

func TestStartWork_GitRepo_ExistingBranch_Force(t *testing.T) {
	client, cleanup := setupStartTestEnv(t)
	defer cleanup()

	cwd, _ := os.Getwd()
	runGit(t, cwd, "init", "-b", "main")
	runGit(t, cwd, "config", "user.email", "test@test.com")
	runGit(t, cwd, "config", "user.name", "Test")
	os.WriteFile("dummy.txt", []byte("init"), 0644)
	runGit(t, cwd, "add", ".")
	runGit(t, cwd, "commit", "-m", "init")

	runGit(t, cwd, "branch", "test-abc123-fix-login-flow")
	createTestIssue(t, "test-abc123", "Fix Login Flow", "PLANNED")

	branch, _, _, err := client.StartWork("test-abc123", true)
	if err != nil {
		t.Fatalf("StartWork(force=true) unexpected error: %v", err)
	}
	if branch != "test-abc123-fix-login-flow" {
		t.Errorf("expected branch 'test-abc123-fix-login-flow', got %q", branch)
	}
}

func TestStartWork_AlreadyDoing_Rejected(t *testing.T) {
	client, cleanup := setupStartTestEnv(t)
	defer cleanup()

	createTestIssue(t, "test-abc123", "Fix login", "DOING")

	_, _, _, err := client.StartWork("test-abc123", false)
	if err == nil {
		t.Fatal("expected error for DOING issue")
	}
	if !strings.Contains(err.Error(), "already in progress") {
		t.Errorf("expected 'already in progress' error, got: %v", err)
	}
}

func TestStartWork_AlreadyDoing_Force(t *testing.T) {
	client, cleanup := setupStartTestEnv(t)
	defer cleanup()

	createTestIssue(t, "test-abc123", "Fix login", "DOING")

	_, _, msgs, err := client.StartWork("test-abc123", true)
	if err != nil {
		t.Fatalf("StartWork(force=true) unexpected error: %v", err)
	}
	foundForceMsg := false
	for _, msg := range msgs {
		if strings.Contains(msg, "Force-claiming") {
			foundForceMsg = true
		}
	}
	if !foundForceMsg {
		t.Errorf("expected force-claiming message in %v", msgs)
	}
}

func TestStartWork_Worktree_CreatesWorktree(t *testing.T) {
	tmpDir := t.TempDir()
	xpoDir := filepath.Join(tmpDir, ".xpo")
	os.MkdirAll(xpoDir, 0755)
	os.WriteFile(filepath.Join(xpoDir, "issues.db"), []byte{}, 0644)

	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	storage.ResetHubRoot()
	defer func() {
		os.Chdir(origDir)
		storage.ResetHubRoot()
	}()

	runGit(t, tmpDir, "init", "-b", "main")
	runGit(t, tmpDir, "config", "user.email", "test@test.com")
	runGit(t, tmpDir, "config", "user.name", "Test")
	os.WriteFile(filepath.Join(tmpDir, "dummy.txt"), []byte("init"), 0644)
	runGit(t, tmpDir, "add", ".")
	runGit(t, tmpDir, "commit", "-m", "init")

	cfg := &config.Config{
		Prefix:           "test-",
		User:             "Test User <test@test.com>",
		EstimationSystem: "fibonacci",
		CountUnestimated: true,
		Version:          2,
		Worktrees:        true,
	}
	client := NewClient(cfg)

	createTestIssue(t, "test-wt001", "Worktree Feature", "PLANNED")

	branch, wtPath, msgs, err := client.StartWork("test-wt001", false)
	if err != nil {
		t.Fatalf("StartWork() unexpected error: %v", err)
	}
	if branch != "test-wt001-worktree-feature" {
		t.Errorf("expected branch 'test-wt001-worktree-feature', got %q", branch)
	}
	if wtPath == "" {
		t.Fatal("expected non-empty worktree path")
	}

	if _, err := os.Stat(wtPath); os.IsNotExist(err) {
		t.Fatalf("worktree directory does not exist: %s", wtPath)
	}

	foundWorktreeMsg := false
	for _, msg := range msgs {
		if strings.Contains(msg, "Created worktree") {
			foundWorktreeMsg = true
		}
	}
	if !foundWorktreeMsg {
		t.Errorf("expected worktree creation message in %v", msgs)
	}

	// Verify primary checkout is still on main
	out, _ := exec.Command("git", "branch", "--show-current").Output()
	currentBranch := strings.TrimSpace(string(out))
	if currentBranch != "main" {
		t.Errorf("expected primary checkout to stay on main, got %q", currentBranch)
	}

	// Verify the branch exists in the worktree
	path, found := FindWorktreeForBranch("test-wt001-worktree-feature")
	if !found {
		t.Fatal("expected to find worktree for branch")
	}
	if path != wtPath {
		t.Errorf("worktree path mismatch: got %q, want %q", path, wtPath)
	}
}
