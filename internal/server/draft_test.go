package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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

// A refused release assignment is the user's to fix, so it answers 400
// with the server's reason instead of a generic 500.
func TestHandleDraft_UpdateReleaseRefusalIs400(t *testing.T) {
	srv := setupTestServer(t)
	mux := srv.SetupRoutes()
	id := seedIssue(t, "Assignable")

	for _, tc := range []struct {
		ref, want string
	}{
		{"@next", "save Current first"},
		{"#99", "not found"},
	} {
		ref := tc.ref
		body, _ := json.Marshal(map[string]interface{}{
			"issue_id": id,
			"type":     "UPDATE",
			"payload":  model.UpdatePayload{ReleaseID: &ref},
		})
		req := httptest.NewRequest("POST", "/api/draft", bytes.NewReader(body))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("%s: expected 400, got %d: %s", ref, w.Code, w.Body.String())
		}
		if !strings.Contains(w.Body.String(), tc.want) {
			t.Errorf("%s: body %q should contain %q", ref, w.Body.String(), tc.want)
		}
	}
}

// with_children next to the payload also moves the epic's open children,
// as the MCP and CLI option does; without it only the epic moves.
func TestHandleDraft_UpdateWithChildren(t *testing.T) {
	for _, withChildren := range []bool{false, true} {
		srv := setupTestServer(t)
		mux := srv.SetupRoutes()
		epic := seedIssue(t, "Epic")
		post := func(req map[string]interface{}) {
			t.Helper()
			body, _ := json.Marshal(req)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, httptest.NewRequest("POST", "/api/draft", bytes.NewReader(body)))
			if w.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
			}
		}
		post(map[string]interface{}{"issue_id": "", "type": "CREATE", "payload": model.CreatePayload{Title: "Child", ParentID: epic, Labels: []string{"feature"}}})

		ref := "@current"
		post(map[string]interface{}{"issue_id": epic, "type": "UPDATE", "payload": model.UpdatePayload{ReleaseID: &ref}, "with_children": withChildren})

		issues, _ := srv.GetProjectedIssues()
		if issues[epic].ReleaseID == "" {
			t.Fatalf("with_children=%v: the epic should be in a release", withChildren)
		}
		for _, iss := range issues {
			if iss.ParentID != epic {
				continue
			}
			moved := iss.ReleaseID == issues[epic].ReleaseID
			if moved != withChildren {
				t.Errorf("with_children=%v: child moved = %v", withChildren, moved)
			}
		}
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
