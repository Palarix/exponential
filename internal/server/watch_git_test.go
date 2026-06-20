package server

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/palarix/exponential/internal/model"
)

func setupFakeGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	refsHeads := filepath.Join(dir, ".git", "refs", "heads")
	if err := os.MkdirAll(refsHeads, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(refsHeads, "main"), []byte("abc123\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func chdir(t *testing.T, dir string) {
	t.Helper()
	orig, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(orig) })
}

func serverWithCache(t *testing.T) *Server {
	t.Helper()
	srv := &Server{
		SSEHub:       NewSSEHub(),
		cachedIssues: map[string]*model.Issue{"test": {ID: "test"}},
	}
	return srv
}

func TestWatchGitRefs_NilSSEHub(t *testing.T) {
	dir := setupFakeGitRepo(t)
	chdir(t, dir)
	srv := &Server{}
	srv.WatchGitRefs() // should not panic
}

func TestWatchGitRefs_NoGitDir(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	srv := &Server{SSEHub: NewSSEHub()}
	srv.WatchGitRefs() // should not panic
}

func TestWatchGitRefs_InvalidateOnRefModify(t *testing.T) {
	dir := setupFakeGitRepo(t)
	chdir(t, dir)
	srv := serverWithCache(t)

	srv.WatchGitRefs()
	time.Sleep(50 * time.Millisecond) // let watcher start

	refFile := filepath.Join(dir, ".git", "refs", "heads", "main")
	os.WriteFile(refFile, []byte("def456\n"), 0o644)

	time.Sleep(300 * time.Millisecond) // debounce + margin

	srv.mu.RLock()
	defer srv.mu.RUnlock()
	if srv.cachedIssues != nil {
		t.Error("expected cachedIssues to be nil after ref file modification")
	}
}

func TestWatchGitRefs_InvalidateOnNewBranch(t *testing.T) {
	dir := setupFakeGitRepo(t)
	chdir(t, dir)
	srv := serverWithCache(t)

	srv.WatchGitRefs()
	time.Sleep(50 * time.Millisecond)

	newRef := filepath.Join(dir, ".git", "refs", "heads", "feature-branch")
	os.WriteFile(newRef, []byte("aaa111\n"), 0o644)

	time.Sleep(300 * time.Millisecond)

	srv.mu.RLock()
	defer srv.mu.RUnlock()
	if srv.cachedIssues != nil {
		t.Error("expected cachedIssues to be nil after new branch creation")
	}
}

func TestWatchGitRefs_InvalidateOnBranchDelete(t *testing.T) {
	dir := setupFakeGitRepo(t)
	chdir(t, dir)

	extraRef := filepath.Join(dir, ".git", "refs", "heads", "to-delete")
	os.WriteFile(extraRef, []byte("bbb222\n"), 0o644)

	srv := serverWithCache(t)
	srv.WatchGitRefs()
	time.Sleep(50 * time.Millisecond)

	os.Remove(extraRef)

	time.Sleep(300 * time.Millisecond)

	srv.mu.RLock()
	defer srv.mu.RUnlock()
	if srv.cachedIssues != nil {
		t.Error("expected cachedIssues to be nil after branch deletion")
	}
}

func TestWatchGitRefs_NestedBranchDir(t *testing.T) {
	dir := setupFakeGitRepo(t)
	chdir(t, dir)
	srv := serverWithCache(t)

	srv.WatchGitRefs()
	time.Sleep(50 * time.Millisecond)

	// Create a hierarchical branch like upe-0000c2/weekly-metrics
	nestedDir := filepath.Join(dir, ".git", "refs", "heads", "upe-0000c2")
	os.MkdirAll(nestedDir, 0o755)
	time.Sleep(100 * time.Millisecond) // let watcher pick up new dir

	os.WriteFile(filepath.Join(nestedDir, "weekly-metrics"), []byte("ccc333\n"), 0o644)

	time.Sleep(300 * time.Millisecond)

	srv.mu.RLock()
	defer srv.mu.RUnlock()
	if srv.cachedIssues != nil {
		t.Error("expected cachedIssues to be nil after nested branch creation")
	}
}

func TestWatchGitRefs_PackedRefs(t *testing.T) {
	dir := setupFakeGitRepo(t)
	chdir(t, dir)

	packedRefs := filepath.Join(dir, ".git", "packed-refs")
	os.WriteFile(packedRefs, []byte("# pack-refs\nabc123 refs/heads/main\n"), 0o644)

	srv := serverWithCache(t)
	srv.WatchGitRefs()
	time.Sleep(50 * time.Millisecond)

	os.WriteFile(packedRefs, []byte("# pack-refs\ndef456 refs/heads/main\n"), 0o644)

	time.Sleep(300 * time.Millisecond)

	srv.mu.RLock()
	defer srv.mu.RUnlock()
	if srv.cachedIssues != nil {
		t.Error("expected cachedIssues to be nil after packed-refs modification")
	}
}

func TestWatchGitRefs_SSEBroadcast(t *testing.T) {
	dir := setupFakeGitRepo(t)
	chdir(t, dir)
	srv := serverWithCache(t)

	ch := srv.SSEHub.subscribe()
	defer srv.SSEHub.unsubscribe(ch)

	srv.WatchGitRefs()
	time.Sleep(50 * time.Millisecond)

	refFile := filepath.Join(dir, ".git", "refs", "heads", "main")
	os.WriteFile(refFile, []byte("eee555\n"), 0o644)

	select {
	case evt := <-ch:
		if evt.Type != "UPDATE" {
			t.Errorf("expected UPDATE event, got %s", evt.Type)
		}
	case <-time.After(500 * time.Millisecond):
		t.Error("timed out waiting for SSE broadcast")
	}
}
