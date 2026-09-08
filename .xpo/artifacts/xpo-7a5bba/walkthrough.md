# Walkthrough: Comprehensive merge test coverage

## What was built

11 new test cases for `MergeIssue` in `internal/exponential/merge_test.go`, plus a shared `setupMergeRepo` helper that creates a git repo with an initial commit, a feature branch with one commit, and an xpo issue in DOING state.

## Test inventory

| Test | Code path exercised |
|---|---|
| `TestMergeIssue_MergeStrategy` | `--no-ff` merge, two-parent commit, amend path (line 159) |
| `TestMergeIssue_FFStrategy` | `--ff-only` merge from feature branch, auto-checkout, single-parent commit |
| `TestMergeIssue_FFStrategy_Diverged` | FF on diverged branches, error + abort path (lines 163-165) |
| `TestMergeIssue_CustomCommitMessage` | `opts.CommitMessage` override (lines 96-106) |
| `TestMergeIssue_NoBranch` | `BranchStats == nil` error (line 49) |
| `TestMergeIssue_ZeroCommits` | `Commits == 0` error (line 52) |
| `TestMergeIssue_Conflict` | Squash merge conflict, error message with stash warning, abort |
| `TestMergeIssue_DeleteBranch` | `DeleteBranch: true`, branch removal (lines 183-186) |
| `TestMergeIssue_WorktreeCleanup` | Real linked worktree, removal after merge (lines 173-181) |
| `TestMergeIssue_WorktreeHubWrongBranch` | Hub on wrong branch in worktree mode, error (line 72) |
| `TestMergeIssue_FFPreservesMainIssuesDB` | FF + diverged issues.db, union preservation (lines 111-131) |

## Key decisions

- **`setupMergeRepo` helper** leaves the caller on the feature branch. Tests that need to be on main (diverged FF, conflict) set up their own repos to control the exact state of issues.db across branches.
- **Conflict test** verifies error message content but not clean tree — `git merge --abort` after a squash conflict doesn't fully reset the index, which is correct behavior (the error tells the user to resolve manually).
- **Worktree tests** create real linked worktrees via `git worktree add` rather than mocking, ensuring the `FindWorktreeForBranch` / `WorktreeRemove` integration is tested end-to-end.

## Follow-up

Six separate issues filed for remaining coverage gaps: xpo-b83765 (git.go), xpo-f20122 (local_transport.go), xpo-5bf8b9 (server handlers), xpo-29495d (start.go edges), xpo-f31793 (MCP start/merge), xpo-ada827 (checks.go + root.go).
