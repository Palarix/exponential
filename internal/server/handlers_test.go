package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/palarix/exponential/internal/exponential"
	"github.com/palarix/exponential/internal/config"
	"github.com/palarix/exponential/internal/model"
	"github.com/palarix/exponential/internal/storage"
)

var testIDCounter atomic.Int64

func setupTestServer(t *testing.T) *Server {
	t.Helper()
	tmpDir := t.TempDir()
	xpoDir := filepath.Join(tmpDir, ".xpo")
	os.MkdirAll(xpoDir, 0755)
	os.WriteFile(filepath.Join(xpoDir, "issues.db"), []byte{}, 0644)
	os.WriteFile(filepath.Join(xpoDir, "config.toml"), []byte{}, 0644)

	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	t.Cleanup(func() { os.Chdir(origDir) })

	cfg := &config.Config{
		Prefix:           "test-",
		User:             "Test User <test@test.com>",
		EstimationSystem: "fibonacci",
		Version:          2,
	}

	return NewServer(cfg, 0, false, 0)
}

func seedIssue(t *testing.T, title string) string {
	t.Helper()
	id := fmt.Sprintf("test-%06x", testIDCounter.Add(1))
	evt := model.Event{
		ID:        id,
		Type:      model.EventTypeCreate,
		Payload:   model.CreatePayload{Title: title, Labels: []string{"feature"}},
		CreatedBy: "Test User <test@test.com>",
		CreatedAt: time.Now(),
	}
	if err := storage.AppendEvent(evt); err != nil {
		t.Fatalf("seedIssue: %v", err)
	}
	return evt.ID
}

func TestHandleInbox(t *testing.T) {
	srv := setupTestServer(t)

	// Issue created by the server's configured user ("Test User").
	id := seedIssue(t, "My Issue")

	// A comment on it by someone else — relevant to me, not my own action.
	if err := storage.AppendEvent(model.Event{
		ID:        id,
		Type:      model.EventTypeComment,
		Payload:   model.CommentPayload{ID: "c1", Text: "looks good"},
		CreatedBy: "Bob Ops <bob@example.com>",
		CreatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("append comment: %v", err)
	}

	mux := srv.SetupRoutes()
	req := httptest.NewRequest("GET", "/api/inbox", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var items []exponential.InboxItem
	json.NewDecoder(w.Body).Decode(&items)
	if len(items) != 1 {
		t.Fatalf("expected 1 inbox item, got %d: %+v", len(items), items)
	}
	if items[0].Type != model.EventTypeComment || items[0].IssueID != id {
		t.Errorf("unexpected inbox item: %+v", items[0])
	}
}

func TestHandleInbox_InvalidSince(t *testing.T) {
	srv := setupTestServer(t)
	mux := srv.SetupRoutes()

	req := httptest.NewRequest("GET", "/api/inbox?since=not-a-time", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleGetIssues_Empty(t *testing.T) {
	srv := setupTestServer(t)
	mux := srv.SetupRoutes()

	req := httptest.NewRequest("GET", "/api/issues", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var issues []IssueResponse
	json.NewDecoder(w.Body).Decode(&issues)
	if issues != nil && len(issues) != 0 {
		t.Errorf("expected empty list, got %d issues", len(issues))
	}
}

func TestHandleGetIssues_WithData(t *testing.T) {
	srv := setupTestServer(t)
	seedIssue(t, "First Issue")
	seedIssue(t, "Second Issue")
	mux := srv.SetupRoutes()

	req := httptest.NewRequest("GET", "/api/issues", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var issues []IssueResponse
	json.NewDecoder(w.Body).Decode(&issues)
	if len(issues) != 2 {
		t.Errorf("expected 2 issues, got %d", len(issues))
	}
}

func TestHandleGetIssue_NotFound(t *testing.T) {
	srv := setupTestServer(t)
	mux := srv.SetupRoutes()

	req := httptest.NewRequest("GET", "/api/issues/test-nonexistent", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestHandleGetIssue_Found(t *testing.T) {
	srv := setupTestServer(t)
	id := seedIssue(t, "My Issue")
	mux := srv.SetupRoutes()

	req := httptest.NewRequest("GET", "/api/issues/"+id, nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var issue IssueResponse
	json.NewDecoder(w.Body).Decode(&issue)
	if issue.Title != "My Issue" {
		t.Errorf("expected title 'My Issue', got %q", issue.Title)
	}
}

func TestHandleDraft_Create(t *testing.T) {
	srv := setupTestServer(t)
	mux := srv.SetupRoutes()

	body, _ := json.Marshal(map[string]interface{}{
		"issue_id": "",
		"type":     "CREATE",
		"payload":  map[string]interface{}{"title": "New Issue", "labels": []string{"bug"}},
	})

	req := httptest.NewRequest("POST", "/api/draft", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["status"] != "ok" {
		t.Errorf("expected status 'ok', got %q", resp["status"])
	}
	if resp["issue_id"] == "" {
		t.Error("expected non-empty issue_id")
	}
}

func TestHandleDraft_BadJSON(t *testing.T) {
	srv := setupTestServer(t)
	mux := srv.SetupRoutes()

	req := httptest.NewRequest("POST", "/api/draft", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleDraft_UnknownType(t *testing.T) {
	srv := setupTestServer(t)
	mux := srv.SetupRoutes()

	body, _ := json.Marshal(map[string]interface{}{
		"issue_id": "test-123456",
		"type":     "INVALID",
		"payload":  map[string]interface{}{},
	})

	req := httptest.NewRequest("POST", "/api/draft", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleGetPending_Empty(t *testing.T) {
	srv := setupTestServer(t)
	mux := srv.SetupRoutes()

	req := httptest.NewRequest("GET", "/api/pending", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp PendingResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.HasPending {
		t.Error("expected no pending events")
	}
}

func TestHandleDiscardPending(t *testing.T) {
	srv := setupTestServer(t)
	srv.AddPendingEvent(model.Event{ID: "test-aaaaaa", Type: model.EventTypeCreate})
	mux := srv.SetupRoutes()

	req := httptest.NewRequest("DELETE", "/api/pending", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if srv.GetPendingCount() != 0 {
		t.Error("expected pending to be cleared")
	}
}

func TestHandleGetConfig(t *testing.T) {
	srv := setupTestServer(t)
	mux := srv.SetupRoutes()

	req := httptest.NewRequest("GET", "/api/config", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["prefix"] != "test-" {
		t.Errorf("expected prefix 'test-', got %q", resp["prefix"])
	}
}

func TestHandleGetIssueHistory_Empty(t *testing.T) {
	srv := setupTestServer(t)
	mux := srv.SetupRoutes()

	req := httptest.NewRequest("GET", "/api/issues/test-000000/history", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestHandleGetIssueHistory_WithEvents(t *testing.T) {
	srv := setupTestServer(t)
	id := seedIssue(t, "History Test")
	mux := srv.SetupRoutes()

	req := httptest.NewRequest("GET", "/api/issues/"+id+"/history", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var history []map[string]interface{}
	json.NewDecoder(w.Body).Decode(&history)
	if len(history) != 1 {
		t.Errorf("expected 1 history event, got %d", len(history))
	}
}

func TestHandleActivity_Empty(t *testing.T) {
	srv := setupTestServer(t)
	mux := srv.SetupRoutes()

	req := httptest.NewRequest("GET", "/api/activity", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestHandleMetrics(t *testing.T) {
	srv := setupTestServer(t)
	mux := srv.SetupRoutes()

	req := httptest.NewRequest("GET", "/api/metrics", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// initServerTestRepo turns the test server's temp dir into a git repo with
// one commit and a worktree for the given branch. Returns the worktree path.
func initServerTestRepo(t *testing.T, branch string) string {
	t.Helper()
	dir, _ := os.Getwd()
	dir, _ = filepath.EvalSymlinks(dir)
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
	git("commit", "--allow-empty", "-m", "init")
	storage.ResetHubRoot()
	t.Cleanup(storage.ResetHubRoot)

	wtPath := filepath.Join(dir, "wt")
	git("worktree", "add", "-b", branch, wtPath, "main")
	return wtPath
}

func TestLocalWorktree_Found(t *testing.T) {
	s := setupTestServer(t)
	wtPath := initServerTestRepo(t, "test-abc123-feature")
	mux := s.SetupRoutes()

	req := httptest.NewRequest("GET", "/api/local/issues/test-abc123/worktree", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var got struct {
		Path   string `json:"path"`
		Branch string `json:"branch"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Path != wtPath || got.Branch != "test-abc123-feature" {
		t.Errorf("unexpected response %+v (want path %s)", got, wtPath)
	}
}

func TestLocalWorktree_NotFound(t *testing.T) {
	s := setupTestServer(t)
	initServerTestRepo(t, "test-abc123-feature")
	mux := s.SetupRoutes()

	req := httptest.NewRequest("GET", "/api/local/issues/test-999999/worktree", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}
