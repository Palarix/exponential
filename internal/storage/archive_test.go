package storage

import (
	"testing"
	"time"

	"github.com/palarix/exponential/internal/model"
)

func makeEvent(id string, evtType model.EventType) model.Event {
	return model.Event{ID: id, Type: evtType, Payload: model.CreatePayload{Title: id}, CreatedAt: time.Now().UTC(), CreatedBy: "test"}
}

func TestArchiveEvents_MovesToArchive(t *testing.T) {
	setupXpoDir(t)

	active := []model.Event{makeEvent("a", model.EventTypeCreate)}
	archived := []model.Event{makeEvent("old", model.EventTypeCreate)}

	AppendEvent(makeEvent("a", model.EventTypeCreate))
	AppendEvent(makeEvent("old", model.EventTypeCreate))

	err := ArchiveEvents(active, archived)
	if err != nil {
		t.Fatal(err)
	}

	remaining, _ := ReadEvents()
	if len(remaining) != 1 || remaining[0].ID != "a" {
		t.Errorf("issues.db should have only active event, got %d events", len(remaining))
	}

	archivedEvts, _ := ReadArchivedEvents()
	if len(archivedEvts) != 1 || archivedEvts[0].ID != "old" {
		t.Errorf("archive.db should have archived event, got %d events", len(archivedEvts))
	}
}

func TestArchiveEvents_AppendsToExistingArchive(t *testing.T) {
	setupXpoDir(t)

	AppendEvent(makeEvent("a", model.EventTypeCreate))
	AppendEvent(makeEvent("b", model.EventTypeCreate))

	ArchiveEvents(
		[]model.Event{makeEvent("b", model.EventTypeCreate)},
		[]model.Event{makeEvent("a", model.EventTypeCreate)},
	)

	AppendEvent(makeEvent("c", model.EventTypeCreate))

	ArchiveEvents(
		[]model.Event{makeEvent("c", model.EventTypeCreate)},
		[]model.Event{makeEvent("b", model.EventTypeCreate)},
	)

	archivedEvts, _ := ReadArchivedEvents()
	if len(archivedEvts) != 2 {
		t.Errorf("archive should have 2 events from two rounds, got %d", len(archivedEvts))
	}
}

func TestArchiveEvents_EmptyArchiveList(t *testing.T) {
	setupXpoDir(t)
	AppendEvent(makeEvent("a", model.EventTypeCreate))

	err := ArchiveEvents(
		[]model.Event{makeEvent("a", model.EventTypeCreate)},
		[]model.Event{},
	)
	if err != nil {
		t.Fatal(err)
	}

	archivedEvts, _ := ReadArchivedEvents()
	if len(archivedEvts) != 0 {
		t.Errorf("archive should be empty, got %d events", len(archivedEvts))
	}
}

func TestArchiveEvents_EmptyActiveList(t *testing.T) {
	setupXpoDir(t)
	AppendEvent(makeEvent("a", model.EventTypeCreate))

	err := ArchiveEvents(
		[]model.Event{},
		[]model.Event{makeEvent("a", model.EventTypeCreate)},
	)
	if err != nil {
		t.Fatal(err)
	}

	remaining, _ := ReadEvents()
	if len(remaining) != 0 {
		t.Errorf("issues.db should be empty, got %d events", len(remaining))
	}
}
