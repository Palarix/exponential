package exponential

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/palarix/exponential/internal/storage"
)

// Read-only git queries for agents (the git_* and code_history MCP tools).
//
// The caller supplies only an issue ID, a repo path, a commit SHA, a line
// range and a limit; the command line is always built here. That closes the
// side doors that make "read-only" git commands unsafe to expose as raw
// Bash: --output writes files, external diff and textconv drivers and
// core.fsmonitor run programs from repo config, and -c sets any config.

const (
	gitReadCap         = 64 * 1024
	gitReadTimeout     = 10 * time.Second
	gitTruncatedMarker = "[truncated at 64 KiB: narrow the request with path or stat]"

	defaultGitLimit = 20
	maxGitLimit     = 100
)

var (
	shaRe = regexp.MustCompile(`^[0-9a-fA-F]{4,40}$`)

	// Environment variables that would point git at another repository or
	// make it run a program.
	gitEnvDenylist = []string{"GIT_DIR", "GIT_WORK_TREE", "GIT_INDEX_FILE", "GIT_EXTERNAL_DIFF", "GIT_CONFIG_PARAMETERS", "GIT_CONFIG_COUNT"}
)

// GitFileChange is one file in `git status`: its two-letter porcelain code
// and path.
type GitFileChange struct {
	Status string `json:"status"`
	Path   string `json:"path"`
}

type GitStatusResult struct {
	IssueID      string          `json:"id"`
	Branch       string          `json:"branch"`
	BaseBranch   string          `json:"base_branch"`
	WorktreePath string          `json:"worktree_path,omitempty"`
	Ahead        int             `json:"ahead"`
	Behind       int             `json:"behind"`
	Staged       []GitFileChange `json:"staged"`
	Unstaged     []GitFileChange `json:"unstaged"`
	Untracked    []string        `json:"untracked"`
}

type GitDiffOptions struct {
	Path string
	Stat bool
}

type GitDiffFile struct {
	Path       string `json:"path"`
	Insertions int    `json:"insertions"`
	Deletions  int    `json:"deletions"`
	Binary     bool   `json:"binary,omitempty"`
}

type GitDiffResult struct {
	IssueID             string        `json:"id"`
	Branch              string        `json:"branch"`
	BaseBranch          string        `json:"base_branch"`
	MergeBase           string        `json:"merge_base"`
	WorktreePath        string        `json:"worktree_path,omitempty"`
	IncludesUncommitted bool          `json:"includes_uncommitted"`
	Files               []GitDiffFile `json:"files,omitempty"`
	Patch               string        `json:"patch,omitempty"`
	Untracked           []string      `json:"untracked,omitempty"`
	Truncated           bool          `json:"truncated,omitempty"`
}

type LogCommit struct {
	SHA      string   `json:"sha"`
	Subject  string   `json:"subject"`
	Author   string   `json:"author"`
	Date     string   `json:"date"`
	IssueIDs []string `json:"issue_ids,omitempty"`
}

type GitLogOptions struct {
	IssueID string
	Path    string
	Limit   int
}

type GitLogResult struct {
	IssueID   string      `json:"id,omitempty"`
	Branch    string      `json:"branch,omitempty"`
	Range     string      `json:"range"`
	Commits   []LogCommit `json:"commits"`
	Truncated bool        `json:"truncated,omitempty"`
}

type GitShowResult struct {
	SHA       string `json:"sha"`
	Text      string `json:"text"`
	Truncated bool   `json:"truncated,omitempty"`
}

type CodeHistoryOptions struct {
	Path      string
	StartLine int
	EndLine   int
	Limit     int
}

type GitLineRange struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

type CodeHistoryCommit struct {
	LogCommit
	Lines []GitLineRange `json:"lines,omitempty"`
}

type CodeHistoryIssue struct {
	ID     string `json:"id"`
	Title  string `json:"title,omitempty"`
	Status string `json:"status,omitempty"`
}

type CodeHistoryResult struct {
	Path      string              `json:"path"`
	Revision  string              `json:"revision"`
	StartLine int                 `json:"start_line,omitempty"`
	EndLine   int                 `json:"end_line,omitempty"`
	Commits   []CodeHistoryCommit `json:"commits"`
	Issues    []CodeHistoryIssue  `json:"issues"`
	Truncated bool                `json:"truncated,omitempty"`
}

// --- Runner ---

// capWriter keeps the first limit bytes and drains the rest, so git never
// blocks on a full pipe.
type capWriter struct {
	buf       bytes.Buffer
	limit     int
	truncated bool
}

func (w *capWriter) Write(p []byte) (int, error) {
	if room := w.limit - w.buf.Len(); room > 0 {
		if len(p) > room {
			w.buf.Write(p[:room])
			w.truncated = true
		} else {
			w.buf.Write(p)
		}
	} else if len(p) > 0 {
		w.truncated = true
	}
	return len(p), nil
}

func gitReadEnv() []string {
	env := make([]string, 0, len(os.Environ())+3)
	for _, kv := range os.Environ() {
		name, _, _ := strings.Cut(kv, "=")
		if !strings.HasPrefix(name, "GIT_CONFIG_KEY_") && !strings.HasPrefix(name, "GIT_CONFIG_VALUE_") &&
			!slices.Contains(gitEnvDenylist, name) {
			env = append(env, kv)
		}
	}
	return append(env, "GIT_OPTIONAL_LOCKS=0", "GIT_TERMINAL_PROMPT=0", "GIT_PAGER=cat")
}

// readGit runs one read-only git command in dir. It returns at most
// gitReadCap bytes of stdout and reports whether it cut the output.
func readGit(dir string, args ...string) (string, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gitReadTimeout)
	defer cancel()

	full := append([]string{
		"-C", dir,
		"-c", "core.fsmonitor=false",
		"-c", "core.hooksPath=/dev/null",
		"-c", "core.quotePath=false",
		"--no-pager",
	}, args...)
	cmd := exec.CommandContext(ctx, "git", full...)
	cmd.Env = gitReadEnv()
	out := &capWriter{limit: gitReadCap}
	stderr := &capWriter{limit: 4096}
	cmd.Stdout = out
	cmd.Stderr = stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return "", false, fmt.Errorf("git %s timed out after %s", args[0], gitReadTimeout)
		}
		msg := strings.TrimSpace(stderr.buf.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", false, fmt.Errorf("git %s: %s", args[0], msg)
	}
	return out.buf.String(), out.truncated, nil
}

func withMarker(s string, truncated bool) string {
	if !truncated {
		return s
	}
	if !strings.HasSuffix(s, "\n") {
		s += "\n"
	}
	return s + gitTruncatedMarker + "\n"
}

// --- Argument validation ---

// validRepoPath cleans a repo-relative path and rejects anything that could
// leave the repo or be read as a git option.
func validRepoPath(p string) (string, error) {
	if p == "" {
		return "", errors.New("path is empty")
	}
	if strings.HasPrefix(p, "-") {
		return "", fmt.Errorf("invalid path %q: must not start with '-'", p)
	}
	if path.IsAbs(p) || strings.HasPrefix(p, `\`) || (len(p) > 1 && p[1] == ':') {
		return "", fmt.Errorf("invalid path %q: must be relative to the repository root", p)
	}
	clean := path.Clean(strings.ReplaceAll(p, `\`, "/"))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("invalid path %q: must stay inside the repository", p)
	}
	return clean, nil
}

func optionalRepoPath(p string) (string, error) {
	if p == "" {
		return "", nil
	}
	return validRepoPath(p)
}

func validSHA(s string) error {
	if !shaRe.MatchString(s) {
		return fmt.Errorf("invalid sha %q: must be 4 to 40 hex characters", s)
	}
	return nil
}

func clampLimit(n int) int {
	switch {
	case n <= 0:
		return defaultGitLimit
	case n > maxGitLimit:
		return maxGitLimit
	default:
		return n
	}
}

// parseIssueIDs returns the distinct issue IDs in s, in order of appearance.
func parseIssueIDs(prefix, s string) []string {
	prefix = strings.TrimSuffix(prefix, "-")
	if prefix == "" {
		prefix = "issue"
	}
	re := regexp.MustCompile(`(?:^|[^A-Za-z0-9_-])(` + regexp.QuoteMeta(prefix) + `-[0-9a-f]{6})(?:$|[^A-Za-z0-9_])`)
	var ids []string
	// FindAll would consume the separator after one ID, missing an ID that
	// starts right after it ("a,b"), so scan from the end of each ID.
	for rest := s; ; {
		m := re.FindStringSubmatchIndex(rest)
		if m == nil {
			break
		}
		id := rest[m[2]:m[3]]
		if !slices.Contains(ids, id) {
			ids = append(ids, id)
		}
		rest = rest[m[3]:]
	}
	return ids
}

func (c *Client) issuePrefix() string {
	if c.Config == nil {
		return ""
	}
	return c.Config.Prefix
}

// --- Issue resolution ---

type issueGitRef struct {
	IssueID  string
	Branch   string
	Worktree string
	Base     string
	Hub      string
}

// issueGitRef resolves an issue through the transport, then finds its local
// branch and worktree.
func (c *Client) issueGitRef(id string) (issueGitRef, error) {
	if id == "" {
		return issueGitRef{}, errors.New("'id' is required")
	}
	issue, err := c.GetIssue(id)
	if err != nil {
		return issueGitRef{}, err
	}
	ref := issueGitRef{IssueID: issue.ID, Base: DefaultBranch(), Hub: storage.HubRoot()}

	if wt, ok := FindWorktreeForIssue(issue.ID); ok {
		ref.Branch, ref.Worktree = wt.Branch, wt.Path
		return ref, nil
	}
	out, _, err := readGit(ref.Hub, "for-each-ref", "--format=%(refname:short)", "refs/heads/")
	if err != nil {
		return issueGitRef{}, err
	}
	for _, b := range strings.Split(strings.TrimSpace(out), "\n") {
		if b != "" && b != ref.Base && branchNameMatchesIssue(b, issue.ID) {
			ref.Branch = b
			return ref, nil
		}
	}
	return issueGitRef{}, fmt.Errorf("issue %s has no branch — start it first", issue.ID)
}

// --- Operations ---

// GitStatus reports the issue branch's position against the default branch
// and, when the issue has a worktree, its uncommitted files.
func (c *Client) GitStatus(id string) (*GitStatusResult, error) {
	ref, err := c.issueGitRef(id)
	if err != nil {
		return nil, err
	}
	res := &GitStatusResult{
		IssueID: ref.IssueID, Branch: ref.Branch, BaseBranch: ref.Base, WorktreePath: ref.Worktree,
		Staged: []GitFileChange{}, Unstaged: []GitFileChange{}, Untracked: []string{},
	}

	out, _, err := readGit(ref.Hub, "rev-list", "--left-right", "--count", "--end-of-options", ref.Base+"..."+ref.Branch)
	if err != nil {
		return nil, err
	}
	if f := strings.Fields(out); len(f) == 2 {
		res.Behind, _ = strconv.Atoi(f[0])
		res.Ahead, _ = strconv.Atoi(f[1])
	}

	if ref.Worktree == "" {
		return res, nil
	}
	out, _, err = readGit(ref.Worktree, "status", "--porcelain=v2", "--untracked-files=all", "--ignore-submodules=all")
	if err != nil {
		return nil, err
	}
	parsePorcelainV2(out, res)
	return res, nil
}

// parsePorcelainV2 sorts `git status --porcelain=v2` entries into staged,
// unstaged and untracked files.
func parsePorcelainV2(out string, res *GitStatusResult) {
	for _, line := range strings.Split(out, "\n") {
		var xy, p string
		switch {
		case strings.HasPrefix(line, "1 "):
			if f := strings.SplitN(line, " ", 9); len(f) == 9 {
				xy, p = f[1], f[8]
			}
		case strings.HasPrefix(line, "2 "):
			if f := strings.SplitN(line, " ", 10); len(f) == 10 {
				xy, p = f[1], f[9]
				p, _, _ = strings.Cut(p, "\t")
			}
		case strings.HasPrefix(line, "u "):
			if f := strings.SplitN(line, " ", 11); len(f) == 11 {
				xy, p = f[1], f[10]
			}
		case strings.HasPrefix(line, "? "):
			res.Untracked = append(res.Untracked, strings.TrimPrefix(line, "? "))
		}
		if len(xy) != 2 {
			continue
		}
		if xy[0] != '.' {
			res.Staged = append(res.Staged, GitFileChange{Status: xy, Path: p})
		}
		if xy[1] != '.' {
			res.Unstaged = append(res.Unstaged, GitFileChange{Status: xy, Path: p})
		}
	}
}

// GitDiff returns the issue's changes since its merge base with the default
// branch. With a worktree it diffs the working tree, so uncommitted changes
// to tracked files are included; untracked files are listed by name.
func (c *Client) GitDiff(id string, opts GitDiffOptions) (*GitDiffResult, error) {
	p, err := optionalRepoPath(opts.Path)
	if err != nil {
		return nil, err
	}
	ref, err := c.issueGitRef(id)
	if err != nil {
		return nil, err
	}
	out, _, err := readGit(ref.Hub, "merge-base", "--end-of-options", ref.Base, ref.Branch)
	if err != nil {
		return nil, err
	}
	mergeBase := strings.TrimSpace(out)
	res := &GitDiffResult{
		IssueID: ref.IssueID, Branch: ref.Branch, BaseBranch: ref.Base, MergeBase: mergeBase,
		WorktreePath: ref.Worktree, IncludesUncommitted: ref.Worktree != "",
	}

	args := []string{"diff", "--no-ext-diff", "--no-textconv", "--no-color"}
	if opts.Stat {
		args = append(args, "--numstat")
	}
	dir := ref.Hub
	if ref.Worktree != "" {
		dir = ref.Worktree
		args = append(args, "--end-of-options", mergeBase)
	} else {
		args = append(args, "--end-of-options", mergeBase, ref.Branch)
	}
	if p != "" {
		args = append(args, "--", p)
	}
	out, truncated, err := readGit(dir, args...)
	if err != nil {
		return nil, err
	}
	res.Truncated = truncated
	if opts.Stat {
		res.Files = parseNumstat(out)
	} else {
		res.Patch = withMarker(out, truncated)
	}

	if ref.Worktree != "" {
		args := []string{"status", "--porcelain=v2", "--untracked-files=all", "--ignore-submodules=all"}
		if p != "" {
			args = append(args, "--", p)
		}
		out, _, err := readGit(ref.Worktree, args...)
		if err != nil {
			return nil, err
		}
		var st GitStatusResult
		parsePorcelainV2(out, &st)
		res.Untracked = st.Untracked
	}
	return res, nil
}

func parseNumstat(out string) []GitDiffFile {
	var files []GitDiffFile
	for _, line := range strings.Split(out, "\n") {
		f := strings.SplitN(line, "\t", 3)
		if len(f) != 3 {
			continue
		}
		file := GitDiffFile{Path: f[2]}
		if f[0] == "-" && f[1] == "-" {
			file.Binary = true
		} else {
			file.Insertions, _ = strconv.Atoi(f[0])
			file.Deletions, _ = strconv.Atoi(f[1])
		}
		files = append(files, file)
	}
	return files
}

const gitLogFormat = "--format=%H%x1f%s%x1f%an%x1f%at%x1e"

// GitLog lists commits: the issue branch's own commits when an issue is
// given, otherwise the default branch, optionally only those touching path.
func (c *Client) GitLog(opts GitLogOptions) (*GitLogResult, error) {
	p, err := optionalRepoPath(opts.Path)
	if err != nil {
		return nil, err
	}
	res := &GitLogResult{Commits: []LogCommit{}}
	rev := DefaultBranch()
	if opts.IssueID != "" {
		ref, err := c.issueGitRef(opts.IssueID)
		if err != nil {
			return nil, err
		}
		res.IssueID, res.Branch = ref.IssueID, ref.Branch
		rev = ref.Base + ".." + ref.Branch
	}
	res.Range = rev

	args := []string{"log", "--no-color", "-n", strconv.Itoa(clampLimit(opts.Limit)), gitLogFormat, "--end-of-options", rev}
	if p != "" {
		args = append(args, "--", p)
	}
	out, truncated, err := readGit(storage.HubRoot(), args...)
	if err != nil {
		return nil, err
	}
	res.Commits = c.parseLog(out)
	res.Truncated = truncated
	return res, nil
}

func (c *Client) parseLog(out string) []LogCommit {
	commits := []LogCommit{}
	for _, rec := range strings.Split(out, "\x1e") {
		f := strings.Split(strings.TrimLeft(rec, "\n"), "\x1f")
		if len(f) != 4 {
			continue
		}
		commits = append(commits, LogCommit{
			SHA: f[0], Subject: f[1], Author: f[2], Date: unixDate(f[3]),
			IssueIDs: parseIssueIDs(c.issuePrefix(), f[1]),
		})
	}
	return commits
}

func unixDate(s string) string {
	n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	if err != nil {
		return ""
	}
	return time.Unix(n, 0).UTC().Format(time.RFC3339)
}

// GitShow returns one commit's metadata, message and patch.
func (c *Client) GitShow(sha, filePath string) (*GitShowResult, error) {
	if err := validSHA(sha); err != nil {
		return nil, err
	}
	p, err := optionalRepoPath(filePath)
	if err != nil {
		return nil, err
	}
	hub := storage.HubRoot()
	full, _, err := readGit(hub, "rev-parse", "--verify", "--quiet", "--end-of-options", sha+"^{commit}")
	if err != nil {
		return nil, fmt.Errorf("commit %s not found", sha)
	}
	full = strings.TrimSpace(full)

	args := []string{"show", "--no-ext-diff", "--no-textconv", "--no-color", "--format=fuller", "--end-of-options", full}
	if p != "" {
		args = append(args, "--", p)
	}
	out, truncated, err := readGit(hub, args...)
	if err != nil {
		return nil, err
	}
	return &GitShowResult{SHA: full, Text: withMarker(out, truncated), Truncated: truncated}, nil
}

// CodeHistory lists the commits, and the issues behind them, that touched a
// file or a line range on the default branch.
func (c *Client) CodeHistory(opts CodeHistoryOptions) (*CodeHistoryResult, error) {
	p, err := validRepoPath(opts.Path)
	if err != nil {
		return nil, err
	}
	hasLines := opts.StartLine != 0 || opts.EndLine != 0
	if hasLines && (opts.StartLine < 1 || opts.EndLine < opts.StartLine) {
		return nil, fmt.Errorf("invalid line range %d-%d: need 1 <= start_line <= end_line", opts.StartLine, opts.EndLine)
	}
	base := DefaultBranch()
	res := &CodeHistoryResult{Path: p, Revision: base, StartLine: opts.StartLine, EndLine: opts.EndLine}
	limit := clampLimit(opts.Limit)

	if hasLines {
		// blame reads "--" after --end-of-options as a revision, so it gets
		// none. base comes from config or git, never from the caller.
		out, truncated, err := readGit(storage.HubRoot(), "blame", "--porcelain", "--no-textconv",
			"-L", fmt.Sprintf("%d,%d", opts.StartLine, opts.EndLine), base, "--", p)
		if err != nil {
			return nil, err
		}
		res.Truncated = truncated
		res.Commits = c.parseBlame(out)
		if len(res.Commits) > limit {
			res.Commits = res.Commits[:limit]
		}
	} else {
		out, truncated, err := readGit(storage.HubRoot(), "log", "--no-color", "--follow",
			"-n", strconv.Itoa(limit), gitLogFormat, "--end-of-options", base, "--", p)
		if err != nil {
			return nil, err
		}
		res.Truncated = truncated
		for _, cm := range c.parseLog(out) {
			res.Commits = append(res.Commits, CodeHistoryCommit{LogCommit: cm})
		}
	}
	if res.Commits == nil {
		res.Commits = []CodeHistoryCommit{}
	}
	res.Issues = c.resolveHistoryIssues(res.Commits)
	return res, nil
}

// parseBlame groups `git blame --porcelain` output by commit, newest first.
func (c *Client) parseBlame(out string) []CodeHistoryCommit {
	byCommit := map[string]*CodeHistoryCommit{}
	var order []string
	var cur *CodeHistoryCommit
	for _, line := range strings.Split(out, "\n") {
		if line == "" || strings.HasPrefix(line, "\t") {
			continue
		}
		f := strings.Fields(line)
		if len(f) >= 3 && len(f[0]) == 40 && shaRe.MatchString(f[0]) {
			sha := f[0]
			finalLine, _ := strconv.Atoi(f[2])
			cur = byCommit[sha]
			if cur == nil {
				cur = &CodeHistoryCommit{LogCommit: LogCommit{SHA: sha}}
				byCommit[sha] = cur
				order = append(order, sha)
			}
			if n := len(cur.Lines); n > 0 && cur.Lines[n-1].End == finalLine-1 {
				cur.Lines[n-1].End = finalLine
			} else {
				cur.Lines = append(cur.Lines, GitLineRange{Start: finalLine, End: finalLine})
			}
			continue
		}
		if cur == nil {
			continue
		}
		key, val, _ := strings.Cut(line, " ")
		switch key {
		case "summary":
			cur.Subject = val
			cur.IssueIDs = parseIssueIDs(c.issuePrefix(), val)
		case "author":
			cur.Author = val
		case "author-time":
			cur.Date = unixDate(val)
		}
	}
	commits := make([]CodeHistoryCommit, 0, len(order))
	for _, sha := range order {
		commits = append(commits, *byCommit[sha])
	}
	sort.SliceStable(commits, func(i, j int) bool { return commits[i].Date > commits[j].Date })
	return commits
}

// resolveHistoryIssues looks up each distinct issue ID once. IDs that no
// longer resolve (deleted, or from another project) are kept without a title.
func (c *Client) resolveHistoryIssues(commits []CodeHistoryCommit) []CodeHistoryIssue {
	issues := []CodeHistoryIssue{}
	seen := map[string]bool{}
	for _, cm := range commits {
		for _, id := range cm.IssueIDs {
			if seen[id] {
				continue
			}
			seen[id] = true
			entry := CodeHistoryIssue{ID: id}
			if issue, err := c.GetIssue(id); err == nil && issue.ID == id {
				entry.Title, entry.Status = issue.Title, string(issue.Status)
			}
			issues = append(issues, entry)
		}
	}
	return issues
}
