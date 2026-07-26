package exponential

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/palarix/exponential/internal/model"
	"github.com/palarix/exponential/internal/storage"
)

type CommitEntry struct {
	SHA     string
	Message string
}

type FileStat struct {
	Status     string
	Path       string
	Insertions int
	Deletions  int
}

// CurrentBranch returns the name of the currently checked-out branch.
func CurrentBranch() string {
	out, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// IssueIDFromBranch attempts to find an issue whose ID matches a segment
// of the given branch name.
func (c *Client) IssueIDFromBranch(branch string) (*model.Issue, error) {
	if c.local == nil {
		return nil, ErrLocalOnly
	}
	events, err := storage.ReadEvents()
	if err != nil {
		return nil, err
	}
	issues := ProjectIssuesWithConfig(events, c.local.Config)

	for _, issue := range issues {
		for _, segment := range strings.Split(branch, "/") {
			if strings.HasPrefix(segment, issue.ID) {
				return issue, nil
			}
		}
	}
	return nil, fmt.Errorf("no issue matches current branch %q", branch)
}

// ResolveReviewIssue resolves an issue for review — either by explicit ID
// or by inferring from the current branch. If the issue has no BranchStats
// from remote detection, it fills them from the local branch.
func (c *Client) ResolveReviewIssue(idOrEmpty string) (*model.Issue, error) {
	if idOrEmpty != "" {
		issue, err := c.Transport.GetIssue(idOrEmpty)
		if err != nil {
			return nil, err
		}
		c.FillLocalBranchStats(issue)
		return issue, nil
	}
	branch := CurrentBranch()
	if branch == "" || branch == "HEAD" {
		return nil, fmt.Errorf("not on a branch — specify an issue ID")
	}
	issue, err := c.IssueIDFromBranch(branch)
	if err != nil {
		return nil, err
	}
	c.FillLocalBranchStats(issue)
	return issue, nil
}

// FillLocalBranchStats computes BranchStats from a local branch when no
// remote branch was detected. Checks both the current branch and any local
// branch whose name contains the issue ID.
func (c *Client) FillLocalBranchStats(issue *model.Issue) {
	if issue.BranchStats != nil {
		return
	}
	branch := findLocalBranch(issue.ID)
	if branch == "" {
		return
	}
	issue.BranchStats = computeBranchStats(branch, DefaultBranch())
}

// findLocalBranch returns a local branch name matching the issue ID,
// checking the current branch first, then scanning all local branches.
func findLocalBranch(issueID string) string {
	current := CurrentBranch()
	if current != "" && branchNameMatchesIssue(current, issueID) {
		return current
	}

	out, err := exec.Command("git", "branch", "--format=%(refname:short)").Output()
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		if branchNameMatchesIssue(line, issueID) {
			return line
		}
	}
	return ""
}

// branchNameMatchesIssue checks if any path segment of a branch name
// starts with the issue ID.
func branchNameMatchesIssue(branch, issueID string) bool {
	for _, segment := range strings.Split(branch, "/") {
		if strings.HasPrefix(segment, issueID) {
			return true
		}
	}
	return false
}

type DetailedCommit struct {
	SHA     string `json:"sha"`
	Message string `json:"message"`
	Author  string `json:"author"`
	Date    string `json:"date"`
}

// ListBranchCommitsDetailed returns commits with author and date info.
func ListBranchCommitsDetailed(branch, base string) []DetailedCommit {
	out, err := exec.Command("git", "log", "--format=%H%n%s%n%an%n%aI", base+".."+branch).Output()
	if err != nil {
		return nil
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var commits []DetailedCommit
	for i := 0; i+3 < len(lines); i += 4 {
		commits = append(commits, DetailedCommit{
			SHA:     lines[i][:12],
			Message: lines[i+1],
			Author:  lines[i+2],
			Date:    lines[i+3],
		})
	}
	return commits
}

// GetCommitDiffText returns the unified diff for a single commit.
func GetCommitDiffText(sha string) string {
	out, err := exec.Command("git", "diff-tree", "-p", sha).Output()
	if err != nil {
		return ""
	}
	return string(out)
}

// GetDiffText returns the unified diff as a string.
func GetDiffText(branch, base string) string {
	out, err := exec.Command("git", "diff", base+"..."+branch).Output()
	if err != nil {
		return ""
	}
	return string(out)
}

// GetWorkingTreeDiffText returns the combined staged + unstaged diff
// relative to HEAD for the currently checked-out branch.
func GetWorkingTreeDiffText() string {
	out, err := exec.Command("git", "diff", "HEAD").Output()
	if err != nil {
		return ""
	}
	return string(out)
}

// ListBranchCommits returns the commits on branch relative to base,
// newest first.
func ListBranchCommits(branch, base string) []CommitEntry {
	out, err := exec.Command("git", "log", "--oneline", "--no-decorate", base+".."+branch).Output()
	if err != nil {
		return nil
	}
	var commits []CommitEntry
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		msg := ""
		if len(parts) == 2 {
			msg = parts[1]
		}
		commits = append(commits, CommitEntry{SHA: parts[0], Message: msg})
	}
	return commits
}

// ListWorkingTreeFilesChanged returns per-file stats for uncommitted
// changes (staged + unstaged) relative to HEAD.
func ListWorkingTreeFilesChanged() []FileStat {
	return listFilesChangedFromDiff("HEAD")
}

// ListFilesChanged returns per-file stats for the branch relative to base.
func ListFilesChanged(branch, base string) []FileStat {
	return listFilesChangedFromDiff(base + "..." + branch)
}

func listFilesChangedFromDiff(diffRef string) []FileStat {
	numOut, err := exec.Command("git", "diff", "--numstat", diffRef).Output()
	if err != nil {
		return nil
	}
	nameOut, _ := exec.Command("git", "diff", "--name-status", diffRef).Output()

	statusMap := make(map[string]string)
	for _, line := range strings.Split(strings.TrimSpace(string(nameOut)), "\n") {
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			statusMap[fields[len(fields)-1]] = fields[0]
		}
	}

	var files []FileStat
	for _, line := range strings.Split(strings.TrimSpace(string(numOut)), "\n") {
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		var ins, del int
		fmt.Sscanf(fields[0], "%d", &ins)
		fmt.Sscanf(fields[1], "%d", &del)
		path := fields[2]

		status := statusMap[path]
		if status == "" {
			status = "M"
		}
		files = append(files, FileStat{
			Status:     status,
			Path:       path,
			Insertions: ins,
			Deletions:  del,
		})
	}
	return files
}

