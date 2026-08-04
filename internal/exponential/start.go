package exponential

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/palarix/exponential/internal/model"
)

// StartWork transitions an issue to DOING and, when in a git repo,
// creates a worktree (default) or checks out a branch (--no-wt).
// Returns the branch name, worktree path (empty when worktrees disabled),
// messages, and any error.
func (c *Client) StartWork(id string, force bool) (branchName, worktreePath string, msgs []string, err error) {
	issue, err := c.Transport.GetIssue(id)
	if err != nil {
		return "", "", nil, err
	}

	switch issue.Status {
	case model.StatusDone:
		return "", "", nil, fmt.Errorf("issue %s is already DONE", id)
	case model.StatusBlocked:
		return "", "", nil, fmt.Errorf("issue %s is BLOCKED", id)
	case model.StatusDoing:
		if !force {
			who := "someone"
			if issue.Assignee != "" {
				who = issue.Assignee
			}
			return "", "", nil, fmt.Errorf("issue %s is already in progress (assigned to %s) — use --force to take over", id, who)
		}
		msgs = append(msgs, fmt.Sprintf("Force-claiming issue %s", id))
	}

	candidateBranch := fmt.Sprintf("%s-%s", issue.ID, Slugify(issue.Title))
	useWorktrees := c.Config.Worktrees && CheckGitRepo()

	if CheckGitRepo() && !force && !useWorktrees {
		if BranchExists(candidateBranch) || RemoteBranchExists(candidateBranch) {
			return "", "", nil, fmt.Errorf("branch already exists: %s — use --force to take over", candidateBranch)
		}
	}

	if issue.Status != model.StatusDoing {
		status := string(model.StatusDoing)
		payload := model.UpdatePayload{Status: &status}
		updateMsgs, err := c.Transport.UpdateIssue(id, payload, "start")
		if err != nil {
			return "", "", nil, err
		}
		msgs = append(msgs, updateMsgs...)
	}

	if !CheckGitRepo() {
		return "", "", msgs, nil
	}

	branchName = candidateBranch

	if useWorktrees {
		wtPath := WorktreeDir(branchName)

		gitErr := WithGitLock(func() error {
			// Force takeover: remove existing worktree for this branch
			if force {
				if existing, ok := FindWorktreeForBranch(branchName); ok {
					if err := WorktreeRemove(existing); err != nil {
						return fmt.Errorf("failed to remove existing worktree %s: %w", existing, err)
					}
					msgs = append(msgs, fmt.Sprintf("Removed existing worktree at %s", existing))
				}
			}

			if BranchExists(branchName) {
				if err := WorktreeAddExisting(wtPath, branchName); err != nil {
					return fmt.Errorf("failed to create worktree for existing branch %s: %w", branchName, err)
				}
				msgs = append(msgs, fmt.Sprintf("Created worktree for existing branch '%s'", branchName))
			} else {
				base := DefaultBranch()
				if err := WorktreeAdd(wtPath, branchName, base); err != nil {
					return fmt.Errorf("failed to create worktree %s from %s: %w", branchName, base, err)
				}
				msgs = append(msgs, fmt.Sprintf("Created worktree with new branch '%s'", branchName))
			}
			return nil
		})
		if gitErr != nil {
			return "", "", msgs, gitErr
		}

		absPath, _ := filepath.Abs(wtPath)
		worktreePath = absPath

		EnsureGitignoreEntry(".xpo/worktrees/")

		if c.Config.WorktreeSetup != "" {
			cmd := exec.Command("sh", "-c", c.Config.WorktreeSetup)
			cmd.Dir = absPath
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				msgs = append(msgs, fmt.Sprintf("Warning: worktree_setup hook failed: %v", err))
			} else {
				msgs = append(msgs, "Ran worktree_setup hook")
			}
		}

		msgs = append(msgs, fmt.Sprintf("Worktree: %s", absPath))
	} else {
		// Classic checkout-based flow
		gitErr := WithGitLock(func() error {
			if BranchExists(branchName) {
				if err := CheckoutBranch(branchName); err != nil {
					return fmt.Errorf("failed to checkout branch %s: %w", branchName, err)
				}
				msgs = append(msgs, fmt.Sprintf("Switched to existing branch '%s'", branchName))
			} else {
				base := DefaultBranch()
				if err := CreateAndCheckoutBranch(branchName, base); err != nil {
					return fmt.Errorf("failed to create branch %s from %s: %w", branchName, base, err)
				}
				msgs = append(msgs, fmt.Sprintf("Created and switched to branch '%s'", branchName))
			}
			return nil
		})
		if gitErr != nil {
			return "", "", msgs, gitErr
		}
	}

	return branchName, worktreePath, msgs, nil
}
