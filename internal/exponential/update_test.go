package exponential

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/palarix/exponential/internal/model"
	"github.com/palarix/exponential/internal/storage"
)

func TestUpdateIssue_StatusChange(t *testing.T) {
	tr := setupLocalTransport(t)
	issue, _ := tr.AddIssue(model.CreatePayload{Title: "updatable"})

	_, err := tr.UpdateIssue(issue.ID, model.UpdatePayload{Status: sp("DOING")}, "update")
	if err != nil {
		t.Fatal(err)
	}

	issues := readAllIssues(t)
	if issues[issue.ID].Status != model.StatusDoing {
		t.Errorf("status = %s, want DOING", issues[issue.ID].Status)
	}
}

func TestUpdateIssue_BlockedByDependency(t *testing.T) {
	tr := setupLocalTransport(t)
	blocker, _ := tr.AddIssue(model.CreatePayload{Title: "blocker"})
	blocked, _ := tr.AddIssue(model.CreatePayload{
		Title: "blocked",
		Dependencies: []model.Dependency{
			{TargetID: blocker.ID, Kind: "blocked_by"},
		},
	})

	_, err := tr.UpdateIssue(blocked.ID, model.UpdatePayload{Status: sp("DOING")}, "update")
	if err == nil {
		t.Error("expected error when starting issue blocked by incomplete blocker")
	}
	if !strings.Contains(err.Error(), "blocked by") {
		t.Errorf("error should mention blocked_by, got: %s", err)
	}
}

func TestUpdateIssue_BlockedByResolved(t *testing.T) {
	tr := setupLocalTransport(t)
	blocker, _ := tr.AddIssue(model.CreatePayload{Title: "blocker"})
	blocked, _ := tr.AddIssue(model.CreatePayload{
		Title: "blocked",
		Dependencies: []model.Dependency{
			{TargetID: blocker.ID, Kind: "blocked_by"},
		},
	})

	tr.UpdateIssue(blocker.ID, model.UpdatePayload{Status: sp("DONE")}, "update")

	_, err := tr.UpdateIssue(blocked.ID, model.UpdatePayload{Status: sp("DOING")}, "update")
	if err != nil {
		t.Errorf("should allow DOING after blocker is DONE, got: %s", err)
	}
}

func TestUpdateIssue_ParentCanCompleteWithIncompleteChildren(t *testing.T) {
	tr := setupLocalTransport(t)
	parent, _ := tr.AddIssue(model.CreatePayload{Title: "parent"})
	child, _ := tr.AddIssue(model.CreatePayload{Title: "child", ParentID: parent.ID})

	_, err := tr.UpdateIssue(parent.ID, model.UpdatePayload{Status: sp("DONE")}, "update")
	if err != nil {
		t.Fatalf("completing parent with incomplete children should succeed, got: %v", err)
	}

	issues := readAllIssues(t)
	if issues[parent.ID].Status != model.StatusDone {
		t.Errorf("parent status = %s, want DONE", issues[parent.ID].Status)
	}
	if issues[child.ID].Status != model.StatusBacklog {
		t.Errorf("child status = %s, want BACKLOG (should not be affected)", issues[child.ID].Status)
	}
}

func TestUpdateIssue_FirstStartTrigger(t *testing.T) {
	tr := setupLocalTransport(t)
	tr.Config.Automations.FirstStart = true
	parent, _ := tr.AddIssue(model.CreatePayload{Title: "epic"})
	child, _ := tr.AddIssue(model.CreatePayload{Title: "story", ParentID: parent.ID})

	msgs, err := tr.UpdateIssue(child.ID, model.UpdatePayload{Status: sp("DOING")}, "start")
	if err != nil {
		t.Fatal(err)
	}

	issues := readAllIssues(t)
	if issues[parent.ID].Status != model.StatusDoing {
		t.Errorf("parent status = %s, want DOING (first-start trigger)", issues[parent.ID].Status)
	}

	autoStarted := false
	for _, m := range msgs {
		if strings.Contains(m, "Auto-started parent") {
			autoStarted = true
		}
	}
	if !autoStarted {
		t.Errorf("expected auto-start message, got: %v", msgs)
	}
}

func TestUpdateIssue_FirstStartTriggerOff(t *testing.T) {
	tr := setupLocalTransport(t)
	parent, _ := tr.AddIssue(model.CreatePayload{Title: "epic"})
	tr.AddIssue(model.CreatePayload{Title: "story", ParentID: parent.ID, Status: "PLANNED"})

	child2, _ := tr.AddIssue(model.CreatePayload{Title: "story2", ParentID: parent.ID})
	tr.UpdateIssue(child2.ID, model.UpdatePayload{Status: sp("DOING")}, "start")

	issues := readAllIssues(t)
	if issues[parent.ID].Status != model.StatusBacklog {
		t.Errorf("parent status = %s, want BACKLOG (first_start disabled)", issues[parent.ID].Status)
	}
}

func TestUpdateIssue_FirstStartSkipsDoingParent(t *testing.T) {
	tr := setupLocalTransport(t)
	tr.Config.Automations.FirstStart = true
	parent, _ := tr.AddIssue(model.CreatePayload{Title: "epic", Status: "DOING"})
	child, _ := tr.AddIssue(model.CreatePayload{Title: "story", ParentID: parent.ID})

	tr.UpdateIssue(child.ID, model.UpdatePayload{Status: sp("DOING")}, "start")

	issues := readAllIssues(t)
	if issues[parent.ID].Status != model.StatusDoing {
		t.Errorf("parent status = %s, want DOING (already in progress)", issues[parent.ID].Status)
	}
}

func TestUpdateIssue_FirstStartSkipsBlocked(t *testing.T) {
	tr := setupLocalTransport(t)
	tr.Config.Automations.FirstStart = true
	parent, _ := tr.AddIssue(model.CreatePayload{Title: "epic", Status: "PLANNED"})
	child, _ := tr.AddIssue(model.CreatePayload{Title: "story", ParentID: parent.ID, Status: "PLANNED"})

	_, err := tr.UpdateIssue(child.ID, model.UpdatePayload{Status: sp("BLOCKED")}, "block")
	if err != nil {
		t.Fatal(err)
	}

	issues := readAllIssues(t)
	if issues[parent.ID].Status != model.StatusPlanned {
		t.Errorf("parent status = %s, want PLANNED (blocked child should not auto-start parent)", issues[parent.ID].Status)
	}
}

func TestUpdateIssue_LastCompletedTrigger(t *testing.T) {
	tr := setupLocalTransport(t)
	tr.Config.Automations.LastCompleted = true
	parent, _ := tr.AddIssue(model.CreatePayload{Title: "epic", Status: "DOING"})
	child1, _ := tr.AddIssue(model.CreatePayload{Title: "story1", ParentID: parent.ID, Status: "DONE"})
	_ = child1
	child2, _ := tr.AddIssue(model.CreatePayload{Title: "story2", ParentID: parent.ID, Status: "DOING"})

	msgs, err := tr.UpdateIssue(child2.ID, model.UpdatePayload{Status: sp("DONE")}, "done")
	if err != nil {
		t.Fatal(err)
	}

	issues := readAllIssues(t)
	if issues[parent.ID].Status != model.StatusDone {
		t.Errorf("parent status = %s, want DONE (last-completed trigger)", issues[parent.ID].Status)
	}

	autoCompleted := false
	for _, m := range msgs {
		if strings.Contains(m, "Auto-completed parent") {
			autoCompleted = true
		}
	}
	if !autoCompleted {
		t.Errorf("expected auto-completed message, got: %v", msgs)
	}
}

func TestUpdateIssue_LastCompletedNotLastChild(t *testing.T) {
	tr := setupLocalTransport(t)
	tr.Config.Automations.LastCompleted = true
	parent, _ := tr.AddIssue(model.CreatePayload{Title: "epic", Status: "DOING"})
	child1, _ := tr.AddIssue(model.CreatePayload{Title: "story1", ParentID: parent.ID, Status: "DOING"})
	tr.AddIssue(model.CreatePayload{Title: "story2", ParentID: parent.ID, Status: "PLANNED"})

	tr.UpdateIssue(child1.ID, model.UpdatePayload{Status: sp("DONE")}, "done")

	issues := readAllIssues(t)
	if issues[parent.ID].Status != model.StatusDoing {
		t.Errorf("parent status = %s, want DOING (not all children done)", issues[parent.ID].Status)
	}
}

func TestUpdateIssue_LastCompletedOff(t *testing.T) {
	tr := setupLocalTransport(t)
	parent, _ := tr.AddIssue(model.CreatePayload{Title: "epic", Status: "DOING"})
	child, _ := tr.AddIssue(model.CreatePayload{Title: "story", ParentID: parent.ID, Status: "DOING"})

	tr.UpdateIssue(child.ID, model.UpdatePayload{Status: sp("DONE")}, "done")

	issues := readAllIssues(t)
	if issues[parent.ID].Status != model.StatusDoing {
		t.Errorf("parent status = %s, want DOING (last_completed disabled)", issues[parent.ID].Status)
	}
}

func TestUpdateIssue_LastCompletedFiresForPlannedParent(t *testing.T) {
	tr := setupLocalTransport(t)
	tr.Config.Automations.LastCompleted = true
	parent, _ := tr.AddIssue(model.CreatePayload{Title: "epic", Status: "PLANNED"})
	child, _ := tr.AddIssue(model.CreatePayload{Title: "story", ParentID: parent.ID, Status: "DOING"})

	tr.UpdateIssue(child.ID, model.UpdatePayload{Status: sp("DONE")}, "done")

	issues := readAllIssues(t)
	if issues[parent.ID].Status != model.StatusDone {
		t.Errorf("parent status = %s, want DONE (last child completed)", issues[parent.ID].Status)
	}
}

func TestUpdateIssue_NonStatusFields(t *testing.T) {
	tr := setupLocalTransport(t)
	issue, _ := tr.AddIssue(model.CreatePayload{Title: "original", Estimate: 3})

	tr.UpdateIssue(issue.ID, model.UpdatePayload{
		Title:       sp("renamed"),
		Description: sp("new desc"),
		Estimate:    ip(5),
		Priority:    ip(2),
		Assignee:    sp("Alice <alice@test.com>"),
		Labels:      []string{"bug", "feature"},
	}, "update")

	issues := readAllIssues(t)
	i := issues[issue.ID]
	if i.Title != "renamed" {
		t.Errorf("title = %q", i.Title)
	}
	if i.Description != "new desc" {
		t.Errorf("description = %q", i.Description)
	}
	if i.Estimate != 5 {
		t.Errorf("estimate = %d", i.Estimate)
	}
	if i.Priority != 2 {
		t.Errorf("priority = %d", i.Priority)
	}
	if i.Assignee != "Alice <alice@test.com>" {
		t.Errorf("assignee = %q", i.Assignee)
	}
	if len(i.Labels) != 2 {
		t.Errorf("labels = %v", i.Labels)
	}
}

func TestUpdateIssue_IssueNotFound(t *testing.T) {
	tr := setupLocalTransport(t)
	_, err := tr.UpdateIssue("nonexistent", model.UpdatePayload{Status: sp("DOING")}, "update")
	if err == nil {
		t.Error("expected error for nonexistent issue")
	}
}

func TestUpdateIssue_RedundantStatusIsNoop(t *testing.T) {
	tr := setupLocalTransport(t)
	issue, _ := tr.AddIssue(model.CreatePayload{Title: "test", Status: "DOING"})

	eventsBefore := countEvents(t, issue.ID)
	msgs, err := tr.UpdateIssue(issue.ID, model.UpdatePayload{Status: sp("DOING")}, "update")
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 0 {
		t.Errorf("expected no messages for noop, got %v", msgs)
	}
	eventsAfter := countEvents(t, issue.ID)
	if eventsAfter != eventsBefore {
		t.Errorf("event count changed: %d → %d (expected no new events)", eventsBefore, eventsAfter)
	}
}

func TestUpdateIssue_RedundantPriorityIsNoop(t *testing.T) {
	tr := setupLocalTransport(t)
	issue, _ := tr.AddIssue(model.CreatePayload{Title: "test", Priority: 2})

	eventsBefore := countEvents(t, issue.ID)
	msgs, err := tr.UpdateIssue(issue.ID, model.UpdatePayload{Priority: ip(2)}, "update")
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 0 {
		t.Errorf("expected no messages for noop, got %v", msgs)
	}
	eventsAfter := countEvents(t, issue.ID)
	if eventsAfter != eventsBefore {
		t.Errorf("event count changed: %d → %d", eventsBefore, eventsAfter)
	}
}

func TestUpdateIssue_RedundantLabelsIsNoop(t *testing.T) {
	tr := setupLocalTransport(t)
	issue, _ := tr.AddIssue(model.CreatePayload{Title: "test", Labels: []string{"bug", "feature"}})

	eventsBefore := countEvents(t, issue.ID)
	// Different order, same set
	msgs, err := tr.UpdateIssue(issue.ID, model.UpdatePayload{Labels: []string{"feature", "bug"}}, "update")
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 0 {
		t.Errorf("expected no messages for noop, got %v", msgs)
	}
	eventsAfter := countEvents(t, issue.ID)
	if eventsAfter != eventsBefore {
		t.Errorf("event count changed: %d → %d", eventsBefore, eventsAfter)
	}
}

func TestUpdateIssue_MixedRedundantAndNewFields(t *testing.T) {
	tr := setupLocalTransport(t)
	issue, _ := tr.AddIssue(model.CreatePayload{Title: "test", Priority: 2})

	msgs, err := tr.UpdateIssue(issue.ID, model.UpdatePayload{
		Priority: ip(2),   // redundant
		Title:    sp("new"), // actual change
	}, "update")
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) == 0 {
		t.Error("expected messages for actual change")
	}
	issues := readAllIssues(t)
	if issues[issue.ID].Title != "new" {
		t.Errorf("title = %q, want %q", issues[issue.ID].Title, "new")
	}
}

func TestUpdateIssue_BlockerCanceled_Unblocks(t *testing.T) {
	tr := setupLocalTransport(t)
	blocker, _ := tr.AddIssue(model.CreatePayload{Title: "blocker"})
	blocked, _ := tr.AddIssue(model.CreatePayload{
		Title: "blocked",
		Dependencies: []model.Dependency{
			{TargetID: blocker.ID, Kind: "blocked_by"},
		},
	})

	tr.UpdateIssue(blocker.ID, model.UpdatePayload{Status: sp("CANCELED")}, "update")

	_, err := tr.UpdateIssue(blocked.ID, model.UpdatePayload{Status: sp("DOING")}, "update")
	if err != nil {
		t.Errorf("should allow DOING after blocker is CANCELED, got: %s", err)
	}
}

func TestUpdateIssue_BlockerDuplicate_Unblocks(t *testing.T) {
	tr := setupLocalTransport(t)
	blocker, _ := tr.AddIssue(model.CreatePayload{Title: "blocker"})
	blocked, _ := tr.AddIssue(model.CreatePayload{
		Title: "blocked",
		Dependencies: []model.Dependency{
			{TargetID: blocker.ID, Kind: "blocked_by"},
		},
	})

	tr.UpdateIssue(blocker.ID, model.UpdatePayload{Status: sp("DUPLICATE")}, "update")

	_, err := tr.UpdateIssue(blocked.ID, model.UpdatePayload{Status: sp("DOING")}, "update")
	if err != nil {
		t.Errorf("should allow DOING after blocker is DUPLICATE, got: %s", err)
	}
}

func TestUpdateIssue_LastCompletedWithCanceledChild(t *testing.T) {
	tr := setupLocalTransport(t)
	tr.Config.Automations.LastCompleted = true
	parent, _ := tr.AddIssue(model.CreatePayload{Title: "epic", Status: "DOING"})
	child1, _ := tr.AddIssue(model.CreatePayload{Title: "story1", ParentID: parent.ID, Status: "DONE"})
	_ = child1
	child2, _ := tr.AddIssue(model.CreatePayload{Title: "story2", ParentID: parent.ID, Status: "DOING"})

	msgs, err := tr.UpdateIssue(child2.ID, model.UpdatePayload{Status: sp("CANCELED")}, "cancel")
	if err != nil {
		t.Fatal(err)
	}

	issues := readAllIssues(t)
	if issues[parent.ID].Status != model.StatusDone {
		t.Errorf("parent status = %s, want DONE (all children terminal)", issues[parent.ID].Status)
	}

	autoCompleted := false
	for _, m := range msgs {
		if strings.Contains(m, "Auto-completed parent") {
			autoCompleted = true
		}
	}
	if !autoCompleted {
		t.Errorf("expected auto-completed message, got: %v", msgs)
	}
}

func TestUpdateIssue_LastCompletedMixedTerminal(t *testing.T) {
	tr := setupLocalTransport(t)
	tr.Config.Automations.LastCompleted = true
	parent, _ := tr.AddIssue(model.CreatePayload{Title: "epic", Status: "DOING"})
	tr.AddIssue(model.CreatePayload{Title: "story1", ParentID: parent.ID, Status: "DONE"})
	tr.AddIssue(model.CreatePayload{Title: "story2", ParentID: parent.ID, Status: "CANCELED"})
	child3, _ := tr.AddIssue(model.CreatePayload{Title: "story3", ParentID: parent.ID, Status: "DOING"})

	tr.UpdateIssue(child3.ID, model.UpdatePayload{Status: sp("DUPLICATE")}, "duplicate")

	issues := readAllIssues(t)
	if issues[parent.ID].Status != model.StatusDone {
		t.Errorf("parent status = %s, want DONE (all children terminal: DONE+CANCELED+DUPLICATE)", issues[parent.ID].Status)
	}
}

func TestUpdateIssue_LastCompletedNotTriggeredByActiveChild(t *testing.T) {
	tr := setupLocalTransport(t)
	tr.Config.Automations.LastCompleted = true
	parent, _ := tr.AddIssue(model.CreatePayload{Title: "epic", Status: "DOING"})
	tr.AddIssue(model.CreatePayload{Title: "story1", ParentID: parent.ID, Status: "DONE"})
	tr.AddIssue(model.CreatePayload{Title: "story2", ParentID: parent.ID, Status: "CANCELED"})
	child3, _ := tr.AddIssue(model.CreatePayload{Title: "story3", ParentID: parent.ID, Status: "DOING"})

	tr.UpdateIssue(child3.ID, model.UpdatePayload{Status: sp("PLANNED")}, "update")

	issues := readAllIssues(t)
	if issues[parent.ID].Status != model.StatusDoing {
		t.Errorf("parent status = %s, want DOING (child3 is PLANNED, not terminal)", issues[parent.ID].Status)
	}
}

func TestUpdateIssue_TerminalToActive(t *testing.T) {
	tr := setupLocalTransport(t)
	issue, _ := tr.AddIssue(model.CreatePayload{Title: "reopenable", Status: "CANCELED"})

	_, err := tr.UpdateIssue(issue.ID, model.UpdatePayload{Status: sp("PLANNED")}, "reopen")
	if err != nil {
		t.Fatalf("should allow reopening CANCELED issue, got: %v", err)
	}

	issues := readAllIssues(t)
	if issues[issue.ID].Status != model.StatusPlanned {
		t.Errorf("status = %s, want PLANNED", issues[issue.ID].Status)
	}
}

func TestUpdateIssue_StatusChange_AssignsSortOrder(t *testing.T) {
	tr := setupLocalTransport(t)

	// Create two issues already in PLANNED with known sort orders
	i1, _ := tr.AddIssue(model.CreatePayload{Title: "first", Status: "PLANNED"})
	tr.UpdateIssue(i1.ID, model.UpdatePayload{SortOrder: sp("a0")}, "sort")
	i2, _ := tr.AddIssue(model.CreatePayload{Title: "second", Status: "PLANNED"})
	tr.UpdateIssue(i2.ID, model.UpdatePayload{SortOrder: sp("a1")}, "sort")

	// Create a third issue in BACKLOG, then move it to PLANNED
	i3, _ := tr.AddIssue(model.CreatePayload{Title: "newcomer", Status: "BACKLOG"})
	_, err := tr.UpdateIssue(i3.ID, model.UpdatePayload{Status: sp("PLANNED")}, "start")
	if err != nil {
		t.Fatal(err)
	}

	issues := readAllIssues(t)
	got := issues[i3.ID].SortOrder
	if got == "" {
		t.Fatal("expected non-empty SortOrder after status transition")
	}
	if got <= "a1" {
		t.Errorf("expected SortOrder > 'a1' (after existing keys), got %q", got)
	}
}

func countEvents(t *testing.T, id string) int {
	t.Helper()
	issues := readAllIssues(t)
	issue, ok := issues[id]
	if !ok {
		t.Fatalf("issue %s not found", id)
	}
	return len(issue.Events)
}

func TestUpdateIssue_DependenciesGetSourceID(t *testing.T) {
	tr := setupLocalTransport(t)
	a, _ := tr.AddIssue(model.CreatePayload{Title: "a"})
	b, _ := tr.AddIssue(model.CreatePayload{Title: "b"})

	_, err := tr.UpdateIssue(a.ID, model.UpdatePayload{
		Dependencies: []model.Dependency{{TargetID: b.ID, Kind: model.DependencyDependsOn}},
	}, "update")
	if err != nil {
		t.Fatal(err)
	}

	events, _ := storage.ReadEvents()
	for _, evt := range events {
		if evt.Type != model.EventTypeUpdate || evt.ID != a.ID {
			continue
		}
		raw, _ := json.Marshal(evt.Payload)
		if strings.Contains(string(raw), `"source_id":""`) {
			t.Errorf("stored update event has empty source_id: %s", raw)
		}
	}
	deps := readAllIssues(t)[a.ID].Dependencies
	if len(deps) != 1 || deps[0].SourceID != a.ID {
		t.Errorf("deps = %+v, want one with SourceID %s", deps, a.ID)
	}
}

func TestUpdateIssue_EmptyDependenciesClears(t *testing.T) {
	tr := setupLocalTransport(t)
	b, _ := tr.AddIssue(model.CreatePayload{Title: "b"})
	a, _ := tr.AddIssue(model.CreatePayload{
		Title:        "a",
		Dependencies: []model.Dependency{{TargetID: b.ID, Kind: model.DependencyDependsOn}},
	})

	if _, err := tr.UpdateIssue(a.ID, model.UpdatePayload{Dependencies: []model.Dependency{}}, "update"); err != nil {
		t.Fatal(err)
	}
	if deps := readAllIssues(t)[a.ID].Dependencies; len(deps) != 0 {
		t.Errorf("expected deps cleared, got %+v", deps)
	}
}

func TestUpdateIssue_EmptyLabelsClears(t *testing.T) {
	tr := setupLocalTransport(t)
	a, _ := tr.AddIssue(model.CreatePayload{Title: "a", Labels: []string{"bug"}})

	if _, err := tr.UpdateIssue(a.ID, model.UpdatePayload{Labels: []string{}}, "update"); err != nil {
		t.Fatal(err)
	}
	if labels := readAllIssues(t)[a.ID].Labels; len(labels) != 0 {
		t.Errorf("expected labels cleared, got %+v", labels)
	}
}

func TestUpdateIssue_RejectsSelfLink(t *testing.T) {
	tr := setupLocalTransport(t)
	a, _ := tr.AddIssue(model.CreatePayload{Title: "a"})

	_, err := tr.UpdateIssue(a.ID, model.UpdatePayload{
		Dependencies: []model.Dependency{{TargetID: a.ID, Kind: "relates_to"}},
	}, "update")
	if err == nil || !strings.Contains(err.Error(), "itself") {
		t.Errorf("expected self-link error, got %v", err)
	}
}

func TestUpdateIssue_RejectsDuplicateLinks(t *testing.T) {
	tr := setupLocalTransport(t)
	a, _ := tr.AddIssue(model.CreatePayload{Title: "a"})
	b, _ := tr.AddIssue(model.CreatePayload{Title: "b"})

	_, err := tr.UpdateIssue(a.ID, model.UpdatePayload{
		Dependencies: []model.Dependency{
			{TargetID: b.ID, Kind: model.DependencyDependsOn},
			{TargetID: b.ID, Kind: model.DependencyDependsOn},
		},
	}, "update")
	if err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Errorf("expected duplicate link error, got %v", err)
	}
}

func TestProjectIssues_FillsEmptyDependencySourceID(t *testing.T) {
	now := time.Now().UTC()
	events := []model.Event{
		{ID: "x-1", Type: model.EventTypeCreate, CreatedAt: now, Payload: model.CreatePayload{
			Title:        "legacy create",
			Dependencies: []model.Dependency{{TargetID: "x-3", Kind: model.DependencyBlocks}},
		}},
		{ID: "x-2", Type: model.EventTypeCreate, CreatedAt: now, Payload: model.CreatePayload{Title: "legacy update"}},
		{ID: "x-3", Type: model.EventTypeCreate, CreatedAt: now, Payload: model.CreatePayload{Title: "target"}},
		{ID: "x-2", Type: model.EventTypeUpdate, CreatedAt: now, Payload: model.UpdatePayload{
			Dependencies: []model.Dependency{{TargetID: "x-3", Kind: model.DependencyDependsOn}},
		}},
	}
	issues := ProjectIssues(events)
	for _, id := range []string{"x-1", "x-2"} {
		deps := issues[id].Dependencies
		if len(deps) != 1 || deps[0].SourceID != id {
			t.Errorf("%s deps = %+v, want SourceID %s", id, deps, id)
		}
	}
}
