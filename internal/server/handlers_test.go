package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/palarix/beats/internal/config"
	"github.com/palarix/beats/internal/model"
	"github.com/palarix/beats/internal/storage"
)

var testIDCounter atomic.Int64

func setupTestServer(t *testing.T) *Server {
	t.Helper()
	tmpDir := t.TempDir()
	beatsDir := filepath.Join(tmpDir, ".beats")
	os.MkdirAll(beatsDir, 0755)
	os.WriteFile(filepath.Join(beatsDir, "issues.db"), []byte{}, 0644)
	os.WriteFile(filepath.Join(beatsDir, "config.toml"), []byte{}, 0644)

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

func TestHandleGetIssues_Empty(t *testing.T) {
	srv := setupTestServer(t)
	mux := srv.setupRoutes()

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
	mux := srv.setupRoutes()

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
	mux := srv.setupRoutes()

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
	mux := srv.setupRoutes()

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
	mux := srv.setupRoutes()

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
	mux := srv.setupRoutes()

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
	mux := srv.setupRoutes()

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
	mux := srv.setupRoutes()

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
	mux := srv.setupRoutes()

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
	mux := srv.setupRoutes()

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
	mux := srv.setupRoutes()

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
	mux := srv.setupRoutes()

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
	mux := srv.setupRoutes()

	req := httptest.NewRequest("GET", "/api/activity", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestHandleMetrics(t *testing.T) {
	srv := setupTestServer(t)
	mux := srv.setupRoutes()

	req := httptest.NewRequest("GET", "/api/metrics", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}
