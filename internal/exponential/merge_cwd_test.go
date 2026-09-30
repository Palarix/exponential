package exponential

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/palarix/exponential/internal/config"
	"github.com/palarix/exponential/internal/model"
	"github.com/palarix/exponential/internal/storage"
)

// setupHubWorktreeRepo builds a hub on main with worktrees enabled and a
// linked worktree for the issue's branch under .xpo/worktrees/, mirroring
// what `xpo start` produces. The issue is created and committed on main.
// The process cwd is left at the hub.
func setupHubWorktreeRepo(t *testing.T, issueID string) (hub, wtPath, branch string, client *Client) {
	t.Helper()
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "init", "-b", "main")
	runGit(t, dir, "config", "user.email", "test@test.com")
	runGit(t, dir, "config", "user.name", "Test")

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	storage.ResetHubRoot()
	t.Cleanup(func() {
		os.Chdir(origDir)
		storage.ResetHubRoot()
	})

	os.MkdirAll(filepath.Join(dir, ".xpo"), 0755)
	os.WriteFile(filepath.Join(dir, ".xpo/config.yaml"), []byte("prefix: test-\nworktrees: true\n"), 0644)
	os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(".xpo/worktrees/\n"), 0644)
	os.WriteFile(filepath.Join(dir, "base.txt"), []byte("base\n"), 0644)
	os.WriteFile(filepath.Join(dir, "shared.txt"), []byte("original\n"), 0644)

	storage.AppendEvent(model.Event{
		ID:        issueID,
		Type:      model.EventTypeCreate,
		Payload:   model.CreatePayload{Title: "Worktree issue", Status: "DOING"},
		CreatedAt: time.Now().UTC(),
		CreatedBy: "Test <test@test.com>",
	})
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "init with issue")

	branch = issueID + "/feature"
	wtPath = filepath.Join(dir, ".xpo", "worktrees", issueID)
	runGit(t, dir, "worktree", "add", "-b", branch, wtPath)
	os.WriteFile(filepath.Join(wtPath, "feature.txt"), []byte("feature\n"), 0644)
	runGit(t, wtPath, "add", "feature.txt")
	runGit(t, wtPath, "commit", "-m", "add feature")

	cfg := &config.Config{Prefix: "test-", User: "Test <test@test.com>", Worktrees: true}
	return dir, wtPath, branch, NewClient(cfg)
}

// chdirWorktree moves the process cwd into the issue's worktree, which is
// where an MCP server launched from a linked session runs.
func chdirWorktree(t *testing.T, wtPath string) {
	t.Helper()
	if err := os.Chdir(wtPath); err != nil {
		t.Fatal(err)
	}
	storage.ResetHubRoot()
}

func gitOut(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// writeHook installs an executable git hook in the hub's shared hooks dir.
func writeHook(t *testing.T, hub, name, body string) {
	t.Helper()
	path := filepath.Join(hub, ".git", "hooks", name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0755); err != nil {
		t.Fatal(err)
	}
}

// foreignEventLine returns a serialized event for another issue, standing
// in for an event another agent wrote to the hub's issues.db.
func foreignEventLine(t *testing.T, id, title string) string {
	t.Helper()
	b, err := json.Marshal(model.Event{
		ID:        id,
		Type:      model.EventTypeCreate,
		Payload:   model.CreatePayload{Title: title, Status: "BACKLOG"},
		CreatedAt: time.Now().UTC(),
		CreatedBy: "Other Agent <other@test.com>",
	})
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func readIssuesDB(t *testing.T, hub string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(hub, ".xpo", "issues.db"))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func hasMergeEvent(t *testing.T, id string) bool {
	t.Helper()
	events, err := storage.ReadEvents()
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range events {
		if e.ID == id && e.Type == model.EventTypeMerge {
			return true
		}
	}
	return false
}

func issueStatus(t *testing.T, id string) model.IssueStatus {
	t.Helper()
	events, err := storage.ReadEvents()
	if err != nil {
		t.Fatal(err)
	}
	issue := ProjectIssues(events)[id]
	if issue == nil {
		t.Fatalf("issue %s not found", id)
	}
	return issue.Status
}

// assertHubClean checks that nothing is staged and no merge is in progress.
func assertHubClean(t *testing.T, hub string) {
	t.Helper()
	out, err := exec.Command("git", "-C", hub, "status", "--porcelain").Output()
	if err != nil {
		t.Fatalf("git status failed: %v", err)
	}
	for _, line := range strings.Split(string(out), "\n") {
		if len(line) >= 2 && line[0] != ' ' && line[0] != '?' {
			t.Errorf("expected nothing staged in hub, found %q", line)
		}
	}
	if _, err := os.Stat(filepath.Join(hub, ".git", "MERGE_HEAD")); err == nil {
		t.Error("expected no merge in progress (MERGE_HEAD exists)")
	}
}

func TestMergeIssue_FromInsideWorktree(t *testing.T) {
	for _, strategy := range []MergeStrategy{MergeStrategySquash, MergeStrategyMerge, MergeStrategyFF} {
		t.Run(string(strategy), func(t *testing.T) {
			hub, wtPath, branch, client := setupHubWorktreeRepo(t, "test-cwd01")
			chdirWorktree(t, wtPath)

			result, err := client.MergeIssue("test-cwd01", MergeOptions{
				Strategy:     strategy,
				DeleteBranch: true,
			})
			if err != nil {
				t.Fatalf("MergeIssue from inside worktree failed: %v", err)
			}
			if result.MergeSHA == "" {
				t.Error("expected MergeSHA to be set")
			}

			if gitOut(t, hub, "show", "main:feature.txt") != "feature" {
				t.Error("expected feature.txt on main")
			}
			committedDB := gitOut(t, hub, "show", "main:.xpo/issues.db")
			if !strings.Contains(committedDB, `"type":"MERGE"`) {
				t.Errorf("expected MERGE event committed on main, got:\n%s", committedDB)
			}

			if strategy != MergeStrategyFF {
				// Code and bookkeeping land in one commit.
				changed := gitOut(t, hub, "diff", "--name-only", "main^1", "main")
				for _, f := range []string{"feature.txt", ".xpo/issues.db"} {
					if !strings.Contains(changed, f) {
						t.Errorf("expected %s in the merge commit, changed files:\n%s", f, changed)
					}
				}
			}

			if got := issueStatus(t, "test-cwd01"); got != model.StatusDone {
				t.Errorf("expected DONE, got %s", got)
			}
			if _, err := os.Stat(wtPath); !os.IsNotExist(err) {
				t.Errorf("expected worktree %s to be removed", wtPath)
			}
			if out := gitOut(t, hub, "branch", "--list", branch); out != "" {
				t.Errorf("expected branch %s deleted, got %q", branch, out)
			}
			assertHubClean(t, hub)
		})
	}
}

func TestMergeIssue_CommitHookFails_WritesNoEvents(t *testing.T) {
	for _, strategy := range []MergeStrategy{MergeStrategySquash, MergeStrategyMerge} {
		t.Run(string(strategy), func(t *testing.T) {
			hub, wtPath, branch, client := setupHubWorktreeRepo(t, "test-hook01")

			// An uncommitted event already on the hub before the merge starts.
			preexisting := foreignEventLine(t, "test-other01", "Pre-existing")
			f, _ := os.OpenFile(filepath.Join(hub, ".xpo", "issues.db"), os.O_APPEND|os.O_WRONLY, 0644)
			f.WriteString(preexisting + "\n")
			f.Close()

			// Another agent appends an event while the commit is running,
			// then the hook rejects the commit.
			concurrent := foreignEventLine(t, "test-other02", "Concurrent")
			writeHook(t, hub, "pre-commit",
				"printf '%s\\n' '"+concurrent+"' >> \"$(git rev-parse --show-toplevel)/.xpo/issues.db\"\n"+
					"echo 'hook says no' >&2\nexit 1")

			mainBefore := gitOut(t, hub, "rev-parse", "main")
			chdirWorktree(t, wtPath)

			_, err := client.MergeIssue("test-hook01", MergeOptions{Strategy: strategy, DeleteBranch: true})
			if err == nil {
				t.Fatal("expected error when pre-commit hook fails")
			}
			msg := err.Error()
			if !strings.Contains(msg, "hook says no") {
				t.Errorf("expected hook output in error, got: %v", err)
			}
			if !strings.Contains(msg, "nothing was merged") {
				t.Errorf("expected 'nothing was merged' in error, got: %v", err)
			}
			if strings.Contains(msg, "Resolve the conflict") {
				t.Errorf("expected no conflict advice for a hook failure, got: %v", err)
			}

			if hasMergeEvent(t, "test-hook01") {
				t.Error("expected no MERGE event after a failed commit")
			}
			if got := issueStatus(t, "test-hook01"); got != model.StatusDoing {
				t.Errorf("expected issue to stay DOING, got %s", got)
			}

			db := readIssuesDB(t, hub)
			if !strings.Contains(db, preexisting) {
				t.Error("pre-existing uncommitted event was lost")
			}
			if !strings.Contains(db, concurrent) {
				t.Error("event appended concurrently by another agent was lost")
			}

			if got := gitOut(t, hub, "rev-parse", "main"); got != mainBefore {
				t.Errorf("expected main unchanged at %s, got %s", mainBefore, got)
			}
			assertHubClean(t, hub)
			if _, err := os.Stat(wtPath); err != nil {
				t.Errorf("expected worktree to survive a failed merge: %v", err)
			}
			if out := gitOut(t, hub, "branch", "--list", branch); out == "" {
				t.Errorf("expected branch %s to survive a failed merge", branch)
			}
		})
	}
}

func TestMergeIssue_ConcurrentAppendDuringCommit_IsCommitted(t *testing.T) {
	hub, wtPath, _, client := setupHubWorktreeRepo(t, "test-hook02")

	preexisting := foreignEventLine(t, "test-other01", "Pre-existing")
	f, _ := os.OpenFile(filepath.Join(hub, ".xpo", "issues.db"), os.O_APPEND|os.O_WRONLY, 0644)
	f.WriteString(preexisting + "\n")
	f.Close()

	concurrent := foreignEventLine(t, "test-other02", "Concurrent")
	writeHook(t, hub, "pre-commit",
		"printf '%s\\n' '"+concurrent+"' >> \"$(git rev-parse --show-toplevel)/.xpo/issues.db\"\nexit 0")

	chdirWorktree(t, wtPath)
	if _, err := client.MergeIssue("test-hook02", MergeOptions{Strategy: MergeStrategySquash}); err != nil {
		t.Fatalf("MergeIssue failed: %v", err)
	}

	committed := gitOut(t, hub, "show", "main:.xpo/issues.db")
	for name, line := range map[string]string{"pre-existing": preexisting, "concurrent": concurrent} {
		if !strings.Contains(committed, line) {
			t.Errorf("expected %s event in the merge commit", name)
		}
	}
	if !strings.Contains(committed, `"type":"MERGE"`) {
		t.Error("expected MERGE event in the merge commit")
	}
	if got := issueStatus(t, "test-hook02"); got != model.StatusDone {
		t.Errorf("expected DONE, got %s", got)
	}
}

func TestMergeIssue_ConflictFromInsideWorktree(t *testing.T) {
	hub, wtPath, _, client := setupHubWorktreeRepo(t, "test-cfl01")

	os.WriteFile(filepath.Join(wtPath, "shared.txt"), []byte("feature version\n"), 0644)
	runGit(t, wtPath, "commit", "-am", "feature change")
	os.WriteFile(filepath.Join(hub, "shared.txt"), []byte("main version\n"), 0644)
	runGit(t, hub, "commit", "-am", "conflicting change on main")

	chdirWorktree(t, wtPath)
	_, err := client.MergeIssue("test-cfl01", MergeOptions{Strategy: MergeStrategySquash})
	if err == nil {
		t.Fatal("expected conflict error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "Resolve the conflict") || !strings.Contains(msg, "shared.txt") {
		t.Errorf("expected conflict advice naming shared.txt, got: %v", err)
	}
	if hasMergeEvent(t, "test-cfl01") {
		t.Error("expected no MERGE event after a conflict")
	}
	if got := issueStatus(t, "test-cfl01"); got != model.StatusDoing {
		t.Errorf("expected issue to stay DOING, got %s", got)
	}
	assertHubClean(t, hub)
}

func TestMergeIssue_BookkeepingCommitFails_MergeSucceedsWithWarning(t *testing.T) {
	hub, wtPath, _, client := setupHubWorktreeRepo(t, "test-bk01")

	// --no-verify does not skip prepare-commit-msg; fail only on the amend
	// (source "commit"), not on the code commit (source "message").
	writeHook(t, hub, "prepare-commit-msg",
		"if [ \"$2\" = commit ]; then echo 'bookkeeping blocked' >&2; exit 1; fi\nexit 0")

	chdirWorktree(t, wtPath)
	result, err := client.MergeIssue("test-bk01", MergeOptions{Strategy: MergeStrategySquash})
	if err != nil {
		t.Fatalf("expected merge to succeed when only the bookkeeping amend fails, got: %v", err)
	}

	warned := false
	for _, m := range result.Messages {
		if strings.Contains(m, "Warning") && strings.Contains(m, "bookkeeping blocked") {
			warned = true
		}
	}
	if !warned {
		t.Errorf("expected a warning carrying the amend failure, got messages: %v", result.Messages)
	}

	if gitOut(t, hub, "show", "main:feature.txt") != "feature" {
		t.Error("expected the code commit on main")
	}
	if strings.Contains(gitOut(t, hub, "show", "main:.xpo/issues.db"), `"type":"MERGE"`) {
		t.Error("expected MERGE event not to be committed when the amend fails")
	}
	if !hasMergeEvent(t, "test-bk01") {
		t.Error("expected MERGE event recorded in the working issues.db")
	}
	if got := issueStatus(t, "test-bk01"); got != model.StatusDone {
		t.Errorf("expected DONE, got %s", got)
	}
	assertHubClean(t, hub)
}
