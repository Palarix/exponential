package exponential

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/palarix/exponential/internal/model"
	"github.com/palarix/exponential/internal/storage"
)

type MergeStrategy string

const (
	MergeStrategyMerge  MergeStrategy = "merge"
	MergeStrategySquash MergeStrategy = "squash"
	MergeStrategyFF     MergeStrategy = "ff"
)

type MergeOptions struct {
	Strategy      MergeStrategy
	CommitMessage string
	DeleteBranch  bool
	KeepBranch    bool
}

type MergeResult struct {
	MergeSHA string
	Messages []string
}

// MergeIssue merges the issue's branch into the default branch,
// records a MERGE event, and transitions the issue to DONE.
func (c *Client) MergeIssue(id string, opts MergeOptions) (*MergeResult, error) {
	if c.local == nil {
		return nil, ErrLocalOnly
	}

	issue, err := c.Transport.GetIssue(id)
	if err != nil {
		return nil, err
	}

	c.FillLocalBranchStats(issue)
	if issue.BranchStats == nil {
		return nil, fmt.Errorf("no branch found for %s", issue.ID)
	}
	if issue.BranchStats.Commits == 0 {
		return nil, fmt.Errorf("branch %s has no commits ahead of %s", issue.BranchStats.Branch, DefaultBranch())
	}

	branch := issue.BranchStats.Branch
	base := DefaultBranch()
	result := &MergeResult{}

	gitErr := WithGitLock(func() error {
		// Capture base SHA before merge
		baseSHA := resolveRef(base)

		// Switch to default branch
		if err := CheckoutBranch(base); err != nil {
			return fmt.Errorf("failed to checkout %s: %w", base, err)
		}

		// Determine the local ref to merge — for remote branches, use the
		// remote tracking ref directly; for local branches, use as-is.
		mergeRef := branch

		// Record MERGE event before committing so it lands in the same commit
		user := c.Transport.GetUser()
		event := model.Event{
			ID:   issue.ID,
			Type: model.EventTypeMerge,
			Payload: model.MergePayload{
				Branch:   branch,
				BaseSHA:  baseSHA,
				MergeSHA: "", // filled after commit
				Strategy: string(opts.Strategy),
			},
			CreatedAt: time.Now().UTC(),
			CreatedBy: user,
		}

		// Default commit messages per strategy
		commitMsg := opts.CommitMessage
		if commitMsg == "" {
			switch opts.Strategy {
			case MergeStrategySquash:
				commitMsg = fmt.Sprintf("%s: %s", issue.ID, issue.Title)
			case MergeStrategyFF:
				commitMsg = fmt.Sprintf("xpo: merge %s", issue.ID)
			default:
				commitMsg = fmt.Sprintf("Merge branch '%s'", branch)
			}
		}

		// Run the merge + include issues.db in the same commit
		var mergeErr error
		switch opts.Strategy {
		case MergeStrategySquash:
			mergeErr = runGitMerge("--squash", mergeRef)
		case MergeStrategyFF:
			mergeErr = runGitMerge("--ff-only", mergeRef)
		default:
			mergeErr = runGitMerge("--no-ff", "-m", commitMsg, mergeRef)
		}

		if mergeErr == nil {
			// Build DONE transition events against pre-merge state
			// (ProjectIssues infers DONE from MERGE, so we must read
			// state before appending the MERGE event).
			preEvents, _ := storage.ReadEvents()
			preMergeState := ProjectIssues(preEvents)

			doneStatus := string(model.StatusDone)
			doneEvents, doneMessages, err := c.local.buildUpdate(
				issue.ID, model.UpdatePayload{Status: &doneStatus}, preMergeState)
			if err != nil {
				return fmt.Errorf("failed to build DONE transition: %w", err)
			}

			// Append MERGE event, then DONE transition events
			if err := c.local.appendEvent(event); err != nil {
				return fmt.Errorf("failed to record merge event: %w", err)
			}
			for _, evt := range doneEvents {
				if err := c.local.appendEvent(evt); err != nil {
					return fmt.Errorf("failed to apply DONE transition: %w", err)
				}
			}
			result.Messages = append(result.Messages, doneMessages...)

			// Commit everything together
			exec.Command("git", "add", ".xpo/issues.db").Run()
			switch opts.Strategy {
			case MergeStrategySquash, MergeStrategyFF:
				mergeErr = exec.Command("git", "commit", "-m", commitMsg).Run()
			default:
				exec.Command("git", "commit", "--amend", "--no-edit").Run()
			}
		}

		if mergeErr != nil {
			exec.Command("git", "merge", "--abort").Run()
			return fmt.Errorf("merge failed: %w\nResolve conflicts manually and re-run, or use a different strategy", mergeErr)
		}

		mergeSHA := resolveRef("HEAD")
		result.MergeSHA = mergeSHA
		result.Messages = append(result.Messages, fmt.Sprintf("Merged %s into %s (%s)", branch, base, opts.Strategy))

		// Delete branch if requested
		if opts.DeleteBranch {
			deleteBranch(branch)
			result.Messages = append(result.Messages, fmt.Sprintf("Deleted branch %s", branch))
		}

		return nil
	})
	if gitErr != nil {
		return nil, gitErr
	}

	return result, nil
}

func IsWorkingTreeClean() bool {
	out, err := exec.Command("git", "status", "--porcelain").Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == ""
}

func resolveRef(ref string) string {
	out, err := exec.Command("git", "rev-parse", "--short", ref).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func runGitMerge(args ...string) error {
	cmd := exec.Command("git", append([]string{"merge"}, args...)...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}

func deleteBranch(branch string) {
	// Delete local branch
	name := branch
	if strings.Contains(branch, "/") {
		parts := strings.SplitN(branch, "/", 2)
		if len(parts) == 2 && (parts[0] == "origin" || strings.Contains(parts[0], "/")) {
			name = parts[1]
		}
	}
	exec.Command("git", "branch", "-D", name).Run()

	// Delete remote tracking branch if it exists
	if strings.HasPrefix(branch, "origin/") {
		remoteBranch := strings.TrimPrefix(branch, "origin/")
		exec.Command("git", "push", "origin", "--delete", remoteBranch).Run()
	}
}
