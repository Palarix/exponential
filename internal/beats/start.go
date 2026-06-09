package beats

import (
	"fmt"

	"github.com/palarix/beats/internal/model"
)

// StartWork transitions an issue to DOING and, when in a git repo,
// creates (or checks out) a branch named <issue-id>/<slugified-title>.
func (c *Client) StartWork(id string) (branchName string, msgs []string, err error) {
	issue, err := c.GetIssue(id)
	if err != nil {
		return "", nil, err
	}

	switch issue.Status {
	case model.StatusDone:
		return "", nil, fmt.Errorf("issue %s is already DONE", id)
	case model.StatusBlocked:
		return "", nil, fmt.Errorf("issue %s is BLOCKED", id)
	}

	if issue.Status != model.StatusDoing {
		status := string(model.StatusDoing)
		payload := model.UpdatePayload{Status: &status}
		updateMsgs, err := c.UpdateIssue(id, payload, "start")
		if err != nil {
			return "", nil, err
		}
		msgs = append(msgs, updateMsgs...)
	}

	if !CheckGitRepo() {
		return "", msgs, nil
	}

	branchName = fmt.Sprintf("%s-%s", issue.ID, Slugify(issue.Title))

	if BranchExists(branchName) {
		if err := CheckoutBranch(branchName); err != nil {
			return "", msgs, fmt.Errorf("failed to checkout branch %s: %w", branchName, err)
		}
		msgs = append(msgs, fmt.Sprintf("Switched to existing branch '%s'", branchName))
	} else {
		base := DefaultBranch()
		if err := CreateAndCheckoutBranch(branchName, base); err != nil {
			return "", msgs, fmt.Errorf("failed to create branch %s from %s: %w", branchName, base, err)
		}
		msgs = append(msgs, fmt.Sprintf("Created and switched to branch '%s'", branchName))
	}

	return branchName, msgs, nil
}
