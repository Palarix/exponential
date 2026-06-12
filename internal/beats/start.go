package beats

import (
	"fmt"

	"github.com/palarix/beats/internal/model"
)

// StartWork transitions an issue to DOING and, when in a git repo,
// creates (or checks out) a branch named <issue-id>/<slugified-title>.
func (c *Client) StartWork(id string, force bool) (branchName string, msgs []string, err error) {
	issue, err := c.Transport.GetIssue(id)
	if err != nil {
		return "", nil, err
	}

	switch issue.Status {
	case model.StatusDone:
		return "", nil, fmt.Errorf("issue %s is already DONE", id)
	case model.StatusBlocked:
		return "", nil, fmt.Errorf("issue %s is BLOCKED", id)
	case model.StatusDoing:
		if !force {
			who := "someone"
			if issue.Assignee != "" {
				who = issue.Assignee
			}
			return "", nil, fmt.Errorf("issue %s is already in progress (assigned to %s) — use --force to take over", id, who)
		}
		msgs = append(msgs, fmt.Sprintf("Force-claiming issue %s", id))
	}

	// Branch pre-flight check (local mode only — server doesn't manage git)
	candidateBranch := fmt.Sprintf("%s-%s", issue.ID, Slugify(issue.Title))
	if CheckGitRepo() && !force {
		if BranchExists(candidateBranch) || RemoteBranchExists(candidateBranch) {
			return "", nil, fmt.Errorf("branch already exists: %s — use --force to take over", candidateBranch)
		}
	}

	if issue.Status != model.StatusDoing {
		status := string(model.StatusDoing)
		payload := model.UpdatePayload{Status: &status}
		updateMsgs, err := c.Transport.UpdateIssue(id, payload, "start")
		if err != nil {
			return "", nil, err
		}
		msgs = append(msgs, updateMsgs...)
	}

	if !CheckGitRepo() {
		return "", msgs, nil
	}

	branchName = candidateBranch

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
		return "", msgs, gitErr
	}

	return branchName, msgs, nil
}
