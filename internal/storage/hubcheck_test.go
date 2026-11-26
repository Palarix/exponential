package storage

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/palarix/exponential/internal/model"
)

// chdirHub moves into dir and resets the hub cache for the test.
func chdirHub(t *testing.T, dir string) {
	t.Helper()
	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	ResetHubRoot()
	t.Cleanup(func() {
		os.Chdir(origDir)
		ResetHubRoot()
	})
}

func runGit(t *testing.T, args ...string) {
	t.Helper()
	if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// bareWorktree returns a worktree of a bare clone of a repository whose
// tree contains .xpo/config.yaml, and the bare repository's path.
func bareWorktree(t *testing.T) (worktree, bare string) {
	t.Helper()
	src := initGitRepo(t)
	os.MkdirAll(filepath.Join(src, ".xpo"), 0755)
	os.WriteFile(filepath.Join(src, ".xpo", "config.yaml"), []byte("prefix: xpo\n"), 0644)
	runGit(t, "-C", src, "add", ".")
	runGit(t, "-C", src, "commit", "-m", "xpo")

	parent, _ := filepath.EvalSymlinks(t.TempDir())
	bare = filepath.Join(parent, "mirror.git")
	runGit(t, "clone", "--bare", src, bare)
	worktree = filepath.Join(parent, "wt")
	runGit(t, "-C", bare, "worktree", "add", "-b", "work", worktree, "main")
	return worktree, bare
}

func TestCheckHub_BareRepositoryWorktree(t *testing.T) {
	wt, bare := bareWorktree(t)
	chdirHub(t, wt)

	err := CheckHub()
	if !errors.Is(err, ErrBareRepository) {
		t.Fatalf("expected ErrBareRepository, got %v", err)
	}
	if !strings.Contains(err.Error(), bare) {
		t.Errorf("error should name the bare repository %s: %v", bare, err)
	}
}

func TestBareRepositoryWorktree_ReadsAndWritesFail(t *testing.T) {
	wt, _ := bareWorktree(t)
	chdirHub(t, wt)

	if _, err := ReadEvents(); !errors.Is(err, ErrBareRepository) {
		t.Errorf("ReadEvents: expected ErrBareRepository, got %v", err)
	}
	if _, err := ReadArchivedEvents(); !errors.Is(err, ErrBareRepository) {
		t.Errorf("ReadArchivedEvents: expected ErrBareRepository, got %v", err)
	}
	if _, err := ValidateEvents(); !errors.Is(err, ErrBareRepository) {
		t.Errorf("ValidateEvents: expected ErrBareRepository, got %v", err)
	}
	if _, err := CheckEventOrder(); !errors.Is(err, ErrBareRepository) {
		t.Errorf("CheckEventOrder: expected ErrBareRepository, got %v", err)
	}
	evt := model.Event{ID: "xpo-000001", Type: model.EventTypeCreate, CreatedAt: time.Now().UTC()}
	if err := AppendEvent(evt); !errors.Is(err, ErrBareRepository) {
		t.Errorf("AppendEvent: expected ErrBareRepository, got %v", err)
	}
}

func TestCheckHub_MissingXpoDir(t *testing.T) {
	dir := initGitRepo(t)
	chdirHub(t, dir)

	err := CheckHub()
	if !errors.Is(err, ErrNoXpoDir) {
		t.Fatalf("expected ErrNoXpoDir, got %v", err)
	}
	if !strings.Contains(err.Error(), dir) {
		t.Errorf("error should name the hub %s: %v", dir, err)
	}
	if _, err := ReadEvents(); !errors.Is(err, ErrNoXpoDir) {
		t.Errorf("ReadEvents: expected ErrNoXpoDir, got %v", err)
	}
}

func TestCheckHub_XpoDirWithoutIssuesDB(t *testing.T) {
	dir := initGitRepo(t)
	os.MkdirAll(filepath.Join(dir, ".xpo"), 0755)
	chdirHub(t, dir)

	if err := CheckHub(); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	events, err := ReadEvents()
	if err != nil || len(events) != 0 {
		t.Errorf("expected no events and no error, got %d events, %v", len(events), err)
	}
}

func TestCheckHub_LinkedWorktreeOfNormalRepo(t *testing.T) {
	dir := initGitRepo(t)
	os.MkdirAll(filepath.Join(dir, ".xpo"), 0755)
	wt := filepath.Join(dir, ".xpo", "worktrees", "feature")
	runGit(t, "-C", dir, "worktree", "add", "-b", "feature", wt)
	chdirHub(t, wt)

	if got := HubRoot(); got != dir {
		t.Errorf("HubRoot: expected %s, got %s", dir, got)
	}
	if err := CheckHub(); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}
