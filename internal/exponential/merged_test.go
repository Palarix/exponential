package exponential

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/palarix/exponential/internal/model"
	"github.com/palarix/exponential/internal/storage"
)

// mergedIssue merges issueID with strategy, adds a later commit on main so the
// merge commit is not HEAD, and returns the projected issue and the merge SHA.
func mergedIssue(t *testing.T, issueID string, strategy MergeStrategy) (string, *model.Issue, string) {
	t.Helper()
	dir, client := setupMergeRepo(t, issueID, nil)
	result, err := client.MergeIssue(issueID, MergeOptions{Strategy: strategy})
	if err != nil {
		t.Fatalf("MergeIssue failed: %v", err)
	}
	os.WriteFile(filepath.Join(dir, "later.txt"), []byte("later\n"), 0644)
	runGit(t, dir, "add", "later.txt")
	runGit(t, dir, "commit", "-m", "later work")

	events, _ := storage.ReadEvents()
	return dir, ProjectIssues(events)[issueID], result.MergeSHA
}

func TestLatestMerge(t *testing.T) {
	_, issue, _ := mergedIssue(t, "test-lm0001", MergeStrategySquash)
	rec, ok := LatestMerge(issue)
	if !ok {
		t.Fatal("expected a merge record")
	}
	if rec.Payload.Branch != "test-lm0001/feature" || rec.Payload.BaseSHA == "" || rec.Payload.Strategy != "squash" {
		t.Errorf("unexpected payload: %+v", rec.Payload)
	}
	if rec.By == "" || rec.At.IsZero() {
		t.Errorf("expected actor and time, got %+v", rec)
	}

	if _, ok := LatestMerge(&model.Issue{ID: "test-none01"}); ok {
		t.Error("expected no merge record for an issue without MERGE events")
	}
}

func TestFindMergeCommit(t *testing.T) {
	for _, strategy := range []MergeStrategy{MergeStrategySquash, MergeStrategyMerge, MergeStrategyFF} {
		t.Run(string(strategy), func(t *testing.T) {
			id := "test-fm" + string(strategy)[:2] + "01"
			_, issue, mergeSHA := mergedIssue(t, id, strategy)
			rec, _ := LatestMerge(issue)

			sha, err := FindMergeCommit(issue.ID, rec.Payload)
			if err != nil {
				t.Fatalf("FindMergeCommit: %v", err)
			}
			if !strings.HasPrefix(sha, mergeSHA) {
				t.Errorf("found %s, want the merge commit %s", sha, mergeSHA)
			}

			changes, err := GetMergedChanges(issue)
			if err != nil {
				t.Fatalf("GetMergedChanges: %v", err)
			}
			if !strings.Contains(changes.Diff, "feature.txt") {
				t.Errorf("expected the merged diff to contain feature.txt, got:\n%s", changes.Diff)
			}
			if strings.Contains(changes.Diff, "later.txt") {
				t.Error("merged diff must not include later commits")
			}
			if len(changes.Commits) == 0 {
				t.Fatal("expected merged commits")
			}
			for _, c := range changes.Commits {
				if strings.HasPrefix(c.Message, "Merge branch") {
					t.Errorf("merge commits must be left out, got %q", c.Message)
				}
			}
			if changes.Strategy != string(strategy) || changes.MergedBy == "" {
				t.Errorf("unexpected metadata: %+v", changes)
			}
		})
	}
}

func TestFindMergeCommit_UsesRecordedSHA(t *testing.T) {
	dir, issue, _ := mergedIssue(t, "test-fmrec01", MergeStrategySquash)
	rec, _ := LatestMerge(issue)
	head := strings.TrimSpace(gitOut(t, dir, "rev-parse", "HEAD"))
	rec.Payload.MergeSHA = head
	sha, err := FindMergeCommit(issue.ID, rec.Payload)
	if err != nil || sha != head {
		t.Errorf("expected the recorded SHA %s, got %s (%v)", head, sha, err)
	}
}

func TestFindMergeCommit_NotFound(t *testing.T) {
	_, issue, _ := mergedIssue(t, "test-fmnf01", MergeStrategySquash)
	rec, _ := LatestMerge(issue)
	if _, err := FindMergeCommit("test-other1", rec.Payload); err == nil {
		t.Error("expected an error when no commit records the issue's merge")
	}
	if _, err := GetMergedChanges(&model.Issue{ID: "test-none01"}); err == nil {
		t.Error("expected an error for an issue that was never merged")
	}
}
