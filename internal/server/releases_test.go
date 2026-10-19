package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/palarix/exponential/internal/model"
)

func releaseReq(t *testing.T, mux http.Handler, method, path string, body interface{}) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	return w
}

func getBoard(t *testing.T, mux http.Handler) ReleasesResponse {
	t.Helper()
	w := releaseReq(t, mux, "GET", "/api/releases", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /api/releases: %d %s", w.Code, w.Body.String())
	}
	var board ReleasesResponse
	if err := json.Unmarshal(w.Body.Bytes(), &board); err != nil {
		t.Fatal(err)
	}
	return board
}

func TestReleasesAPI_EmptyBoardHasPlaceholders(t *testing.T) {
	srv := setupTestServer(t)
	board := getBoard(t, srv.SetupRoutes())
	if board.Current == nil || !board.Current.Placeholder || board.Current.Number != 1 {
		t.Errorf("current = %+v", board.Current)
	}
	if board.Next == nil || !board.Next.Placeholder || board.Next.Number != 2 || !board.Next.Locked {
		t.Errorf("next = %+v", board.Next)
	}
	if board.Upcoming == nil || board.Past == nil {
		t.Error("upcoming and past are empty arrays, not null")
	}
}

func TestReleasesAPI_Lifecycle(t *testing.T) {
	srv := setupTestServer(t)
	mux := srv.SetupRoutes()

	w := releaseReq(t, mux, "POST", "/api/releases", map[string]string{"label": "1.5", "goal": "ship"})
	if w.Code != http.StatusOK {
		t.Fatalf("create: %d %s", w.Code, w.Body.String())
	}
	var created ReleaseResponse
	json.Unmarshal(w.Body.Bytes(), &created)
	if created.ID == "" || created.Name != "1.5" || created.StartedAt == nil {
		t.Fatalf("created = %+v", created)
	}

	// Duplicate open label.
	if w := releaseReq(t, mux, "POST", "/api/releases", map[string]string{"label": "1.5"}); w.Code != http.StatusBadRequest {
		t.Errorf("duplicate label: %d", w.Code)
	}

	// Writing to the Next placeholder saves it.
	w = releaseReq(t, mux, "PUT", "/api/releases/"+url.PathEscape("@next"), map[string]string{"label": "1.6"})
	if w.Code != http.StatusOK {
		t.Fatalf("update placeholder: %d %s", w.Code, w.Body.String())
	}
	board := getBoard(t, mux)
	if board.Next.Placeholder || board.Next.Name != "1.6" {
		t.Errorf("next = %+v", board.Next)
	}

	// Assign an issue via the issue draft endpoint with a release ref.
	id := seedIssue(t, "in release")
	w = releaseReq(t, mux, "POST", "/api/draft", map[string]interface{}{
		"issue_id": id, "type": "UPDATE", "payload": map[string]string{"release_id": "@current"},
	})
	if w.Code != http.StatusOK {
		t.Fatalf("assign: %d %s", w.Code, w.Body.String())
	}
	w = releaseReq(t, mux, "GET", "/api/issues/"+id, nil)
	var issue IssueResponse
	json.Unmarshal(w.Body.Bytes(), &issue)
	if issue.ReleaseID != created.ID || issue.Release != "1.5" {
		t.Errorf("issue release = %q / %q", issue.ReleaseID, issue.Release)
	}
	if got := getBoard(t, mux).Current; got.Total != 1 || got.Counts["BACKLOG"] != 1 {
		t.Errorf("current counts = %+v", got)
	}

	// Progress.
	w = releaseReq(t, mux, "GET", "/api/releases/"+created.ID+"/progress", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("progress: %d %s", w.Code, w.Body.String())
	}
	var prog ReleaseProgressResponse
	json.Unmarshal(w.Body.Bytes(), &prog)
	if prog.Total != 1 || len(prog.Days) == 0 || prog.Days[len(prog.Days)-1].Open != 1 {
		t.Errorf("progress = %+v", prog)
	}

	// Reorder: 1.6 before 1.5.
	w = releaseReq(t, mux, "POST", "/api/releases/reorder", map[string]string{"id": "1.6", "before": "1.5"})
	if w.Code != http.StatusOK {
		t.Fatalf("reorder: %d %s", w.Code, w.Body.String())
	}
	if getBoard(t, mux).Current.Name != "1.6" {
		t.Error("1.6 should be Current after the reorder")
	}

	// Complete 1.5 (now Next): its open issue moves on to the slot after it.
	w = releaseReq(t, mux, "POST", "/api/releases/"+created.ID+"/complete", map[string]string{"issues": model.ReleaseMoveToNext})
	if w.Code != http.StatusOK {
		t.Fatalf("complete: %d %s", w.Code, w.Body.String())
	}
	board = getBoard(t, mux)
	if len(board.Past) != 1 || board.Past[0].State != "RELEASED" {
		t.Errorf("past = %+v", board.Past)
	}
	if w := releaseReq(t, mux, "POST", "/api/releases/"+created.ID+"/complete", nil); w.Code != http.StatusBadRequest {
		t.Errorf("completing twice: %d", w.Code)
	}
	if w := releaseReq(t, mux, "DELETE", "/api/releases/"+created.ID, nil); w.Code != http.StatusBadRequest {
		t.Errorf("deleting a released release: %d", w.Code)
	}

	// 1.5 was Next, so its open issue moved to the slot after it: a saved #3.
	if board.Next.Number != 3 || board.Next.Placeholder || board.Next.Total != 1 {
		t.Errorf("next = %+v", board.Next)
	}

	// Delete #3 and remove its issues.
	w = releaseReq(t, mux, "DELETE", "/api/releases/"+url.PathEscape("#3")+"?issues=remove", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("delete: %d %s", w.Code, w.Body.String())
	}
	w = releaseReq(t, mux, "GET", "/api/issues/"+id, nil)
	issue = IssueResponse{}
	json.Unmarshal(w.Body.Bytes(), &issue)
	if issue.ReleaseID != "" {
		t.Errorf("issue should be in no release, got %q", issue.ReleaseID)
	}
}

func TestReleasesAPI_PlaceholderCannotBeCompletedOrDeleted(t *testing.T) {
	srv := setupTestServer(t)
	mux := srv.SetupRoutes()
	if w := releaseReq(t, mux, "POST", "/api/releases/"+url.PathEscape("@current")+"/complete", nil); w.Code != http.StatusBadRequest {
		t.Errorf("complete placeholder: %d", w.Code)
	}
	if w := releaseReq(t, mux, "DELETE", "/api/releases/"+url.PathEscape("@next"), nil); w.Code != http.StatusBadRequest {
		t.Errorf("delete placeholder: %d", w.Code)
	}
	if w := releaseReq(t, mux, "GET", "/api/releases/"+url.PathEscape("#9")+"/progress", nil); w.Code != http.StatusNotFound {
		t.Errorf("progress of unknown release: %d", w.Code)
	}
}

func TestActivity_NamesTheRelease(t *testing.T) {
	srv := setupTestServer(t)
	mux := srv.SetupRoutes()
	releaseReq(t, mux, "POST", "/api/releases", map[string]string{"label": "1.5"})
	id := seedIssue(t, "x")
	releaseReq(t, mux, "POST", "/api/draft", map[string]interface{}{
		"issue_id": id, "type": "UPDATE", "payload": map[string]string{"release_id": "1.5"},
	})

	w := releaseReq(t, mux, "GET", "/api/activity", nil)
	var items []map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &items)
	if len(items) == 0 || items[0]["release"] != "1.5" {
		t.Errorf("latest activity should name the release: %v", items)
	}
	for _, it := range items {
		if it["issue_id"] == nil || len(it["issue_id"].(string)) > 4 && it["issue_id"].(string)[:4] == model.ReleaseIDPrefix {
			t.Errorf("release events are not issue activity: %v", it)
		}
	}
}

func TestActivity_CompleteIsOneEntry(t *testing.T) {
	srv := setupTestServer(t)
	mux := srv.SetupRoutes()
	releaseReq(t, mux, "POST", "/api/releases", map[string]string{"label": "1.5"})
	releaseReq(t, mux, "POST", "/api/releases", map[string]string{"label": "1.6"})
	for _, title := range []string{"a", "b", "c"} {
		id := seedIssue(t, title)
		releaseReq(t, mux, "POST", "/api/draft", map[string]interface{}{
			"issue_id": id, "type": "UPDATE", "payload": map[string]string{"release_id": "1.5"},
		})
	}
	releaseReq(t, mux, "POST", "/api/releases/1.5/complete", nil)

	w := releaseReq(t, mux, "GET", "/api/activity", nil)
	var items []map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &items)
	if len(items) == 0 || items[0]["type"] != string(model.EventTypeReleaseComplete) || items[0]["release"] != "1.5" || items[0]["moved_to"] != "1.6" {
		t.Fatalf("latest activity should be the release: %v", items[0])
	}
	if items[1]["type"] == string(model.EventTypeUpdate) && items[1]["release"] == "1.6" {
		t.Errorf("the caused moves must not appear: %v", items[1])
	}
}
