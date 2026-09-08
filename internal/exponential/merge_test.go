package exponential

import (
	"os"
	"path/filepath"
	"strings"
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

func TestMergeIssue_SquashPreservesMainIssuesDB(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init", "-b", "main")
	runGit(t, dir, "config", "user.email", "test@test.com")
	runGit(t, dir, "config", "user.name", "Test")

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	t.Cleanup(func() {
		os.Chdir(origDir)
		storage.ResetHubRoot()
	})

	os.MkdirAll(filepath.Join(dir, ".xpo"), 0755)
	os.WriteFile(filepath.Join(dir, ".xpo/config.yaml"), []byte("prefix: test-\nworktrees: false\n"), 0644)
	os.WriteFile(filepath.Join(dir, ".gitattributes"), []byte(".xpo/issues.db merge=union\n"), 0644)

	cfg := &config.Config{Prefix: "test-", User: "Test <test@test.com>", Worktrees: false}
	client := NewClient(cfg)

	// Seed an initial issue on main
	storage.ResetHubRoot()
	evt0 := model.Event{
		ID:   "test-000000",
		Type: model.EventTypeCreate,
		Payload: model.CreatePayload{
			Title:  "Background issue",
			Status: "DOING",
		},
		CreatedBy: "Test <test@test.com>",
	}
	storage.AppendEvent(evt0)

	os.WriteFile(filepath.Join(dir, "base.txt"), []byte("base\n"), 0644)
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "init")

	// Create a feature branch
	runGit(t, dir, "checkout", "-b", "test-abc123/feature")

	// Add a branch-only event and code change
	evtBranch := model.Event{
		ID:   "test-abc123",
		Type: model.EventTypeCreate,
		Payload: model.CreatePayload{
			Title:  "Feature issue",
			Status: "DOING",
		},
		CreatedBy: "Test <test@test.com>",
	}
	storage.ResetHubRoot()
	storage.AppendEvent(evtBranch)
	os.WriteFile(filepath.Join(dir, "feature.txt"), []byte("feature\n"), 0644)
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "feature work")

	// Go back to main and add a main-only event
	runGit(t, dir, "checkout", "main")
	storage.ResetHubRoot()
	evtMain := model.Event{
		ID:   "test-000000",
		Type: model.EventTypeComment,
		Payload: model.CommentPayload{
			Text: "Main-only comment after branch diverged",
		},
		CreatedBy: "Test <test@test.com>",
	}
	storage.AppendEvent(evtMain)
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "main-only event")

	// Switch to the feature branch for merge
	runGit(t, dir, "checkout", "test-abc123/feature")
	storage.ResetHubRoot()

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

	// Verify all events survived the merge
	storage.ResetHubRoot()
	events, err := storage.ReadEvents()
	if err != nil {
		t.Fatalf("ReadEvents failed: %v", err)
	}

	hasBackgroundCreate := false
	hasMainComment := false
	hasBranchCreate := false
	hasMerge := false
	for _, e := range events {
		if e.ID == "test-000000" && e.Type == model.EventTypeCreate {
			hasBackgroundCreate = true
		}
		if e.ID == "test-000000" && e.Type == model.EventTypeComment {
			hasMainComment = true
		}
		if e.ID == "test-abc123" && e.Type == model.EventTypeCreate {
			hasBranchCreate = true
		}
		if e.ID == "test-abc123" && e.Type == model.EventTypeMerge {
			hasMerge = true
		}
	}

	if !hasBackgroundCreate {
		t.Error("lost the initial create event from main")
	}
	if !hasMainComment {
		t.Error("lost the comment event added on main after branch diverged")
	}
	if !hasBranchCreate {
		t.Error("lost the create event from the feature branch")
	}
	if !hasMerge {
		t.Error("expected MERGE event in issue history")
	}
}

func TestMergeIssue_BranchModeWithWorktreesEnabled(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init", "-b", "main")
	runGit(t, dir, "config", "user.email", "test@test.com")
	runGit(t, dir, "config", "user.name", "Test")

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	t.Cleanup(func() {
		os.Chdir(origDir)
		storage.ResetHubRoot()
	})

	os.MkdirAll(filepath.Join(dir, ".xpo"), 0755)
	os.WriteFile(filepath.Join(dir, ".xpo/config.yaml"), []byte("prefix: test-\n"), 0644)

	os.WriteFile(filepath.Join(dir, "base.txt"), []byte("base\n"), 0644)
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "init")

	runGit(t, dir, "checkout", "-b", "test-abc123/feature")
	os.WriteFile(filepath.Join(dir, "feature.txt"), []byte("feature\n"), 0644)
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "add feature")

	storage.ResetHubRoot()
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

	// Worktrees enabled but no worktree exists — this is branch mode.
	// The hub is on the feature branch; merge must auto-checkout main.
	cfg := &config.Config{Prefix: "test-", User: "Test <test@test.com>", Worktrees: true}
	client := NewClient(cfg)

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

	branch := CurrentBranch()
	if branch != "main" {
		t.Errorf("expected to be on main, got %s", branch)
	}
}

func TestUnionLines(t *testing.T) {
	tests := []struct {
		name     string
		base     string
		theirs   string
		expected string
	}{
		{
			name:     "disjoint lines",
			base:     "a\nb\n",
			theirs:   "c\nd\n",
			expected: "a\nb\nc\nd\n",
		},
		{
			name:     "overlapping lines",
			base:     "a\nb\nc\n",
			theirs:   "a\nb\nd\n",
			expected: "a\nb\nc\nd\n",
		},
		{
			name:     "identical",
			base:     "a\nb\n",
			theirs:   "a\nb\n",
			expected: "a\nb\n",
		},
		{
			name:     "empty base",
			base:     "",
			theirs:   "a\nb\n",
			expected: "a\nb\n",
		},
		{
			name:     "empty theirs",
			base:     "a\nb\n",
			theirs:   "",
			expected: "a\nb\n",
		},
		{
			name:     "both empty",
			base:     "",
			theirs:   "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := unionLines([]byte(tt.base), []byte(tt.theirs))
			gotStr := string(got)
			if tt.expected == "" {
				if got != nil {
					t.Errorf("expected nil, got %q", gotStr)
				}
				return
			}
			if gotStr != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, gotStr)
			}
		})
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

func setupHubCleanForMergeRepo(t *testing.T) (dir string, cleanup func()) {
	t.Helper()
	dir = t.TempDir()
	runGit(t, dir, "init", "-b", "main")
	runGit(t, dir, "config", "user.email", "test@test.com")
	runGit(t, dir, "config", "user.name", "Test")

	os.MkdirAll(filepath.Join(dir, ".xpo"), 0755)
	os.WriteFile(filepath.Join(dir, "base.txt"), []byte("base\n"), 0644)
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "init")

	runGit(t, dir, "checkout", "-b", "feature-branch")
	os.WriteFile(filepath.Join(dir, "feature.txt"), []byte("feature\n"), 0644)
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "add feature")
	runGit(t, dir, "checkout", "main")

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	storage.ResetHubRoot()
	return dir, func() {
		os.Chdir(origDir)
		storage.ResetHubRoot()
	}
}

func TestHubCleanForMerge_CleanTree(t *testing.T) {
	_, cleanup := setupHubCleanForMergeRepo(t)
	defer cleanup()

	if err := HubCleanForMerge("feature-branch"); err != nil {
		t.Errorf("expected clean merge, got: %v", err)
	}
}

func TestHubCleanForMerge_UntrackedFilesAllowed(t *testing.T) {
	dir, cleanup := setupHubCleanForMergeRepo(t)
	defer cleanup()

	os.WriteFile(filepath.Join(dir, "idea.md"), []byte("some ideas\n"), 0644)
	os.WriteFile(filepath.Join(dir, "thoughts.md"), []byte("some thoughts\n"), 0644)

	if err := HubCleanForMerge("feature-branch"); err != nil {
		t.Errorf("untracked files should not block merge, got: %v", err)
	}
}

func TestHubCleanForMerge_XpoChangesAllowed(t *testing.T) {
	dir, cleanup := setupHubCleanForMergeRepo(t)
	defer cleanup()

	os.WriteFile(filepath.Join(dir, ".xpo", "issues.db"), []byte("event data\n"), 0644)

	if err := HubCleanForMerge("feature-branch"); err != nil {
		t.Errorf(".xpo/ changes should not block merge, got: %v", err)
	}
}

func TestHubCleanForMerge_ModifiedNonConflicting(t *testing.T) {
	dir, cleanup := setupHubCleanForMergeRepo(t)
	defer cleanup()

	os.WriteFile(filepath.Join(dir, "base.txt"), []byte("modified base\n"), 0644)

	if err := HubCleanForMerge("feature-branch"); err != nil {
		t.Errorf("modified file not touched by branch should not block merge, got: %v", err)
	}
}

func TestHubCleanForMerge_ModifiedConflicting(t *testing.T) {
	dir, cleanup := setupHubCleanForMergeRepo(t)
	defer cleanup()

	// feature-branch adds feature.txt — modify it locally on main to create a conflict
	os.WriteFile(filepath.Join(dir, "feature.txt"), []byte("local conflict\n"), 0644)
	runGit(t, dir, "add", "feature.txt")

	err := HubCleanForMerge("feature-branch")
	if err == nil {
		t.Fatal("expected error for conflicting modified file")
	}
	if !strings.Contains(err.Error(), "feature.txt") {
		t.Errorf("error should list the conflicting file, got: %v", err)
	}
	if !strings.Contains(err.Error(), "Do NOT stash") {
		t.Errorf("error should warn against stashing, got: %v", err)
	}
}
