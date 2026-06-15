package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/palarix/beats/internal/model"
)

func setupBeatsDir(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".beats"), 0755)
	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	t.Cleanup(func() { os.Chdir(oldWd) })
}

func TestAppendEvent_CreatesFile(t *testing.T) {
	setupBeatsDir(t)

	evt := model.Event{
		ID: "x", Type: model.EventTypeCreate,
		Payload:   model.CreatePayload{Title: "X"},
		CreatedAt: time.Now().UTC(), CreatedBy: "test",
	}
	if err := AppendEvent(evt); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(".beats", "issues.db"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"id":"x"`) {
		t.Error("event not found in file")
	}
}

func TestAppendEvent_AppendsToExisting(t *testing.T) {
	setupBeatsDir(t)

	AppendEvent(model.Event{ID: "a", Type: model.EventTypeCreate, Payload: model.CreatePayload{Title: "A"}, CreatedAt: time.Now().UTC(), CreatedBy: "test"})
	AppendEvent(model.Event{ID: "b", Type: model.EventTypeCreate, Payload: model.CreatePayload{Title: "B"}, CreatedAt: time.Now().UTC(), CreatedBy: "test"})

	events, err := ReadEvents()
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 {
		t.Errorf("expected 2 events, got %d", len(events))
	}
}

func TestAppendEvent_NewlineSeparated(t *testing.T) {
	setupBeatsDir(t)

	AppendEvent(model.Event{ID: "a", Type: model.EventTypeCreate, Payload: model.CreatePayload{Title: "A"}, CreatedAt: time.Now().UTC(), CreatedBy: "test"})
	AppendEvent(model.Event{ID: "b", Type: model.EventTypeCreate, Payload: model.CreatePayload{Title: "B"}, CreatedAt: time.Now().UTC(), CreatedBy: "test"})

	data, err := os.ReadFile(filepath.Join(".beats", "issues.db"))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 lines, got %d", len(lines))
	}
	for i, line := range lines {
		var evt model.Event
		if err := json.Unmarshal([]byte(line), &evt); err != nil {
			t.Errorf("line %d is not valid JSON: %v", i, err)
		}
	}
}

func TestAppendEvent_PreservesPayload(t *testing.T) {
	setupBeatsDir(t)

	AppendEvent(model.Event{
		ID: "x", Type: model.EventTypeCreate,
		Payload:   model.CreatePayload{Title: "Hello", Status: "PLANNED", Estimate: 5, Labels: []string{"bug"}},
		CreatedAt: time.Now().UTC(), CreatedBy: "test",
	})

	events, _ := ReadEvents()
	if len(events) != 1 {
		t.Fatal("expected 1 event")
	}
	b, _ := json.Marshal(events[0].Payload)
	var p model.CreatePayload
	json.Unmarshal(b, &p)
	if p.Title != "Hello" || p.Status != "PLANNED" || p.Estimate != 5 {
		t.Errorf("payload fields lost: %+v", p)
	}
}
