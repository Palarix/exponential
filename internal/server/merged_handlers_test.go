package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/palarix/exponential/internal/exponential"
	"github.com/palarix/exponential/internal/storage"
)

// seedMergedIssue makes the test server's dir a git repo, seeds an issue,
// merges a one-commit branch for it, and returns the issue ID.
func seedMergedIssue(t *testing.T, srv *Server) string {
	t.Helper()
	dir, _ := os.Getwd()
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	git("init", "-b", "main")
	git("config", "user.email", "test@test.com")
	git("config", "user.name", "Test")
	storage.ResetHubRoot()
	t.Cleanup(storage.ResetHubRoot)

	id := seedIssue(t, "Merged issue")
	git("add", ".")
	git("commit", "-m", "init")
	git("checkout", "-b", id+"-feature")
	os.WriteFile(filepath.Join(dir, "feature.txt"), []byte("feature\n"), 0644)
	git("add", "feature.txt")
	git("commit", "-m", "add feature")
	git("checkout", "main")

	if _, err := exponential.NewClient(srv.Config).MergeIssue(id, exponential.MergeOptions{Strategy: exponential.MergeStrategySquash}); err != nil {
		t.Fatalf("MergeIssue: %v", err)
	}
	return id
}

func getJSON(t *testing.T, srv *Server, path string, out any) int {
	t.Helper()
	w := httptest.NewRecorder()
	srv.SetupRoutes().ServeHTTP(w, httptest.NewRequest("GET", path, nil))
	if out != nil {
		json.Unmarshal(w.Body.Bytes(), out)
	}
	return w.Code
}

func TestHandleMergedChanges(t *testing.T) {
	srv := setupTestServer(t)
	id := seedMergedIssue(t, srv)

	var got exponential.MergedChanges
	if code := getJSON(t, srv, "/api/issues/"+id+"/merged", &got); code != http.StatusOK {
		t.Fatalf("expected 200, got %d", code)
	}
	if got.MergeSHA == "" || got.Strategy != "squash" || got.Branch != id+"-feature" {
		t.Errorf("unexpected metadata: %+v", got)
	}
	if !strings.Contains(got.Diff, "feature.txt") {
		t.Errorf("expected the merged diff, got:\n%s", got.Diff)
	}
	if len(got.Commits) != 1 {
		t.Errorf("expected the squash commit, got %+v", got.Commits)
	}

	var issue struct {
		Merged bool `json:"merged"`
	}
	getJSON(t, srv, "/api/issues/"+id, &issue)
	if !issue.Merged {
		t.Error("expected merged: true on a merged issue")
	}
}

func TestHandleMergedChanges_NotMerged(t *testing.T) {
	srv := setupTestServer(t)
	id := seedIssue(t, "Never merged")

	var body struct {
		Error string `json:"error"`
	}
	if code := getJSON(t, srv, "/api/issues/"+id+"/merged", &body); code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", code)
	}
	if !strings.Contains(body.Error, "has not been merged") {
		t.Errorf("expected a readable error, got %q", body.Error)
	}

	var issue map[string]any
	getJSON(t, srv, "/api/issues/"+id, &issue)
	if _, ok := issue["merged"]; ok {
		t.Error("expected no merged field on an unmerged issue")
	}
}
