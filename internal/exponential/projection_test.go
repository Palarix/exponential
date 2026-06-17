package exponential

import (
	"testing"
	"time"

	"github.com/palarix/exponential/internal/model"
)

func makeIssue(id string, status model.IssueStatus, sortOrder string, createdAt time.Time) *model.Issue {
	return &model.Issue{
		ID:        id,
		Status:    status,
		SortOrder: sortOrder,
		CreatedAt: createdAt,
	}
}

func TestBackfillSortOrderGlobal(t *testing.T) {
	now := time.Now()
	issues := map[string]*model.Issue{
		"a": makeIssue("a", model.StatusBacklog, "", now),
		"b": makeIssue("b", model.StatusPlanned, "", now.Add(time.Second)),
		"c": makeIssue("c", model.StatusDoing, "", now.Add(2*time.Second)),
	}

	backfillSortOrder(issues)

	for id, issue := range issues {
		if issue.SortOrder == "" {
			t.Fatalf("issue %s has empty sort_order after backfill", id)
		}
	}

	if issues["a"].SortOrder >= issues["b"].SortOrder {
		t.Fatalf("expected a < b, got %s >= %s", issues["a"].SortOrder, issues["b"].SortOrder)
	}
	if issues["b"].SortOrder >= issues["c"].SortOrder {
		t.Fatalf("expected b < c, got %s >= %s", issues["b"].SortOrder, issues["c"].SortOrder)
	}
}

func TestBackfillSortOrderPreservesExisting(t *testing.T) {
	now := time.Now()
	issues := map[string]*model.Issue{
		"a": makeIssue("a", model.StatusBacklog, "a5", now),
		"b": makeIssue("b", model.StatusPlanned, "a8", now.Add(time.Second)),
		"c": makeIssue("c", model.StatusDoing, "", now.Add(2*time.Second)),
	}

	backfillSortOrder(issues)

	if issues["a"].SortOrder != "a5" {
		t.Fatalf("expected a to keep a5, got %s", issues["a"].SortOrder)
	}
	if issues["b"].SortOrder != "a8" {
		t.Fatalf("expected b to keep a8, got %s", issues["b"].SortOrder)
	}
	if issues["c"].SortOrder <= "a8" {
		t.Fatalf("expected c > a8 (global max), got %s", issues["c"].SortOrder)
	}
}

func TestBackfillSortOrderMixedStatuses(t *testing.T) {
	now := time.Now()
	issues := map[string]*model.Issue{
		"a": makeIssue("a", model.StatusBacklog, "a5", now),
		"b": makeIssue("b", model.StatusPlanned, "", now.Add(time.Second)),
		"c": makeIssue("c", model.StatusDoing, "", now.Add(2*time.Second)),
	}

	backfillSortOrder(issues)

	if issues["a"].SortOrder != "a5" {
		t.Fatalf("keyed issue changed: expected a5, got %s", issues["a"].SortOrder)
	}
	if issues["b"].SortOrder <= "a5" {
		t.Fatalf("expected b > a5, got %s", issues["b"].SortOrder)
	}
	if issues["c"].SortOrder <= issues["b"].SortOrder {
		t.Fatalf("expected c > b, got c=%s b=%s", issues["c"].SortOrder, issues["b"].SortOrder)
	}
}

func TestBackfillSortOrderEmpty(t *testing.T) {
	issues := map[string]*model.Issue{}
	backfillSortOrder(issues)
}

func TestBackfillSortOrderAllKeyed(t *testing.T) {
	now := time.Now()
	issues := map[string]*model.Issue{
		"a": makeIssue("a", model.StatusBacklog, "a1", now),
		"b": makeIssue("b", model.StatusPlanned, "a3", now.Add(time.Second)),
		"c": makeIssue("c", model.StatusDoing, "a5", now.Add(2*time.Second)),
	}

	backfillSortOrder(issues)

	if issues["a"].SortOrder != "a1" {
		t.Fatalf("expected a1, got %s", issues["a"].SortOrder)
	}
	if issues["b"].SortOrder != "a3" {
		t.Fatalf("expected a3, got %s", issues["b"].SortOrder)
	}
	if issues["c"].SortOrder != "a5" {
		t.Fatalf("expected a5, got %s", issues["c"].SortOrder)
	}
}
