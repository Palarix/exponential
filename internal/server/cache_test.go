package server

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/palarix/beats/internal/model"
	"github.com/palarix/beats/internal/storage"
)

func TestProjectionCache_HitOnRepeatCall(t *testing.T) {
	srv := setupTestServer(t)
	seedIssue(t, "Cached Issue")

	issues1, err := srv.GetProjectedIssues()
	if err != nil {
		t.Fatal(err)
	}
	issues2, err := srv.GetProjectedIssues()
	if err != nil {
		t.Fatal(err)
	}

	if len(issues1) != 1 || len(issues2) != 1 {
		t.Fatalf("expected 1 issue each call, got %d and %d", len(issues1), len(issues2))
	}

	// Same pointer means cache was reused.
	for k := range issues1 {
		if issues1[k] != issues2[k] {
			t.Error("expected same pointer on cache hit")
		}
	}
}

func TestProjectionCache_InvalidatedByFileChange(t *testing.T) {
	srv := setupTestServer(t)
	seedIssue(t, "Original")

	issues1, err := srv.GetProjectedIssues()
	if err != nil {
		t.Fatal(err)
	}
	if len(issues1) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues1))
	}

	// Ensure mtime changes (some filesystems have 1s granularity).
	time.Sleep(10 * time.Millisecond)
	seedIssue(t, "Added Later")

	issues2, err := srv.GetProjectedIssues()
	if err != nil {
		t.Fatal(err)
	}
	if len(issues2) != 2 {
		t.Fatalf("expected 2 issues after file change, got %d", len(issues2))
	}
}

func TestProjectionCache_InvalidatedByPendingEvent(t *testing.T) {
	srv := setupTestServer(t)
	id := seedIssue(t, "Base Issue")

	issues1, err := srv.GetProjectedIssues()
	if err != nil {
		t.Fatal(err)
	}
	if issues1[id].Title != "Base Issue" {
		t.Fatalf("unexpected title: %s", issues1[id].Title)
	}

	newTitle := "Updated via Pending"
	srv.AddPendingEvent(model.Event{
		ID:        id,
		Type:      model.EventTypeUpdate,
		Payload:   model.UpdatePayload{Title: &newTitle},
		CreatedBy: "test",
		CreatedAt: time.Now(),
	})

	issues2, err := srv.GetProjectedIssues()
	if err != nil {
		t.Fatal(err)
	}
	if issues2[id].Title != newTitle {
		t.Fatalf("expected pending title %q, got %q", newTitle, issues2[id].Title)
	}
}

func TestProjectionCache_DiscardPendingInvalidates(t *testing.T) {
	srv := setupTestServer(t)
	id := seedIssue(t, "Original Title")

	newTitle := "Pending Title"
	srv.AddPendingEvent(model.Event{
		ID:        id,
		Type:      model.EventTypeUpdate,
		Payload:   model.UpdatePayload{Title: &newTitle},
		CreatedBy: "test",
		CreatedAt: time.Now(),
	})

	issues1, err := srv.GetProjectedIssues()
	if err != nil {
		t.Fatal(err)
	}
	if issues1[id].Title != newTitle {
		t.Fatalf("expected pending title, got %q", issues1[id].Title)
	}

	srv.DiscardPending()

	issues2, err := srv.GetProjectedIssues()
	if err != nil {
		t.Fatal(err)
	}
	if issues2[id].Title != "Original Title" {
		t.Fatalf("expected original title after discard, got %q", issues2[id].Title)
	}
}

func TestGetAllEvents_CachesPersistedEvents(t *testing.T) {
	srv := setupTestServer(t)
	seedIssue(t, "Event Issue")

	events1, err := srv.GetAllEvents()
	if err != nil {
		t.Fatal(err)
	}
	events2, err := srv.GetAllEvents()
	if err != nil {
		t.Fatal(err)
	}

	if len(events1) != len(events2) {
		t.Fatalf("expected same event count, got %d and %d", len(events1), len(events2))
	}
	if len(events1) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events1))
	}
}

func TestGetAllEvents_IncludesPending(t *testing.T) {
	srv := setupTestServer(t)
	id := seedIssue(t, "Persisted")

	srv.AddPendingEvent(model.Event{
		ID:        id,
		Type:      model.EventTypeComment,
		Payload:   model.CommentPayload{Text: "pending comment"},
		CreatedBy: "test",
		CreatedAt: time.Now(),
	})

	events, err := srv.GetAllEvents()
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events (1 persisted + 1 pending), got %d", len(events))
	}
}

func TestEventCache_DetectsDeletedDB(t *testing.T) {
	srv := setupTestServer(t)
	seedIssue(t, "Will Be Deleted")

	issues1, err := srv.GetProjectedIssues()
	if err != nil {
		t.Fatal(err)
	}
	if len(issues1) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues1))
	}

	os.Remove(filepath.Join(".beats", "issues.db"))

	issues2, err := srv.GetProjectedIssues()
	if err != nil {
		t.Fatal(err)
	}
	if len(issues2) != 0 {
		t.Fatalf("expected 0 issues after DB deletion, got %d", len(issues2))
	}
}

func TestEventCache_EmptyDBCaches(t *testing.T) {
	srv := setupTestServer(t)

	// First call with empty DB.
	issues1, err := srv.GetProjectedIssues()
	if err != nil {
		t.Fatal(err)
	}
	if len(issues1) != 0 {
		t.Fatalf("expected 0 issues, got %d", len(issues1))
	}

	// Second call should hit cache (same empty result).
	issues2, err := srv.GetProjectedIssues()
	if err != nil {
		t.Fatal(err)
	}
	if len(issues2) != 0 {
		t.Fatalf("expected 0 issues on cache hit, got %d", len(issues2))
	}
}

func TestSaveAndSync_InvalidatesCache(t *testing.T) {
	srv := setupTestServer(t)
	id := seedIssue(t, "Original")

	// Prime the cache.
	_, err := srv.GetProjectedIssues()
	if err != nil {
		t.Fatal(err)
	}

	newTitle := "Saved Title"
	srv.AddPendingEvent(model.Event{
		ID:        id,
		Type:      model.EventTypeUpdate,
		Payload:   model.UpdatePayload{Title: &newTitle},
		CreatedBy: "test",
		CreatedAt: time.Now(),
	})

	if err := srv.SaveAndSync("test save"); err != nil {
		t.Fatal(err)
	}

	// After save, pending is empty and event is on disk.
	if srv.GetPendingCount() != 0 {
		t.Fatal("expected 0 pending after save")
	}

	issues, err := srv.GetProjectedIssues()
	if err != nil {
		t.Fatal(err)
	}
	if issues[id].Title != newTitle {
		t.Fatalf("expected saved title %q, got %q", newTitle, issues[id].Title)
	}
}

func TestExternalWrite_DetectedByMtime(t *testing.T) {
	srv := setupTestServer(t)
	seedIssue(t, "First")

	issues1, err := srv.GetProjectedIssues()
	if err != nil {
		t.Fatal(err)
	}
	if len(issues1) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues1))
	}

	// Simulate external write (CLI or MCP) by appending directly.
	time.Sleep(10 * time.Millisecond)
	storage.AppendEvent(model.Event{
		ID:        "external-001",
		Type:      model.EventTypeCreate,
		Payload:   model.CreatePayload{Title: "External Issue", Labels: []string{"bug"}},
		CreatedBy: "external@cli",
		CreatedAt: time.Now(),
	})

	issues2, err := srv.GetProjectedIssues()
	if err != nil {
		t.Fatal(err)
	}
	if len(issues2) != 2 {
		t.Fatalf("expected 2 issues after external write, got %d", len(issues2))
	}
}
