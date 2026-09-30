package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/palarix/exponential/internal/model"
)

func TestReadEvents_EmptyFile(t *testing.T) {
	setupXpoDir(t)
	os.WriteFile(filepath.Join(".xpo", "issues.db"), []byte{}, 0644)

	events, err := ReadEvents()
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 0 {
		t.Errorf("expected 0 events from empty file, got %d", len(events))
	}
}

func TestReadEvents_FileNotFound(t *testing.T) {
	setupXpoDir(t)

	events, err := ReadEvents()
	if err != nil {
		t.Fatal("missing file should not error")
	}
	if len(events) != 0 {
		t.Errorf("expected 0 events when file missing, got %d", len(events))
	}
}

func TestReadEvents_MalformedJSON(t *testing.T) {
	setupXpoDir(t)
	os.WriteFile(filepath.Join(".xpo", "issues.db"), []byte("not json\n"), 0644)

	_, err := ReadEvents()
	if err == nil {
		t.Error("expected error for malformed JSON")
	}
}

func TestReadEvents_MultipleEvents(t *testing.T) {
	setupXpoDir(t)
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
	setupXpoDir(t)

	events, err := ReadArchivedEvents()
	if err != nil {
		t.Fatal("missing archive should not error")
	}
	if len(events) != 0 {
		t.Errorf("expected 0, got %d", len(events))
	}
}

func TestReadArchivedEvents_WithData(t *testing.T) {
	setupXpoDir(t)
	path := filepath.Join(".xpo", "archive.db")
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

func TestReadEvents_IgnoresTornFinalLine(t *testing.T) {
	setupXpoDir(t)
	AppendEvent(model.Event{ID: "a", Type: model.EventTypeCreate, Payload: model.CreatePayload{Title: "A"}, CreatedAt: time.Now().UTC(), CreatedBy: "t"})

	// Another process is halfway through appending the next event.
	f, _ := os.OpenFile(filepath.Join(".xpo", "issues.db"), os.O_APPEND|os.O_WRONLY, 0644)
	f.WriteString(`{"id":"b","type":"CRE`)
	f.Close()

	events, err := ReadEvents()
	if err != nil {
		t.Fatalf("torn final line should be ignored, got %v", err)
	}
	if len(events) != 1 || events[0].ID != "a" {
		t.Errorf("expected only event a, got %d events", len(events))
	}
}

func TestReadEvents_KeepsValidUnterminatedFinalLine(t *testing.T) {
	setupXpoDir(t)
	os.WriteFile(filepath.Join(".xpo", "issues.db"), []byte(`{"id":"a","type":"CREATE"}`), 0644)

	events, err := ReadEvents()
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Errorf("a complete final line without newline should still be read, got %d events", len(events))
	}
}

func TestReadEvents_ErrorsOnMalformedLineBeforeTail(t *testing.T) {
	setupXpoDir(t)
	os.WriteFile(filepath.Join(".xpo", "issues.db"), []byte("{\"id\":\"a\",\"ty\n{\"id\":\"b\",\"type\":\"CREATE\"}\n"), 0644)

	if _, err := ReadEvents(); err == nil {
		t.Error("a malformed line that is not the torn tail must still error")
	}
}
