package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/palarix/beats/internal/model"
)

func TestReadEvents_EmptyFile(t *testing.T) {
	setupBeatsDir(t)
	os.WriteFile(filepath.Join(".beats", "issues.db"), []byte{}, 0644)

	events, err := ReadEvents()
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 0 {
		t.Errorf("expected 0 events from empty file, got %d", len(events))
	}
}

func TestReadEvents_FileNotFound(t *testing.T) {
	setupBeatsDir(t)

	events, err := ReadEvents()
	if err != nil {
		t.Fatal("missing file should not error")
	}
	if len(events) != 0 {
		t.Errorf("expected 0 events when file missing, got %d", len(events))
	}
}

func TestReadEvents_MalformedJSON(t *testing.T) {
	setupBeatsDir(t)
	os.WriteFile(filepath.Join(".beats", "issues.db"), []byte("not json\n"), 0644)

	_, err := ReadEvents()
	if err == nil {
		t.Error("expected error for malformed JSON")
	}
}

func TestReadEvents_MultipleEvents(t *testing.T) {
	setupBeatsDir(t)
	now := time.Now().UTC()
	AppendEvent(model.Event{ID: "a", Type: model.EventTypeCreate, Payload: model.CreatePayload{Title: "A"}, CreatedAt: now, CreatedBy: "t"})
	AppendEvent(model.Event{ID: "b", Type: model.EventTypeCreate, Payload: model.CreatePayload{Title: "B"}, CreatedAt: now, CreatedBy: "t"})
	AppendEvent(model.Event{ID: "a", Type: model.EventTypeUpdate, Payload: model.UpdatePayload{Status: strptr("DOING")}, CreatedAt: now, CreatedBy: "t"})

	events, err := ReadEvents()
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 3 {
		t.Errorf("expected 3 events, got %d", len(events))
	}
}

func TestReadArchivedEvents_FileNotFound(t *testing.T) {
	setupBeatsDir(t)

	events, err := ReadArchivedEvents()
	if err != nil {
		t.Fatal("missing archive should not error")
	}
	if len(events) != 0 {
		t.Errorf("expected 0, got %d", len(events))
	}
}

func TestReadArchivedEvents_WithData(t *testing.T) {
	setupBeatsDir(t)
	path := filepath.Join(".beats", "archive.db")
	f, _ := os.Create(path)
	for _, id := range []string{"x", "y"} {
		evt := model.Event{ID: id, Type: model.EventTypeCreate, Payload: model.CreatePayload{Title: id}, CreatedAt: time.Now().UTC(), CreatedBy: "t"}
		b, _ := json.Marshal(evt)
		f.Write(b)
		f.WriteString("\n")
	}
	f.Close()

	events, err := ReadArchivedEvents()
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 {
		t.Errorf("expected 2 archived events, got %d", len(events))
	}
}
