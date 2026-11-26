package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func gitT(t *testing.T, args ...string) {
	t.Helper()
	if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// bareRepoWorktree returns a worktree of a bare clone whose tree holds an
// xpo project.
func bareRepoWorktree(t *testing.T) string {
	t.Helper()
	src := setupProject(t)
	t.Cleanup(func() { os.RemoveAll(src) })
	gitT(t, "-C", src, "init", "-b", "main")
	gitT(t, "-C", src, "-c", "user.email=t@t", "-c", "user.name=T", "add", ".")
	gitT(t, "-C", src, "-c", "user.email=t@t", "-c", "user.name=T", "commit", "-m", "init")

	parent := t.TempDir()
	bare := filepath.Join(parent, "mirror.git")
	gitT(t, "clone", "--bare", src, bare)
	wt := filepath.Join(parent, "wt")
	gitT(t, "-C", bare, "worktree", "add", "-b", "work", wt, "main")
	return wt
}

func TestBareRepoWorktree_ListFails(t *testing.T) {
	wt := bareRepoWorktree(t)
	stdout, stderr, code := runXPO(t, wt, "list")
	if code == 0 {
		t.Fatalf("expected non-zero exit, got 0\nstdout: %s", stdout)
	}
	if !strings.Contains(stderr, "bare repository") {
		t.Errorf("stderr should name the bare repository case: %s\nstdout: %s", stderr, stdout)
	}
}

func TestBareRepoWorktree_ListJSONFails(t *testing.T) {
	wt := bareRepoWorktree(t)
	stdout, _, code := runXPO(t, wt, "list", "--json")
	assertJSONError(t, stdout, code, "bare repository")
}
