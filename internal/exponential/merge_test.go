package exponential

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/palarix/exponential/internal/config"
	"github.com/palarix/exponential/internal/model"
	"github.com/palarix/exponential/internal/storage"
)

func TestMergeIssue_Squash(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init", "-b", "main")
	runGit(t, dir, "config", "user.email", "test@test.com")
	runGit(t, dir, "config", "user.name", "Test")

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	// Initialize xpo
	os.MkdirAll(filepath.Join(dir, ".xpo"), 0755)
	os.WriteFile(filepath.Join(dir, ".xpo/config.yaml"), []byte("prefix: test-\n"), 0644)

	os.WriteFile(filepath.Join(dir, "base.txt"), []byte("base\n"), 0644)
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "init")

	// Create a feature branch with commits
	runGit(t, dir, "checkout", "-b", "test-abc123/feature")
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a\n"), 0644)
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "add a")
	os.WriteFile(filepath.Join(dir, "b.txt"), []byte("b\n"), 0644)
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "add b")

	// Create a xpo issue
	cfg := &config.Config{Prefix: "test-", User: "Test <test@test.com>"}
	client := NewClient(cfg)

	evt := model.Event{
		ID:   "test-abc123",
		Type: model.EventTypeCreate,
		Payload: model.CreatePayload{
			Title:  "Test issue",
			Status: "DOING",
		},
		CreatedBy: "Test <test@test.com>",
	}
	storage.AppendEvent(evt)
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "add issue")

	result, err := client.MergeIssue("test-abc123", MergeOptions{
		Strategy:   MergeStrategySquash,
		KeepBranch: true,
	})
	if err != nil {
		t.Fatalf("MergeIssue failed: %v", err)
	}

	if result.MergeSHA == "" {
		t.Error("expected MergeSHA to be set")
	}

	// Verify we're on main
	branch := CurrentBranch()
	if branch != "main" {
		t.Errorf("expected to be on main, got %s", branch)
	}

	// Verify the merge event was recorded
	events, _ := storage.ReadEvents()
	issues := ProjectIssues(events)
	issue := issues["test-abc123"]
	if issue == nil {
		t.Fatal("issue not found after merge")
	}
	if issue.Status != model.StatusDone {
		t.Errorf("expected DONE, got %s", issue.Status)
	}

	// Verify event ordering: MERGE then DONE (UPDATE with status)
	hasMerge := false
	hasDone := false
	mergeIdx, doneIdx := -1, -1
	for idx, e := range events {
		if e.ID != issue.ID {
			continue
		}
		if e.Type == model.EventTypeMerge {
			hasMerge = true
			mergeIdx = idx
		}
		if e.Type == model.EventTypeUpdate {
			// After JSON roundtrip the payload may be a map
			if p, ok := e.Payload.(model.UpdatePayload); ok {
				if p.Status != nil && *p.Status == string(model.StatusDone) {
					hasDone = true
					doneIdx = idx
				}
			} else if m, ok := e.Payload.(map[string]interface{}); ok {
				if s, ok := m["status"].(string); ok && s == string(model.StatusDone) {
					hasDone = true
					doneIdx = idx
				}
			}
		}
	}
	if !hasMerge {
		t.Error("expected MERGE event in issue history")
	}
	if !hasDone {
		t.Error("expected DONE update event in issue history")
	}
	if hasMerge && hasDone && mergeIdx >= doneIdx {
		t.Errorf("expected MERGE event (idx %d) before DONE event (idx %d)", mergeIdx, doneIdx)
	}
}

func TestIsWorkingTreeClean(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init", "-b", "main")
	runGit(t, dir, "config", "user.email", "test@test.com")
	runGit(t, dir, "config", "user.name", "Test")

	os.WriteFile(filepath.Join(dir, "base.txt"), []byte("base\n"), 0644)
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "init")

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	if !IsWorkingTreeClean() {
		t.Error("expected clean tree after commit")
	}

	os.WriteFile(filepath.Join(dir, "dirty.txt"), []byte("dirty\n"), 0644)

	if IsWorkingTreeClean() {
		t.Error("expected dirty tree after adding untracked file")
	}
}
