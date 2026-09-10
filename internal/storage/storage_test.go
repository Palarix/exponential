package storage

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/palarix/exponential/internal/model"
)

func setupXpoDir(t *testing.T) {
	t.Helper()
	setupRefStore(t)
}

func setupRefStore(t *testing.T) {
	t.Helper()
	dir := initGitRepo(t)
	chdir(t, dir)
	ResetHubRoot()
	ResetRefStore()
	t.Cleanup(func() {
		ResetHubRoot()
		ResetRefStore()
	})
	if err := InitRefStore(); err != nil {
		t.Fatalf("InitRefStore: %v", err)
	}
}

func chdir(t *testing.T, dir string) {
	t.Helper()
	prev, _ := os.Getwd()
	os.Chdir(dir)
	t.Cleanup(func() { os.Chdir(prev) })
}

func TestAppendEvent_WritesToRef(t *testing.T) {
	setupXpoDir(t)

	evt := model.Event{
		ID: "x", Type: model.EventTypeCreate,
		Payload:   model.CreatePayload{Title: "X"},
		CreatedAt: time.Now().UTC(), CreatedBy: "test",
	}
	if err := AppendEvent(evt); err != nil {
		t.Fatal(err)
	}

	events, err := ReadEvents()
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].ID != "x" {
		t.Errorf("expected 1 event with ID 'x', got %d events", len(events))
	}
}

func TestAppendEvent_AppendsToExisting(t *testing.T) {
	setupXpoDir(t)

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

func TestAppendEvent_PreservesPayload(t *testing.T) {
	setupXpoDir(t)

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
