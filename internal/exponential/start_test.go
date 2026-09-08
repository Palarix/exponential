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

func TestStartWork_CanceledIssue(t *testing.T) {
	client, cleanup := setupStartTestEnv(t)
	defer cleanup()

	createTestIssue(t, "test-abc123", "Fix login", "CANCELED")

	_, _, _, err := client.StartWork("test-abc123", false)
	if err == nil {
		t.Fatal("expected error for CANCELED issue")
	}
	if !strings.Contains(err.Error(), "CANCELED") {
		t.Errorf("expected error to mention CANCELED, got: %v", err)
	}
}

func TestStartWork_DuplicateIssue(t *testing.T) {
	client, cleanup := setupStartTestEnv(t)
	defer cleanup()

	createTestIssue(t, "test-abc123", "Fix login", "DUPLICATE")

	_, _, _, err := client.StartWork("test-abc123", false)
	if err == nil {
		t.Fatal("expected error for DUPLICATE issue")
	}
	if !strings.Contains(err.Error(), "DUPLICATE") {
		t.Errorf("expected error to mention DUPLICATE, got: %v", err)
	}
}

func TestStartWork_BacklogIssue_Rejected(t *testing.T) {
	client, cleanup := setupStartTestEnv(t)
	defer cleanup()

	createTestIssue(t, "test-abc123", "Fix login", "BACKLOG")

	_, _, _, err := client.StartWork("test-abc123", false)
	if err == nil {
		t.Fatal("expected error for BACKLOG issue")
	}
	if !strings.Contains(err.Error(), "BACKLOG") {
		t.Errorf("expected error to mention BACKLOG, got: %v", err)
	}
	if !strings.Contains(err.Error(), "PLANNED") {
		t.Errorf("expected error to mention PLANNED, got: %v", err)
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

func setupWorktreeTestRepo(t *testing.T) (string, *Client) {
	t.Helper()
	tmpDir := t.TempDir()
	tmpDir, _ = filepath.EvalSymlinks(tmpDir)
	xpoDir := filepath.Join(tmpDir, ".xpo")
	os.MkdirAll(xpoDir, 0755)
	os.WriteFile(filepath.Join(xpoDir, "issues.db"), []byte{}, 0644)

	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	storage.ResetHubRoot()
	t.Cleanup(func() {
		os.Chdir(origDir)
		storage.ResetHubRoot()
	})

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
	return tmpDir, NewClient(cfg)
}

func TestStartWork_Worktree_ExistingBranch(t *testing.T) {
	tmpDir, client := setupWorktreeTestRepo(t)

	// Create a branch manually (simulates a previous start that was cleaned up
	// but the branch survived)
	runGit(t, tmpDir, "branch", "test-exist01-existing-branch")

	createTestIssue(t, "test-exist01", "Existing Branch", "PLANNED")

	branch, wtPath, msgs, err := client.StartWork("test-exist01", false)
	if err != nil {
		t.Fatalf("StartWork() unexpected error: %v", err)
	}
	if branch != "test-exist01-existing-branch" {
		t.Errorf("expected branch 'test-exist01-existing-branch', got %q", branch)
	}
	if wtPath == "" {
		t.Fatal("expected non-empty worktree path")
	}

	foundMsg := false
	for _, msg := range msgs {
		if strings.Contains(msg, "existing branch") {
			foundMsg = true
		}
	}
	if !foundMsg {
		t.Errorf("expected 'existing branch' message in %v", msgs)
	}

	// Verify worktree uses the pre-existing branch
	_, found := FindWorktreeForBranch("test-exist01-existing-branch")
	if !found {
		t.Fatal("expected worktree for existing branch")
	}
}

func TestStartWork_Worktree_ForceRemovesExisting(t *testing.T) {
	_, client := setupWorktreeTestRepo(t)
	createTestIssue(t, "test-force01", "Force Takeover", "PLANNED")

	// First start creates the worktree
	_, wtPath1, _, err := client.StartWork("test-force01", false)
	if err != nil {
		t.Fatalf("first StartWork() failed: %v", err)
	}
	if _, err := os.Stat(wtPath1); os.IsNotExist(err) {
		t.Fatal("first worktree should exist")
	}

	// Force start should remove and recreate
	_, wtPath2, msgs, err := client.StartWork("test-force01", true)
	if err != nil {
		t.Fatalf("force StartWork() failed: %v", err)
	}

	foundRemoveMsg := false
	for _, msg := range msgs {
		if strings.Contains(msg, "Removed existing worktree") {
			foundRemoveMsg = true
		}
	}
	if !foundRemoveMsg {
		t.Errorf("expected 'Removed existing worktree' message in %v", msgs)
	}

	if wtPath2 == "" {
		t.Fatal("expected non-empty worktree path after force")
	}
}

func TestStartWork_Worktree_SetupHook(t *testing.T) {
	_, client := setupWorktreeTestRepo(t)
	client.Config.WorktreeSetup = "touch .setup-ran"

	createTestIssue(t, "test-hook01", "Setup Hook", "PLANNED")

	_, wtPath, msgs, err := client.StartWork("test-hook01", false)
	if err != nil {
		t.Fatalf("StartWork() unexpected error: %v", err)
	}

	// Verify the marker file was created in the worktree
	markerPath := filepath.Join(wtPath, ".setup-ran")
	if _, err := os.Stat(markerPath); os.IsNotExist(err) {
		t.Error("expected setup hook to create .setup-ran in worktree")
	}

	foundHookMsg := false
	for _, msg := range msgs {
		if strings.Contains(msg, "worktree_setup hook") {
			foundHookMsg = true
		}
	}
	if !foundHookMsg {
		t.Errorf("expected setup hook message in %v", msgs)
	}
}

func TestStartWork_Worktree_SetupHookFails(t *testing.T) {
	_, client := setupWorktreeTestRepo(t)
	client.Config.WorktreeSetup = "exit 1"

	createTestIssue(t, "test-hookfail01", "Setup Hook Fail", "PLANNED")

	_, wtPath, msgs, err := client.StartWork("test-hookfail01", false)
	if err != nil {
		t.Fatalf("StartWork() should succeed even if hook fails: %v", err)
	}
	if wtPath == "" {
		t.Fatal("expected non-empty worktree path despite hook failure")
	}

	foundWarning := false
	for _, msg := range msgs {
		if strings.Contains(msg, "Warning: worktree_setup hook failed") {
			foundWarning = true
		}
	}
	if !foundWarning {
		t.Errorf("expected hook failure warning in %v", msgs)
	}
}
