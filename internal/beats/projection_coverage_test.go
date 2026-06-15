package beats

import (
	"testing"
	"time"

	"github.com/palarix/beats/internal/model"
)

func TestProjection_CreateWithStatus(t *testing.T) {
	events := []model.Event{
		{ID: "a", Type: model.EventTypeCreate, Payload: model.CreatePayload{Title: "A", Status: "PLANNED"}},
	}
	issues := ProjectIssues(events)
	if issues["a"].Status != model.StatusPlanned {
		t.Errorf("projected status = %s, want PLANNED", issues["a"].Status)
	}
}

func TestProjection_CreateDefaultsToBacklog(t *testing.T) {
	events := []model.Event{
		{ID: "a", Type: model.EventTypeCreate, Payload: model.CreatePayload{Title: "A"}},
	}
	issues := ProjectIssues(events)
	if issues["a"].Status != model.StatusBacklog {
		t.Errorf("projected status = %s, want BACKLOG", issues["a"].Status)
	}
}

func TestProjection_UpdateOnMissingIssue(t *testing.T) {
	events := []model.Event{
		{ID: "ghost", Type: model.EventTypeUpdate, Payload: model.UpdatePayload{Status: sp("DOING")}},
	}
	issues := ProjectIssues(events)
	if _, exists := issues["ghost"]; exists {
		t.Error("update on nonexistent issue should not create it")
	}
}

func TestProjection_DeleteRemovesFromMap(t *testing.T) {
	events := []model.Event{
		{ID: "a", Type: model.EventTypeCreate, Payload: model.CreatePayload{Title: "A"}},
		{ID: "a", Type: model.EventTypeDelete, Payload: model.DeletePayload{Reason: "test"}},
	}
	issues := ProjectIssues(events)
	if _, exists := issues["a"]; exists {
		t.Error("deleted issue should be removed from projected map")
	}
}

func TestProjection_CommentAppended(t *testing.T) {
	events := []model.Event{
		{ID: "a", Type: model.EventTypeCreate, Payload: model.CreatePayload{Title: "A"}},
		{ID: "a", Type: model.EventTypeComment, Payload: model.CommentPayload{ID: "c1", Text: "hello"}, CreatedBy: "bob"},
	}
	issues := ProjectIssues(events)
	if len(issues["a"].Comments) != 1 || issues["a"].Comments[0].Text != "hello" {
		t.Errorf("comment not projected correctly: %+v", issues["a"].Comments)
	}
}

func TestProjection_CommentOnMissingIssue(t *testing.T) {
	events := []model.Event{
		{ID: "ghost", Type: model.EventTypeComment, Payload: model.CommentPayload{ID: "c1", Text: "hello"}},
	}
	issues := ProjectIssues(events)
	if _, exists := issues["ghost"]; exists {
		t.Error("comment on nonexistent issue should not create it")
	}
}

func TestProjection_AllUpdateFields(t *testing.T) {
	events := []model.Event{
		{ID: "a", Type: model.EventTypeCreate, Payload: model.CreatePayload{Title: "A"}, CreatedAt: time.Now()},
		{ID: "a", Type: model.EventTypeUpdate, Payload: model.UpdatePayload{
			Title:       sp("B"),
			Description: sp("desc"),
			Status:      sp("DOING"),
			ParentID:    sp("parent"),
			Estimate:    ip(5),
			Priority:    ip(2),
			SortOrder:   sp("z1"),
			Assignee:    sp("Alice <a@b.com>"),
			CycleID:     sp("2026-06-01"),
			Labels:      []string{"bug"},
			Dependencies: []model.Dependency{
				{SourceID: "a", TargetID: "b", Kind: "depends_on"},
			},
		}, CreatedAt: time.Now()},
	}
	issues := ProjectIssues(events)
	i := issues["a"]
	if i.Title != "B" {
		t.Errorf("Title = %q", i.Title)
	}
	if i.Description != "desc" {
		t.Errorf("Description = %q", i.Description)
	}
	if i.Status != model.StatusDoing {
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
	if len(i.Dependencies) != 1 || i.Dependencies[0].TargetID != "b" {
		t.Errorf("Dependencies = %v", i.Dependencies)
	}
}

func TestProjection_UpdatedAtTracked(t *testing.T) {
	t1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	events := []model.Event{
		{ID: "a", Type: model.EventTypeCreate, Payload: model.CreatePayload{Title: "A"}, CreatedAt: t1},
		{ID: "a", Type: model.EventTypeUpdate, Payload: model.UpdatePayload{Status: sp("DOING")}, CreatedAt: t2},
	}
	issues := ProjectIssues(events)
	if !issues["a"].UpdatedAt.Equal(t2) {
		t.Errorf("UpdatedAt = %v, want %v", issues["a"].UpdatedAt, t2)
	}
}

func TestProjection_MultipleIssuesIndependent(t *testing.T) {
	events := []model.Event{
		{ID: "a", Type: model.EventTypeCreate, Payload: model.CreatePayload{Title: "A"}},
		{ID: "b", Type: model.EventTypeCreate, Payload: model.CreatePayload{Title: "B", Status: "DOING"}},
		{ID: "a", Type: model.EventTypeUpdate, Payload: model.UpdatePayload{Status: sp("PLANNED")}},
	}
	issues := ProjectIssues(events)
	if issues["a"].Status != model.StatusPlanned {
		t.Errorf("a.Status = %s, want PLANNED", issues["a"].Status)
	}
	if issues["b"].Status != model.StatusDoing {
		t.Errorf("b.Status = %s, want DOING (should not be affected by a's update)", issues["b"].Status)
	}
}
