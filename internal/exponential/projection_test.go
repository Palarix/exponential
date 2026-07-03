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

func TestProjectIssues_ArtifactCreated(t *testing.T) {
	events := []model.Event{
		{
			ID:        "test-abc",
			Type:      model.EventTypeCreate,
			Payload:   model.CreatePayload{Title: "test issue"},
			CreatedAt: time.Now(),
			CreatedBy: "Tester <test@example.com>",
		},
		{
			ID:   "test-abc",
			Type: model.EventTypeArtifact,
			Payload: model.ArtifactPayload{
				ArtifactType: "spec",
				Filename:     "spec.md",
				Action:       "created",
			},
			CreatedAt: time.Now(),
			CreatedBy: "Tester <test@example.com>",
		},
	}

	issues := ProjectIssues(events)
	issue := issues["test-abc"]
	if issue == nil {
		t.Fatal("issue not found")
	}
	if len(issue.Artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(issue.Artifacts))
	}
	a := issue.Artifacts[0]
	if a.ArtifactType != "spec" {
		t.Errorf("artifact type = %q, want %q", a.ArtifactType, "spec")
	}
	if a.Filename != "spec.md" {
		t.Errorf("filename = %q, want %q", a.Filename, "spec.md")
	}
}

func TestProjectIssues_ArtifactUpdated(t *testing.T) {
	now := time.Now()
	events := []model.Event{
		{
			ID:        "test-abc",
			Type:      model.EventTypeCreate,
			Payload:   model.CreatePayload{Title: "test issue"},
			CreatedAt: now,
			CreatedBy: "Tester <test@example.com>",
		},
		{
			ID:   "test-abc",
			Type: model.EventTypeArtifact,
			Payload: model.ArtifactPayload{
				ArtifactType: "spec",
				Filename:     "spec.md",
				Action:       "created",
			},
			CreatedAt: now,
			CreatedBy: "Alice <alice@example.com>",
		},
		{
			ID:   "test-abc",
			Type: model.EventTypeArtifact,
			Payload: model.ArtifactPayload{
				ArtifactType: "spec",
				Filename:     "spec.md",
				Action:       "updated",
			},
			CreatedAt: now.Add(time.Minute),
			CreatedBy: "Bob <bob@example.com>",
		},
	}

	issues := ProjectIssues(events)
	issue := issues["test-abc"]
	if len(issue.Artifacts) != 1 {
		t.Fatalf("expected 1 artifact after update, got %d", len(issue.Artifacts))
	}
	a := issue.Artifacts[0]
	if a.UpdatedBy != "Bob <bob@example.com>" {
		t.Errorf("updated_by = %q, want Bob", a.UpdatedBy)
	}
	if !a.UpdatedAt.Equal(now.Add(time.Minute)) {
		t.Errorf("updated_at not advanced")
	}
}

func TestProjectIssues_ArtifactDeleted(t *testing.T) {
	events := []model.Event{
		{
			ID:        "test-abc",
			Type:      model.EventTypeCreate,
			Payload:   model.CreatePayload{Title: "test issue"},
			CreatedAt: time.Now(),
			CreatedBy: "Tester <test@example.com>",
		},
		{
			ID:   "test-abc",
			Type: model.EventTypeArtifact,
			Payload: model.ArtifactPayload{
				ArtifactType: "spec",
				Filename:     "spec.md",
				Action:       "created",
			},
			CreatedAt: time.Now(),
			CreatedBy: "Tester <test@example.com>",
		},
		{
			ID:   "test-abc",
			Type: model.EventTypeArtifact,
			Payload: model.ArtifactPayload{
				ArtifactType: "spec",
				Filename:     "spec.md",
				Action:       "deleted",
			},
			CreatedAt: time.Now(),
			CreatedBy: "Tester <test@example.com>",
		},
	}

	issues := ProjectIssues(events)
	issue := issues["test-abc"]
	if len(issue.Artifacts) != 0 {
		t.Fatalf("expected 0 artifacts after delete, got %d", len(issue.Artifacts))
	}
}

func TestProjectIssues_ArtifactOnMissingIssue(t *testing.T) {
	events := []model.Event{
		{
			ID:   "test-nonexistent",
			Type: model.EventTypeArtifact,
			Payload: model.ArtifactPayload{
				ArtifactType: "spec",
				Filename:     "spec.md",
				Action:       "created",
			},
			CreatedAt: time.Now(),
			CreatedBy: "Tester <test@example.com>",
		},
	}

	issues := ProjectIssues(events)
	if len(issues) != 0 {
		t.Fatalf("expected no issues, got %d", len(issues))
	}
}
