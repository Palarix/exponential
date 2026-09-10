package exponential

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/palarix/exponential/internal/config"
	"github.com/palarix/exponential/internal/model"
	"github.com/palarix/exponential/internal/storage"
)

func setupTestEnv(t *testing.T) (*Client, func()) {
	t.Helper()
	tmpDir := t.TempDir()
	tmpDir, _ = filepath.EvalSymlinks(tmpDir)

	exec.Command("git", "init", "-b", "main", tmpDir).Run()
	exec.Command("git", "-C", tmpDir, "config", "user.email", "test@test.com").Run()
	exec.Command("git", "-C", tmpDir, "config", "user.name", "Test").Run()
	os.WriteFile(filepath.Join(tmpDir, "init.txt"), []byte("init"), 0644)
	exec.Command("git", "-C", tmpDir, "add", ".").Run()
	exec.Command("git", "-C", tmpDir, "commit", "-m", "init").Run()

	xpoDir := filepath.Join(tmpDir, ".xpo")
	os.MkdirAll(xpoDir, 0755)

	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	storage.ResetHubRoot()
	storage.ResetRefStore()
	if err := storage.InitRefStore(); err != nil {
		t.Fatalf("InitRefStore: %v", err)
	}

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
		storage.ResetHubRoot()
		storage.ResetRefStore()
	}

	return client, cleanup
}

func TestAddIssue(t *testing.T) {
	client, cleanup := setupTestEnv(t)
	defer cleanup()

	issue, err := client.AddIssue(model.CreatePayload{
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

	parent, err := client.AddIssue(model.CreatePayload{
		Title:  "Parent Issue",
		Labels: []string{"epic"},
	})
	if err != nil {
		t.Fatalf("AddIssue (parent) failed: %v", err)
	}

	child, err := client.AddIssue(model.CreatePayload{
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

	issue, _ := client.AddIssue(model.CreatePayload{
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

	parent, _ := client.AddIssue(model.CreatePayload{Title: "Parent", Labels: []string{"epic"}})
	child1, _ := client.AddIssue(model.CreatePayload{Title: "Child 1", ParentID: parent.ID})
	child2, _ := client.AddIssue(model.CreatePayload{Title: "Child 2", ParentID: parent.ID})

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

func TestBlockingRules(t *testing.T) {
	client, cleanup := setupTestEnv(t)
	defer cleanup()

	blocker, _ := client.AddIssue(model.CreatePayload{Title: "Blocker Issue"})
	blocked, _ := client.AddIssue(model.CreatePayload{
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

func TestEstimateAggregation(t *testing.T) {
	cfg := &config.Config{
		Prefix:           "test-",
		User:             "Test User <test@test.com>",
		EstimationSystem: "fibonacci",
		CountUnestimated: true,
		Version:          2,
	}

	tmpDir := t.TempDir()
	tmpDir, _ = filepath.EvalSymlinks(tmpDir)
	exec.Command("git", "init", "-b", "main", tmpDir).Run()
	exec.Command("git", "-C", tmpDir, "config", "user.email", "test@test.com").Run()
	exec.Command("git", "-C", tmpDir, "config", "user.name", "Test").Run()
	os.WriteFile(filepath.Join(tmpDir, "init.txt"), []byte("init"), 0644)
	exec.Command("git", "-C", tmpDir, "add", ".").Run()
	exec.Command("git", "-C", tmpDir, "commit", "-m", "init").Run()
	os.MkdirAll(filepath.Join(tmpDir, ".xpo"), 0755)
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	storage.ResetHubRoot()
	storage.ResetRefStore()
	if err := storage.InitRefStore(); err != nil {
		t.Fatalf("InitRefStore: %v", err)
	}
	defer func() {
		os.Chdir(origDir)
		storage.ResetHubRoot()
		storage.ResetRefStore()
	}()

	client := NewClient(cfg)

	parent, _ := client.AddIssue(model.CreatePayload{Title: "Parent"})
	client.AddIssue(model.CreatePayload{Title: "Child 1", ParentID: parent.ID, Estimate: 3})
	client.AddIssue(model.CreatePayload{Title: "Child 2", ParentID: parent.ID, Estimate: 5})
	client.AddIssue(model.CreatePayload{Title: "Child 3 (unestimated)", ParentID: parent.ID})

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
	tmpDir, _ = filepath.EvalSymlinks(tmpDir)
	exec.Command("git", "init", "-b", "main", tmpDir).Run()
	exec.Command("git", "-C", tmpDir, "config", "user.email", "test@test.com").Run()
	exec.Command("git", "-C", tmpDir, "config", "user.name", "Test").Run()
	os.WriteFile(filepath.Join(tmpDir, "init.txt"), []byte("init"), 0644)
	exec.Command("git", "-C", tmpDir, "add", ".").Run()
	exec.Command("git", "-C", tmpDir, "commit", "-m", "init").Run()
	os.MkdirAll(filepath.Join(tmpDir, ".xpo"), 0755)
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	storage.ResetHubRoot()
	storage.ResetRefStore()
	if err := storage.InitRefStore(); err != nil {
		t.Fatalf("InitRefStore: %v", err)
	}
	defer func() {
		os.Chdir(origDir)
		storage.ResetHubRoot()
		storage.ResetRefStore()
	}()

	client := NewClient(cfg)

	parent, _ := client.AddIssue(model.CreatePayload{Title: "Parent"})
	client.AddIssue(model.CreatePayload{Title: "Child 1", ParentID: parent.ID, Estimate: 3})
	client.AddIssue(model.CreatePayload{Title: "Child 2 (unestimated)", ParentID: parent.ID})

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

	issue, _ := client.AddIssue(model.CreatePayload{Title: "Comment Test"})

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

	issue1, _ := client.AddIssue(model.CreatePayload{Title: "Issue 1"})
	issue2, _ := client.AddIssue(model.CreatePayload{
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

func TestDeleteParentUnparentsChildren(t *testing.T) {
	client, cleanup := setupTestEnv(t)
	defer cleanup()

	parent, _ := client.AddIssue(model.CreatePayload{Title: "Parent", Labels: []string{"epic"}})
	child1, _ := client.AddIssue(model.CreatePayload{Title: "Child 1", ParentID: parent.ID})
	child2, _ := client.AddIssue(model.CreatePayload{Title: "Child 2", ParentID: parent.ID})

	err := client.DeleteIssue(parent.ID, "testing auto-unparent")
	if err != nil {
		t.Fatalf("DeleteIssue failed: %v", err)
	}

	events, _ := storage.ReadEvents()
	issues := ProjectIssues(events)

	if _, exists := issues[parent.ID]; exists {
		t.Error("Expected parent to be deleted")
	}
	if issues[child1.ID].ParentID != "" {
		t.Errorf("Expected child1 ParentID to be cleared, got %q", issues[child1.ID].ParentID)
	}
	if issues[child2.ID].ParentID != "" {
		t.Errorf("Expected child2 ParentID to be cleared, got %q", issues[child2.ID].ParentID)
	}
}

func TestDeleteParentCascadeDeletesChildren(t *testing.T) {
	client, cleanup := setupTestEnv(t)
	defer cleanup()

	parent, _ := client.AddIssue(model.CreatePayload{Title: "Parent", Labels: []string{"epic"}})
	child1, _ := client.AddIssue(model.CreatePayload{Title: "Child 1", ParentID: parent.ID})
	child2, _ := client.AddIssue(model.CreatePayload{Title: "Child 2", ParentID: parent.ID})
	unrelated, _ := client.AddIssue(model.CreatePayload{Title: "Unrelated"})

	err := client.DeleteIssue(parent.ID, "cascade test", true)
	if err != nil {
		t.Fatalf("DeleteIssue failed: %v", err)
	}

	events, _ := storage.ReadEvents()
	issues := ProjectIssues(events)

	if _, exists := issues[parent.ID]; exists {
		t.Error("Expected parent to be deleted")
	}
	if _, exists := issues[child1.ID]; exists {
		t.Error("Expected child1 to be cascade-deleted")
	}
	if _, exists := issues[child2.ID]; exists {
		t.Error("Expected child2 to be cascade-deleted")
	}
	if _, exists := issues[unrelated.ID]; !exists {
		t.Error("Unrelated issue should not be deleted")
	}
}

func TestAddIssueWithStatus(t *testing.T) {
	client, cleanup := setupTestEnv(t)
	defer cleanup()

	issue, err := client.AddIssue(model.CreatePayload{
		Title:  "Planned at creation",
		Status: string(model.StatusPlanned),
	})
	if err != nil {
		t.Fatalf("AddIssue failed: %v", err)
	}
	if issue.Status != model.StatusPlanned {
		t.Errorf("expected status PLANNED on returned issue, got %s", issue.Status)
	}

	events, _ := storage.ReadEvents()
	projected := ProjectIssues(events)[issue.ID]
	if projected.Status != model.StatusPlanned {
		t.Errorf("expected projected status PLANNED, got %s", projected.Status)
	}
}

func TestAddIssueDefaultStatus(t *testing.T) {
	client, cleanup := setupTestEnv(t)
	defer cleanup()

	issue, err := client.AddIssue(model.CreatePayload{Title: "Default"})
	if err != nil {
		t.Fatalf("AddIssue failed: %v", err)
	}
	if issue.Status != model.StatusBacklog {
		t.Errorf("expected default status BACKLOG, got %s", issue.Status)
	}
}

func TestAddIssueAutoFillsSourceID(t *testing.T) {
	client, cleanup := setupTestEnv(t)
	defer cleanup()

	target, _ := client.AddIssue(model.CreatePayload{Title: "Target"})

	issue, err := client.AddIssue(model.CreatePayload{
		Title: "With deps",
		Dependencies: []model.Dependency{
			{TargetID: target.ID, Kind: model.DependencyBlocks},
			{TargetID: target.ID, Kind: model.DependencyBlockedBy},
		},
	})
	if err != nil {
		t.Fatalf("AddIssue failed: %v", err)
	}
	for i, d := range issue.Dependencies {
		if d.SourceID != issue.ID {
			t.Errorf("dep[%d] SourceID: got %q want %q", i, d.SourceID, issue.ID)
		}
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
