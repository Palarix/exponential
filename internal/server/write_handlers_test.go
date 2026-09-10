package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/palarix/exponential/internal/config"
	"github.com/palarix/exponential/internal/storage"
)

// --- handleGetArtifact ---

func TestHandleGetArtifact_NotFound(t *testing.T) {
	srv := setupTestServer(t)
	id := seedIssue(t, "Artifact issue")
	mux := srv.SetupRoutes()

	req := httptest.NewRequest("GET", "/api/issues/"+id+"/artifacts/nonexistent.md", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleGetArtifact_Found(t *testing.T) {
	srv := setupTestServer(t)
	id := seedIssue(t, "Artifact found")
	mux := srv.SetupRoutes()

	// Write an artifact to the ref store
	if err := storage.WriteArtifact(id, "notes.md", "hello"); err != nil {
		t.Fatalf("WriteArtifact: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/issues/"+id+"/artifacts/notes.md", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["content"] != "hello" {
		t.Errorf("expected content 'hello', got %q", resp["content"])
	}
}

// --- handleListInstances ---

func TestHandleListInstances(t *testing.T) {
	srv := setupTestServer(t)
	mux := srv.SetupRoutes()

	req := httptest.NewRequest("GET", "/api/instances", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp []json.RawMessage
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Errorf("expected JSON array, got error: %v", err)
	}
}

// --- handleGetCycles ---

func TestHandleGetCycles_Enabled(t *testing.T) {
	srv := setupTestServer(t)
	srv.Config.Cycles = config.CycleConfig{
		Enabled:   true,
		Duration:  "2w",
		StartDay:  "monday",
	}
	mux := srv.SetupRoutes()

	req := httptest.NewRequest("GET", "/api/cycles", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["enabled"] != true {
		t.Errorf("expected enabled=true, got %v", resp["enabled"])
	}
	cycles, ok := resp["cycles"].([]interface{})
	if !ok {
		t.Fatal("expected cycles array")
	}
	if len(cycles) == 0 {
		t.Error("expected at least one cycle")
	}
}

// --- handleCycleProgress ---

func TestHandleCycleProgress_Disabled(t *testing.T) {
	srv := setupTestServer(t)
	mux := srv.SetupRoutes()

	req := httptest.NewRequest("GET", "/api/cycles/2026-09-01/progress", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

// --- handleTimeline ---

func TestHandleTimeline_Empty(t *testing.T) {
	srv := setupTestServer(t)
	mux := srv.SetupRoutes()

	req := httptest.NewRequest("GET", "/api/timeline", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleTimeline_WithData(t *testing.T) {
	srv := setupTestServer(t)
	seedIssue(t, "Timeline issue")
	mux := srv.SetupRoutes()

	req := httptest.NewRequest("GET", "/api/timeline?limit=5", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp []interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if len(resp) == 0 {
		t.Error("expected at least one timeline entry")
	}
}

// --- handleStartWork (validation) ---

func TestHandleStartWork_MissingID(t *testing.T) {
	srv := setupTestServer(t)
	mux := srv.SetupRoutes()

	// POST to the route pattern without an ID — the mux won't match {id},
	// so we test with a nonexistent issue instead
	req := httptest.NewRequest("POST", "/api/issues/test-nonexistent999/start", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	// Should fail because issue doesn't exist (409 conflict from StartWork error)
	if w.Code == http.StatusOK {
		t.Error("expected error for nonexistent issue")
	}
}

// --- handleMergeIssue (validation) ---

func TestHandleMergeIssue_InvalidStrategy(t *testing.T) {
	srv := setupTestServer(t)
	id := seedIssue(t, "Merge test")
	mux := srv.SetupRoutes()

	body, _ := json.Marshal(map[string]string{"strategy": "rebase"})
	req := httptest.NewRequest("POST", "/api/issues/"+id+"/merge", bytes.NewReader(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleMergeIssue_BadJSON(t *testing.T) {
	srv := setupTestServer(t)
	id := seedIssue(t, "Merge bad json")
	mux := srv.SetupRoutes()

	req := httptest.NewRequest("POST", "/api/issues/"+id+"/merge", bytes.NewReader([]byte("not json")))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

// --- handleCommitDetail (validation) ---

func TestHandleCommitDetail_NotFound(t *testing.T) {
	srv := setupTestServer(t)
	mux := srv.SetupRoutes()

	req := httptest.NewRequest("GET", "/api/commits/deadbeef", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}
