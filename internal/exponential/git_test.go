package exponential

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/palarix/exponential/internal/storage"
)

func initTestRepo(t *testing.T, defaultBranch string) string {
	t.Helper()
	dir := t.TempDir()
	// Resolve symlinks (macOS /var -> /private/var) so paths match git output
	dir, _ = filepath.EvalSymlinks(dir)
	runGit(t, dir, "init", "-b", defaultBranch)
	runGit(t, dir, "config", "user.email", "test@test.com")
	runGit(t, dir, "config", "user.name", "Test")
	os.WriteFile(filepath.Join(dir, "init.txt"), []byte("init\n"), 0644)
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "init")

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	storage.ResetHubRoot()
	t.Cleanup(func() {
		os.Chdir(origDir)
		storage.ResetHubRoot()
	})
	return dir
}

func TestDefaultBranch_Main(t *testing.T) {
	initTestRepo(t, "main")
	// No remote configured, so DefaultBranch falls back to "main"
	got := DefaultBranch()
	if got != "main" {
		t.Errorf("expected main, got %s", got)
	}
}

func TestDefaultBranch_LocalBranchWithoutRemote(t *testing.T) {
	dir := initTestRepo(t, "trunk")
	runGit(t, dir, "config", "init.defaultBranch", "trunk")
	got := DefaultBranch()
	if got != "trunk" {
		t.Errorf("expected trunk, got %s", got)
	}
}

func TestDefaultBranch_NoGitRepo(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	t.Cleanup(func() { os.Chdir(origDir) })

	got := DefaultBranch()
	if got != "main" {
		t.Errorf("expected fallback to main, got %s", got)
	}
}

func TestDefaultBranch_WithRemote(t *testing.T) {
	remoteDir := t.TempDir()
	remoteDir, _ = filepath.EvalSymlinks(remoteDir)
	runGit(t, remoteDir, "init", "--bare", "-b", "develop")

	dir := t.TempDir()
	dir, _ = filepath.EvalSymlinks(dir)
	runGit(t, dir, "init", "-b", "develop")
	runGit(t, dir, "config", "user.email", "test@test.com")
	runGit(t, dir, "config", "user.name", "Test")
	os.WriteFile(filepath.Join(dir, "init.txt"), []byte("init\n"), 0644)
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "init")
	runGit(t, dir, "remote", "add", "origin", remoteDir)
	runGit(t, dir, "push", "-u", "origin", "develop")
	runGit(t, dir, "remote", "set-head", "origin", "develop")

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	storage.ResetHubRoot()
	t.Cleanup(func() {
		os.Chdir(origDir)
		storage.ResetHubRoot()
	})

	got := DefaultBranch()
	if got != "develop" {
		t.Errorf("expected develop, got %s", got)
	}
}

func TestBranchExists_True(t *testing.T) {
	initTestRepo(t, "main")
	if !BranchExists("main") {
		t.Error("expected main to exist")
	}
}

func TestBranchExists_False(t *testing.T) {
	initTestRepo(t, "main")
	if BranchExists("nonexistent") {
		t.Error("expected nonexistent branch to not exist")
	}
}

func TestCreateAndCheckoutBranch(t *testing.T) {
	initTestRepo(t, "main")

	err := CreateAndCheckoutBranch("feature", "main")
	if err != nil {
		t.Fatalf("CreateAndCheckoutBranch failed: %v", err)
	}

	got := CurrentBranch()
	if got != "feature" {
		t.Errorf("expected to be on feature, got %s", got)
	}

	if !BranchExists("feature") {
		t.Error("expected feature branch to exist")
	}
}

func TestCreateAndCheckoutBranch_AlreadyExists(t *testing.T) {
	initTestRepo(t, "main")
	runGit(t, ".", "checkout", "-b", "feature")
	runGit(t, ".", "checkout", "main")

	err := CreateAndCheckoutBranch("feature", "main")
	if err == nil {
		t.Error("expected error when branch already exists")
	}
}

func TestCheckoutBranch(t *testing.T) {
	initTestRepo(t, "main")
	runGit(t, ".", "checkout", "-b", "feature")
	runGit(t, ".", "checkout", "main")

	err := CheckoutBranch("feature")
	if err != nil {
		t.Fatalf("CheckoutBranch failed: %v", err)
	}

	got := CurrentBranch()
	if got != "feature" {
		t.Errorf("expected to be on feature, got %s", got)
	}
}

func TestCheckoutBranch_Nonexistent(t *testing.T) {
	initTestRepo(t, "main")

	err := CheckoutBranch("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent branch")
	}
}

func TestWorktreeLifecycle(t *testing.T) {
	dir := initTestRepo(t, "main")

	wtPath := filepath.Join(dir, "wt-feature")
	err := WorktreeAdd(wtPath, "feature", "main")
	if err != nil {
		t.Fatalf("WorktreeAdd failed: %v", err)
	}

	// Verify directory exists
	if _, err := os.Stat(wtPath); os.IsNotExist(err) {
		t.Fatal("worktree directory was not created")
	}

	// Verify it appears in list with correct branch
	entries, err := WorktreeList()
	if err != nil {
		t.Fatalf("WorktreeList failed: %v", err)
	}

	found := false
	for _, e := range entries {
		if e.Branch == "feature" {
			found = true
			if e.Path != wtPath {
				t.Errorf("expected path %s, got %s", wtPath, e.Path)
			}
		}
	}
	if !found {
		t.Error("worktree for 'feature' not found in list")
	}

	// Remove it
	err = WorktreeRemove(wtPath)
	if err != nil {
		t.Fatalf("WorktreeRemove failed: %v", err)
	}

	// Verify directory is gone
	if _, err := os.Stat(wtPath); !os.IsNotExist(err) {
		t.Error("worktree directory still exists after remove")
	}

	// Verify it's gone from list
	entries, err = WorktreeList()
	if err != nil {
		t.Fatalf("WorktreeList after remove failed: %v", err)
	}
	for _, e := range entries {
		if e.Branch == "feature" {
			t.Error("worktree for 'feature' still in list after remove")
		}
	}
}

func TestWorktreeList_IncludesHub(t *testing.T) {
	dir := initTestRepo(t, "main")

	entries, err := WorktreeList()
	if err != nil {
		t.Fatalf("WorktreeList failed: %v", err)
	}

	// The hub checkout should always appear
	found := false
	for _, e := range entries {
		if e.Path == dir && e.Branch == "main" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected hub entry (path=%s, branch=main) in worktree list", dir)
	}
}

func TestFindWorktreeForBranch_Found(t *testing.T) {
	dir := initTestRepo(t, "main")

	wtPath := filepath.Join(dir, "wt-feature")
	err := WorktreeAdd(wtPath, "feature", "main")
	if err != nil {
		t.Fatalf("WorktreeAdd failed: %v", err)
	}
	t.Cleanup(func() {
		exec.Command("git", "worktree", "remove", "--force", wtPath).Run()
		exec.Command("git", "worktree", "prune").Run()
	})

	gotPath, ok := FindWorktreeForBranch("feature")
	if !ok {
		t.Fatal("expected to find worktree for feature")
	}
	if gotPath != wtPath {
		t.Errorf("expected path %s, got %s", wtPath, gotPath)
	}
}

func TestFindWorktreeForBranch_NotFound(t *testing.T) {
	initTestRepo(t, "main")

	_, ok := FindWorktreeForBranch("nonexistent")
	if ok {
		t.Error("expected not to find worktree for nonexistent branch")
	}
}

func TestFindWorktreeForBranch_HubCheckout(t *testing.T) {
	dir := initTestRepo(t, "main")
	runGit(t, dir, "checkout", "-b", "feature")

	// Hub is on "feature" — FindWorktreeForBranch should find it
	// (returns the hub path; caller decides whether to treat as worktree)
	gotPath, ok := FindWorktreeForBranch("feature")
	if !ok {
		t.Fatal("expected to find hub checkout as worktree entry for feature")
	}
	if gotPath != dir {
		t.Errorf("expected hub path %s, got %s", dir, gotPath)
	}
}
