package beats

import (
	"os/exec"
	"strings"

	"github.com/kuyio/beats/internal/model"
)

// applyBranchInference scans remote git branches for issue IDs and
// synthetically marks matching issues as DOING when their recorded
// status is BACKLOG or PLANNED. The status change is projection-only
// and never persisted to the event log.
func applyBranchInference(issues map[string]*model.Issue) {
	if !CheckGitRepo() {
		return
	}

	branches := listRemoteBranches()
	if len(branches) == 0 {
		return
	}

	for _, issue := range issues {
		if issue.Status == model.StatusDone || issue.Status == model.StatusBlocked || issue.Status == model.StatusDoing {
			continue
		}
		if branchMatchesIssue(branches, issue.ID) {
			issue.Status = model.StatusDoing
			issue.InferredStatus = true
		}
	}
}

// listRemoteBranches runs `git branch -r` and returns the trimmed
// branch names, skipping alias lines (e.g. "origin/HEAD -> origin/main").
func listRemoteBranches() []string {
	out, err := exec.Command("git", "branch", "-r").Output()
	if err != nil {
		return nil
	}

	var branches []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.Contains(line, "->") {
			continue
		}
		branches = append(branches, line)
	}
	return branches
}

// branchMatchesIssue checks if any remote branch has a path segment
// that starts with the given issue ID.
func branchMatchesIssue(branches []string, issueID string) bool {
	for _, branch := range branches {
		// Strip the remote prefix (e.g. "origin/") to get the branch path
		parts := strings.SplitN(branch, "/", 2)
		if len(parts) < 2 {
			continue
		}
		branchPath := parts[1]

		for _, segment := range strings.Split(branchPath, "/") {
			if strings.HasPrefix(segment, issueID) {
				return true
			}
		}
	}
	return false
}
