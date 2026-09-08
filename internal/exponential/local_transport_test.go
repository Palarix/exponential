package exponential

import (
	"strings"
	"testing"
	"time"

	"github.com/palarix/exponential/internal/model"
	"github.com/palarix/exponential/internal/storage"
)

func TestGetIssue_Found(t *testing.T) {
	tr := setupLocalTransport(t)
	created, err := tr.AddIssue(model.CreatePayload{Title: "Get test", Status: "DOING"})
	if err != nil {
		t.Fatal(err)
	}

	got, err := tr.GetIssue(created.ID)
	if err != nil {
		t.Fatalf("GetIssue failed: %v", err)
	}
	if got.Title != "Get test" {
		t.Errorf("expected title 'Get test', got %q", got.Title)
	}
	if got.Status != model.StatusDoing {
		t.Errorf("expected DOING, got %s", got.Status)
	}
}

func TestGetIssue_NotFound(t *testing.T) {
	tr := setupLocalTransport(t)
	_, err := tr.GetIssue("test-nonexistent999")
	if err == nil {
		t.Fatal("expected error for nonexistent issue")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' in error, got: %s", err)
	}
}

func TestGetIssue_ShortID(t *testing.T) {
	tr := setupLocalTransport(t)
	created, err := tr.AddIssue(model.CreatePayload{Title: "Short ID test"})
	if err != nil {
		t.Fatal(err)
	}

	shortID := strings.TrimPrefix(created.ID, "test-")
	got, err := tr.GetIssue(shortID)
	if err != nil {
		t.Fatalf("GetIssue with short ID failed: %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("expected %s, got %s", created.ID, got.ID)
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
