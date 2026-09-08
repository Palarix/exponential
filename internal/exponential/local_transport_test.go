package exponential

import (
	"strings"
	"testing"
	"time"

	"github.com/palarix/exponential/internal/model"
	"github.com/palarix/exponential/internal/storage"
)

func TestGetIssue_FieldsPreserved(t *testing.T) {
	tr := setupLocalTransport(t)
	created, err := tr.AddIssue(model.CreatePayload{Title: "Field test", Status: "DOING", Estimate: 5})
	if err != nil {
		t.Fatal(err)
	}

	got, err := tr.GetIssue(created.ID)
	if err != nil {
		t.Fatalf("GetIssue failed: %v", err)
	}
	if got.Title != "Field test" {
		t.Errorf("expected title 'Field test', got %q", got.Title)
	}
	if got.Status != model.StatusDoing {
		t.Errorf("expected DOING, got %s", got.Status)
	}
	if got.Estimate != 5 {
		t.Errorf("expected estimate 5, got %d", got.Estimate)
	}
}

func TestGetIssue_AmbiguousShortID(t *testing.T) {
	tr := setupLocalTransport(t)
	tr.AddIssue(model.CreatePayload{Title: "first"})
	tr.AddIssue(model.CreatePayload{Title: "second"})

	// "test-" is a prefix common to all generated IDs
	_, err := tr.GetIssue("test-")
	if err == nil {
		t.Fatal("expected ambiguity error")
	}
	if !strings.Contains(err.Error(), "ambiguous") {
		t.Errorf("expected 'ambiguous' in error, got: %s", err)
	}
}

func TestFindIssue_ActiveWithChildren(t *testing.T) {
	tr := setupLocalTransport(t)
	parent, err := tr.AddIssue(model.CreatePayload{Title: "Parent", Status: "DOING"})
	if err != nil {
		t.Fatal(err)
	}
	child, err := tr.AddIssue(model.CreatePayload{Title: "Child", Status: "PLANNED", ParentID: parent.ID})
	if err != nil {
		t.Fatal(err)
	}

	issue, children, archived, err := tr.FindIssue(parent.ID)
	if err != nil {
		t.Fatalf("FindIssue failed: %v", err)
	}
	if issue.ID != parent.ID {
		t.Errorf("expected %s, got %s", parent.ID, issue.ID)
	}
	if archived {
		t.Error("expected active, got archived")
	}
	if len(children) != 1 {
		t.Errorf("expected 1 child, got %d", len(children))
	} else if children[0].ID != child.ID {
		t.Errorf("expected child %s, got %s", child.ID, children[0].ID)
	}
}

func TestFindIssue_ActiveNoChildren(t *testing.T) {
	tr := setupLocalTransport(t)
	created, err := tr.AddIssue(model.CreatePayload{Title: "Lonely", Status: "DOING"})
	if err != nil {
		t.Fatal(err)
	}

	issue, children, archived, err := tr.FindIssue(created.ID)
	if err != nil {
		t.Fatalf("FindIssue failed: %v", err)
	}
	if issue.ID != created.ID {
		t.Errorf("expected %s, got %s", created.ID, issue.ID)
	}
	if archived {
		t.Error("expected active, got archived")
	}
	if len(children) != 0 {
		t.Errorf("expected 0 children, got %d", len(children))
	}
}

func TestFindIssue_ArchivedFallback(t *testing.T) {
	tr := setupLocalTransport(t)

	// Create an issue, then archive it by writing to archive.db and removing from issues.db
	created, err := tr.AddIssue(model.CreatePayload{Title: "Archived issue", Status: "DONE"})
	if err != nil {
		t.Fatal(err)
	}

	// Read all events, move them to archive
	events, _ := storage.ReadEvents()
	storage.ArchiveEvents(nil, events)

	// FindIssue should find it in the archive
	issue, _, archived, err := tr.FindIssue(created.ID)
	if err != nil {
		t.Fatalf("FindIssue should find archived issue: %v", err)
	}
	if !archived {
		t.Error("expected archived=true")
	}
	if issue.ID != created.ID {
		t.Errorf("expected %s, got %s", created.ID, issue.ID)
	}
}

func TestFindIssue_NotFound(t *testing.T) {
	tr := setupLocalTransport(t)
	_, _, _, err := tr.FindIssue("test-nonexistent999")
	if err == nil {
		t.Fatal("expected error for nonexistent issue")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' in error, got: %s", err)
	}
}

func TestAppendEvent_Basic(t *testing.T) {
	tr := setupLocalTransport(t)
	evt := model.Event{
		ID:        "test-append01",
		Type:      model.EventTypeCreate,
		Payload:   model.CreatePayload{Title: "Appended", Status: "BACKLOG"},
		CreatedAt: time.Now().UTC(),
		CreatedBy: "Test <test@test.com>",
	}
	if err := tr.appendEvent(evt); err != nil {
		t.Fatalf("appendEvent failed: %v", err)
	}

	events, err := storage.ReadEvents()
	if err != nil {
		t.Fatalf("ReadEvents failed: %v", err)
	}
	found := false
	for _, e := range events {
		if e.ID == "test-append01" {
			found = true
		}
	}
	if !found {
		t.Error("appended event not found in storage")
	}
}

func TestAppendEvent_StampsSource(t *testing.T) {
	tr := setupLocalTransport(t)
	tr.Source = "mcp"
	evt := model.Event{
		ID:        "test-src01",
		Type:      model.EventTypeCreate,
		Payload:   model.CreatePayload{Title: "Source", Status: "BACKLOG"},
		CreatedAt: time.Now().UTC(),
		CreatedBy: "Test <test@test.com>",
	}
	if err := tr.appendEvent(evt); err != nil {
		t.Fatalf("appendEvent failed: %v", err)
	}

	events, _ := storage.ReadEvents()
	for _, e := range events {
		if e.ID == "test-src01" {
			if e.Source != "mcp" {
				t.Errorf("expected Source 'mcp', got %q", e.Source)
			}
			return
		}
	}
	t.Error("event not found")
}

func TestAppendEvent_StampsOnBehalfOf(t *testing.T) {
	tr := setupLocalTransport(t)
	tr.OnBehalfOf = "Agent <agent@test.com>"
	evt := model.Event{
		ID:        "test-obo01",
		Type:      model.EventTypeCreate,
		Payload:   model.CreatePayload{Title: "OnBehalfOf", Status: "BACKLOG"},
		CreatedAt: time.Now().UTC(),
		CreatedBy: "Test <test@test.com>",
	}
	if err := tr.appendEvent(evt); err != nil {
		t.Fatalf("appendEvent failed: %v", err)
	}

	events, _ := storage.ReadEvents()
	for _, e := range events {
		if e.ID == "test-obo01" {
			if e.OnBehalfOf != "Agent <agent@test.com>" {
				t.Errorf("expected OnBehalfOf 'Agent <agent@test.com>', got %q", e.OnBehalfOf)
			}
			return
		}
	}
	t.Error("event not found")
}

func TestGetInbox_Empty(t *testing.T) {
	tr := setupLocalTransport(t)
	items, err := tr.GetInbox(time.Time{})
	if err != nil {
		t.Fatalf("GetInbox failed: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("expected empty inbox, got %d items", len(items))
	}
}
