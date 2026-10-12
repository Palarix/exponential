package exponential

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/palarix/exponential/internal/config"
)

// setupGitReadEnv creates a git repo with an .xpo workspace and returns a
// local client bound to it.
func setupGitReadEnv(t *testing.T) (*Client, string) {
	t.Helper()
	dir := initTestRepo(t, "main")
	os.MkdirAll(filepath.Join(dir, ".xpo"), 0755)
	os.WriteFile(filepath.Join(dir, ".xpo", "issues.db"), []byte{}, 0644)
	commitFile(t, dir, ".gitignore", ".xpo/\n", "ignore .xpo")
	cfg := &config.Config{
		Prefix:           "test",
		User:             "Test User <test@test.com>",
		EstimationSystem: "fibonacci",
		Version:          2,
	}
	return NewClient(cfg), dir
}

func commitFile(t *testing.T, dir, name, content, msg string) {
	t.Helper()
	path := filepath.Join(dir, name)
	os.MkdirAll(filepath.Dir(path), 0755)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "add", name)
	runGit(t, dir, "commit", "-m", msg)
}

// startGitReadIssue creates a PLANNED issue and starts it, returning the
// worktree path.
func startGitReadIssue(t *testing.T, c *Client, id, title string) string {
	t.Helper()
	createTestIssue(t, id, title, "PLANNED")
	_, wt, _, err := c.StartWork(id, false)
	if err != nil {
		t.Fatalf("StartWork: %v", err)
	}
	if wt == "" {
		t.Fatal("expected a worktree")
	}
	return wt
}

func changePaths(cs []GitFileChange) []string {
	var out []string
	for _, c := range cs {
		out = append(out, c.Path)
	}
	return out
}

// --- git_status ---

func TestGitStatus_Worktree(t *testing.T) {
	c, dir := setupGitReadEnv(t)
	wt := startGitReadIssue(t, c, "test-a1b2c3", "Status")

	commitFile(t, wt, "feature.txt", "one\n", "test-a1b2c3: add feature")
	commitFile(t, dir, "main-only.txt", "main\n", "later main commit")

	os.WriteFile(filepath.Join(wt, "feature.txt"), []byte("two\n"), 0644)
	os.WriteFile(filepath.Join(wt, "staged.txt"), []byte("s\n"), 0644)
	runGit(t, wt, "add", "staged.txt")
	os.WriteFile(filepath.Join(wt, "scratch.txt"), []byte("u\n"), 0644)

	got, err := c.GitStatus("a1b2c3")
	if err != nil {
		t.Fatalf("GitStatus: %v", err)
	}
	if got.IssueID != "test-a1b2c3" {
		t.Errorf("IssueID = %q, want the canonical ID", got.IssueID)
	}
	if !strings.HasPrefix(got.Branch, "test-a1b2c3") {
		t.Errorf("Branch = %q", got.Branch)
	}
	if got.BaseBranch != "main" {
		t.Errorf("BaseBranch = %q, want main", got.BaseBranch)
	}
	if got.WorktreePath != wt {
		t.Errorf("WorktreePath = %q, want %q", got.WorktreePath, wt)
	}
	if got.Ahead != 1 || got.Behind != 1 {
		t.Errorf("ahead/behind = %d/%d, want 1/1", got.Ahead, got.Behind)
	}
	if !slices.Contains(changePaths(got.Staged), "staged.txt") {
		t.Errorf("Staged = %v, want staged.txt", got.Staged)
	}
	if !slices.Contains(changePaths(got.Unstaged), "feature.txt") {
		t.Errorf("Unstaged = %v, want feature.txt", got.Unstaged)
	}
	if !slices.Contains(got.Untracked, "scratch.txt") {
		t.Errorf("Untracked = %v, want scratch.txt", got.Untracked)
	}
}

func TestGitStatus_NoBranch(t *testing.T) {
	c, _ := setupGitReadEnv(t)
	createTestIssue(t, "test-0000aa", "Not started", "PLANNED")

	_, err := c.GitStatus("test-0000aa")
	if err == nil {
		t.Fatal("expected an error for an issue with no branch")
	}
	if !strings.Contains(err.Error(), "test-0000aa") || !strings.Contains(err.Error(), "no branch") {
		t.Errorf("error = %q, want it to name the issue and say no branch", err)
	}
}

func TestGitStatus_UnknownIssue(t *testing.T) {
	c, _ := setupGitReadEnv(t)
	if _, err := c.GitStatus("test-ffffff"); err == nil {
		t.Fatal("expected an error for an unknown issue")
	}
}

// --- git_diff ---

func TestGitDiff_WorktreeIncludesUncommitted(t *testing.T) {
	c, dir := setupGitReadEnv(t)
	wt := startGitReadIssue(t, c, "test-b1b2c3", "Diff")

	commitFile(t, wt, "committed.txt", "committed line\n", "test-b1b2c3: commit")
	os.WriteFile(filepath.Join(wt, "init.txt"), []byte("init\nuncommitted line\n"), 0644)
	os.WriteFile(filepath.Join(wt, "new-untracked.txt"), []byte("x\n"), 0644)
	commitFile(t, dir, "main-only.txt", "main only line\n", "later main commit")

	got, err := c.GitDiff("test-b1b2c3", GitDiffOptions{})
	if err != nil {
		t.Fatalf("GitDiff: %v", err)
	}
	if !strings.Contains(got.Patch, "committed line") {
		t.Error("patch is missing the committed change")
	}
	if !strings.Contains(got.Patch, "uncommitted line") {
		t.Error("patch is missing the uncommitted change")
	}
	if strings.Contains(got.Patch, "main only line") {
		t.Error("patch contains a change made on main after the merge base")
	}
	if !got.IncludesUncommitted {
		t.Error("IncludesUncommitted = false, want true with a worktree")
	}
	if !slices.Contains(got.Untracked, "new-untracked.txt") {
		t.Errorf("Untracked = %v, want new-untracked.txt", got.Untracked)
	}
	if got.MergeBase == "" {
		t.Error("MergeBase is empty")
	}
}

func TestGitDiff_BranchWithoutWorktree(t *testing.T) {
	c, _ := setupGitReadEnv(t)
	wt := startGitReadIssue(t, c, "test-c1b2c3", "No worktree")
	commitFile(t, wt, "branch.txt", "on the branch\n", "test-c1b2c3: branch work")
	if err := WorktreeRemove(wt); err != nil {
		t.Fatalf("WorktreeRemove: %v", err)
	}

	got, err := c.GitDiff("test-c1b2c3", GitDiffOptions{})
	if err != nil {
		t.Fatalf("GitDiff: %v", err)
	}
	if !strings.Contains(got.Patch, "on the branch") {
		t.Errorf("patch = %q, want the branch change", got.Patch)
	}
	if got.WorktreePath != "" || got.IncludesUncommitted {
		t.Errorf("WorktreePath=%q IncludesUncommitted=%v, want none", got.WorktreePath, got.IncludesUncommitted)
	}
}

func TestGitDiff_Stat(t *testing.T) {
	c, _ := setupGitReadEnv(t)
	wt := startGitReadIssue(t, c, "test-d1b2c3", "Stat")
	commitFile(t, wt, "a.txt", "1\n2\n3\n", "test-d1b2c3: a")
	os.WriteFile(filepath.Join(wt, "init.txt"), []byte("changed\n"), 0644)
	os.WriteFile(filepath.Join(wt, "bin.dat"), []byte{0, 1, 2, 0, 3}, 0644)
	runGit(t, wt, "add", "bin.dat")

	got, err := c.GitDiff("test-d1b2c3", GitDiffOptions{Stat: true})
	if err != nil {
		t.Fatalf("GitDiff: %v", err)
	}
	if got.Patch != "" {
		t.Errorf("Patch = %q, want empty with stat", got.Patch)
	}
	files := map[string]GitDiffFile{}
	for _, f := range got.Files {
		files[f.Path] = f
	}
	if f := files["a.txt"]; f.Insertions != 3 || f.Deletions != 0 {
		t.Errorf("a.txt = %+v, want +3 -0", f)
	}
	if f := files["init.txt"]; f.Insertions != 1 || f.Deletions != 1 {
		t.Errorf("init.txt = %+v, want +1 -1", f)
	}
	if f := files["bin.dat"]; !f.Binary {
		t.Errorf("bin.dat = %+v, want Binary", f)
	}
}

func TestGitDiff_Path(t *testing.T) {
	c, _ := setupGitReadEnv(t)
	wt := startGitReadIssue(t, c, "test-e1b2c3", "Path")
	commitFile(t, wt, "keep.txt", "keep me\n", "test-e1b2c3: keep")
	commitFile(t, wt, "skip.txt", "skip me\n", "test-e1b2c3: skip")

	got, err := c.GitDiff("test-e1b2c3", GitDiffOptions{Path: "keep.txt"})
	if err != nil {
		t.Fatalf("GitDiff: %v", err)
	}
	if !strings.Contains(got.Patch, "keep me") || strings.Contains(got.Patch, "skip me") {
		t.Errorf("patch = %q, want only keep.txt", got.Patch)
	}
}

func TestGitDiff_Truncates(t *testing.T) {
	c, _ := setupGitReadEnv(t)
	wt := startGitReadIssue(t, c, "test-f1b2c3", "Big")
	big := strings.Repeat("a line of text that makes the diff large\n", 4000)
	commitFile(t, wt, "big.txt", big, "test-f1b2c3: big")

	got, err := c.GitDiff("test-f1b2c3", GitDiffOptions{})
	if err != nil {
		t.Fatalf("GitDiff: %v", err)
	}
	if !got.Truncated {
		t.Fatal("Truncated = false, want true for a diff over 64 KiB")
	}
	if len(got.Patch) > gitReadCap+200 {
		t.Errorf("patch is %d bytes, want about %d", len(got.Patch), gitReadCap)
	}
	if !strings.HasSuffix(strings.TrimSpace(got.Patch), gitTruncatedMarker) {
		t.Error("patch does not end with the truncation marker")
	}
}

// --- git_log ---

func TestGitLog_Issue(t *testing.T) {
	c, dir := setupGitReadEnv(t)
	wt := startGitReadIssue(t, c, "test-a2b2c3", "Log")
	commitFile(t, wt, "one.txt", "1\n", "test-a2b2c3: first, see test-0b0b0b")
	commitFile(t, wt, "two.txt", "2\n", "second without prefix")
	commitFile(t, dir, "main.txt", "m\n", "test-999999: main work")

	got, err := c.GitLog(GitLogOptions{IssueID: "test-a2b2c3"})
	if err != nil {
		t.Fatalf("GitLog: %v", err)
	}
	if len(got.Commits) != 2 {
		t.Fatalf("got %d commits, want 2 branch commits: %+v", len(got.Commits), got.Commits)
	}
	if got.Commits[0].Subject != "second without prefix" {
		t.Errorf("first commit = %q, want newest first", got.Commits[0].Subject)
	}
	first := got.Commits[1]
	if !slices.Equal(first.IssueIDs, []string{"test-a2b2c3", "test-0b0b0b"}) {
		t.Errorf("IssueIDs = %v", first.IssueIDs)
	}
	if len(first.SHA) != 40 || first.Author != "Test" || first.Date == "" {
		t.Errorf("commit = %+v", first)
	}
	if _, err := time.Parse(time.RFC3339, first.Date); err != nil {
		t.Errorf("Date %q is not RFC 3339", first.Date)
	}
}

func TestGitLog_PathOnDefaultBranch(t *testing.T) {
	c, dir := setupGitReadEnv(t)
	commitFile(t, dir, "src/a.go", "package a\n", "test-a3b2c3: add a")
	commitFile(t, dir, "src/b.go", "package b\n", "test-b3b2c3: add b")

	got, err := c.GitLog(GitLogOptions{Path: "src/a.go"})
	if err != nil {
		t.Fatalf("GitLog: %v", err)
	}
	if len(got.Commits) != 1 || got.Commits[0].Subject != "test-a3b2c3: add a" {
		t.Errorf("commits = %+v, want only the a.go commit", got.Commits)
	}
}

func TestGitLog_LimitClamp(t *testing.T) {
	c, dir := setupGitReadEnv(t)
	for i := range 3 {
		commitFile(t, dir, "f.txt", strings.Repeat("x", i+1), "commit")
	}

	got, err := c.GitLog(GitLogOptions{Limit: 2})
	if err != nil {
		t.Fatalf("GitLog: %v", err)
	}
	if len(got.Commits) != 2 {
		t.Errorf("got %d commits, want 2", len(got.Commits))
	}
	if clampLimit(0) != 20 || clampLimit(-5) != 20 || clampLimit(500) != 100 || clampLimit(7) != 7 {
		t.Error("clampLimit does not default to 20 and clamp to 1–100")
	}
}

// --- git_show ---

func TestGitShow(t *testing.T) {
	c, dir := setupGitReadEnv(t)
	commitFile(t, dir, "shown.txt", "shown content\n", "test-a4b2c3: show me")
	sha := strings.TrimSpace(gitOut(t, dir, "rev-parse", "HEAD"))

	got, err := c.GitShow(sha[:10], "")
	if err != nil {
		t.Fatalf("GitShow: %v", err)
	}
	if !strings.Contains(got.Text, "test-a4b2c3: show me") || !strings.Contains(got.Text, "+shown content") {
		t.Errorf("text = %q, want message and patch", got.Text)
	}
	if got.SHA != sha {
		t.Errorf("SHA = %q, want the full SHA %q", got.SHA, sha)
	}
}

func TestGitShow_RejectsBadSHA(t *testing.T) {
	c, _ := setupGitReadEnv(t)
	for _, bad := range []string{"", "HEAD", "--output=/tmp/x", "abc", "main", "a1b2c3d4..e5f6"} {
		if _, err := c.GitShow(bad, ""); err == nil {
			t.Errorf("GitShow(%q) succeeded, want an error", bad)
		}
	}
}

// --- code_history ---

func TestCodeHistory_File(t *testing.T) {
	c, dir := setupGitReadEnv(t)
	createTestIssue(t, "test-a5b2c3", "Add the parser", "DONE")
	commitFile(t, dir, "parser.go", "v1\n", "test-a5b2c3: Add the parser")
	commitFile(t, dir, "parser.go", "v2\n", "test-0dead0: Gone issue")
	commitFile(t, dir, "other.go", "x\n", "test-a6b2c3: unrelated")

	got, err := c.CodeHistory(CodeHistoryOptions{Path: "parser.go"})
	if err != nil {
		t.Fatalf("CodeHistory: %v", err)
	}
	if len(got.Commits) != 2 {
		t.Fatalf("got %d commits, want 2", len(got.Commits))
	}
	issues := map[string]CodeHistoryIssue{}
	for _, i := range got.Issues {
		issues[i.ID] = i
	}
	if i := issues["test-a5b2c3"]; i.Title != "Add the parser" || i.Status != "DONE" {
		t.Errorf("resolved issue = %+v", i)
	}
	if i, ok := issues["test-0dead0"]; !ok || i.Title != "" {
		t.Errorf("unresolved issue = %+v (present %v), want it kept with no title", i, ok)
	}
	if _, ok := issues["test-a6b2c3"]; ok {
		t.Error("issues include a commit that did not touch the file")
	}
}

func TestCodeHistory_Lines(t *testing.T) {
	c, dir := setupGitReadEnv(t)
	commitFile(t, dir, "lines.txt", "a\nb\nc\nd\n", "test-a7b2c3: first")
	commitFile(t, dir, "lines.txt", "a\nB\nC\nd\n", "test-b7b2c3: second")

	got, err := c.CodeHistory(CodeHistoryOptions{Path: "lines.txt", StartLine: 2, EndLine: 3})
	if err != nil {
		t.Fatalf("CodeHistory: %v", err)
	}
	if len(got.Commits) != 1 {
		t.Fatalf("commits = %+v, want only the second commit", got.Commits)
	}
	cm := got.Commits[0]
	if cm.Subject != "test-b7b2c3: second" {
		t.Errorf("subject = %q", cm.Subject)
	}
	if !slices.Equal(cm.Lines, []GitLineRange{{Start: 2, End: 3}}) {
		t.Errorf("Lines = %+v, want [2-3]", cm.Lines)
	}

	got, err = c.CodeHistory(CodeHistoryOptions{Path: "lines.txt", StartLine: 1, EndLine: 4})
	if err != nil {
		t.Fatalf("CodeHistory: %v", err)
	}
	if len(got.Commits) != 2 {
		t.Fatalf("commits = %+v, want both", got.Commits)
	}
	i := slices.IndexFunc(got.Commits, func(c CodeHistoryCommit) bool { return c.Subject == "test-a7b2c3: first" })
	if i < 0 {
		t.Fatalf("first commit missing: %+v", got.Commits)
	}
	if first := got.Commits[i]; !slices.Equal(first.Lines, []GitLineRange{{1, 1}, {4, 4}}) {
		t.Errorf("first commit lines = %+v, want [1-1 4-4]", first.Lines)
	}
}

func TestCodeHistory_BadLineRange(t *testing.T) {
	c, dir := setupGitReadEnv(t)
	commitFile(t, dir, "x.txt", "a\n", "x")
	for _, o := range []CodeHistoryOptions{
		{Path: "x.txt", StartLine: 3, EndLine: 1},
		{Path: "x.txt", StartLine: 0, EndLine: 2},
		{Path: "x.txt", EndLine: 2},
		{Path: ""},
	} {
		if _, err := c.CodeHistory(o); err == nil {
			t.Errorf("CodeHistory(%+v) succeeded, want an error", o)
		}
	}
}

// --- argument validation ---

func TestGitRead_RejectsBadPaths(t *testing.T) {
	c, _ := setupGitReadEnv(t)
	startGitReadIssue(t, c, "test-a8b2c3", "Paths")
	for _, bad := range []string{"../outside", "a/../../outside", "/etc/passwd", "--output=/tmp/x", "-p", "..", "."} {
		if _, err := c.GitDiff("test-a8b2c3", GitDiffOptions{Path: bad}); err == nil {
			t.Errorf("GitDiff path %q succeeded, want an error", bad)
		}
		if _, err := c.GitLog(GitLogOptions{Path: bad}); err == nil {
			t.Errorf("GitLog path %q succeeded, want an error", bad)
		}
		if _, err := c.GitShow("abcd1234", bad); err == nil {
			t.Errorf("GitShow path %q succeeded, want an error", bad)
		}
		if _, err := c.CodeHistory(CodeHistoryOptions{Path: bad}); err == nil {
			t.Errorf("CodeHistory path %q succeeded, want an error", bad)
		}
	}
}

func TestValidRepoPath_CleansGoodPaths(t *testing.T) {
	got, err := validRepoPath("src//a/./b.go")
	if err != nil || got != "src/a/b.go" {
		t.Errorf("validRepoPath = %q, %v; want src/a/b.go", got, err)
	}
}

// --- side effects ---

// hostileRepoConfig sets diff.external, a textconv driver and core.fsmonitor
// to scripts that create marker files, and returns the marker directory.
func hostileRepoConfig(t *testing.T, dir string) string {
	t.Helper()
	markers := t.TempDir()
	script := func(name string) string {
		p := filepath.Join(markers, name+".sh")
		body := "#!/bin/sh\ntouch " + filepath.Join(markers, name+".ran") + "\ncat \"$1\" 2>/dev/null\nexit 0\n"
		if err := os.WriteFile(p, []byte(body), 0755); err != nil {
			t.Fatal(err)
		}
		return p
	}
	runGit(t, dir, "config", "diff.external", script("external"))
	runGit(t, dir, "config", "diff.evil.textconv", script("textconv"))
	runGit(t, dir, "config", "core.fsmonitor", script("fsmonitor"))
	commitFile(t, dir, ".gitattributes", "*.txt diff=evil\n", "attributes")
	return markers
}

func ranMarkers(t *testing.T, markers string) []string {
	t.Helper()
	m, _ := filepath.Glob(filepath.Join(markers, "*.ran"))
	return m
}

func TestGitRead_RunsNoRepoPrograms(t *testing.T) {
	c, dir := setupGitReadEnv(t)
	markers := hostileRepoConfig(t, dir)
	wt := startGitReadIssue(t, c, "test-a9b2c3", "Hostile")
	commitFile(t, wt, "f.txt", "one\n", "test-a9b2c3: f")
	os.WriteFile(filepath.Join(wt, "f.txt"), []byte("two\n"), 0644)
	sha := strings.TrimSpace(gitOut(t, wt, "rev-parse", "HEAD"))
	commitFile(t, dir, "g.txt", "a\nb\n", "test-b9b2c3: g")

	// Sanity check: plain git runs each script with this config.
	gitOut(t, wt, "diff", "HEAD")
	gitOut(t, wt, "diff", "--no-ext-diff", "HEAD")
	gitOut(t, wt, "status")
	for _, name := range []string{"external", "textconv", "fsmonitor"} {
		if _, err := os.Stat(filepath.Join(markers, name+".ran")); err != nil {
			t.Fatalf("plain git did not run the %s script; the test proves nothing", name)
		}
	}
	for _, m := range ranMarkers(t, markers) {
		os.Remove(m)
	}

	calls := map[string]func() error{
		"status": func() error { _, err := c.GitStatus("test-a9b2c3"); return err },
		"diff":   func() error { _, err := c.GitDiff("test-a9b2c3", GitDiffOptions{}); return err },
		"stat":   func() error { _, err := c.GitDiff("test-a9b2c3", GitDiffOptions{Stat: true}); return err },
		"log":    func() error { _, err := c.GitLog(GitLogOptions{IssueID: "test-a9b2c3"}); return err },
		"show":   func() error { _, err := c.GitShow(sha, ""); return err },
		"history": func() error {
			_, err := c.CodeHistory(CodeHistoryOptions{Path: "g.txt"})
			return err
		},
		"blame": func() error {
			_, err := c.CodeHistory(CodeHistoryOptions{Path: "g.txt", StartLine: 1, EndLine: 2})
			return err
		},
	}
	for name, call := range calls {
		if err := call(); err != nil {
			t.Errorf("%s: %v", name, err)
		}
		if ran := ranMarkers(t, markers); len(ran) > 0 {
			t.Errorf("%s ran repo-configured programs: %v", name, ran)
			for _, m := range ran {
				os.Remove(m)
			}
		}
	}
}

func TestGitRead_DoesNotWriteIndex(t *testing.T) {
	c, dir := setupGitReadEnv(t)
	wt := startGitReadIssue(t, c, "test-c9b2c3", "Index")
	commitFile(t, wt, "f.txt", "one\n", "test-c9b2c3: f")

	// Make the stat info stale so a plain `git status` would refresh the index.
	future := time.Now().Add(time.Hour)
	os.Chtimes(filepath.Join(wt, "f.txt"), future, future)
	os.Chtimes(filepath.Join(dir, "init.txt"), future, future)

	wtGitDir := strings.TrimSpace(gitOut(t, wt, "rev-parse", "--absolute-git-dir"))
	indexes := []string{filepath.Join(dir, ".git", "index"), filepath.Join(wtGitDir, "index")}
	before := map[string]time.Time{}
	for _, p := range indexes {
		st, err := os.Stat(p)
		if err != nil {
			t.Fatal(err)
		}
		before[p] = st.ModTime()
	}

	if _, err := c.GitStatus("test-c9b2c3"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.GitDiff("test-c9b2c3", GitDiffOptions{}); err != nil {
		t.Fatal(err)
	}

	for _, p := range indexes {
		st, _ := os.Stat(p)
		if !st.ModTime().Equal(before[p]) {
			t.Errorf("%s was modified", p)
		}
		if _, err := os.Stat(p + ".lock"); err == nil {
			t.Errorf("%s.lock exists", p)
		}
	}
}

// --- transports ---

func TestGitDiff_RemoteTransport(t *testing.T) {
	local, _ := setupGitReadEnv(t)
	wt := startGitReadIssue(t, local, "test-d9b2c3", "Remote")
	commitFile(t, wt, "remote.txt", "from the local worktree\n", "test-d9b2c3: work")

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/issues/{id}", func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix("test-d9b2c3", r.PathValue("id")) {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(apiIssue{ID: "test-d9b2c3", Title: "Remote", Status: "DOING"})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	cfg := &config.Config{Prefix: "test", Version: 2}
	cfg.Remote.URL = srv.URL
	remote := NewClient(cfg)

	got, err := remote.GitDiff("d9b2c3", GitDiffOptions{})
	if err != nil {
		t.Fatalf("GitDiff over a remote transport: %v", err)
	}
	if got.IssueID != "test-d9b2c3" || !strings.Contains(got.Patch, "from the local worktree") {
		t.Errorf("got %+v, want the local worktree diff", got)
	}
}

func TestParseIssueIDs(t *testing.T) {
	got := parseIssueIDs("xpo", "xpo-a1b2c3: fix, see xpo-d4e5f6 and xpo-a1b2c3; not xpo-zzzzzz or xpo-a1b2c3d or myxpo-111111")
	if !slices.Equal(got, []string{"xpo-a1b2c3", "xpo-d4e5f6"}) {
		t.Errorf("parseIssueIDs = %v", got)
	}
	if got := parseIssueIDs("", "issue-abcdef: default prefix"); !slices.Equal(got, []string{"issue-abcdef"}) {
		t.Errorf("default prefix: %v", got)
	}
}
