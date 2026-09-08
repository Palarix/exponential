package exponential

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
	useWorktrees := c.Config.Worktrees && CheckGitRepo()
	if useWorktrees {
		if _, ok := FindWorktreeForBranch(branch); !ok {
			useWorktrees = false
		}
	}
	result := &MergeResult{}

	gitErr := WithGitLock(func() error {
		// Capture base SHA before merge
		baseSHA := resolveRef(base)

		if useWorktrees {
			current := HubBranch()
			if current != base {
				return fmt.Errorf("hub checkout is on %s, not %s — park the hub on the default branch before merging", current, base)
			}
		} else {
			if err := CheckoutBranch(base); err != nil {
				return fmt.Errorf("failed to checkout %s: %w", base, err)
			}
		}

		mergeRef := branch

		user := c.Transport.GetUser()
		event := model.Event{
			ID:   issue.ID,
			Type: model.EventTypeMerge,
			Payload: model.MergePayload{
				Branch:   branch,
				BaseSHA:  baseSHA,
				MergeSHA: "",
				Strategy: string(opts.Strategy),
			},
			CreatedAt: time.Now().UTC(),
			CreatedBy: user,
		}

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

		// Squash and ff-only merges do not invoke the merge=union driver,
		// so the branch's issues.db would overwrite main's. Save main's
		// version before the merge so we can union them afterward.
		var mainIssuesDB []byte
		if !useWorktrees && opts.Strategy != MergeStrategyMerge {
			mainIssuesDB, _ = os.ReadFile(filepath.Join(storage.XpoDir(), "issues.db"))
		}

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
			if mainIssuesDB != nil {
				branchIssuesDB, _ := os.ReadFile(filepath.Join(storage.XpoDir(), "issues.db"))
				merged := unionLines(mainIssuesDB, branchIssuesDB)
				os.WriteFile(filepath.Join(storage.XpoDir(), "issues.db"), merged, 0644)
			}

			preEvents, _ := storage.ReadEvents()
			preMergeState := ProjectIssues(preEvents)

			doneStatus := string(model.StatusDone)
			doneEvents, doneMessages, err := c.local.buildUpdate(
				issue.ID, model.UpdatePayload{Status: &doneStatus}, preMergeState)
			if err != nil {
				return fmt.Errorf("failed to build DONE transition: %w", err)
			}

			if err := c.local.appendEvent(event); err != nil {
				return fmt.Errorf("failed to record merge event: %w", err)
			}
			for _, evt := range doneEvents {
				if err := c.local.appendEvent(evt); err != nil {
					return fmt.Errorf("failed to apply DONE transition: %w", err)
				}
			}
			result.Messages = append(result.Messages, doneMessages...)

			exec.Command("git", "-C", storage.HubRoot(), "add", ".xpo/issues.db").Run()
			exec.Command("git", "-C", storage.HubRoot(), "add", filepath.Join(".xpo", "artifacts", issue.ID)).Run()
			switch opts.Strategy {
			case MergeStrategySquash, MergeStrategyFF:
				mergeErr = exec.Command("git", "commit", "-m", commitMsg).Run()
			default:
				exec.Command("git", "commit", "--amend", "--no-edit").Run()
			}
		}

		if mergeErr != nil {
			exec.Command("git", "merge", "--abort").Run()
			return fmt.Errorf("merge failed: %w\nDo NOT stash or reset. Resolve the conflict in the listed files, then re-run xpo merge.", mergeErr)
		}

		mergeSHA := resolveRef("HEAD")
		result.MergeSHA = mergeSHA
		result.Messages = append(result.Messages, fmt.Sprintf("Merged %s into %s (%s)", branch, base, opts.Strategy))

		// Clean up worktree (always — worktrees are ephemeral, even with --keep-branch)
		if useWorktrees {
			if wtPath, ok := FindWorktreeForBranch(branch); ok {
				if err := WorktreeRemove(wtPath); err != nil {
					result.Messages = append(result.Messages, fmt.Sprintf("Warning: failed to remove worktree %s: %v", wtPath, err))
				} else {
					result.Messages = append(result.Messages, fmt.Sprintf("Removed worktree %s", wtPath))
				}
			}
		}

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

// HubCleanForMerge checks whether the hub working tree is safe to merge the
// given branch. Untracked files and .xpo/ changes are ignored — only modified
// tracked files that overlap with the incoming branch's changes block the merge.
func HubCleanForMerge(branch string) error {
	hub := storage.HubRoot()
	out, err := exec.Command("git", "-C", hub, "status", "--porcelain").Output()
	if err != nil {
		return fmt.Errorf("failed to check working tree status: %w", err)
	}

	var modifiedTracked []string
	for _, line := range strings.Split(string(out), "\n") {
		if len(line) < 3 {
			continue
		}
		statusCode := line[:2]
		path := strings.TrimSpace(line[3:])

		if strings.HasPrefix(path, ".xpo/") {
			continue
		}
		if statusCode == "??" {
			continue
		}
		modifiedTracked = append(modifiedTracked, path)
	}

	if len(modifiedTracked) == 0 {
		return nil
	}

	base := DefaultBranch()
	branchOut, err := exec.Command("git", "-C", hub, "diff", "--name-only", base+"..."+branch).Output()
	if err != nil {
		return fmt.Errorf("failed to diff branch %s against %s: %w", branch, base, err)
	}

	branchFiles := make(map[string]struct{})
	for _, f := range strings.Split(strings.TrimSpace(string(branchOut)), "\n") {
		f = strings.TrimSpace(f)
		if f != "" {
			branchFiles[f] = struct{}{}
		}
	}

	var conflicting []string
	for _, path := range modifiedTracked {
		if _, overlap := branchFiles[path]; overlap {
			conflicting = append(conflicting, path)
		}
	}

	if len(conflicting) == 0 {
		return nil
	}

	msg := "these tracked files have local changes that conflict with the incoming branch:\n"
	for _, f := range conflicting {
		msg += fmt.Sprintf("  - %s (modified locally, also changed on %s)\n", f, branch)
	}
	msg += "Commit or remove these changes before merging.\n"
	msg += "Do NOT stash — .xpo/issues.db must not be stashed."
	return fmt.Errorf("%s", msg)
}

// HubBranch returns the current branch of the hub (primary checkout),
// regardless of which worktree the caller is in.
func HubBranch() string {
	hub := storage.HubRoot()
	out, err := exec.Command("git", "-C", hub, "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func resolveRef(ref string) string {
	out, err := exec.Command("git", "rev-parse", "--short", ref).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// unionLines merges two line-delimited byte slices, keeping all lines from
// base and appending any lines from theirs that are not already present.
// This simulates git's merge=union driver for squash/ff merges.
func unionLines(base, theirs []byte) []byte {
	baseStr := strings.TrimRight(string(base), "\n")
	theirStr := strings.TrimRight(string(theirs), "\n")

	if baseStr == "" && theirStr == "" {
		return nil
	}

	var baseLines []string
	if baseStr != "" {
		baseLines = strings.Split(baseStr, "\n")
	}
	var theirLines []string
	if theirStr != "" {
		theirLines = strings.Split(theirStr, "\n")
	}

	seen := make(map[string]struct{}, len(baseLines))
	for _, line := range baseLines {
		seen[line] = struct{}{}
	}

	result := make([]string, len(baseLines))
	copy(result, baseLines)

	for _, line := range theirLines {
		if _, ok := seen[line]; !ok {
			result = append(result, line)
			seen[line] = struct{}{}
		}
	}

	return []byte(strings.Join(result, "\n") + "\n")
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
