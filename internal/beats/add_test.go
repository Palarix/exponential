package beats

import (
	"strings"
	"testing"

	"github.com/palarix/beats/internal/model"
)

func TestAddIssue_HappyPath(t *testing.T) {
	tr := setupLocalTransport(t)
	issue, err := tr.AddIssue(model.CreatePayload{Title: "hello"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(issue.ID, "test-") {
		t.Errorf("expected test- prefix, got %s", issue.ID)
	}
	if issue.Status != model.StatusBacklog {
		t.Errorf("expected BACKLOG default, got %s", issue.Status)
	}
	issues := readAllIssues(t)
	if _, ok := issues[issue.ID]; !ok {
		t.Error("issue not persisted to event log")
	}
}

func TestAddIssue_WithStatus(t *testing.T) {
	tr := setupLocalTransport(t)
	issue, err := tr.AddIssue(model.CreatePayload{Title: "planned", Status: "PLANNED"})
	if err != nil {
		t.Fatal(err)
	}
	if issue.Status != model.StatusPlanned {
		t.Errorf("expected PLANNED, got %s", issue.Status)
	}
	issues := readAllIssues(t)
	if issues[issue.ID].Status != model.StatusPlanned {
		t.Errorf("projected status = %s, want PLANNED", issues[issue.ID].Status)
	}
}

func TestAddIssue_DependencySourceIDAutofill(t *testing.T) {
	tr := setupLocalTransport(t)
	issue, err := tr.AddIssue(model.CreatePayload{
		Title: "with dep",
		Dependencies: []model.Dependency{
			{TargetID: "other", Kind: "depends_on"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	issues := readAllIssues(t)
	deps := issues[issue.ID].Dependencies
	if len(deps) != 1 || deps[0].SourceID != issue.ID {
		t.Errorf("expected SourceID autofilled to %s, got %+v", issue.ID, deps)
	}
}

func TestAddIssue_PrefixFromConfig(t *testing.T) {
	tr := setupLocalTransport(t)
	tr.Config.Prefix = "proj-"
	issue, err := tr.AddIssue(model.CreatePayload{Title: "custom prefix"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(issue.ID, "proj-") {
		t.Errorf("expected proj- prefix, got %s", issue.ID)
	}
}

func TestAddIssue_AllFields(t *testing.T) {
	tr := setupLocalTransport(t)
	issue, err := tr.AddIssue(model.CreatePayload{
		Title:       "full",
		Description: "desc",
		Status:      "DOING",
		Estimate:    5,
		Priority:    2,
		Assignee:    "Alice <a@b.com>",
		Labels:      []string{"bug"},
	})
	if err != nil {
		t.Fatal(err)
	}
	issues := readAllIssues(t)
	i := issues[issue.ID]
	if i.Description != "desc" || i.Estimate != 5 || i.Priority != 2 || i.Assignee != "Alice <a@b.com>" {
		t.Errorf("fields not projected: %+v", i)
	}
	if len(i.Labels) != 1 || i.Labels[0] != "bug" {
		t.Errorf("labels = %v", i.Labels)
	}
}
