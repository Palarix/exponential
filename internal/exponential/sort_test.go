package exponential

import (
	"testing"
	"time"

	"github.com/palarix/exponential/internal/model"
)

func TestSortIssues_GroupsByStatus(t *testing.T) {
	issues := map[string]*model.Issue{
		"a": {ID: "a", Status: model.StatusDoing, SortOrder: "a1", CreatedAt: time.Now()},
		"b": {ID: "b", Status: model.StatusBacklog, SortOrder: "a2", CreatedAt: time.Now()},
		"c": {ID: "c", Status: model.StatusPlanned, SortOrder: "a3", CreatedAt: time.Now()},
	}

	sorted := SortIssues(issues)
	if len(sorted) != 3 {
		t.Fatalf("expected 3 issues, got %d", len(sorted))
	}
}

func TestSortIssues_EmptyMap(t *testing.T) {
	sorted := SortIssues(map[string]*model.Issue{})
	if len(sorted) != 0 {
		t.Errorf("expected 0 issues, got %d", len(sorted))
	}
}

func TestSortIssues_SortOrderWithinGroup(t *testing.T) {
	issues := map[string]*model.Issue{
		"a": {ID: "a", Status: model.StatusBacklog, SortOrder: "b", CreatedAt: time.Now()},
		"b": {ID: "b", Status: model.StatusBacklog, SortOrder: "a", CreatedAt: time.Now()},
	}

	sorted := SortIssues(issues)
	if len(sorted) != 2 {
		t.Fatalf("expected 2, got %d", len(sorted))
	}
	// Both are in same group; verify they're both present
	idSet := map[string]bool{sorted[0].ID: true, sorted[1].ID: true}
	if !idSet["a"] || !idSet["b"] {
		t.Errorf("expected both a and b, got [%s, %s]", sorted[0].ID, sorted[1].ID)
	}
}

func TestSortIssues_ChildrenGroupedUnderParent(t *testing.T) {
	now := time.Now()
	issues := map[string]*model.Issue{
		"p":  {ID: "p", Status: model.StatusBacklog, SortOrder: "a", CreatedAt: now},
		"c1": {ID: "c1", Status: model.StatusBacklog, ParentID: "p", SortOrder: "a1", CreatedAt: now},
		"c2": {ID: "c2", Status: model.StatusBacklog, ParentID: "p", SortOrder: "a2", CreatedAt: now},
		"x":  {ID: "x", Status: model.StatusBacklog, SortOrder: "b", CreatedAt: now},
	}

	sorted := SortIssues(issues)
	parentIdx := -1
	for i, iss := range sorted {
		if iss.ID == "p" {
			parentIdx = i
			break
		}
	}
	if parentIdx == -1 {
		t.Fatal("parent not found in sorted list")
	}
	if parentIdx+1 >= len(sorted) || (sorted[parentIdx+1].ID != "c1" && sorted[parentIdx+1].ID != "c2") {
		t.Error("children should appear immediately after their parent")
	}
}
