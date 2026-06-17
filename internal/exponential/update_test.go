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

func TestUpdateIssue_CannotCompleteWithIncompleteChildren(t *testing.T) {
	tr := setupLocalTransport(t)
	tr.Config.Automations.AutoCloseSubIssues = false
	parent, _ := tr.AddIssue(model.CreatePayload{Title: "parent"})
	tr.AddIssue(model.CreatePayload{Title: "child", ParentID: parent.ID})

	_, err := tr.UpdateIssue(parent.ID, model.UpdatePayload{Status: sp("DONE")}, "update")
	if err == nil {
		t.Error("expected error when completing parent with incomplete children")
	}
	if !strings.Contains(err.Error(), "unfinished sub-issues") {
		t.Errorf("error should mention sub-issues, got: %s", err)
	}
}

func TestUpdateIssue_AutoCloseSubIssues(t *testing.T) {
	tr := setupLocalTransport(t)
	parent, _ := tr.AddIssue(model.CreatePayload{Title: "parent"})
	child1, _ := tr.AddIssue(model.CreatePayload{Title: "child1", ParentID: parent.ID})
	child2, _ := tr.AddIssue(model.CreatePayload{Title: "child2", ParentID: parent.ID})

	msgs, err := tr.UpdateIssue(parent.ID, model.UpdatePayload{Status: sp("DONE")}, "update")
	if err != nil {
		t.Fatal(err)
	}

	issues := readAllIssues(t)
	if issues[child1.ID].Status != model.StatusDone {
		t.Errorf("child1 status = %s, want DONE", issues[child1.ID].Status)
	}
	if issues[child2.ID].Status != model.StatusDone {
		t.Errorf("child2 status = %s, want DONE", issues[child2.ID].Status)
	}

	autoMsgs := 0
	for _, m := range msgs {
		if strings.Contains(m, "Auto-closed") {
			autoMsgs++
		}
	}
	if autoMsgs != 2 {
		t.Errorf("expected 2 auto-close messages, got %d", autoMsgs)
	}
}

func TestUpdateIssue_AutoProgressChildren_BacklogToPlanned(t *testing.T) {
	tr := setupLocalTransport(t)
	parent, _ := tr.AddIssue(model.CreatePayload{Title: "parent"})
	child, _ := tr.AddIssue(model.CreatePayload{Title: "child", ParentID: parent.ID})

	tr.UpdateIssue(parent.ID, model.UpdatePayload{Status: sp("PLANNED")}, "update")

	issues := readAllIssues(t)
	if issues[child.ID].Status != model.StatusPlanned {
		t.Errorf("child status = %s, want PLANNED (auto-progressed from BACKLOG)", issues[child.ID].Status)
	}
}

func TestUpdateIssue_AutoProgressChildren_PlannedToDoing(t *testing.T) {
	tr := setupLocalTransport(t)
	parent, _ := tr.AddIssue(model.CreatePayload{Title: "parent"})
	child, _ := tr.AddIssue(model.CreatePayload{Title: "child", ParentID: parent.ID, Status: "PLANNED"})

	tr.UpdateIssue(parent.ID, model.UpdatePayload{Status: sp("DOING")}, "update")

	issues := readAllIssues(t)
	if issues[child.ID].Status != model.StatusDoing {
		t.Errorf("child status = %s, want DOING (auto-progressed from PLANNED)", issues[child.ID].Status)
	}
}

func TestUpdateIssue_AutoProgressSkipsNonMatchingChildren(t *testing.T) {
	tr := setupLocalTransport(t)
	parent, _ := tr.AddIssue(model.CreatePayload{Title: "parent"})
	doingChild, _ := tr.AddIssue(model.CreatePayload{Title: "already doing", ParentID: parent.ID, Status: "DOING"})

	tr.UpdateIssue(parent.ID, model.UpdatePayload{Status: sp("PLANNED")}, "update")

	issues := readAllIssues(t)
	if issues[doingChild.ID].Status != model.StatusDoing {
		t.Errorf("DOING child should not regress to PLANNED, got %s", issues[doingChild.ID].Status)
	}
}

func TestUpdateIssue_AutomationsOff(t *testing.T) {
	tr := setupLocalTransport(t)
	tr.Config.Automations.AutoCloseSubIssues = false
	tr.Config.Automations.AutoProgressSubIssues = false
	parent, _ := tr.AddIssue(model.CreatePayload{Title: "parent"})
	child, _ := tr.AddIssue(model.CreatePayload{Title: "child", ParentID: parent.ID})

	tr.UpdateIssue(parent.ID, model.UpdatePayload{Status: sp("PLANNED")}, "update")

	issues := readAllIssues(t)
	if issues[child.ID].Status != model.StatusBacklog {
		t.Errorf("child status = %s, want BACKLOG (automations disabled)", issues[child.ID].Status)
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
