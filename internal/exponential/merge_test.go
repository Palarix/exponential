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

	// Create a xpo issue (hub state lives on main)
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

	// Create a feature branch with commits, then park the hub back on main
	runGit(t, dir, "checkout", "-b", "test-abc123/feature")
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a\n"), 0644)
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "add a")
	os.WriteFile(filepath.Join(dir, "b.txt"), []byte("b\n"), 0644)
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "add b")
	runGit(t, dir, "checkout", "main")

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

// The hub is never switched: merging while it is parked on another branch
// fails before any git state changes.
func TestMergeIssue_HubNotOnBase_NoCheckout(t *testing.T) {
	dir, client := setupMergeRepo(t, "test-hub01", nil)
	runGit(t, dir, "checkout", "test-hub01/feature")

	_, err := client.MergeIssue("test-hub01", MergeOptions{Strategy: MergeStrategySquash, KeepBranch: true})
	if err == nil {
		t.Fatal("expected an error when the hub is not on main")
	}
	if !strings.Contains(err.Error(), "hub checkout is on") {
		t.Errorf("expected 'hub checkout is on' error, got: %v", err)
	}
	if got := CurrentBranch(); got != "test-hub01/feature" {
		t.Errorf("merge must not check out another branch, hub is on %q", got)
	}
}

// A branch without a worktree (made by hand, or whose worktree was removed)
// still merges from the hub.
func TestMergeIssue_BranchWithoutWorktree(t *testing.T) {
	_, client := setupMergeRepo(t, "test-nowt01", nil)
	if _, ok := FindWorktreeForBranch("test-nowt01/feature"); ok {
		t.Fatal("setup should not create a worktree")
	}

	result, err := client.MergeIssue("test-nowt01", MergeOptions{Strategy: MergeStrategySquash})
	if err != nil {
		t.Fatalf("MergeIssue failed: %v", err)
	}
	if result.MergeSHA == "" {
		t.Error("expected MergeSHA to be set")
	}
	if got := CurrentBranch(); got != "main" {
		t.Errorf("expected hub on main, got %s", got)
	}
}

// setupMergeRepo creates a git repo with an initial commit and an xpo issue in
// DOING state on main, plus a feature branch (no worktree) with one commit. The
// working directory is set to the repo, and the hub is left on main.
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

	branchName := issueID + "/feature"
	runGit(t, dir, "checkout", "-b", branchName)
	os.WriteFile(filepath.Join(dir, "feature.txt"), []byte("feature\n"), 0644)
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "add feature")
	runGit(t, dir, "checkout", "main")

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
	os.WriteFile(filepath.Join(dir, ".xpo/config.yaml"), []byte("prefix: test-\n"), 0644)

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

	cfg := &config.Config{Prefix: "test-", User: "Test <test@test.com>"}
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
	os.WriteFile(filepath.Join(dir, ".xpo/config.yaml"), []byte("prefix: test-\n"), 0644)

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

	cfg := &config.Config{Prefix: "test-", User: "Test <test@test.com>"}
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

// --- CheckMergeConflicts tests ---

func setupMergeConflictRepo(t *testing.T) (dir string, cleanup func()) {
	t.Helper()
	dir = t.TempDir()
	runGit(t, dir, "init", "-b", "main")
	runGit(t, dir, "config", "user.email", "test@test.com")
	runGit(t, dir, "config", "user.name", "Test")

	os.MkdirAll(filepath.Join(dir, ".xpo"), 0755)
	os.WriteFile(filepath.Join(dir, "shared.txt"), []byte("original\n"), 0644)
	os.WriteFile(filepath.Join(dir, "base.txt"), []byte("base\n"), 0644)
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "init")

	runGit(t, dir, "checkout", "-b", "feature-branch")
	os.WriteFile(filepath.Join(dir, "feature.txt"), []byte("feature\n"), 0644)
	os.WriteFile(filepath.Join(dir, "shared.txt"), []byte("feature change\n"), 0644)
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "feature work")
	runGit(t, dir, "checkout", "main")

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	storage.ResetHubRoot()
	return dir, func() {
		os.Chdir(origDir)
		storage.ResetHubRoot()
	}
}

func TestCheckMergeConflicts_CleanMerge(t *testing.T) {
	_, cleanup := setupMergeConflictRepo(t)
	defer cleanup()

	conflicts, err := CheckMergeConflicts("feature-branch")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(conflicts) != 0 {
		t.Errorf("expected no conflicts, got: %v", conflicts)
	}
}

func TestCheckMergeConflicts_CommittedConflict(t *testing.T) {
	dir, cleanup := setupMergeConflictRepo(t)
	defer cleanup()

	// Commit a conflicting change to shared.txt on main
	os.WriteFile(filepath.Join(dir, "shared.txt"), []byte("main change\n"), 0644)
	runGit(t, dir, "add", "shared.txt")
	runGit(t, dir, "commit", "-m", "conflicting change on main")

	conflicts, err := CheckMergeConflicts("feature-branch")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(conflicts) == 0 {
		t.Fatal("expected conflicts for divergent committed changes")
	}
	found := false
	for _, f := range conflicts {
		if f == "shared.txt" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected shared.txt in conflicts, got: %v", conflicts)
	}
}

func TestCheckMergeConflicts_DirtyTreeIgnored(t *testing.T) {
	dir, cleanup := setupMergeConflictRepo(t)
	defer cleanup()

	// Dirty the working tree with changes to a file the branch also touches —
	// but don't commit. Committed refs are clean, so mergeability should pass.
	os.WriteFile(filepath.Join(dir, "shared.txt"), []byte("uncommitted local edit\n"), 0644)

	conflicts, err := CheckMergeConflicts("feature-branch")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(conflicts) != 0 {
		t.Errorf("uncommitted changes should not cause conflicts, got: %v", conflicts)
	}
}

// --- WorktreeRequireClean tests ---

func setupWorktreeCleanRepo(t *testing.T) (hubDir string, wtDir string, cleanup func()) {
	t.Helper()
	hubDir = t.TempDir()
	runGit(t, hubDir, "init", "-b", "main")
	runGit(t, hubDir, "config", "user.email", "test@test.com")
	runGit(t, hubDir, "config", "user.name", "Test")

	os.MkdirAll(filepath.Join(hubDir, ".xpo"), 0755)
	os.WriteFile(filepath.Join(hubDir, "base.txt"), []byte("base\n"), 0644)
	runGit(t, hubDir, "add", ".")
	runGit(t, hubDir, "commit", "-m", "init")

	wtDir = filepath.Join(t.TempDir(), "wt")
	runGit(t, hubDir, "worktree", "add", "-b", "feature-branch", wtDir, "main")

	os.WriteFile(filepath.Join(wtDir, "feature.txt"), []byte("feature\n"), 0644)
	runGit(t, wtDir, "add", ".")
	runGit(t, wtDir, "commit", "-m", "add feature")

	origDir, _ := os.Getwd()
	os.Chdir(hubDir)
	storage.ResetHubRoot()
	return hubDir, wtDir, func() {
		os.Chdir(origDir)
		storage.ResetHubRoot()
	}
}

func TestWorktreeRequireClean_CleanWorktree(t *testing.T) {
	_, _, cleanup := setupWorktreeCleanRepo(t)
	defer cleanup()

	if err := WorktreeRequireClean("feature-branch"); err != nil {
		t.Errorf("expected clean worktree, got: %v", err)
	}
}

func TestWorktreeRequireClean_UntrackedFile(t *testing.T) {
	_, wtDir, cleanup := setupWorktreeCleanRepo(t)
	defer cleanup()

	os.WriteFile(filepath.Join(wtDir, "foo"), []byte("stray file\n"), 0644)

	err := WorktreeRequireClean("feature-branch")
	if err == nil {
		t.Fatal("expected error for untracked file in worktree")
	}
	if !strings.Contains(err.Error(), "foo") {
		t.Errorf("error should list the untracked file, got: %v", err)
	}
}

func TestWorktreeRequireClean_ModifiedTrackedFile(t *testing.T) {
	_, wtDir, cleanup := setupWorktreeCleanRepo(t)
	defer cleanup()

	os.WriteFile(filepath.Join(wtDir, "feature.txt"), []byte("uncommitted edit\n"), 0644)

	err := WorktreeRequireClean("feature-branch")
	if err == nil {
		t.Fatal("expected error for modified tracked file in worktree")
	}
	if !strings.Contains(err.Error(), "feature.txt") {
		t.Errorf("error should list the dirty file, got: %v", err)
	}
}

func TestWorktreeRequireClean_XpoChangesAllowed(t *testing.T) {
	_, wtDir, cleanup := setupWorktreeCleanRepo(t)
	defer cleanup()

	os.MkdirAll(filepath.Join(wtDir, ".xpo"), 0755)
	os.WriteFile(filepath.Join(wtDir, ".xpo", "issues.db"), []byte("event data\n"), 0644)

	if err := WorktreeRequireClean("feature-branch"); err != nil {
		t.Errorf(".xpo/ changes should not block, got: %v", err)
	}
}

func TestWorktreeRequireClean_NoWorktree(t *testing.T) {
	_, cleanup := setupHubCleanForMergeRepo(t)
	defer cleanup()

	if err := WorktreeRequireClean("feature-branch"); err != nil {
		t.Errorf("no worktree should not block (branch-only mode), got: %v", err)
	}
}

// --- HubRequireCleanTree tests ---

func TestHubRequireCleanTree_CleanTree(t *testing.T) {
	_, cleanup := setupHubCleanForMergeRepo(t)
	defer cleanup()

	if err := HubRequireCleanTree(); err != nil {
		t.Errorf("expected clean tree, got: %v", err)
	}
}

func TestHubRequireCleanTree_UntrackedFilesAllowed(t *testing.T) {
	dir, cleanup := setupHubCleanForMergeRepo(t)
	defer cleanup()

	os.WriteFile(filepath.Join(dir, "idea.md"), []byte("some ideas\n"), 0644)
	os.WriteFile(filepath.Join(dir, "thoughts.md"), []byte("some thoughts\n"), 0644)

	if err := HubRequireCleanTree(); err != nil {
		t.Errorf("untracked files should not block, got: %v", err)
	}
}

func TestHubRequireCleanTree_XpoChangesAllowed(t *testing.T) {
	dir, cleanup := setupHubCleanForMergeRepo(t)
	defer cleanup()

	os.WriteFile(filepath.Join(dir, ".xpo", "issues.db"), []byte("event data\n"), 0644)

	if err := HubRequireCleanTree(); err != nil {
		t.Errorf(".xpo/ changes should not block, got: %v", err)
	}
}

func TestHubRequireCleanTree_ModifiedTrackedFile(t *testing.T) {
	dir, cleanup := setupHubCleanForMergeRepo(t)
	defer cleanup()

	os.WriteFile(filepath.Join(dir, "base.txt"), []byte("modified\n"), 0644)

	err := HubRequireCleanTree()
	if err == nil {
		t.Fatal("expected error for modified tracked file")
	}
	if !strings.Contains(err.Error(), "uncommitted changes") {
		t.Errorf("error should mention uncommitted changes, got: %v", err)
	}
	if !strings.Contains(err.Error(), "base.txt") {
		t.Errorf("error should list the dirty file, got: %v", err)
	}
	if !strings.Contains(err.Error(), "Do NOT stash") {
		t.Errorf("error should warn against stashing, got: %v", err)
	}
}

// --- HubCleanForMerge (legacy, delegates to new functions) ---

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
}
