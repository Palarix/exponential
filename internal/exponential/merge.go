package exponential

import (
	"fmt"
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
	result := &MergeResult{}

	gitErr := WithGitLock(func() error {
		// Every git call below targets the hub via hubGit, never the process
		// cwd: an MCP server may be running inside the issue's worktree.
		baseSHA := resolveRef(base)

		// The hub is never switched: .xpo/ is hub state, and checking out
		// another branch would carry it along.
		if current := HubBranch(); current != base {
			return fmt.Errorf("hub checkout is on %s, not %s — park the hub on the default branch before merging", current, base)
		}

		commitMsg := opts.CommitMessage
		if commitMsg == "" {
			switch opts.Strategy {
			case MergeStrategySquash:
				commitMsg = fmt.Sprintf("%s: %s", issue.ID, issue.Title)
			case MergeStrategyFF:
				// A fast-forward creates no commit; the branch tip keeps its
				// message unless one was given (applied when amending below).
			default:
				commitMsg = fmt.Sprintf("Merge branch '%s'", branch)
			}
		}

		// Merge and commit the code before recording anything. issues.db is
		// shared by every agent on the hub, so events can never be taken
		// back: a failure up to here must leave the event log untouched.
		var mergeErr error
		switch opts.Strategy {
		case MergeStrategySquash:
			mergeErr = runGitMerge("--squash", branch)
		case MergeStrategyFF:
			mergeErr = runGitMerge("--ff-only", branch)
		default:
			mergeErr = runGitMerge("--no-ff", "--no-commit", branch)
		}
		if mergeErr != nil {
			abortMerge()
			return mergeFailure(mergeErr)
		}

		if opts.Strategy != MergeStrategyFF {
			if err := runHubCommit("-m", commitMsg); err != nil {
				abortMerge()
				return fmt.Errorf("commit failed — nothing was merged, %s is still %s: %w%s",
					issue.ID, issue.Status, err, errDetail(err))
			}
		}
		mergeSHA := resolveRef("HEAD")

		if err := c.recordMerge(issue, branch, baseSHA, opts.Strategy, result); err != nil {
			return fmt.Errorf("merged %s into %s as %s but failed to record the merge: %w\nThe code is merged — do not re-run merge. Transition %s to DONE manually.",
				branch, base, mergeSHA, err, issue.ID)
		}

		// Fold the xpo bookkeeping into the merge commit, or for ff into the
		// fast-forwarded branch tip (the branch always has commits ahead, and
		// it and its worktree are removed below). Hooks already ran on the
		// code commit, so skip them here. If this fails the events stay
		// recorded but uncommitted, and ride along with the next commit.
		hubGit("add", ".xpo/issues.db").Run()
		hubGit("add", filepath.Join(".xpo", "artifacts", issue.ID)).Run()
		amendArgs := []string{"--amend", "--no-edit", "--no-verify"}
		if opts.Strategy == MergeStrategyFF && opts.CommitMessage != "" {
			amendArgs = []string{"--amend", "--no-verify", "-m", opts.CommitMessage}
		}
		if bookkeepErr := runHubCommit(amendArgs...); bookkeepErr != nil {
			hubGit("reset", "-q", "--", ".xpo").Run()
			result.Messages = append(result.Messages, fmt.Sprintf(
				"Warning: merge recorded but .xpo changes were not committed (%v%s) — they will be included in the next commit",
				bookkeepErr, errDetail(bookkeepErr)))
		}

		result.MergeSHA = resolveRef("HEAD")
		result.Messages = append(result.Messages, fmt.Sprintf("Merged %s into %s (%s)", branch, base, opts.Strategy))

		// Clean up worktree (always — worktrees are ephemeral, even with --keep-branch)
		if wtPath, ok := FindWorktreeForBranch(branch); ok {
			if err := WorktreeRemove(wtPath); err != nil {
				result.Messages = append(result.Messages, fmt.Sprintf("Warning: failed to remove worktree %s: %v", wtPath, err))
			} else {
				result.Messages = append(result.Messages, fmt.Sprintf("Removed worktree %s", wtPath))
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

// recordMerge appends the MERGE event and the DONE transition. It must only
// run once the merge commit exists on the base branch.
func (c *Client) recordMerge(issue *model.Issue, branch, baseSHA string, strategy MergeStrategy, result *MergeResult) error {
	preEvents, err := storage.ReadEvents()
	if err != nil {
		return err
	}
	doneStatus := string(model.StatusDone)
	doneEvents, doneMessages, err := c.local.buildUpdate(
		issue.ID, model.UpdatePayload{Status: &doneStatus}, ProjectIssues(preEvents))
	if err != nil {
		return fmt.Errorf("failed to build DONE transition: %w", err)
	}

	event := model.Event{
		ID:   issue.ID,
		Type: model.EventTypeMerge,
		Payload: model.MergePayload{
			Branch:   branch,
			BaseSHA:  baseSHA,
			MergeSHA: "",
			Strategy: string(strategy),
		},
		CreatedAt: time.Now().UTC(),
		CreatedBy: c.Transport.GetUser(),
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
	return nil
}

func IsWorkingTreeClean() bool {
	out, err := exec.Command("git", "status", "--porcelain").Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == ""
}

// CheckMergeConflicts tests whether the committed branch tip can merge cleanly
// into the default branch using git merge-tree. It compares only committed refs
// and is unaffected by working-directory state. Returns the list of conflicting
// files (empty if the merge is clean).
func CheckMergeConflicts(branch string) ([]string, error) {
	hub := storage.HubRoot()
	base := DefaultBranch()

	cmd := exec.Command("git", "-C", hub, "merge-tree", "--write-tree", base, branch)
	out, err := cmd.CombinedOutput()
	if err == nil {
		return nil, nil
	}

	exitErr, ok := err.(*exec.ExitError)
	if !ok || exitErr.ExitCode() > 1 {
		return nil, fmt.Errorf("git merge-tree failed: %w\n%s", err, out)
	}

	// Exit code 1 means conflicts. Parse CONFLICT lines for file names.
	var conflicts []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "CONFLICT") {
			continue
		}
		// Format: "CONFLICT (content): Merge conflict in <file>"
		if idx := strings.Index(line, "Merge conflict in "); idx >= 0 {
			conflicts = append(conflicts, strings.TrimSpace(line[idx+len("Merge conflict in "):]))
			continue
		}
		// Format: "CONFLICT (modify/delete): <file> deleted in ... and modified in ..."
		if idx := strings.Index(line, "): "); idx >= 0 {
			rest := line[idx+3:]
			if sp := strings.IndexByte(rest, ' '); sp >= 0 {
				conflicts = append(conflicts, rest[:sp])
			}
		}
	}

	if len(conflicts) == 0 {
		conflicts = append(conflicts, "unknown conflict (could not parse merge-tree output)")
	}
	return conflicts, nil
}

// HubRequireCleanTree checks that the hub working tree has no uncommitted
// tracked changes (excluding .xpo/ paths and untracked files). Use this as
// a precondition before performing an actual merge operation.
func HubRequireCleanTree() error {
	hub := storage.HubRoot()
	out, err := exec.Command("git", "-C", hub, "status", "--porcelain").Output()
	if err != nil {
		return fmt.Errorf("failed to check working tree status: %w", err)
	}

	var dirty []string
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
		dirty = append(dirty, path)
	}

	if len(dirty) == 0 {
		return nil
	}

	msg := "uncommitted changes — commit or discard before merging:\n"
	for _, f := range dirty {
		msg += fmt.Sprintf("  - %s\n", f)
	}
	msg += "Do NOT stash — .xpo/issues.db must not be stashed."
	return fmt.Errorf("%s", msg)
}

// HubDirtyTrackedFiles returns the list of modified tracked files in the hub
// working tree, excluding .xpo/ paths and untracked files.
func HubDirtyTrackedFiles() []string {
	hub := storage.HubRoot()
	out, err := exec.Command("git", "-C", hub, "status", "--porcelain").Output()
	if err != nil {
		return nil
	}

	var dirty []string
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
		dirty = append(dirty, path)
	}
	return dirty
}

// WorktreeDirtyFiles returns uncommitted and untracked files in the worktree
// for the given branch (excluding .xpo/). Returns nil if no worktree exists.
func WorktreeDirtyFiles(branch string) []string {
	wtPath, ok := FindWorktreeForBranch(branch)
	if !ok {
		return nil
	}

	out, err := exec.Command("git", "-C", wtPath, "status", "--porcelain").Output()
	if err != nil {
		return nil
	}

	var dirty []string
	for _, line := range strings.Split(string(out), "\n") {
		if len(line) < 3 {
			continue
		}
		path := strings.TrimSpace(line[3:])
		if strings.HasPrefix(path, ".xpo/") {
			continue
		}
		dirty = append(dirty, path)
	}
	return dirty
}

// WorktreeRequireClean checks that the worktree for the given branch has no
// uncommitted or untracked files (excluding .xpo/). This prevents silent data
// loss when the worktree is removed after merge. Returns nil if no worktree
// exists for the branch (branch-only mode).
func WorktreeRequireClean(branch string) error {
	dirty := WorktreeDirtyFiles(branch)
	if len(dirty) == 0 {
		return nil
	}

	msg := "worktree has uncommitted or untracked files that would be lost on merge:\n"
	for _, f := range dirty {
		msg += fmt.Sprintf("  - %s\n", f)
	}
	msg += "Commit, move, or remove these files before merging."
	return fmt.Errorf("%s", msg)
}

// HubCleanForMerge checks whether the hub is ready to merge the given branch.
// It checks committed-ref mergeability, hub working-tree, and worktree cleanliness.
func HubCleanForMerge(branch string) error {
	conflicts, err := CheckMergeConflicts(branch)
	if err != nil {
		return err
	}
	if len(conflicts) > 0 {
		msg := "merge conflicts between committed refs:\n"
		for _, f := range conflicts {
			msg += fmt.Sprintf("  - %s\n", f)
		}
		return fmt.Errorf("%s", msg)
	}
	if err := HubRequireCleanTree(); err != nil {
		return err
	}
	return WorktreeRequireClean(branch)
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
	out, err := hubGit("rev-parse", "--short", ref).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

type mergeError struct {
	err    error
	output string
}

func (e *mergeError) Error() string { return e.err.Error() }
func (e *mergeError) Unwrap() error { return e.err }

// hubGit runs git against the hub checkout regardless of the process cwd.
// Never os.Chdir instead: cwd is process-global and servers handle
// requests concurrently.
func hubGit(args ...string) *exec.Cmd {
	return exec.Command("git", append([]string{"-C", storage.HubRoot()}, args...)...)
}

func runGitMerge(args ...string) error {
	return runHubGit(append([]string{"merge"}, args...)...)
}

func runHubCommit(args ...string) error {
	return runHubGit(append([]string{"commit"}, args...)...)
}

func runHubGit(args ...string) error {
	out, err := hubGit(args...).CombinedOutput()
	if err != nil {
		return &mergeError{err: err, output: strings.TrimSpace(string(out))}
	}
	return nil
}

// abortMerge cleans up the hub's index after a failed merge or commit.
// --no-ff sets MERGE_HEAD so --abort works; squash does not, so fall back
// to reset --merge. Neither touches files that differ only between the
// index and the working tree, so concurrent issues.db appends survive.
func abortMerge() {
	if err := hubGit("merge", "--abort").Run(); err != nil {
		hubGit("reset", "--merge").Run()
	}
}

// errDetail returns git's captured output for err, prefixed with a newline.
func errDetail(err error) string {
	if me, ok := err.(*mergeError); ok && me.output != "" {
		return "\n" + me.output
	}
	return ""
}

// mergeFailure builds the error for a failed `git merge`. Only a real
// conflict gets the resolve-the-conflict guidance.
func mergeFailure(err error) error {
	detail := errDetail(err)
	if strings.Contains(detail, "CONFLICT") {
		return fmt.Errorf("merge failed: %w%s\nDo NOT stash or reset. Resolve the conflict in the listed files, then re-run xpo merge.", err, detail)
	}
	return fmt.Errorf("merge failed — nothing was merged: %w%s", err, detail)
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
	hubGit("branch", "-D", name).Run()

	// Delete remote tracking branch if it exists
	if strings.HasPrefix(branch, "origin/") {
		remoteBranch := strings.TrimPrefix(branch, "origin/")
		hubGit("push", "origin", "--delete", remoteBranch).Run()
	}
}
