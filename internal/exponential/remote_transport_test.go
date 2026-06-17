package exponential

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/palarix/exponential/internal/model"
)

func newTestServer(t *testing.T) (*httptest.Server, *RemoteTransport) {
	t.Helper()

	mux := http.NewServeMux()

	now := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)

	issues := []apiIssue{
		{
			ID: "test-aaa111", Title: "First issue", Status: "BACKLOG",
			Labels: []string{"feature"}, Estimate: 3,
			CreatedAt: now, CreatedBy: "Alice <alice@test.com>", UpdatedAt: now,
		},
		{
			ID: "test-bbb222", Title: "Second issue", Status: "DOING",
			ParentID: "test-aaa111", Labels: []string{"bug"},
			CreatedAt: now, CreatedBy: "Bob <bob@test.com>", UpdatedAt: now,
		},
	}

	mux.HandleFunc("GET /api/issues", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(issues)
	})

	mux.HandleFunc("GET /api/issues/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		for _, i := range issues {
			if i.ID == id {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(i)
				return
			}
		}
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
	})

	mux.HandleFunc("POST /api/draft", func(w http.ResponseWriter, r *http.Request) {
		var req draftRequest
		json.NewDecoder(r.Body).Decode(&req)

		issueID := req.IssueID
		if req.Type == string(model.EventTypeCreate) {
			issueID = "test-new123"
			issues = append(issues, apiIssue{
				ID: issueID, Title: "New issue", Status: "BACKLOG",
				CreatedAt: now, CreatedBy: "Test <test@test.com>", UpdatedAt: now,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(draftResponse{Status: "ok", IssueID: issueID})
	})

	mux.HandleFunc("GET /api/user", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(apiUser{User: "Test <test@test.com>"})
	})

	srv := httptest.NewServer(mux)

	rt := &RemoteTransport{
		BaseURL:    srv.URL,
		Token:      "test-token",
		HTTPClient: srv.Client(),
	}

	return srv, rt
}

func TestRemoteTransport_ListIssues(t *testing.T) {
	srv, rt := newTestServer(t)
	defer srv.Close()

	issues, err := rt.ListIssues(FilterOptions{})
	if err != nil {
		t.Fatalf("ListIssues: %v", err)
	}
	if len(issues) != 2 {
		t.Fatalf("expected 2 issues, got %d", len(issues))
	}
	if issues[0].ID != "test-aaa111" {
		t.Errorf("expected first issue ID test-aaa111, got %s", issues[0].ID)
	}
}

func TestRemoteTransport_ListIssues_Filtered(t *testing.T) {
	srv, rt := newTestServer(t)
	defer srv.Close()

	issues, err := rt.ListIssues(FilterOptions{Statuses: []string{"DOING"}})
	if err != nil {
		t.Fatalf("ListIssues: %v", err)
	}
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues))
	}
	if issues[0].ID != "test-bbb222" {
		t.Errorf("expected test-bbb222, got %s", issues[0].ID)
	}
}

func TestRemoteTransport_GetIssue(t *testing.T) {
	srv, rt := newTestServer(t)
	defer srv.Close()

	issue, err := rt.GetIssue("test-aaa111")
	if err != nil {
		t.Fatalf("GetIssue: %v", err)
	}
	if issue.Title != "First issue" {
		t.Errorf("expected 'First issue', got %q", issue.Title)
	}
	if issue.Estimate != 3 {
		t.Errorf("expected estimate 3, got %d", issue.Estimate)
	}
}

func TestRemoteTransport_GetIssue_NotFound(t *testing.T) {
	srv, rt := newTestServer(t)
	defer srv.Close()

	_, err := rt.GetIssue("test-zzz999")
	if err == nil {
		t.Fatal("expected error for non-existent issue")
	}
}

func TestRemoteTransport_FindIssue(t *testing.T) {
	srv, rt := newTestServer(t)
	defer srv.Close()

	issue, children, archived, err := rt.FindIssue("test-aaa111")
	if err != nil {
		t.Fatalf("FindIssue: %v", err)
	}
	if issue.ID != "test-aaa111" {
		t.Errorf("expected test-aaa111, got %s", issue.ID)
	}
	if len(children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(children))
	}
	if children[0].ID != "test-bbb222" {
		t.Errorf("expected child test-bbb222, got %s", children[0].ID)
	}
	if archived {
		t.Error("expected archived=false")
	}
}

func TestRemoteTransport_AddIssue(t *testing.T) {
	srv, rt := newTestServer(t)
	defer srv.Close()

	issue, err := rt.AddIssue(model.CreatePayload{Title: "New issue"})
	if err != nil {
		t.Fatalf("AddIssue: %v", err)
	}
	if issue.ID != "test-new123" {
		t.Errorf("expected test-new123, got %s", issue.ID)
	}
}

func TestRemoteTransport_UpdateIssue(t *testing.T) {
	srv, rt := newTestServer(t)
	defer srv.Close()

	status := "DONE"
	msgs, err := rt.UpdateIssue("test-aaa111", model.UpdatePayload{Status: &status}, "update")
	if err != nil {
		t.Fatalf("UpdateIssue: %v", err)
	}
	if len(msgs) == 0 {
		t.Error("expected at least one message")
	}
}

func TestRemoteTransport_AddComment(t *testing.T) {
	srv, rt := newTestServer(t)
	defer srv.Close()

	err := rt.AddComment("test-aaa111", "hello world")
	if err != nil {
		t.Fatalf("AddComment: %v", err)
	}
}

func TestRemoteTransport_DeleteIssue(t *testing.T) {
	srv, rt := newTestServer(t)
	defer srv.Close()

	err := rt.DeleteIssue("test-aaa111", "no longer needed", false)
	if err != nil {
		t.Fatalf("DeleteIssue: %v", err)
	}
}

func TestRemoteTransport_GetUser(t *testing.T) {
	srv, rt := newTestServer(t)
	defer srv.Close()

	user := rt.GetUser()
	if user != "Test <test@test.com>" {
		t.Errorf("expected 'Test <test@test.com>', got %q", user)
	}
}

func TestRemoteTransport_GetUser_Cached(t *testing.T) {
	srv, rt := newTestServer(t)
	defer srv.Close()

	rt.User = "Cached <cached@test.com>"
	user := rt.GetUser()
	if user != "Cached <cached@test.com>" {
		t.Errorf("expected cached user, got %q", user)
	}
}

func TestRemoteTransport_AuthHeader(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(apiUser{User: "test"})
	}))
	defer srv.Close()

	rt := &RemoteTransport{
		BaseURL:    srv.URL,
		Token:      "my-secret-token",
		HTTPClient: srv.Client(),
	}
	rt.GetUser()

	if gotAuth != "Bearer my-secret-token" {
		t.Errorf("expected 'Bearer my-secret-token', got %q", gotAuth)
	}
}
