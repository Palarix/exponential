package beats

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kuyio/beats/internal/config"
	"github.com/kuyio/beats/internal/model"
	"github.com/kuyio/beats/internal/storage"
)

func setupTestEnv(t *testing.T) (*Client, func()) {
	t.Helper()
	tmpDir := t.TempDir()
	beatsDir := filepath.Join(tmpDir, ".beats")
	os.MkdirAll(beatsDir, 0755)

	issuesDB := filepath.Join(beatsDir, "issues.db")
	os.WriteFile(issuesDB, []byte{}, 0644)

	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)

	cfg := &config.Config{
		Prefix:           "test-",
		User:             "Test User <test@test.com>",
		EstimationSystem: "fibonacci",
		CountUnestimated: true,
		Version:          2,
	}

	client := NewClient(cfg)
	cleanup := func() {
		os.Chdir(origDir)
	}

	return client, cleanup
}

func TestAddIssue(t *testing.T) {
	client, cleanup := setupTestEnv(t)
	defer cleanup()

	issue, err := client.AddIssue(AddOptions{
		Title:    "Test Issue",
		Labels:   []string{"feature"},
		Assignee: "Dev <dev@test.com>",
		Estimate: 3,
	})
	if err != nil {
		t.Fatalf("AddIssue failed: %v", err)
	}

	if issue.Title != "Test Issue" {
		t.Errorf("Expected title 'Test Issue', got %q", issue.Title)
	}
	if issue.Status != model.StatusBacklog {
		t.Errorf("Expected status BACKLOG, got %s", issue.Status)
	}
	if len(issue.Labels) != 1 || issue.Labels[0] != "feature" {
		t.Errorf("Expected labels [feature], got %v", issue.Labels)
	}
	if issue.Assignee != "Dev <dev@test.com>" {
		t.Errorf("Expected assignee 'Dev <dev@test.com>', got %q", issue.Assignee)
	}
	if issue.Estimate != 3 {
		t.Errorf("Expected estimate 3, got %d", issue.Estimate)
	}
}

func TestAddIssueWithParent(t *testing.T) {
	client, cleanup := setupTestEnv(t)
	defer cleanup()

	parent, err := client.AddIssue(AddOptions{
		Title:  "Parent Issue",
		Labels: []string{"epic"},
	})
	if err != nil {
		t.Fatalf("AddIssue (parent) failed: %v", err)
	}

	child, err := client.AddIssue(AddOptions{
		Title:    "Child Issue",
		ParentID: parent.ID,
	})
	if err != nil {
		t.Fatalf("AddIssue (child) failed: %v", err)
	}

	if child.ParentID != parent.ID {
		t.Errorf("Expected ParentID %s, got %s", parent.ID, child.ParentID)
	}
}

func TestUpdateIssue(t *testing.T) {
	client, cleanup := setupTestEnv(t)
	defer cleanup()

	issue, _ := client.AddIssue(AddOptions{
		Title:  "Original Title",
		Labels: []string{"bug"},
	})

	newTitle := "Updated Title"
	newLabels := []string{"bug", "critical"}
	newAssignee := "Dev <dev@test.com>"
	payload := model.UpdatePayload{
		Title:    &newTitle,
		Labels:   newLabels,
		Assignee: &newAssignee,
	}

	_, err := client.UpdateIssue(issue.ID, payload, "update")
	if err != nil {
		t.Fatalf("UpdateIssue failed: %v", err)
	}

	// Verify by reading back
	events, _ := storage.ReadEvents()
	issues := ProjectIssues(events)
	updated := issues[issue.ID]

	if updated.Title != "Updated Title" {
		t.Errorf("Expected title 'Updated Title', got %q", updated.Title)
	}
	if len(updated.Labels) != 2 {
		t.Errorf("Expected 2 labels, got %d", len(updated.Labels))
	}
	if updated.Assignee != "Dev <dev@test.com>" {
		t.Errorf("Expected assignee 'Dev <dev@test.com>', got %q", updated.Assignee)
	}
}

func TestParentChildRelationship(t *testing.T) {
	client, cleanup := setupTestEnv(t)
	defer cleanup()

	parent, _ := client.AddIssue(AddOptions{Title: "Parent", Labels: []string{"epic"}})
	child1, _ := client.AddIssue(AddOptions{Title: "Child 1", ParentID: parent.ID})
	child2, _ := client.AddIssue(AddOptions{Title: "Child 2", ParentID: parent.ID})

	// Verify children
	_, children, _, err := client.FindIssue(parent.ID)
	if err != nil {
		t.Fatalf("FindIssue failed: %v", err)
	}
	if len(children) != 2 {
		t.Errorf("Expected 2 children, got %d", len(children))
	}

	// Verify parent reference
	events, _ := storage.ReadEvents()
	issues := ProjectIssues(events)
	if issues[child1.ID].ParentID != parent.ID {
		t.Errorf("Child 1 ParentID mismatch")
	}
	if issues[child2.ID].ParentID != parent.ID {
		t.Errorf("Child 2 ParentID mismatch")
	}
}

func TestCannotCompleteParentWithIncompleteChildren(t *testing.T) {
	client, cleanup := setupTestEnv(t)
	defer cleanup()

	parent, _ := client.AddIssue(AddOptions{Title: "Parent"})
	client.AddIssue(AddOptions{Title: "Child", ParentID: parent.ID})

	// Try to complete parent - should fail
	status := string(model.StatusDone)
	_, err := client.UpdateIssue(parent.ID, model.UpdatePayload{Status: &status}, "done")
	if err == nil {
		t.Error("Expected error when completing parent with incomplete children")
	}
}

func TestBlockingRules(t *testing.T) {
	client, cleanup := setupTestEnv(t)
	defer cleanup()

	blocker, _ := client.AddIssue(AddOptions{Title: "Blocker Issue"})
	blocked, _ := client.AddIssue(AddOptions{
		Title: "Blocked Issue",
		Dependencies: []model.Dependency{
			{SourceID: "", TargetID: blocker.ID, Kind: model.DependencyBlockedBy},
		},
	})

	// Try to start blocked issue - should fail
	status := string(model.StatusDoing)
	_, err := client.UpdateIssue(blocked.ID, model.UpdatePayload{Status: &status}, "start")
	if err == nil {
		t.Error("Expected error when starting blocked issue")
	}

	// Complete blocker
	doneStatus := string(model.StatusDone)
	client.UpdateIssue(blocker.ID, model.UpdatePayload{Status: &doneStatus}, "done")

	// Now should succeed
	_, err = client.UpdateIssue(blocked.ID, model.UpdatePayload{Status: &status}, "start")
	if err != nil {
		t.Errorf("Expected success starting issue after blocker done: %v", err)
	}
}

func TestAutoCompleteParent(t *testing.T) {
	cfg := &config.Config{
		Prefix:           "test-",
		User:             "Test User <test@test.com>",
		EstimationSystem: "fibonacci",
		CountUnestimated: true,
		Version:          2,
		Automations: config.Automations{
			AutoCompleteParent: true,
		},
	}

	tmpDir := t.TempDir()
	beatsDir := filepath.Join(tmpDir, ".beats")
	os.MkdirAll(beatsDir, 0755)
	os.WriteFile(filepath.Join(beatsDir, "issues.db"), []byte{}, 0644)
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	client := NewClient(cfg)

	parent, _ := client.AddIssue(AddOptions{Title: "Parent"})
	child, _ := client.AddIssue(AddOptions{Title: "Child", ParentID: parent.ID})

	// Complete the child
	doneStatus := string(model.StatusDone)
	client.UpdateIssue(child.ID, model.UpdatePayload{Status: &doneStatus}, "done")

	// Read with config - parent should be auto-completed
	events, _ := storage.ReadEvents()
	issues := ProjectIssuesWithConfig(events, cfg)

	if issues[parent.ID].Status != model.StatusDone {
		t.Errorf("Expected parent to be auto-completed, got status %s", issues[parent.ID].Status)
	}
}

func TestAutoCloseSubIssues(t *testing.T) {
	cfg := &config.Config{
		Prefix:           "test-",
		User:             "Test User <test@test.com>",
		EstimationSystem: "fibonacci",
		CountUnestimated: true,
		Version:          2,
		Automations: config.Automations{
			AutoCloseSubIssues: true,
		},
	}

	tmpDir := t.TempDir()
	beatsDir := filepath.Join(tmpDir, ".beats")
	os.MkdirAll(beatsDir, 0755)
	os.WriteFile(filepath.Join(beatsDir, "issues.db"), []byte{}, 0644)
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	client := NewClient(cfg)

	parent, _ := client.AddIssue(AddOptions{Title: "Parent"})
	child, _ := client.AddIssue(AddOptions{Title: "Child", ParentID: parent.ID})

	// Complete the parent - should auto-close child
	doneStatus := string(model.StatusDone)
	msgs, err := client.UpdateIssue(parent.ID, model.UpdatePayload{Status: &doneStatus}, "done")
	if err != nil {
		t.Fatalf("UpdateIssue failed: %v", err)
	}

	// Should have auto-close message
	found := false
	for _, msg := range msgs {
		if msg == "Auto-closed sub-issue "+child.ID {
			found = true
		}
	}
	if !found {
		t.Errorf("Expected auto-close message for child, got: %v", msgs)
	}

	// Verify child is done
	events, _ := storage.ReadEvents()
	issues := ProjectIssues(events)
	if issues[child.ID].Status != model.StatusDone {
		t.Errorf("Expected child to be auto-closed, got status %s", issues[child.ID].Status)
	}
}

func TestAutomationsOff(t *testing.T) {
	client, cleanup := setupTestEnv(t)
	defer cleanup()

	parent, _ := client.AddIssue(AddOptions{Title: "Parent"})
	child, _ := client.AddIssue(AddOptions{Title: "Child", ParentID: parent.ID})

	// Complete the child
	doneStatus := string(model.StatusDone)
	client.UpdateIssue(child.ID, model.UpdatePayload{Status: &doneStatus}, "done")

	// Read WITHOUT automations - parent should NOT be auto-completed
	events, _ := storage.ReadEvents()
	issues := ProjectIssues(events)

	if issues[parent.ID].Status == model.StatusDone {
		t.Error("Parent should NOT be auto-completed when automations are off")
	}
}

func TestEstimateAggregation(t *testing.T) {
	cfg := &config.Config{
		Prefix:           "test-",
		User:             "Test User <test@test.com>",
		EstimationSystem: "fibonacci",
		CountUnestimated: true,
		Version:          2,
	}

	tmpDir := t.TempDir()
	beatsDir := filepath.Join(tmpDir, ".beats")
	os.MkdirAll(beatsDir, 0755)
	os.WriteFile(filepath.Join(beatsDir, "issues.db"), []byte{}, 0644)
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	client := NewClient(cfg)

	parent, _ := client.AddIssue(AddOptions{Title: "Parent"})
	client.AddIssue(AddOptions{Title: "Child 1", ParentID: parent.ID, Estimate: 3})
	client.AddIssue(AddOptions{Title: "Child 2", ParentID: parent.ID, Estimate: 5})
	client.AddIssue(AddOptions{Title: "Child 3 (unestimated)", ParentID: parent.ID})

	events, _ := storage.ReadEvents()
	issues := ProjectIssuesWithConfig(events, cfg)

	// Parent should sum: 3 + 5 + 1 (unestimated default) = 9
	if issues[parent.ID].Estimate != 9 {
		t.Errorf("Expected parent estimate 9, got %d", issues[parent.ID].Estimate)
	}
}

func TestCountUnestimatedFalse(t *testing.T) {
	cfg := &config.Config{
		Prefix:           "test-",
		User:             "Test User <test@test.com>",
		EstimationSystem: "fibonacci",
		CountUnestimated: false,
		Version:          2,
	}

	tmpDir := t.TempDir()
	beatsDir := filepath.Join(tmpDir, ".beats")
	os.MkdirAll(beatsDir, 0755)
	os.WriteFile(filepath.Join(beatsDir, "issues.db"), []byte{}, 0644)
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	client := NewClient(cfg)

	parent, _ := client.AddIssue(AddOptions{Title: "Parent"})
	client.AddIssue(AddOptions{Title: "Child 1", ParentID: parent.ID, Estimate: 3})
	client.AddIssue(AddOptions{Title: "Child 2 (unestimated)", ParentID: parent.ID})

	events, _ := storage.ReadEvents()
	issues := ProjectIssuesWithConfig(events, cfg)

	// Parent should sum: 3 + 0 (unestimated not counted) = 3
	if issues[parent.ID].Estimate != 3 {
		t.Errorf("Expected parent estimate 3, got %d", issues[parent.ID].Estimate)
	}
}

func TestComments(t *testing.T) {
	client, cleanup := setupTestEnv(t)
	defer cleanup()

	issue, _ := client.AddIssue(AddOptions{Title: "Comment Test"})

	// Add comment via event
	evt := model.Event{
		ID:   issue.ID,
		Type: model.EventTypeComment,
		Payload: model.CommentPayload{
			ID:   "c1",
			Text: "Hello, world!",
		},
		CreatedBy: "Test User <test@test.com>",
	}
	storage.AppendEvent(evt)

	events, _ := storage.ReadEvents()
	issues := ProjectIssues(events)
	updated := issues[issue.ID]

	if len(updated.Comments) != 1 {
		t.Fatalf("Expected 1 comment, got %d", len(updated.Comments))
	}
	if updated.Comments[0].Text != "Hello, world!" {
		t.Errorf("Expected comment text 'Hello, world!', got %q", updated.Comments[0].Text)
	}
}

func TestDependencies(t *testing.T) {
	client, cleanup := setupTestEnv(t)
	defer cleanup()

	issue1, _ := client.AddIssue(AddOptions{Title: "Issue 1"})
	issue2, _ := client.AddIssue(AddOptions{
		Title: "Issue 2",
		Dependencies: []model.Dependency{
			{TargetID: issue1.ID, Kind: model.DependencyDependsOn},
		},
	})

	events, _ := storage.ReadEvents()
	issues := ProjectIssues(events)

	deps := issues[issue2.ID].Dependencies
	if len(deps) != 1 {
		t.Fatalf("Expected 1 dependency, got %d", len(deps))
	}
	if deps[0].Kind != model.DependencyDependsOn {
		t.Errorf("Expected depends_on, got %s", deps[0].Kind)
	}
	if deps[0].TargetID != issue1.ID {
		t.Errorf("Expected target %s, got %s", issue1.ID, deps[0].TargetID)
	}
}

func TestInverseKind(t *testing.T) {
	pairs := map[model.DependencyKind]model.DependencyKind{
		model.DependencyDependsOn:    model.DependencyDependencyOf,
		model.DependencyDependencyOf: model.DependencyDependsOn,
		model.DependencyBlockedBy:    model.DependencyBlocks,
		model.DependencyBlocks:       model.DependencyBlockedBy,
		model.DependencyDuplicatedBy: model.DependencyDuplicates,
		model.DependencyDuplicates:   model.DependencyDuplicatedBy,
	}

	for input, expected := range pairs {
		result := model.InverseKind(input)
		if result != expected {
			t.Errorf("InverseKind(%s) = %s, want %s", input, result, expected)
		}
	}
}
