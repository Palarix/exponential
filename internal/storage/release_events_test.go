package storage

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/palarix/exponential/internal/model"
)

func releaseCreate(id string, number int) model.Event {
	return model.Event{
		ID:        id,
		Type:      model.EventTypeReleaseCreate,
		Payload:   model.ReleaseCreatePayload{Number: number},
		CreatedAt: time.Now().UTC(),
		CreatedBy: "test",
	}
}

func releaseUpdate(id, goal string) model.Event {
	return model.Event{
		ID:        id,
		Type:      model.EventTypeReleaseUpdate,
		Payload:   model.ReleaseUpdatePayload{Goal: &goal},
		CreatedAt: time.Now().UTC(),
		CreatedBy: "test",
	}
}

func TestAppendEventCollapsed_KeepsReleaseEvents(t *testing.T) {
	setupTestRepo(t)

	for _, evt := range []model.Event{
		releaseCreate("rel-a", 1),
		releaseUpdate("rel-a", "first"),
		releaseUpdate("rel-a", "first"),
		{ID: "rel-a", Type: model.EventTypeReleaseComplete, Payload: model.ReleaseCompletePayload{OpenIssues: model.ReleaseRemove}, CreatedAt: time.Now().UTC(), CreatedBy: "test"},
	} {
		if err := AppendEventCollapsed(evt); err != nil {
			t.Fatal(err)
		}
	}

	events, _ := ReadEvents()
	if len(events) != 4 {
		t.Fatalf("release events must not be collapsed or pruned: got %d events, want 4", len(events))
	}
}

func TestAppendEventCollapsed_KeepsReleaseIDOnIssueUpdates(t *testing.T) {
	setupTestRepo(t)

	AppendEventCollapsed(makeEvent("i1", model.EventTypeCreate))
	title := "new title"
	AppendEventCollapsed(model.Event{ID: "i1", Type: model.EventTypeUpdate, Payload: model.UpdatePayload{Title: &title}, CreatedAt: time.Now().UTC(), CreatedBy: "test"})
	rel := "rel-a"
	AppendEventCollapsed(model.Event{ID: "i1", Type: model.EventTypeUpdate, Payload: model.UpdatePayload{ReleaseID: &rel}, CreatedAt: time.Now().UTC(), CreatedBy: "test"})

	events, _ := ReadEvents()
	if len(events) != 2 {
		t.Fatalf("got %d events, want create + one merged update", len(events))
	}
	b, _ := json.Marshal(events[1].Payload)
	var p model.UpdatePayload
	json.Unmarshal(b, &p)
	if p.ReleaseID == nil || *p.ReleaseID != "rel-a" || p.Title == nil {
		t.Errorf("merged update lost a field: %s", b)
	}

	// Setting the same release again is a no-op and is pruned.
	AppendEventCollapsed(model.Event{ID: "i1", Type: model.EventTypeUpdate, Payload: model.UpdatePayload{ReleaseID: &rel}, CreatedAt: time.Now().UTC(), CreatedBy: "other"})
	events, _ = ReadEvents()
	if len(events) != 2 {
		t.Errorf("no-op release update should be pruned, got %d events", len(events))
	}
}

func TestArchiveEvents_NeverArchivesReleaseEvents(t *testing.T) {
	setupXpoDir(t)

	AppendEvent(releaseCreate("rel-a", 1))
	AppendEvent(makeDone("old"))
	AppendEvent(model.Event{ID: "rel-a", Type: model.EventTypeReleaseComplete, Payload: model.ReleaseCompletePayload{}, CreatedAt: time.Now().UTC(), CreatedBy: "test"})

	res, err := ArchiveEvents(ids("old", "rel-a"))
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Archived) != 1 || res.Archived[0] != "old" {
		t.Errorf("archived = %v, want only [old]", res.Archived)
	}
	remaining, _ := ReadEvents()
	if len(remaining) != 2 || remaining[0].ID != "rel-a" || remaining[1].ID != "rel-a" {
		t.Errorf("release events must stay in issues.db, got %v", eventIDs(remaining))
	}
}

func TestAppendEvents_WritesBatchInOrder(t *testing.T) {
	setupXpoDir(t)

	AppendEvent(makeEvent("a", model.EventTypeCreate))
	now := time.Now().UTC().Add(-time.Hour) // earlier than the existing line
	err := AppendEvents(false, func(current []model.Event) ([]model.Event, error) {
		if len(current) != 1 {
			t.Errorf("builder should see the current log, got %d events", len(current))
		}
		return []model.Event{
			{ID: "b", Type: model.EventTypeCreate, Payload: model.CreatePayload{Title: "b"}, CreatedAt: now, CreatedBy: "test"},
			{ID: "c", Type: model.EventTypeCreate, Payload: model.CreatePayload{Title: "c"}, CreatedAt: now, CreatedBy: "test"},
		}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	events, _ := ReadEvents()
	if got := eventIDs(events); len(got) != 3 || got[1] != "b" || got[2] != "c" {
		t.Fatalf("events = %v, want [a b c]", got)
	}
	if !events[1].CreatedAt.After(events[0].CreatedAt) || !events[2].CreatedAt.After(events[1].CreatedAt) {
		t.Error("batch timestamps must strictly increase in file order")
	}
}

func TestAppendEvents_BuilderErrorWritesNothing(t *testing.T) {
	setupXpoDir(t)

	AppendEvent(makeEvent("a", model.EventTypeCreate))
	want := errors.New("nope")
	err := AppendEvents(false, func([]model.Event) ([]model.Event, error) {
		return []model.Event{makeEvent("b", model.EventTypeCreate)}, want
	})
	if !errors.Is(err, want) {
		t.Fatalf("err = %v, want builder error", err)
	}
	events, _ := ReadEvents()
	if len(events) != 1 {
		t.Errorf("a failed batch must write nothing, got %d events", len(events))
	}
}

func TestAppendEvents_Collapsed(t *testing.T) {
	setupTestRepo(t)

	AppendEventCollapsed(makeEvent("a", model.EventTypeCreate))
	title := "x"
	err := AppendEvents(true, func([]model.Event) ([]model.Event, error) {
		return []model.Event{
			releaseCreate("rel-a", 1),
			{ID: "a", Type: model.EventTypeUpdate, Payload: model.UpdatePayload{Title: &title}, CreatedAt: time.Now().UTC(), CreatedBy: "test"},
		}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	events, _ := ReadEvents()
	if got := eventIDs(events); len(got) != 3 || got[1] != "rel-a" || got[2] != "a" {
		t.Errorf("events = %v, want [a rel-a a]", got)
	}
}

func TestAppendEventCollapsed_DoesNotFoldAcrossCauses(t *testing.T) {
	setupTestRepo(t)

	AppendEventCollapsed(makeEvent("i1", model.EventTypeCreate))
	title := "edited"
	AppendEventCollapsed(model.Event{ID: "i1", Type: model.EventTypeUpdate, Payload: model.UpdatePayload{Title: &title}, CreatedAt: time.Now().UTC(), CreatedBy: "test"})
	rel := "rel-b"
	AppendEventCollapsed(model.Event{ID: "i1", Type: model.EventTypeUpdate, Payload: model.UpdatePayload{ReleaseID: &rel}, CreatedAt: time.Now().UTC(), CreatedBy: "test", Cause: "rel-a"})

	events, _ := ReadEvents()
	if len(events) != 3 || events[2].Cause != "rel-a" {
		t.Fatalf("a caused update must stay its own event: %+v", events)
	}
}

func TestAppendEventCollapsed_KeepsChildrenOnlyUpdate(t *testing.T) {
	setupTestRepo(t)

	rel := "rel-a"
	AppendEventCollapsed(model.Event{ID: "e1", Type: model.EventTypeCreate, Payload: model.CreatePayload{Title: "epic", ReleaseID: rel}, CreatedAt: time.Now().UTC(), CreatedBy: "test"})
	AppendEventCollapsed(model.Event{ID: "e1", Type: model.EventTypeUpdate, Payload: model.UpdatePayload{ReleaseID: &rel, Children: &model.ChildrenMove{Count: 2, ReleaseID: rel}}, CreatedAt: time.Now().UTC(), CreatedBy: "test"})

	events, _ := ReadEvents()
	if len(events) != 2 {
		t.Fatalf("got %d events, want the children-only update kept", len(events))
	}
	b, _ := json.Marshal(events[1].Payload)
	var p model.UpdatePayload
	json.Unmarshal(b, &p)
	if p.ReleaseID != nil || p.Children == nil || p.Children.Count != 2 {
		t.Errorf("payload = %s, want only children_moved", b)
	}
}
