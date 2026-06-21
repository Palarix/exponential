package exponential

import (
	"strings"
	"testing"

	"github.com/palarix/exponential/internal/model"
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
