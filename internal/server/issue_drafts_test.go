package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/palarix/exponential/internal/config"
	"github.com/palarix/exponential/internal/model"
)

func draftRequest(t *testing.T, mux *http.ServeMux, method, path string, body any, token string) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	return w
}

func decodeDraft(t *testing.T, w *httptest.ResponseRecorder) model.IssueDraft {
	t.Helper()
	var d model.IssueDraft
	if err := json.NewDecoder(w.Body).Decode(&d); err != nil {
		t.Fatalf("decode draft: %v", err)
	}
	return d
}

func listDrafts(t *testing.T, mux *http.ServeMux, token string) []model.IssueDraft {
	t.Helper()
	w := draftRequest(t, mux, "GET", "/api/issue-drafts", nil, token)
	if w.Code != http.StatusOK {
		t.Fatalf("list: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var out []model.IssueDraft
	json.NewDecoder(w.Body).Decode(&out)
	return out
}

func TestIssueDrafts_CRUD(t *testing.T) {
	srv := setupTestServer(t)
	mux := srv.SetupRoutes()

	if got := listDrafts(t, mux, ""); len(got) != 0 {
		t.Fatalf("expected empty list, got %d", len(got))
	}

	w := draftRequest(t, mux, "POST", "/api/issue-drafts", model.IssueDraftPayload{Title: "first", Labels: []string{"feature"}}, "")
	if w.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", w.Code, w.Body.String())
	}
	created := decodeDraft(t, w)
	if created.CreatedBy != "Test User <test@test.com>" || created.Payload.Title != "first" {
		t.Errorf("unexpected draft: %+v", created)
	}

	w = draftRequest(t, mux, "PUT", "/api/issue-drafts/"+created.ID, model.IssueDraftPayload{Title: "second"}, "")
	if w.Code != http.StatusOK {
		t.Fatalf("update: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if u := decodeDraft(t, w); u.ID != created.ID || u.Payload.Title != "second" {
		t.Errorf("update wrong: %+v", u)
	}

	if got := listDrafts(t, mux, ""); len(got) != 1 || got[0].Payload.Title != "second" {
		t.Fatalf("list after update wrong: %+v", got)
	}

	w = draftRequest(t, mux, "DELETE", "/api/issue-drafts/"+created.ID, nil, "")
	if w.Code != http.StatusNoContent {
		t.Fatalf("delete: expected 204, got %d: %s", w.Code, w.Body.String())
	}
	if got := listDrafts(t, mux, ""); len(got) != 0 {
		t.Errorf("expected empty list after delete, got %d", len(got))
	}
}

func TestIssueDrafts_UpdateMissingIs404(t *testing.T) {
	srv := setupTestServer(t)
	mux := srv.SetupRoutes()
	w := draftRequest(t, mux, "PUT", "/api/issue-drafts/draft-deadbeef", model.IssueDraftPayload{Title: "x"}, "")
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
	if got := listDrafts(t, mux, ""); len(got) != 0 {
		t.Errorf("update recreated the draft")
	}
}

func TestIssueDrafts_InvalidIDIs404(t *testing.T) {
	srv := setupTestServer(t)
	mux := srv.SetupRoutes()
	for _, method := range []string{"PUT", "DELETE"} {
		w := draftRequest(t, mux, method, "/api/issue-drafts/..%2Fissues", model.IssueDraftPayload{}, "")
		if w.Code != http.StatusNotFound {
			t.Errorf("%s: expected 404, got %d", method, w.Code)
		}
	}
}

func TestIssueDrafts_ValidationIs400(t *testing.T) {
	srv := setupTestServer(t)
	mux := srv.SetupRoutes()
	w := draftRequest(t, mux, "POST", "/api/issue-drafts", model.IssueDraftPayload{Title: strings.Repeat("x", 501)}, "")
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
	req := httptest.NewRequest("POST", "/api/issue-drafts", strings.NewReader("{not json"))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("bad JSON: expected 400, got %d", rec.Code)
	}
}

func TestIssueDrafts_NotifyDraftsOnly(t *testing.T) {
	srv := setupTestServer(t)
	sub := subscribeLive(t, srv)
	mux := srv.SetupRoutes()

	var d model.IssueDraft
	steps := map[string]func(){
		"create": func() {
			d = decodeDraft(t, draftRequest(t, mux, "POST", "/api/issue-drafts", model.IssueDraftPayload{Title: "x"}, ""))
		},
		"update": func() {
			draftRequest(t, mux, "PUT", "/api/issue-drafts/"+d.ID, model.IssueDraftPayload{Title: "y"}, "")
		},
		"delete": func() { draftRequest(t, mux, "DELETE", "/api/issue-drafts/"+d.ID, nil, "") },
	}
	for _, name := range []string{"create", "update", "delete"} {
		steps[name]()
		if got := signals(sub, 50*time.Millisecond); got[TopicDrafts] != 1 || len(got) != 1 {
			t.Errorf("%s: signals = %v, want drafts only", name, got)
		}
	}
}

func TestIssueDrafts_AuthScopesToOwner(t *testing.T) {
	perms := config.PermissionsConfig{
		Roles: map[string][]string{"writer": {"issue.read", "issue.create"}},
		Users: map[string]string{"alice@test.com": "writer", "bob@test.com": "writer"},
	}
	srv, priv := setupCapServer(t, perms)
	mux := srv.SetupRoutes()
	alice := tokenFor(priv, "Alice <alice@test.com>")
	bob := tokenFor(priv, "Bob <bob@test.com>")

	w := draftRequest(t, mux, "POST", "/api/issue-drafts", model.IssueDraftPayload{Title: "alice's"}, alice)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", w.Code, w.Body.String())
	}
	d := decodeDraft(t, w)
	if d.CreatedBy != "Alice <alice@test.com>" {
		t.Errorf("created_by = %q", d.CreatedBy)
	}

	if got := listDrafts(t, mux, bob); len(got) != 0 {
		t.Errorf("bob sees %d drafts, want 0", len(got))
	}
	if got := listDrafts(t, mux, alice); len(got) != 1 {
		t.Errorf("alice sees %d drafts, want 1", len(got))
	}
	if w := draftRequest(t, mux, "PUT", "/api/issue-drafts/"+d.ID, model.IssueDraftPayload{Title: "x"}, bob); w.Code != http.StatusNotFound {
		t.Errorf("bob update: expected 404, got %d", w.Code)
	}
	if w := draftRequest(t, mux, "DELETE", "/api/issue-drafts/"+d.ID, nil, bob); w.Code != http.StatusNotFound {
		t.Errorf("bob delete: expected 404, got %d", w.Code)
	}
	if w := draftRequest(t, mux, "DELETE", "/api/issue-drafts/"+d.ID, nil, alice); w.Code != http.StatusNoContent {
		t.Errorf("alice delete: expected 204, got %d", w.Code)
	}
}

func TestIssueDrafts_RequiresCreateCapability(t *testing.T) {
	perms := config.PermissionsConfig{
		Roles: map[string][]string{"viewer": {"issue.read"}},
		Users: map[string]string{"bob@test.com": "viewer"},
	}
	srv, priv := setupCapServer(t, perms)
	mux := srv.SetupRoutes()
	w := draftRequest(t, mux, "POST", "/api/issue-drafts", model.IssueDraftPayload{Title: "x"}, tokenFor(priv, "Bob <bob@test.com>"))
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}
