package storage

import (
	"testing"
	"time"

	"github.com/palarix/exponential/internal/model"
)

func TestProjectCommittedState_CreateAllFields(t *testing.T) {
	events := []model.Event{
		{ID: "x", Type: model.EventTypeCreate, Payload: model.CreatePayload{
			Title:       "Title",
			Description: "Desc",
			Status:      "PLANNED",
			ParentID:    "parent",
			Estimate:    5,
			Priority:    2,
			SortOrder:   "z1",
			Assignee:    "Alice <a@b.com>",
			CycleID:     "2026-06-01",
			Labels:      []string{"bug"},
			Dependencies: []model.Dependency{
				{SourceID: "x", TargetID: "y", Kind: "blocks"},
			},
		}, CreatedAt: time.Now().UTC()},
	}

	state := projectCommittedState(events)
	i := state["x"]
	if i == nil {
		t.Fatal("issue not in state")
	}
	if i.Title != "Title" {
		t.Errorf("Title = %q", i.Title)
	}
	if i.Description != "Desc" {
		t.Errorf("Description = %q", i.Description)
	}
	if i.Status != model.StatusPlanned {
		t.Errorf("Status = %s", i.Status)
	}
	if i.ParentID != "parent" {
		t.Errorf("ParentID = %q", i.ParentID)
	}
	if i.Estimate != 5 {
		t.Errorf("Estimate = %d", i.Estimate)
	}
	if i.Priority != 2 {
		t.Errorf("Priority = %d", i.Priority)
	}
	if i.SortOrder != "z1" {
		t.Errorf("SortOrder = %q", i.SortOrder)
	}
	if i.Assignee != "Alice <a@b.com>" {
		t.Errorf("Assignee = %q", i.Assignee)
	}
	if i.CycleID != "2026-06-01" {
		t.Errorf("CycleID = %q", i.CycleID)
	}
	if len(i.Labels) != 1 || i.Labels[0] != "bug" {
		t.Errorf("Labels = %v", i.Labels)
	}
	if len(i.Dependencies) != 1 || i.Dependencies[0].TargetID != "y" {
		t.Errorf("Dependencies = %v", i.Dependencies)
	}
}

func TestProjectCommittedState_DefaultBacklog(t *testing.T) {
	events := []model.Event{
		{ID: "x", Type: model.EventTypeCreate, Payload: model.CreatePayload{Title: "No status"}, CreatedAt: time.Now().UTC()},
	}
	state := projectCommittedState(events)
	if state["x"].Status != model.StatusBacklog {
		t.Errorf("empty status should default to BACKLOG, got %s", state["x"].Status)
	}
}

func TestProjectCommittedState_UpdateApplied(t *testing.T) {
	events := []model.Event{
		{ID: "x", Type: model.EventTypeCreate, Payload: model.CreatePayload{Title: "X", Status: "BACKLOG"}, CreatedAt: time.Now().UTC()},
		{ID: "x", Type: model.EventTypeUpdate, Payload: model.UpdatePayload{
			Title:       strptr("Y"),
			Description: strptr("desc"),
			Status:      strptr("DOING"),
			ParentID:    strptr("p"),
			Estimate:    intptr(3),
			Priority:    intptr(1),
			SortOrder:   strptr("a1"),
			Assignee:    strptr("Bob"),
			CycleID:     strptr("c1"),
			Labels:      []string{"feat"},
			Dependencies: []model.Dependency{{SourceID: "x", TargetID: "z", Kind: "depends_on"}},
		}, CreatedAt: time.Now().UTC()},
	}
	state := projectCommittedState(events)
	i := state["x"]
	if i.Title != "Y" {
		t.Errorf("Title not updated: %q", i.Title)
	}
	if i.Status != model.StatusDoing {
		t.Errorf("Status not updated: %s", i.Status)
	}
	if i.Assignee != "Bob" {
		t.Errorf("Assignee not updated: %q", i.Assignee)
	}
	if len(i.Labels) != 1 || i.Labels[0] != "feat" {
		t.Errorf("Labels not updated: %v", i.Labels)
	}
}

func TestProjectCommittedState_DeleteMarksDeleted(t *testing.T) {
	events := []model.Event{
		{ID: "x", Type: model.EventTypeCreate, Payload: model.CreatePayload{Title: "X"}, CreatedAt: time.Now().UTC()},
		{ID: "x", Type: model.EventTypeDelete, Payload: model.DeletePayload{Reason: "test"}, CreatedAt: time.Now().UTC()},
	}
	state := projectCommittedState(events)
	if !state["x"].Deleted {
		t.Error("deleted issue should be marked Deleted")
	}
}

func TestProjectCommittedState_UpdateOnMissing_Ignored(t *testing.T) {
	events := []model.Event{
		{ID: "ghost", Type: model.EventTypeUpdate, Payload: model.UpdatePayload{Status: strptr("DOING")}, CreatedAt: time.Now().UTC()},
	}
	state := projectCommittedState(events)
	if _, exists := state["ghost"]; exists {
		t.Error("update on nonexistent issue should not create it")
	}
}

func TestProjectCommittedState_Empty(t *testing.T) {
	state := projectCommittedState(nil)
	if len(state) != 0 {
		t.Errorf("empty events should produce empty state, got %d", len(state))
	}
}
