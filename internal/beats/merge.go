package beats

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/palarix/beats/internal/model"
)

type MergeStrategy string

const (
	MergeStrategyMerge  MergeStrategy = "merge"
	MergeStrategySquash MergeStrategy = "squash"
	MergeStrategyFF     MergeStrategy = "ff"
)

type MergeOptions struct {
	Strategy     MergeStrategy
	DeleteBranch bool
	KeepBranch   bool
}

type MergeResult struct {
	MergeSHA string
	Messages []string
}

// MergeIssue merges the issue's branch into the default branch,
// records a MERGE event, and transitions the issue to DONE.
func (c *Client) MergeIssue(id string, opts MergeOptions) (*MergeResult, error) {
	issue, err := c.GetIssue(id)
	if err != nil {
		return nil, err
	}

	// Find the branch (local or remote)
	c.fillLocalBranchStats(issue)
	if issue.BranchStats == nil {
		return nil, fmt.Errorf("no branch found for %s", issue.ID)
	}
	if issue.BranchStats.Commits == 0 {
		return nil, fmt.Errorf("branch %s has no commits ahead of %s", issue.BranchStats.Branch, DefaultBranch())
	}

	branch := issue.BranchStats.Branch
	base := DefaultBranch()
	result := &MergeResult{}

	// Capture base SHA before merge
	baseSHA := resolveRef(base)

	// Switch to default branch
	if err := CheckoutBranch(base); err != nil {
		return nil, fmt.Errorf("failed to checkout %s: %w", base, err)
	}

	// Determine the local ref to merge — for remote branches, use the
	// remote tracking ref directly; for local branches, use as-is.
	mergeRef := branch

	// Run the merge
	var mergeErr error
	switch opts.Strategy {
	case MergeStrategySquash:
		mergeErr = runGitMerge("--squash", mergeRef)
		if mergeErr == nil {
			msg := fmt.Sprintf("%s: %s", issue.ID, issue.Title)
			mergeErr = exec.Command("git", "commit", "-m", msg).Run()
		}
	case MergeStrategyFF:
		mergeErr = runGitMerge("--ff-only", mergeRef)
	default:
		mergeErr = runGitMerge("--no-ff", "-m",
			fmt.Sprintf("Merge branch '%s'", branch), mergeRef)
	}

	if mergeErr != nil {
		exec.Command("git", "merge", "--abort").Run()
		return nil, fmt.Errorf("merge failed: %w\nResolve conflicts manually and re-run, or use a different strategy", mergeErr)
	}

	// Capture merge SHA
	mergeSHA := resolveRef("HEAD")
	result.MergeSHA = mergeSHA

	result.Messages = append(result.Messages, fmt.Sprintf("Merged %s into %s (%s)", branch, base, opts.Strategy))

	// Record MERGE event
	user := c.GetUser()
	event := model.Event{
		ID:   issue.ID,
		Type: model.EventTypeMerge,
		Payload: model.MergePayload{
			Branch:   branch,
			BaseSHA:  baseSHA,
			MergeSHA: mergeSHA,
			Strategy: string(opts.Strategy),
		},
		CreatedAt: time.Now().UTC(),
		CreatedBy: user,
	}
	if err := c.appendEvent(event); err != nil {
		return result, fmt.Errorf("merge succeeded but failed to record event: %w", err)
	}

	// Auto-commit the issues.db change
	if c.Config.AutoCommit {
		_ = exec.Command("git", "add", ".beats/issues.db").Run()
		_ = exec.Command("git", "commit", "-m", fmt.Sprintf("beats: merge %s", issue.ID)).Run()
	}

	result.Messages = append(result.Messages, fmt.Sprintf("Recorded MERGE event on %s", issue.ID))

	// Delete branch if requested
	if opts.DeleteBranch {
		deleteBranch(branch)
		result.Messages = append(result.Messages, fmt.Sprintf("Deleted branch %s", branch))
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
	exec.Command("git", "branch", "-d", name).Run()

	// Delete remote tracking branch if it exists
	if strings.HasPrefix(branch, "origin/") {
		remoteBranch := strings.TrimPrefix(branch, "origin/")
		exec.Command("git", "push", "origin", "--delete", remoteBranch).Run()
	}
}
