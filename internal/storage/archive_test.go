package storage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/palarix/beats/internal/model"
)

func makeEvent(id string, evtType model.EventType) model.Event {
	return model.Event{ID: id, Type: evtType, Payload: model.CreatePayload{Title: id}, CreatedAt: time.Now().UTC(), CreatedBy: "test"}
}

func TestArchiveEvents_MovesToArchive(t *testing.T) {
	setupBeatsDir(t)

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

func TestArchiveEvents_CreatesBackup(t *testing.T) {
	setupBeatsDir(t)

	AppendEvent(makeEvent("a", model.EventTypeCreate))

	ArchiveEvents(
		[]model.Event{makeEvent("a", model.EventTypeCreate)},
		[]model.Event{},
	)

	entries, _ := os.ReadDir(".beats")
	backupFound := false
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".bak") {
			backupFound = true
		}
	}
	if !backupFound {
		t.Error("backup file not created")
	}
}

func TestArchiveEvents_AppendsToExistingArchive(t *testing.T) {
	setupBeatsDir(t)

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
	setupBeatsDir(t)
	AppendEvent(makeEvent("a", model.EventTypeCreate))

	err := ArchiveEvents(
		[]model.Event{makeEvent("a", model.EventTypeCreate)},
		[]model.Event{},
	)
	if err != nil {
		t.Fatal(err)
	}

	archPath := filepath.Join(".beats", "archive.db")
	info, err := os.Stat(archPath)
	if err != nil {
		t.Fatal("archive.db should be created even if empty archive list")
	}
	if info.Size() != 0 {
		t.Error("archive.db should be empty when no events archived")
	}
}

func TestArchiveEvents_EmptyActiveList(t *testing.T) {
	setupBeatsDir(t)
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
