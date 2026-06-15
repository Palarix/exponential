package ui

import (
	"testing"
	"time"

	"github.com/palarix/beats/internal/model"
)

func TestGetRecentDoneIDs_Empty(t *testing.T) {
	result := GetRecentDoneIDs(nil, 3)
	if len(result) != 0 {
		t.Errorf("expected empty map, got %d", len(result))
	}
}

func TestGetRecentDoneIDs_NoDone(t *testing.T) {
	issues := []*model.Issue{
		{ID: "a", Status: model.StatusDoing, UpdatedAt: time.Now()},
		{ID: "b", Status: model.StatusPlanned, UpdatedAt: time.Now()},
	}
	result := GetRecentDoneIDs(issues, 3)
	if len(result) != 0 {
		t.Errorf("expected 0 done IDs, got %d", len(result))
	}
}

func TestGetRecentDoneIDs_TopN(t *testing.T) {
	now := time.Now()
	issues := []*model.Issue{
		{ID: "old", Status: model.StatusDone, UpdatedAt: now.Add(-3 * time.Hour)},
		{ID: "mid", Status: model.StatusDone, UpdatedAt: now.Add(-2 * time.Hour)},
		{ID: "new", Status: model.StatusDone, UpdatedAt: now.Add(-1 * time.Hour)},
		{ID: "newest", Status: model.StatusDone, UpdatedAt: now},
	}
	result := GetRecentDoneIDs(issues, 2)
	if len(result) != 2 {
		t.Fatalf("expected 2, got %d", len(result))
	}
	if !result["newest"] || !result["new"] {
		t.Errorf("expected newest and new, got %v", result)
	}
}

func TestGetRecentDoneIDs_CountExceedsAvailable(t *testing.T) {
	issues := []*model.Issue{
		{ID: "a", Status: model.StatusDone, UpdatedAt: time.Now()},
	}
	result := GetRecentDoneIDs(issues, 5)
	if len(result) != 1 {
		t.Errorf("expected 1, got %d", len(result))
	}
}

func TestGetRecentDoneIDs_MixedStatuses(t *testing.T) {
	now := time.Now()
	issues := []*model.Issue{
		{ID: "doing", Status: model.StatusDoing, UpdatedAt: now},
		{ID: "done1", Status: model.StatusDone, UpdatedAt: now.Add(-1 * time.Hour)},
		{ID: "planned", Status: model.StatusPlanned, UpdatedAt: now},
		{ID: "done2", Status: model.StatusDone, UpdatedAt: now},
	}
	result := GetRecentDoneIDs(issues, 3)
	if len(result) != 2 {
		t.Errorf("expected 2 done issues, got %d", len(result))
	}
	if !result["done1"] || !result["done2"] {
		t.Errorf("expected done1 and done2, got %v", result)
	}
}

func TestGetRecentDoneIDsFromMap(t *testing.T) {
	now := time.Now()
	issuesMap := map[string]*model.Issue{
		"a": {ID: "a", Status: model.StatusDone, UpdatedAt: now},
		"b": {ID: "b", Status: model.StatusDoing, UpdatedAt: now},
	}
	result := GetRecentDoneIDsFromMap(issuesMap, 3)
	if len(result) != 1 || !result["a"] {
		t.Errorf("expected {a: true}, got %v", result)
	}
}
