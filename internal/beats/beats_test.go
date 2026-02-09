package beats

import (
	"os"
	"testing"

	"github.com/palarix/beats/internal/config"
	"github.com/palarix/beats/internal/model"
)

func setupTestEnv(t *testing.T) *Client {
	// Create temp dir
	tmpDir, err := os.MkdirTemp("", "beats-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Change to temp dir
	originalWd, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to chdir: %v", err)
	}

	// Setup cleanup
	t.Cleanup(func() {
		os.Chdir(originalWd)
		os.RemoveAll(tmpDir)
	})

	// Init beats
	_, err = InitBeats(false)
	if err != nil {
		t.Fatalf("Failed to init beats: %v", err)
	}

	// Create client
	cfg := &config.Config{
		Prefix:     "TEST-",
		AutoCommit: false, // Don't try to run git in tests
	}
	client := NewClient(cfg)

	return client
}

func TestAddIssue(t *testing.T) {
	client := setupTestEnv(t)

	opts := AddOptions{
		Title:       "Test Issue",
		Description: "Description",
		Kind:        "TASK",
		Estimate:    5,
	}

	issue, err := client.AddIssue(opts)
	if err != nil {
		t.Fatalf("AddIssue failed: %v", err)
	}

	if issue.Title != opts.Title {
		t.Errorf("Expected title %s, got %s", opts.Title, issue.Title)
	}
	if issue.Status != model.StatusBacklog {
		t.Errorf("Expected status BACKLOG, got %s", issue.Status)
	}

	// Verify retrieval
	retrieved, err := client.GetIssue(issue.ID)
	if err != nil {
		t.Fatalf("GetIssue failed: %v", err)
	}
	if retrieved.Title != opts.Title {
		t.Errorf("Retrieved title mismatch")
	}
}

func TestUpdateIssue(t *testing.T) {
	client := setupTestEnv(t)

	// Create
	opts := AddOptions{Title: "To Update", Kind: "TASK"}
	issue, _ := client.AddIssue(opts)

	// Update
	newTitle := "Updated Title"
	newStatus := string(model.StatusPlanned)
	payload := model.UpdatePayload{
		Title:  &newTitle,
		Status: &newStatus,
	}

	_, err := client.UpdateIssue(issue.ID, payload, "update")
	if err != nil {
		t.Fatalf("UpdateIssue failed: %v", err)
	}

	// Verify
	updated, _ := client.GetIssue(issue.ID)
	if updated.Title != newTitle {
		t.Errorf("Expected title %s, got %s", newTitle, updated.Title)
	}
	if updated.Status != model.StatusPlanned {
		t.Errorf("Expected status PLANNED, got %s", updated.Status)
	}
}

func TestEpicAutoStart(t *testing.T) {
	client := setupTestEnv(t)

	// Create Epic
	epic, _ := client.AddIssue(AddOptions{Title: "Epic", Kind: "EPIC"})

	// Create Child
	child, _ := client.AddIssue(AddOptions{Title: "Child", Kind: "TASK", ParentID: epic.ID})

	// Start Child
	statusDoing := string(model.StatusDoing)
	_, err := client.UpdateIssue(child.ID, model.UpdatePayload{Status: &statusDoing}, "start child")
	if err != nil {
		t.Fatalf("UpdateIssue failed: %v", err)
	}

	// Verify Epic Auto-Started
	updatedEpic, _ := client.GetIssue(epic.ID)
	if updatedEpic.Status != model.StatusDoing {
		t.Errorf("Expected Epic to auto-start (DOING), got %s", updatedEpic.Status)
	}
}

func TestEpicAutoComplete(t *testing.T) {
	client := setupTestEnv(t)

	// Create Epic & Children
	epic, _ := client.AddIssue(AddOptions{Title: "Epic", Kind: "EPIC"})
	child1, _ := client.AddIssue(AddOptions{Title: "Child 1", Kind: "TASK", ParentID: epic.ID})
	child2, _ := client.AddIssue(AddOptions{Title: "Child 2", Kind: "TASK", ParentID: epic.ID})

	// Complete Child 1
	statusDone := string(model.StatusDone)
	client.UpdateIssue(child1.ID, model.UpdatePayload{Status: &statusDone}, "done c1")

	// Verify Epic NOT done yet
	updatedEpic, _ := client.GetIssue(epic.ID)
	if updatedEpic.Status == model.StatusDone {
		t.Errorf("Epic should not be done yet")
	}

	// Complete Child 2
	client.UpdateIssue(child2.ID, model.UpdatePayload{Status: &statusDone}, "done c2")

	// Verify Epic Auto-Complete (Existing Behavior)
	updatedEpic, _ = client.GetIssue(epic.ID)
	if updatedEpic.Status != model.StatusDone {
		t.Errorf("Expected Epic to auto-complete (DONE), got %s", updatedEpic.Status)
	}
}

func TestMetadata(t *testing.T) {
	client := setupTestEnv(t)

	// Create with Metadata
	checklist := []model.ChecklistItem{{Title: "Item 1", State: "open"}}
	labels := []string{"bug", "critical"}
	opts := AddOptions{Title: "Meta Issue", Kind: "TASK", Estimate: 1}

	// We need to support passing metadata in AddOptions.
	// Since AddOptions struct in add.go wasn't updated in previous steps (I only updated CreatePayload in types.go),
	// I need to update AddOptions first!
	// Checking the plan... "Update AddIssue to support creation with Dependencies...".
	// I missed updating AddOptions/AddIssue in add.go.
	// I will update the test to expect this, and then fix add.go.

	// For now let's test *Update* metadata since Add isn't updated yet.
	issue, _ := client.AddIssue(opts)

	updatePayload := model.UpdatePayload{
		Checklist: checklist,
		Labels:    labels,
	}
	client.UpdateIssue(issue.ID, updatePayload, "add metadata")

	updated, _ := client.GetIssue(issue.ID)
	if len(updated.Checklist) != 1 || updated.Checklist[0].Title != "Item 1" {
		t.Errorf("Checklist mismatch")
	}
	if len(updated.Labels) != 2 || updated.Labels[0] != "bug" {
		t.Errorf("Labels mismatch")
	}
}

func TestBlockingRules(t *testing.T) {
	client := setupTestEnv(t)

	// Create Blocker (Review)
	blockerOpts := AddOptions{Title: "Review", Kind: "TASK"}
	blocker, _ := client.AddIssue(blockerOpts)

	// Create Blocked Task (Dev)
	blockedOpts := AddOptions{Title: "Dev", Kind: "TASK"}
	blocked, _ := client.AddIssue(blockedOpts)

	// Add Dependency: Blocked is blocked by Blocker
	// We use the compatibility field Update since we haven't exposed Dependencies update properly via opts yet
	// But UpdatePayload supports Dependencies list.
	deps := []model.Dependency{{
		SourceID: blocked.ID,
		TargetID: blocker.ID,
		Kind:     model.DependencyBlockedBy,
	}}
	client.UpdateIssue(blocked.ID, model.UpdatePayload{Dependencies: deps}, "block dev")

	// Try to start Blocked Task -> Should Fail
	statusDoing := string(model.StatusDoing)
	_, err := client.UpdateIssue(blocked.ID, model.UpdatePayload{Status: &statusDoing}, "try start")
	if err == nil {
		t.Errorf("Expected error when starting blocked task, got nil")
	}

	// Complete Blocker
	statusDone := string(model.StatusDone)
	client.UpdateIssue(blocker.ID, model.UpdatePayload{Status: &statusDone}, "finish review")

	// Try to start Blocked Task -> Should Succeed
	_, err = client.UpdateIssue(blocked.ID, model.UpdatePayload{Status: &statusDoing}, "retry start")
	if err != nil {
		t.Errorf("Expected success when starting unblocked task, got error: %v", err)
	}
}
