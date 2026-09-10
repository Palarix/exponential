package storage

import (
	"testing"
	"time"

	"github.com/palarix/exponential/internal/model"
)

func TestReadEvents_Empty(t *testing.T) {
	setupXpoDir(t)

	events, err := ReadEvents()
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 0 {
		t.Errorf("expected 0 events from empty ref, got %d", len(events))
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

func TestReadArchivedEvents_Empty(t *testing.T) {
	setupXpoDir(t)

	events, err := ReadArchivedEvents()
	if err != nil {
		t.Fatal("missing archive should not error")
	}
	if len(events) != 0 {
		t.Errorf("expected 0, got %d", len(events))
	}
}

func strptr(s string) *string { return &s }
