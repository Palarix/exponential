package beats

import (
	"testing"
	"time"

	"github.com/palarix/beats/internal/model"
)

func TestRunMigrations_AlreadyUpToDate(t *testing.T) {
	version, err := RunMigrations(3)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if version != 3 {
		t.Errorf("Expected version 3, got %d", version)
	}
}

func TestRunMigrations_OldVersion(t *testing.T) {
	_, err := RunMigrations(1)
	if err == nil {
		t.Error("Expected error for old version")
	}
}

func TestRunMigrations_V0(t *testing.T) {
	_, err := RunMigrations(0)
	if err == nil {
		t.Error("Expected error for v0")
	}
}

func TestBuildGlobalOrder(t *testing.T) {
	now := time.Now()
	issues := map[string]*model.Issue{
		"a": {ID: "a", Status: model.StatusDoing, SortOrder: "a1", CreatedAt: now},
		"b": {ID: "b", Status: model.StatusBacklog, SortOrder: "a0", CreatedAt: now},
		"c": {ID: "c", Status: model.StatusPlanned, SortOrder: "a2", CreatedAt: now},
		"d": {ID: "d", Status: model.StatusDone, SortOrder: "a3", CreatedAt: now},
	}

	ordered := buildGlobalOrder(issues)

	if len(ordered) != 4 {
		t.Fatalf("expected 4 issues, got %d", len(ordered))
	}

	expectedOrder := []string{"b", "c", "a", "d"}
	for i, id := range expectedOrder {
		if ordered[i].ID != id {
			t.Errorf("position %d: expected %s, got %s", i, id, ordered[i].ID)
		}
	}
}

func TestBuildGlobalOrder_WithinGroupSort(t *testing.T) {
	now := time.Now()
	issues := map[string]*model.Issue{
		"a": {ID: "a", Status: model.StatusPlanned, SortOrder: "a5", CreatedAt: now},
		"b": {ID: "b", Status: model.StatusPlanned, SortOrder: "a2", CreatedAt: now},
		"c": {ID: "c", Status: model.StatusPlanned, SortOrder: "a8", CreatedAt: now},
	}

	ordered := buildGlobalOrder(issues)

	if ordered[0].ID != "b" || ordered[1].ID != "a" || ordered[2].ID != "c" {
		t.Errorf("expected b,a,c got %s,%s,%s", ordered[0].ID, ordered[1].ID, ordered[2].ID)
	}
}

func TestBuildGlobalOrder_Empty(t *testing.T) {
	ordered := buildGlobalOrder(map[string]*model.Issue{})
	if len(ordered) != 0 {
		t.Fatalf("expected empty, got %d", len(ordered))
	}
}
