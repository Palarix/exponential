package exponential

import (
	"os"
	"os/exec"
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
	storage.ResetHubRoot()
	defer func() {
		os.Chdir(origDir)
		storage.ResetHubRoot()
	}()

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

// setupMergeRepo creates a git repo with an initial commit on main, a feature
// branch with one commit, and an xpo issue in DOING state. The working directory
// is set to the repo. Returns the dir, a cleanup function, and the issue ID.
// The caller is left on the feature branch.
func setupMergeRepo(t *testing.T, issueID string, cfgOverrides *config.Config) (string, *Client) {
	t.Helper()
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

	branchName := issueID + "/feature"
	runGit(t, dir, "checkout", "-b", branchName)
	os.WriteFile(filepath.Join(dir, "feature.txt"), []byte("feature\n"), 0644)
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "add feature")

	storage.ResetHubRoot()
	evt := model.Event{
		ID:   issueID,
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

	cfg := &config.Config{Prefix: "test-", User: "Test <test@test.com>"}
	if cfgOverrides != nil {
		*cfg = *cfgOverrides
		if cfg.Prefix == "" {
			cfg.Prefix = "test-"
		}
		if cfg.User == "" {
			cfg.User = "Test <test@test.com>"
		}
	}
	return dir, NewClient(cfg)
}

func TestMergeIssue_MergeStrategy(t *testing.T) {
	_, client := setupMergeRepo(t, "test-merge01", nil)

	result, err := client.MergeIssue("test-merge01", MergeOptions{
		Strategy:   MergeStrategyMerge,
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

	// Verify merge commit has two parents (--no-ff)
	out, err := exec.Command("git", "cat-file", "-p", "HEAD").Output()
	if err != nil {
		t.Fatalf("git cat-file failed: %v", err)
	}
	parentCount := strings.Count(string(out), "\nparent ")
	if parentCount < 2 {
		t.Errorf("expected merge commit with 2 parents, got %d", parentCount)
	}

	events, _ := storage.ReadEvents()
	issues := ProjectIssues(events)
	issue := issues["test-merge01"]
	if issue == nil {
		t.Fatal("issue not found after merge")
	}
	if issue.Status != model.StatusDone {
		t.Errorf("expected DONE, got %s", issue.Status)
	}
}

func TestMergeIssue_FFStrategy(t *testing.T) {
	// FF needs no divergence, so we stay on the feature branch and let
	// MergeIssue auto-checkout main.
	_, client := setupMergeRepo(t, "test-ff01", nil)

	result, err := client.MergeIssue("test-ff01", MergeOptions{
		Strategy:   MergeStrategyFF,
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

	// Verify no merge commit (should be a fast-forward with a single commit on top)
	out, err := exec.Command("git", "cat-file", "-p", "HEAD").Output()
	if err != nil {
		t.Fatalf("git cat-file failed: %v", err)
	}
	parentCount := strings.Count(string(out), "\nparent ")
	if parentCount != 1 {
		t.Errorf("expected single-parent commit (ff), got %d parents", parentCount)
	}
}

func TestMergeIssue_FFStrategy_Diverged(t *testing.T) {
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

	// Create issue on main so it's visible from both branches
	storage.ResetHubRoot()
	evt := model.Event{
		ID:   "test-ffdiv01",
		Type: model.EventTypeCreate,
		Payload: model.CreatePayload{
			Title:  "FF diverged issue",
			Status: "DOING",
		},
		CreatedBy: "Test <test@test.com>",
	}
	storage.AppendEvent(evt)
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "init with issue")

	// Create feature branch with a commit
	runGit(t, dir, "checkout", "-b", "test-ffdiv01/feature")
	os.WriteFile(filepath.Join(dir, "feature.txt"), []byte("feature\n"), 0644)
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "add feature")

	// Diverge main
	runGit(t, dir, "checkout", "main")
	os.WriteFile(filepath.Join(dir, "main-only.txt"), []byte("main\n"), 0644)
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "diverge main")
	storage.ResetHubRoot()

	cfg := &config.Config{Prefix: "test-", User: "Test <test@test.com>"}
	client := NewClient(cfg)

	_, err := client.MergeIssue("test-ffdiv01", MergeOptions{
		Strategy:   MergeStrategyFF,
		KeepBranch: true,
	})
	if err == nil {
		t.Fatal("expected error for FF merge on diverged branches")
	}
	if !strings.Contains(err.Error(), "merge failed") {
		t.Errorf("expected 'merge failed' error, got: %v", err)
	}

	// Verify we're still on main and tree is clean (abort succeeded)
	branch := CurrentBranch()
	if branch != "main" {
		t.Errorf("expected to be on main after failed merge, got %s", branch)
	}
}

func TestMergeIssue_CustomCommitMessage(t *testing.T) {
	_, client := setupMergeRepo(t, "test-msg01", nil)

	customMsg := "feat: my custom merge message"
	result, err := client.MergeIssue("test-msg01", MergeOptions{
		Strategy:      MergeStrategySquash,
		CommitMessage: customMsg,
		KeepBranch:    true,
	})
	if err != nil {
		t.Fatalf("MergeIssue failed: %v", err)
	}
	if result.MergeSHA == "" {
		t.Error("expected MergeSHA to be set")
	}

	out, err := exec.Command("git", "log", "-1", "--format=%s").Output()
	if err != nil {
		t.Fatalf("git log failed: %v", err)
	}
	subject := strings.TrimSpace(string(out))
	if subject != customMsg {
		t.Errorf("expected commit message %q, got %q", customMsg, subject)
	}
}

func TestMergeIssue_NoBranch(t *testing.T) {
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

	storage.ResetHubRoot()
	evt := model.Event{
		ID:   "test-nobranch",
		Type: model.EventTypeCreate,
		Payload: model.CreatePayload{
			Title:  "No branch issue",
			Status: "DOING",
		},
		CreatedBy: "Test <test@test.com>",
	}
	storage.AppendEvent(evt)
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "add issue")

	cfg := &config.Config{Prefix: "test-", User: "Test <test@test.com>"}
	client := NewClient(cfg)

	_, err := client.MergeIssue("test-nobranch", MergeOptions{
		Strategy:   MergeStrategySquash,
		KeepBranch: true,
	})
	if err == nil {
		t.Fatal("expected error for issue with no branch")
	}
	if !strings.Contains(err.Error(), "no branch found") {
		t.Errorf("expected 'no branch found' error, got: %v", err)
	}
}

func TestMergeIssue_ZeroCommits(t *testing.T) {
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

	// Create a branch with the issue ID but no additional commits
	runGit(t, dir, "checkout", "-b", "test-zerocmt/feature")
	storage.ResetHubRoot()
	evt := model.Event{
		ID:   "test-zerocmt",
		Type: model.EventTypeCreate,
		Payload: model.CreatePayload{
			Title:  "Zero commits issue",
			Status: "DOING",
		},
		CreatedBy: "Test <test@test.com>",
	}
	storage.AppendEvent(evt)
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "add issue")

	// The issue event commit is on the branch, but it only touches .xpo/ —
	// BranchStats.Commits counts commits ahead of main, so this is 1.
	// To get zero, we need to also commit it on main.
	runGit(t, dir, "checkout", "main")
	runGit(t, dir, "merge", "test-zerocmt/feature")
	runGit(t, dir, "checkout", "test-zerocmt/feature")
	storage.ResetHubRoot()

	cfg := &config.Config{Prefix: "test-", User: "Test <test@test.com>"}
	client := NewClient(cfg)

	_, err := client.MergeIssue("test-zerocmt", MergeOptions{
		Strategy:   MergeStrategySquash,
		KeepBranch: true,
	})
	if err == nil {
		t.Fatal("expected error for branch with zero commits")
	}
	if !strings.Contains(err.Error(), "no commits ahead") {
		t.Errorf("expected 'no commits ahead' error, got: %v", err)
	}
}

func TestMergeIssue_Conflict(t *testing.T) {
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
	os.WriteFile(filepath.Join(dir, "shared.txt"), []byte("original\n"), 0644)

	// Create issue on main so it's visible after checkout
	storage.ResetHubRoot()
	evt := model.Event{
		ID:   "test-conflict01",
		Type: model.EventTypeCreate,
		Payload: model.CreatePayload{
			Title:  "Conflict issue",
			Status: "DOING",
		},
		CreatedBy: "Test <test@test.com>",
	}
	storage.AppendEvent(evt)
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "init with issue")

	// Feature branch modifies shared.txt
	runGit(t, dir, "checkout", "-b", "test-conflict01/feature")
	os.WriteFile(filepath.Join(dir, "shared.txt"), []byte("feature version\n"), 0644)
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "feature change")

	// Main also modifies shared.txt — creates a conflict
	runGit(t, dir, "checkout", "main")
	os.WriteFile(filepath.Join(dir, "shared.txt"), []byte("main version\n"), 0644)
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "conflicting change on main")
	storage.ResetHubRoot()

	cfg := &config.Config{Prefix: "test-", User: "Test <test@test.com>"}
	client := NewClient(cfg)

	_, err := client.MergeIssue("test-conflict01", MergeOptions{
		Strategy:   MergeStrategySquash,
		KeepBranch: true,
	})
	if err == nil {
		t.Fatal("expected error for conflicting merge")
	}
	errMsg := err.Error()
	if !strings.Contains(errMsg, "merge failed") {
		t.Errorf("expected 'merge failed' error, got: %v", err)
	}
	if !strings.Contains(errMsg, "Do NOT stash") {
		t.Errorf("expected stash warning in error, got: %v", err)
	}
	if !strings.Contains(errMsg, "shared.txt") {
		t.Errorf("expected conflicting filename 'shared.txt' in error, got: %v", err)
	}

	// Verify we're on main
	branch := CurrentBranch()
	if branch != "main" {
		t.Errorf("expected to be on main after failed merge, got %s", branch)
	}

	// Verify merge conflict is resolved (reset --merge cleans up squash index)
	statusOut, _ := exec.Command("git", "status", "--porcelain").Output()
	for _, line := range strings.Split(string(statusOut), "\n") {
		if len(line) >= 3 {
			code := line[:2]
			file := strings.TrimSpace(line[3:])
			if code == "UU" || code == "AA" || code == "DD" {
				t.Errorf("expected no merge conflicts after abort, found %s %s", code, file)
			}
			if file == "shared.txt" && (code[0] == 'M' || code[1] == 'M') {
				t.Errorf("expected shared.txt to be clean after abort, got status %s", code)
			}
		}
	}
}

func TestMergeIssue_DeleteBranch(t *testing.T) {
	_, client := setupMergeRepo(t, "test-delbr01", nil)

	branchName := "test-delbr01/feature"
	result, err := client.MergeIssue("test-delbr01", MergeOptions{
		Strategy:     MergeStrategySquash,
		DeleteBranch: true,
	})
	if err != nil {
		t.Fatalf("MergeIssue failed: %v", err)
	}
	if result.MergeSHA == "" {
		t.Error("expected MergeSHA to be set")
	}

	// Verify branch was deleted
	out, err := exec.Command("git", "branch", "--list", branchName).Output()
	if err != nil {
		t.Fatalf("git branch --list failed: %v", err)
	}
	if strings.TrimSpace(string(out)) != "" {
		t.Errorf("expected branch %s to be deleted, but it still exists", branchName)
	}
}

func TestMergeIssue_WorktreeCleanup(t *testing.T) {
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
	os.WriteFile(filepath.Join(dir, ".xpo/config.yaml"), []byte("prefix: test-\nworktrees: true\n"), 0644)

	os.WriteFile(filepath.Join(dir, "base.txt"), []byte("base\n"), 0644)
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "init")

	// Create a linked worktree for the feature branch
	branchName := "test-wt01/feature"
	wtPath := filepath.Join(dir, ".worktrees", "test-wt01")
	runGit(t, dir, "worktree", "add", "-b", branchName, wtPath)

	// Add a commit in the worktree
	os.WriteFile(filepath.Join(wtPath, "feature.txt"), []byte("feature\n"), 0644)
	runGit(t, wtPath, "add", ".")
	runGit(t, wtPath, "commit", "-m", "add feature")

	// Create the issue
	storage.ResetHubRoot()
	evt := model.Event{
		ID:   "test-wt01",
		Type: model.EventTypeCreate,
		Payload: model.CreatePayload{
			Title:  "Worktree issue",
			Status: "DOING",
		},
		CreatedBy: "Test <test@test.com>",
	}
	storage.AppendEvent(evt)
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "add issue")

	cfg := &config.Config{Prefix: "test-", User: "Test <test@test.com>", Worktrees: true}
	client := NewClient(cfg)

	result, err := client.MergeIssue("test-wt01", MergeOptions{
		Strategy:   MergeStrategySquash,
		KeepBranch: true,
	})
	if err != nil {
		t.Fatalf("MergeIssue failed: %v", err)
	}
	if result.MergeSHA == "" {
		t.Error("expected MergeSHA to be set")
	}

	// Verify worktree was removed
	if _, err := os.Stat(wtPath); !os.IsNotExist(err) {
		t.Errorf("expected worktree %s to be removed", wtPath)
	}

	// Verify worktree is no longer listed
	entries, _ := WorktreeList()
	for _, e := range entries {
		if e.Branch == branchName {
			t.Errorf("worktree for %s still listed after merge", branchName)
		}
	}
}

func TestMergeIssue_WorktreeHubWrongBranch(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init", "-b", "main")
	runGit(t, dir, "config", "user.email", "test@test.com")
	runGit(t, dir, "config", "user.name", "Test")

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	t.Cleanup(func() {
		// Clean up: go back to main before worktree prune
		exec.Command("git", "-C", dir, "checkout", "main").Run()
		exec.Command("git", "-C", dir, "worktree", "prune").Run()
		os.Chdir(origDir)
		storage.ResetHubRoot()
	})

	os.MkdirAll(filepath.Join(dir, ".xpo"), 0755)
	os.WriteFile(filepath.Join(dir, ".xpo/config.yaml"), []byte("prefix: test-\nworktrees: true\n"), 0644)

	os.WriteFile(filepath.Join(dir, "base.txt"), []byte("base\n"), 0644)
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "init")

	// Create a linked worktree for the feature branch
	branchName := "test-wthub01/feature"
	wtPath := filepath.Join(dir, ".worktrees", "test-wthub01")
	runGit(t, dir, "worktree", "add", "-b", branchName, wtPath)

	// Add a commit in the worktree
	os.WriteFile(filepath.Join(wtPath, "feature.txt"), []byte("feature\n"), 0644)
	runGit(t, wtPath, "add", ".")
	runGit(t, wtPath, "commit", "-m", "add feature")

	// Create the issue
	storage.ResetHubRoot()
	evt := model.Event{
		ID:   "test-wthub01",
		Type: model.EventTypeCreate,
		Payload: model.CreatePayload{
			Title:  "Worktree hub issue",
			Status: "DOING",
		},
		CreatedBy: "Test <test@test.com>",
	}
	storage.AppendEvent(evt)
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "add issue")

	// Move hub off main to simulate wrong branch
	runGit(t, dir, "checkout", "-b", "some-other-branch")

	cfg := &config.Config{Prefix: "test-", User: "Test <test@test.com>", Worktrees: true}
	client := NewClient(cfg)

	_, err := client.MergeIssue("test-wthub01", MergeOptions{
		Strategy:   MergeStrategySquash,
		KeepBranch: true,
	})
	if err == nil {
		t.Fatal("expected error when hub is on wrong branch in worktree mode")
	}
	if !strings.Contains(err.Error(), "hub checkout is on") {
		t.Errorf("expected 'hub checkout' error, got: %v", err)
	}
}

func TestMergeIssue_FFPreservesMainIssuesDB(t *testing.T) {
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

	// Create a feature branch (no divergence so FF is possible)
	runGit(t, dir, "checkout", "-b", "test-ffdb01/feature")

	evtBranch := model.Event{
		ID:   "test-ffdb01",
		Type: model.EventTypeCreate,
		Payload: model.CreatePayload{
			Title:  "FF issue",
			Status: "DOING",
		},
		CreatedBy: "Test <test@test.com>",
	}
	storage.ResetHubRoot()
	storage.AppendEvent(evtBranch)
	os.WriteFile(filepath.Join(dir, "feature.txt"), []byte("feature\n"), 0644)
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "feature work")

	storage.ResetHubRoot()

	result, err := client.MergeIssue("test-ffdb01", MergeOptions{
		Strategy:   MergeStrategyFF,
		KeepBranch: true,
	})
	if err != nil {
		t.Fatalf("MergeIssue failed: %v", err)
	}
	if result.MergeSHA == "" {
		t.Error("expected MergeSHA to be set")
	}

	// Verify all events survived
	storage.ResetHubRoot()
	events, err := storage.ReadEvents()
	if err != nil {
		t.Fatalf("ReadEvents failed: %v", err)
	}

	hasBackgroundCreate := false
	hasBranchCreate := false
	hasMerge := false
	for _, e := range events {
		if e.ID == "test-000000" && e.Type == model.EventTypeCreate {
			hasBackgroundCreate = true
		}
		if e.ID == "test-ffdb01" && e.Type == model.EventTypeCreate {
			hasBranchCreate = true
		}
		if e.ID == "test-ffdb01" && e.Type == model.EventTypeMerge {
			hasMerge = true
		}
	}

	if !hasBackgroundCreate {
		t.Error("lost the initial create event from main")
	}
	if !hasBranchCreate {
		t.Error("lost the create event from the feature branch")
	}
	if !hasMerge {
		t.Error("expected MERGE event in issue history")
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
