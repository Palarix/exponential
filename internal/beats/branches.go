package beats

import (
	"os/exec"
	"strconv"
	"strings"

	"github.com/palarix/beats/internal/model"
)

// applyBranchInference scans remote git branches for issue IDs and
// synthetically marks matching issues as DOING when their recorded
// status is BACKLOG or PLANNED. It also computes branch change stats
// (commits, files changed, insertions/deletions) for any issue with
// a matching branch regardless of status.
func applyBranchInference(issues map[string]*model.Issue) {
	if !CheckGitRepo() {
		return
	}

	remoteBranches := listRemoteBranches()
	localBranches := listLocalBranches()
	base := DefaultBranch()

	for _, issue := range issues {
		// Status inference: only from remote branches (discovers work on other clones)
		if matchingBranch(remoteBranches, issue.ID) != "" {
			if issue.Status != model.StatusDone && issue.Status != model.StatusBlocked && issue.Status != model.StatusDoing {
				issue.Status = model.StatusDoing
				issue.InferredStatus = true
			}
		}

		// Stats: prefer local branch (has unpushed commits), fall back to remote
		localBranch := matchLocalBranch(localBranches, issue.ID, base)
		if localBranch != "" {
			issue.BranchStats = computeBranchStats(localBranch, base)
		} else {
			remoteBranch := matchingBranch(remoteBranches, issue.ID)
			if remoteBranch != "" {
				issue.BranchStats = computeBranchStats(remoteBranch, base)
			}
		}
	}
}

func listLocalBranches() []string {
	out, err := exec.Command("git", "branch", "--format=%(refname:short)").Output()
	if err != nil {
		return nil
	}
	var branches []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line != "" {
			branches = append(branches, line)
		}
	}
	return branches
}

func matchLocalBranch(branches []string, issueID, base string) string {
	for _, branch := range branches {
		if branch == base {
			continue
		}
		if branchNameMatchesIssue(branch, issueID) {
			return branch
		}
	}
	return ""
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

// matchingBranch returns the first remote branch that has a path segment
// starting with the given issue ID, or empty string if none match.
func matchingBranch(branches []string, issueID string) string {
	for _, branch := range branches {
		parts := strings.SplitN(branch, "/", 2)
		if len(parts) < 2 {
			continue
		}
		branchPath := parts[1]

		for _, segment := range strings.Split(branchPath, "/") {
			if strings.HasPrefix(segment, issueID) {
				return branch
			}
		}
	}
	return ""
}

// branchMatchesIssue checks if any remote branch has a path segment
// that starts with the given issue ID.
func branchMatchesIssue(branches []string, issueID string) bool {
	return matchingBranch(branches, issueID) != ""
}

// computeBranchStats returns commit count, files changed, and line
// insertions/deletions for a branch relative to the base branch.
func computeBranchStats(branch, base string) *model.BranchStats {
	stats := &model.BranchStats{Branch: branch}

	if out, err := exec.Command("git", "rev-parse", branch).Output(); err == nil {
		sha := strings.TrimSpace(string(out))
		if len(sha) > 12 {
			sha = sha[:12]
		}
		stats.HeadSHA = sha
	}

	if out, err := exec.Command("git", "rev-list", "--count", base+".."+branch).Output(); err == nil {
		if n, err := strconv.Atoi(strings.TrimSpace(string(out))); err == nil {
			stats.Commits = n
		}
	}

	if out, err := exec.Command("git", "diff", "--shortstat", base+"..."+branch).Output(); err == nil {
		parseShortstat(strings.TrimSpace(string(out)), stats)
	}

	return stats
}

// parseShortstat parses output like "7 files changed, 142 insertions(+), 38 deletions(-)"
func parseShortstat(line string, stats *model.BranchStats) {
	if line == "" {
		return
	}
	for _, part := range strings.Split(line, ",") {
		part = strings.TrimSpace(part)
		fields := strings.Fields(part)
		if len(fields) < 2 {
			continue
		}
		n, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}
		switch {
		case strings.Contains(fields[1], "file"):
			stats.FilesChanged = n
		case strings.Contains(fields[1], "insertion"):
			stats.Insertions = n
		case strings.Contains(fields[1], "deletion"):
			stats.Deletions = n
		}
	}
}
