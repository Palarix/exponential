package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/palarix/exponential/internal/model"
)

func TestHandleDraft_Update(t *testing.T) {
	srv := setupTestServer(t)
	mux := srv.SetupRoutes()
	id := seedIssue(t, "Updatable")

	status := "DOING"
	body, _ := json.Marshal(map[string]interface{}{
		"issue_id": id,
		"type":     "UPDATE",
		"payload":  model.UpdatePayload{Status: &status},
	})

	req := httptest.NewRequest("POST", "/api/draft", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	issues, _ := srv.GetProjectedIssues()
	if issues[id].Status != model.StatusDoing {
		t.Errorf("status = %s, want DOING", issues[id].Status)
	}
}

func TestHandleDraft_Comment(t *testing.T) {
	srv := setupTestServer(t)
	mux := srv.SetupRoutes()
	id := seedIssue(t, "Commentable")

	body, _ := json.Marshal(map[string]interface{}{
		"issue_id": id,
		"type":     "COMMENT",
		"payload":  map[string]interface{}{"id": "c1", "text": "hello"},
	})

	req := httptest.NewRequest("POST", "/api/draft", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	issues, _ := srv.GetProjectedIssues()
	if len(issues[id].Comments) != 1 {
		t.Errorf("expected 1 comment, got %d", len(issues[id].Comments))
	}
}

func TestHandleDraft_Delete(t *testing.T) {
	srv := setupTestServer(t)
	mux := srv.SetupRoutes()
	id := seedIssue(t, "Deletable")

	body, _ := json.Marshal(map[string]interface{}{
		"issue_id": id,
		"type":     "DELETE",
		"payload":  map[string]interface{}{"reason": "testing"},
	})

	req := httptest.NewRequest("POST", "/api/draft", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	issues, _ := srv.GetProjectedIssues()
	if _, exists := issues[id]; exists {
		t.Error("deleted issue should not appear in projected issues")
	}
}

func TestHandleDraft_UpdateNonexistent(t *testing.T) {
	srv := setupTestServer(t)
	mux := srv.SetupRoutes()

	status := "DOING"
	body, _ := json.Marshal(map[string]interface{}{
		"issue_id": "nonexistent",
		"type":     "UPDATE",
		"payload":  model.UpdatePayload{Status: &status},
	})

	req := httptest.NewRequest("POST", "/api/draft", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 for nonexistent issue, got %d", w.Code)
	}
}
