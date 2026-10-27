package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/palarix/exponential/internal/exponential"
	"github.com/palarix/exponential/internal/model"
)

func milestoneReq(t *testing.T, mux http.Handler, method, path string, body interface{}) *httptest.ResponseRecorder {
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

func getBoard(t *testing.T, mux http.Handler) MilestonesResponse {
	t.Helper()
	w := milestoneReq(t, mux, "GET", "/api/milestones", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /api/milestones: %d %s", w.Code, w.Body.String())
	}
	var board MilestonesResponse
	if err := json.Unmarshal(w.Body.Bytes(), &board); err != nil {
		t.Fatal(err)
	}
	return board
}

func TestMilestonesAPI_EmptyBoardHasPlaceholders(t *testing.T) {
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

func TestMilestonesAPI_Lifecycle(t *testing.T) {
	srv := setupTestServer(t)
	mux := srv.SetupRoutes()

	w := milestoneReq(t, mux, "POST", "/api/milestones", map[string]string{"label": "1.5", "goal": "ship"})
	if w.Code != http.StatusOK {
		t.Fatalf("create: %d %s", w.Code, w.Body.String())
	}
	var created MilestoneResponse
	json.Unmarshal(w.Body.Bytes(), &created)
	if created.ID == "" || created.Name != "1.5" || created.StartedAt == nil {
		t.Fatalf("created = %+v", created)
	}

	// Duplicate open label.
	if w := milestoneReq(t, mux, "POST", "/api/milestones", map[string]string{"label": "1.5"}); w.Code != http.StatusBadRequest {
		t.Errorf("duplicate label: %d", w.Code)
	}

	// Writing to the Next placeholder saves it.
	w = milestoneReq(t, mux, "PUT", "/api/milestones/"+url.PathEscape("@next"), map[string]string{"label": "1.6"})
	if w.Code != http.StatusOK {
		t.Fatalf("update placeholder: %d %s", w.Code, w.Body.String())
	}
	board := getBoard(t, mux)
	if board.Next.Placeholder || board.Next.Name != "1.6" {
		t.Errorf("next = %+v", board.Next)
	}

	// Assign an issue via the issue draft endpoint with a milestone ref.
	id := seedIssue(t, "in milestone")
	w = milestoneReq(t, mux, "POST", "/api/draft", map[string]interface{}{
		"issue_id": id, "type": "UPDATE", "payload": map[string]string{"milestone_id": "@current"},
	})
	if w.Code != http.StatusOK {
		t.Fatalf("assign: %d %s", w.Code, w.Body.String())
	}
	w = milestoneReq(t, mux, "GET", "/api/issues/"+id, nil)
	var issue IssueResponse
	json.Unmarshal(w.Body.Bytes(), &issue)
	if issue.MilestoneID != created.ID || issue.Milestone != "1.5" {
		t.Errorf("issue milestone = %q / %q", issue.MilestoneID, issue.Milestone)
	}
	if got := getBoard(t, mux).Current; got.Total != 1 || got.Counts["BACKLOG"] != 1 || got.Points == nil || got.Points.Total != 0 {
		t.Errorf("current counts = %+v", got)
	}

	// Progress.
	w = milestoneReq(t, mux, "GET", "/api/milestones/"+created.ID+"/progress", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("progress: %d %s", w.Code, w.Body.String())
	}
	var prog MilestoneProgressResponse
	json.Unmarshal(w.Body.Bytes(), &prog)
	if prog.Total != 1 || len(prog.Days) == 0 || prog.Days[len(prog.Days)-1].Open != 1 {
		t.Errorf("progress = %+v", prog)
	}

	// Reorder: 1.6 before 1.5.
	w = milestoneReq(t, mux, "POST", "/api/milestones/reorder", map[string]string{"id": "1.6", "before": "1.5"})
	if w.Code != http.StatusOK {
		t.Fatalf("reorder: %d %s", w.Code, w.Body.String())
	}
	if getBoard(t, mux).Current.Name != "1.6" {
		t.Error("1.6 should be Current after the reorder")
	}

	// Complete 1.5 (now Next, the last milestone): its open issue moves to the milestone before it.
	w = milestoneReq(t, mux, "POST", "/api/milestones/"+created.ID+"/complete", map[string]string{"issues": model.MilestoneMoveToNext})
	if w.Code != http.StatusOK {
		t.Fatalf("complete: %d %s", w.Code, w.Body.String())
	}
	board = getBoard(t, mux)
	if len(board.Past) != 1 || board.Past[0].State != "COMPLETED" {
		t.Errorf("past = %+v", board.Past)
	}
	if w := milestoneReq(t, mux, "POST", "/api/milestones/"+created.ID+"/complete", nil); w.Code != http.StatusBadRequest {
		t.Errorf("completing twice: %d", w.Code)
	}
	if w := milestoneReq(t, mux, "DELETE", "/api/milestones/"+created.ID, nil); w.Code != http.StatusBadRequest {
		t.Errorf("deleting a completed milestone: %d", w.Code)
	}

	// Nothing came after 1.5, so its open issue moved to 1.6 and no milestone was saved for it.
	if board.Current.Name != "1.6" || board.Current.Total != 1 || !board.Next.Placeholder {
		t.Errorf("current = %+v, next = %+v", board.Current, board.Next)
	}

	// Delete 1.6 and remove its issues.
	w = milestoneReq(t, mux, "DELETE", "/api/milestones/1.6?issues=remove", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("delete: %d %s", w.Code, w.Body.String())
	}
	w = milestoneReq(t, mux, "GET", "/api/issues/"+id, nil)
	issue = IssueResponse{}
	json.Unmarshal(w.Body.Bytes(), &issue)
	if issue.MilestoneID != "" {
		t.Errorf("issue should be in no milestone, got %q", issue.MilestoneID)
	}
}

func TestMilestoneResponse_PointsOnlyWhenKnown(t *testing.T) {
	r := &model.Milestone{ID: "ms-a", Number: 1, State: model.MilestoneCompleted}
	with, _ := json.Marshal(milestoneToResponse(r, "past", exponential.MilestoneScope{Points: &model.MilestonePoints{Total: 8, Done: 3}}))
	if !bytes.Contains(with, []byte(`"points":{"total":8,"done":3}`)) {
		t.Errorf("points missing: %s", with)
	}
	without, _ := json.Marshal(milestoneToResponse(r, "past", exponential.MilestoneScope{}))
	if bytes.Contains(without, []byte(`"points"`)) {
		t.Errorf("a milestone without points leaves the field out: %s", without)
	}
}

func TestMilestonesAPI_CompleteAndDeleteMoveTo(t *testing.T) {
	srv := setupTestServer(t)
	mux := srv.SetupRoutes()
	for _, label := range []string{"1.5", "1.6", "1.7"} {
		milestoneReq(t, mux, "POST", "/api/milestones", map[string]string{"label": label})
	}
	id := seedIssue(t, "x")
	milestoneReq(t, mux, "POST", "/api/draft", map[string]interface{}{
		"issue_id": id, "type": "UPDATE", "payload": map[string]string{"milestone_id": "1.5"},
	})
	milestoneOf := func() string {
		w := milestoneReq(t, mux, "GET", "/api/issues/"+id, nil)
		var issue IssueResponse
		json.Unmarshal(w.Body.Bytes(), &issue)
		return issue.Milestone
	}
	if got := milestoneOf(); got != "1.5" {
		t.Fatalf("issue should start in 1.5, got %q", got)
	}

	if w := milestoneReq(t, mux, "POST", "/api/milestones/1.5/complete", map[string]string{"to": "1.7"}); w.Code != http.StatusOK {
		t.Fatalf("complete to: %d %s", w.Code, w.Body.String())
	}
	if got := milestoneOf(); got != "1.7" {
		t.Errorf("after complete to 1.7 the issue is in %q", got)
	}
	if w := milestoneReq(t, mux, "DELETE", "/api/milestones/1.7?to=1.6", nil); w.Code != http.StatusOK {
		t.Fatalf("delete to: %d %s", w.Code, w.Body.String())
	}
	if got := milestoneOf(); got != "1.6" {
		t.Errorf("after delete to 1.6 the issue is in %q", got)
	}
	if w := milestoneReq(t, mux, "DELETE", "/api/milestones/1.6?issues=remove&to=1.6", nil); w.Code != http.StatusBadRequest {
		t.Errorf("to with remove: %d", w.Code)
	}
}

func TestMilestonesAPI_PlaceholderCannotBeCompletedOrDeleted(t *testing.T) {
	srv := setupTestServer(t)
	mux := srv.SetupRoutes()
	if w := milestoneReq(t, mux, "POST", "/api/milestones/"+url.PathEscape("@current")+"/complete", nil); w.Code != http.StatusBadRequest {
		t.Errorf("complete placeholder: %d", w.Code)
	}
	if w := milestoneReq(t, mux, "DELETE", "/api/milestones/"+url.PathEscape("@next"), nil); w.Code != http.StatusBadRequest {
		t.Errorf("delete placeholder: %d", w.Code)
	}
	if w := milestoneReq(t, mux, "GET", "/api/milestones/"+url.PathEscape("#9")+"/progress", nil); w.Code != http.StatusNotFound {
		t.Errorf("progress of unknown milestone: %d", w.Code)
	}
}

func TestActivity_NamesTheMilestone(t *testing.T) {
	srv := setupTestServer(t)
	mux := srv.SetupRoutes()
	milestoneReq(t, mux, "POST", "/api/milestones", map[string]string{"label": "1.5"})
	id := seedIssue(t, "x")
	milestoneReq(t, mux, "POST", "/api/draft", map[string]interface{}{
		"issue_id": id, "type": "UPDATE", "payload": map[string]string{"milestone_id": "1.5"},
	})

	w := milestoneReq(t, mux, "GET", "/api/activity", nil)
	var items []map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &items)
	if len(items) == 0 || items[0]["milestone"] != "1.5" {
		t.Errorf("latest activity should name the milestone: %v", items)
	}
	for _, it := range items {
		if it["issue_id"] == nil || len(it["issue_id"].(string)) > 4 && it["issue_id"].(string)[:4] == model.MilestoneIDPrefix {
			t.Errorf("milestone events are not issue activity: %v", it)
		}
	}
}

func TestActivity_CompleteIsOneEntry(t *testing.T) {
	srv := setupTestServer(t)
	mux := srv.SetupRoutes()
	milestoneReq(t, mux, "POST", "/api/milestones", map[string]string{"label": "1.5"})
	milestoneReq(t, mux, "POST", "/api/milestones", map[string]string{"label": "1.6"})
	for _, title := range []string{"a", "b", "c"} {
		id := seedIssue(t, title)
		milestoneReq(t, mux, "POST", "/api/draft", map[string]interface{}{
			"issue_id": id, "type": "UPDATE", "payload": map[string]string{"milestone_id": "1.5"},
		})
	}
	milestoneReq(t, mux, "POST", "/api/milestones/1.5/complete", nil)

	w := milestoneReq(t, mux, "GET", "/api/activity", nil)
	var items []map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &items)
	if len(items) == 0 || items[0]["type"] != string(model.EventTypeMilestoneComplete) || items[0]["milestone"] != "1.5" || items[0]["moved_to"] != "1.6" {
		t.Fatalf("latest activity should be the milestone: %v", items[0])
	}
	if items[1]["type"] == string(model.EventTypeUpdate) && items[1]["milestone"] == "1.6" {
		t.Errorf("the caused moves must not appear: %v", items[1])
	}
}
