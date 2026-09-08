package storage

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func initGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	dir, _ = filepath.EvalSymlinks(dir)
	cmd := exec.Command("git", "init", "-b", "main", dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %v\n%s", err, out)
	}
	// Need at least one commit for git rev-parse to work
	cmd = exec.Command("git", "-C", dir, "config", "user.email", "test@test.com")
	cmd.Run()
	cmd = exec.Command("git", "-C", dir, "config", "user.name", "Test")
	cmd.Run()
	os.WriteFile(filepath.Join(dir, "init.txt"), []byte("init"), 0644)
	cmd = exec.Command("git", "-C", dir, "add", ".")
	cmd.Run()
	cmd = exec.Command("git", "-C", dir, "commit", "-m", "init")
	cmd.Run()
	return dir
}

func TestHubRoot_InGitRepo(t *testing.T) {
	dir := initGitRepo(t)

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	ResetHubRoot()
	t.Cleanup(func() {
		os.Chdir(origDir)
		ResetHubRoot()
	})

	got := HubRoot()
	if got != dir {
		t.Errorf("expected %s, got %s", dir, got)
	}
}

func TestHubRoot_NonGitDir(t *testing.T) {
	dir := t.TempDir()

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	ResetHubRoot()
	t.Cleanup(func() {
		os.Chdir(origDir)
		ResetHubRoot()
	})

	got := HubRoot()
	if got != "." {
		t.Errorf("expected fallback '.', got %s", got)
	}
}

func TestHubRoot_Caching(t *testing.T) {
	dir := initGitRepo(t)

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	ResetHubRoot()
	t.Cleanup(func() {
		os.Chdir(origDir)
		ResetHubRoot()
	})

	first := HubRoot()
	// Change directory — cached value should persist
	os.Chdir(origDir)
	second := HubRoot()
	if first != second {
		t.Errorf("expected cached value %s, got %s", first, second)
	}
}

func TestResetHubRoot_ClearsCachedValue(t *testing.T) {
	dir := initGitRepo(t)

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	ResetHubRoot()
	t.Cleanup(func() {
		os.Chdir(origDir)
		ResetHubRoot()
	})

	first := HubRoot()
	if first != dir {
		t.Errorf("expected %s, got %s", dir, first)
	}

	// Reset and change to a non-git dir — should re-discover
	nonGitDir := t.TempDir()
	os.Chdir(nonGitDir)
	ResetHubRoot()

	second := HubRoot()
	if second != "." {
		t.Errorf("expected fallback '.' after reset, got %s", second)
	}
}

func TestXpoDir(t *testing.T) {
	dir := initGitRepo(t)

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	ResetHubRoot()
	t.Cleanup(func() {
		os.Chdir(origDir)
		ResetHubRoot()
	})

	got := XpoDir()
	expected := filepath.Join(dir, ".xpo")
	if got != expected {
		t.Errorf("expected %s, got %s", expected, got)
	}
}

func TestHubRoot_InWorktree(t *testing.T) {
	dir := initGitRepo(t)

	// Create a linked worktree
	wtPath := filepath.Join(dir, "wt-feature")
	cmd := exec.Command("git", "-C", dir, "worktree", "add", "-b", "feature", wtPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git worktree add failed: %v\n%s", err, out)
	}
	t.Cleanup(func() {
		exec.Command("git", "-C", dir, "worktree", "remove", "--force", wtPath).Run()
	})

	origDir, _ := os.Getwd()
	os.Chdir(wtPath)
	ResetHubRoot()
	t.Cleanup(func() {
		os.Chdir(origDir)
		ResetHubRoot()
	})

	got := HubRoot()
	if got != dir {
		t.Errorf("expected hub root %s from worktree, got %s", dir, got)
	}
}
