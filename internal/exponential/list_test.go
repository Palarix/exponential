package exponential

import (
	"testing"
	"time"

	"github.com/palarix/exponential/internal/model"
)

func makeTestIssues() []*model.Issue {
	now := time.Now()
	return []*model.Issue{
		{ID: "a", Title: "Bug in auth", Status: model.StatusBacklog, Labels: []string{"bug"}, Assignee: "Alice <alice@test.com>", CreatedBy: "Alice <alice@test.com>", ParentID: "epic-1", UpdatedAt: now},
		{ID: "b", Title: "Add search", Status: model.StatusPlanned, Labels: []string{"feature"}, Assignee: "Bob <bob@test.com>", CreatedBy: "Bob <bob@test.com>", UpdatedAt: now},
		{ID: "c", Title: "Fix typo", Status: model.StatusDoing, Labels: []string{"bug"}, CreatedBy: "Alice <alice@test.com>", UpdatedAt: now},
		{ID: "d", Title: "Refactor", Status: model.StatusDone, Labels: []string{"improvement"}, CreatedBy: "Bob <bob@test.com>", UpdatedAt: now},
	}
}

func TestFilterIssues_ByStatus(t *testing.T) {
	issues := makeTestIssues()
	result := FilterIssues(issues, FilterOptions{Statuses: []string{"PLANNED"}, All: true}, "")
	if len(result) != 1 || result[0].ID != "b" {
		t.Errorf("expected [b], got %v", ids(result))
	}
}

func TestFilterIssues_MultiStatus(t *testing.T) {
	issues := makeTestIssues()
	result := FilterIssues(issues, FilterOptions{Statuses: []string{"BACKLOG", "DOING"}, All: true}, "")
	if len(result) != 2 {
		t.Errorf("expected 2 issues, got %d", len(result))
	}
}

func TestFilterIssues_ByMatch(t *testing.T) {
	issues := makeTestIssues()
	result := FilterIssues(issues, FilterOptions{Match: "auth", All: true}, "")
	if len(result) != 1 || result[0].ID != "a" {
		t.Errorf("expected [a], got %v", ids(result))
	}
}

func TestFilterIssues_ByMatchCaseInsensitive(t *testing.T) {
	issues := makeTestIssues()
	result := FilterIssues(issues, FilterOptions{Match: "AUTH", All: true}, "")
	if len(result) != 1 {
		t.Errorf("match should be case-insensitive, got %d results", len(result))
	}
}

func TestFilterIssues_ByLabel(t *testing.T) {
	issues := makeTestIssues()
	result := FilterIssues(issues, FilterOptions{Label: "bug", All: true}, "")
	if len(result) != 2 {
		t.Errorf("expected 2 bug issues, got %d", len(result))
	}
}

func TestFilterIssues_ByAssignee(t *testing.T) {
	issues := makeTestIssues()
	result := FilterIssues(issues, FilterOptions{Assignee: "alice", All: true}, "")
	if len(result) != 1 || result[0].ID != "a" {
		t.Errorf("expected [a], got %v", ids(result))
	}
}

func TestFilterIssues_ByParent(t *testing.T) {
	issues := makeTestIssues()
	result := FilterIssues(issues, FilterOptions{ParentID: "epic-1", All: true}, "")
	if len(result) != 1 || result[0].ID != "a" {
		t.Errorf("expected [a], got %v", ids(result))
	}
}

func TestFilterIssues_NoFilters_DoneCountLimited(t *testing.T) {
	now := time.Now()
	issues := []*model.Issue{
		{ID: "a", Title: "A", Status: model.StatusBacklog, UpdatedAt: now},
		{ID: "d1", Title: "D1", Status: model.StatusDone, UpdatedAt: now.Add(-1 * time.Hour)},
		{ID: "d2", Title: "D2", Status: model.StatusDone, UpdatedAt: now.Add(-2 * time.Hour)},
		{ID: "d3", Title: "D3", Status: model.StatusDone, UpdatedAt: now.Add(-3 * time.Hour)},
		{ID: "d4", Title: "D4", Status: model.StatusDone, UpdatedAt: now.Add(-4 * time.Hour)},
	}
	result := FilterIssues(issues, FilterOptions{}, "")
	doneCount := 0
	for _, i := range result {
		if i.Status == model.StatusDone {
			doneCount++
		}
	}
	if doneCount == 0 {
		t.Error("expected some DONE issues to be included (most recent)")
	}
	if doneCount > 3 {
		t.Errorf("without All flag, at most 3 recent DONE should show, got %d", doneCount)
	}
}

func TestFilterIssues_All_ShowsDone(t *testing.T) {
	issues := makeTestIssues()
	result := FilterIssues(issues, FilterOptions{All: true}, "")
	found := false
	for _, i := range result {
		if i.ID == "d" {
			found = true
		}
	}
	if !found {
		t.Error("DONE issue should be shown with All flag")
	}
}

func TestFilterIssues_ExplicitDoneStatus(t *testing.T) {
	issues := makeTestIssues()
	result := FilterIssues(issues, FilterOptions{Statuses: []string{"DONE"}}, "")
	if len(result) != 1 || result[0].ID != "d" {
		t.Errorf("expected [d], got %v", ids(result))
	}
}

func TestParseTimeFilter_Days(t *testing.T) {
	got, err := ParseTimeFilter("7d")
	if err != nil {
		t.Fatal(err)
	}
	expected := time.Now().Add(-7 * 24 * time.Hour)
	if got.Sub(expected).Abs() > time.Second {
		t.Errorf("7d parsed to %v, expected ~%v", got, expected)
	}
}

func TestParseTimeFilter_Weeks(t *testing.T) {
	got, err := ParseTimeFilter("2w")
	if err != nil {
		t.Fatal(err)
	}
	expected := time.Now().Add(-14 * 24 * time.Hour)
	if got.Sub(expected).Abs() > time.Second {
		t.Errorf("2w parsed to %v, expected ~%v", got, expected)
	}
}

func TestParseTimeFilter_Hours(t *testing.T) {
	got, err := ParseTimeFilter("4h")
	if err != nil {
		t.Fatal(err)
	}
	expected := time.Now().Add(-4 * time.Hour)
	if got.Sub(expected).Abs() > time.Second {
		t.Errorf("4h parsed to %v, expected ~%v", got, expected)
	}
}

func TestParseTimeFilter_Date(t *testing.T) {
	got, err := ParseTimeFilter("2026-01-15")
	if err != nil {
		t.Fatal(err)
	}
	expected := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	if !got.Equal(expected) {
		t.Errorf("date parsed to %v, expected %v", got, expected)
	}
}

func TestParseTimeFilter_RFC3339(t *testing.T) {
	got, err := ParseTimeFilter("2026-01-15T10:30:00Z")
	if err != nil {
		t.Fatal(err)
	}
	expected := time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)
	if !got.Equal(expected) {
		t.Errorf("RFC3339 parsed to %v, expected %v", got, expected)
	}
}

func TestParseTimeFilter_Invalid(t *testing.T) {
	_, err := ParseTimeFilter("not-a-time")
	if err == nil {
		t.Error("expected error for invalid time string")
	}
}

func ids(issues []*model.Issue) []string {
	out := make([]string, len(issues))
	for i, iss := range issues {
		out[i] = iss.ID
	}
	return out
}
