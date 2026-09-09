package exponential

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
	out, err := exec.Command("git", "diff-tree", "-p", "--src-prefix=a/", "--dst-prefix=b/", sha).Output()
	if err != nil {
		return ""
	}
	return string(out)
}

// GetDiffText returns the unified diff as a string.
func GetDiffText(branch, base string) string {
	out, err := exec.Command("git", "diff", "--src-prefix=a/", "--dst-prefix=b/", base+"..."+branch).Output()
	if err != nil {
		return ""
	}
	return string(out)
}

// GetWorkingTreeDiffText returns the combined staged + unstaged diff
// relative to HEAD. When dir is non-empty, targets that directory
// via git -C (for worktree support).
func GetWorkingTreeDiffText(dir string) string {
	args := []string{}
	if dir != "" {
		args = append(args, "-C", dir)
	}
	args = append(args, "diff", "--src-prefix=a/", "--dst-prefix=b/", "HEAD")
	out, err := exec.Command("git", args...).Output()
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
// changes (staged + unstaged) relative to HEAD. When dir is non-empty,
// targets that directory via git -C (for worktree support).
func ListWorkingTreeFilesChanged(dir string) []FileStat {
	return listFilesChangedFromDiff("HEAD", dir)
}

// ListFilesChanged returns per-file stats for the branch relative to base.
func ListFilesChanged(branch, base string) []FileStat {
	return listFilesChangedFromDiff(base+"..."+branch, "")
}

func listFilesChangedFromDiff(diffRef string, dir string) []FileStat {
	args := []string{}
	if dir != "" {
		args = append(args, "-C", dir)
	}
	numOut, err := exec.Command("git", append(args, "diff", "--numstat", diffRef)...).Output()
	if err != nil {
		return nil
	}
	nameOut, _ := exec.Command("git", append(args, "diff", "--name-status", diffRef)...).Output()

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

const maxUntrackedFileSize = 1 << 20 // 1 MB

func listUntrackedFiles(dir string) []string {
	args := []string{}
	if dir != "" {
		args = append(args, "-C", dir)
	}
	args = append(args, "ls-files", "--others", "--exclude-standard")
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		return nil
	}
	var files []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" || strings.HasPrefix(line, ".xpo/") {
			continue
		}
		files = append(files, line)
	}
	return files
}

func isBinary(data []byte) bool {
	check := data
	if len(check) > 512 {
		check = check[:512]
	}
	return bytes.ContainsRune(check, 0)
}

func buildSyntheticDiff(path string, content []byte) string {
	lines := strings.Split(string(content), "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	if len(lines) == 0 {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "diff --git a/%s b/%s\n", path, path)
	b.WriteString("new file mode 100644\n")
	b.WriteString("--- /dev/null\n")
	fmt.Fprintf(&b, "+++ b/%s\n", path)
	fmt.Fprintf(&b, "@@ -0,0 +1,%d @@\n", len(lines))
	for _, line := range lines {
		b.WriteByte('+')
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}

// GetWorkingTreeFullDiffText returns the combined staged + unstaged diff
// relative to HEAD, augmented with synthetic diff entries for untracked files.
// Binary files and files over 1 MB are skipped. Paths under .xpo/ are filtered.
func GetWorkingTreeFullDiffText(dir string) string {
	tracked := GetWorkingTreeDiffText(dir)
	untracked := listUntrackedFiles(dir)

	var parts []string
	if tracked != "" {
		parts = append(parts, tracked)
	}

	for _, path := range untracked {
		fullPath := path
		if dir != "" {
			fullPath = filepath.Join(dir, path)
		}
		data, err := os.ReadFile(fullPath)
		if err != nil {
			continue
		}
		if int64(len(data)) > maxUntrackedFileSize {
			continue
		}
		if isBinary(data) {
			continue
		}
		if synth := buildSyntheticDiff(path, data); synth != "" {
			parts = append(parts, synth)
		}
	}

	return strings.Join(parts, "")
}

// ListWorkingTreeAllFilesChanged returns per-file stats for all uncommitted
// changes (staged + unstaged) plus untracked files. Paths under .xpo/ are filtered.
func ListWorkingTreeAllFilesChanged(dir string) []FileStat {
	files := ListWorkingTreeFilesChanged(dir)

	seen := make(map[string]bool, len(files))
	for _, f := range files {
		seen[f.Path] = true
	}

	for _, path := range listUntrackedFiles(dir) {
		if seen[path] {
			continue
		}
		fullPath := path
		if dir != "" {
			fullPath = filepath.Join(dir, path)
		}
		data, err := os.ReadFile(fullPath)
		if err != nil {
			continue
		}
		lineCount := 0
		if len(data) > 0 {
			lineCount = strings.Count(string(data), "\n")
			if data[len(data)-1] != '\n' {
				lineCount++
			}
		}
		files = append(files, FileStat{
			Status:     "A",
			Path:       path,
			Insertions: lineCount,
			Deletions:  0,
		})
	}

	return files
}

