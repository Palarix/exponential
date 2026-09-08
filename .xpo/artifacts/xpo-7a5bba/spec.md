# Spec: Add comprehensive merge test coverage

## What

Fill the test gaps in `internal/exponential/merge_test.go` for `MergeIssue` and related functions.

## Why

Only one happy path (squash + branch mode + keep branch) is tested. The auto-checkout regression (xpo-1fccf2) shipped because no test exercised branch-mode merge with `Worktrees: true`. Multiple strategies, error paths, and worktree flows are untested.

## Tests to Add

### Merge strategies
1. **TestMergeIssue_MergeStrategy** — `MergeStrategyMerge` (--no-ff). Verify merge commit is created (two parents), issue transitions to DONE, events recorded.
2. **TestMergeIssue_FFStrategy** — `MergeStrategyFF` on a fast-forwardable branch. Verify HEAD advances without a merge commit.
3. **TestMergeIssue_FFStrategy_Diverged** — FF on a diverged branch. Verify error, branch state unchanged.
4. **TestMergeIssue_CustomCommitMessage** — squash with `opts.CommitMessage` set. Verify the commit message matches.

### Error paths
5. **TestMergeIssue_NoBranch** — issue with no `BranchStats`. Verify descriptive error.
6. **TestMergeIssue_ZeroCommits** — branch exists but has zero commits ahead. Verify error or no-op.
7. **TestMergeIssue_Conflict** — create a true merge conflict. Verify error contains conflicting file name, merge is aborted, branch state is clean.

### Branch deletion
8. **TestMergeIssue_DeleteBranch** — `DeleteBranch: true`. Verify branch is removed after merge.

### Worktree mode
9. **TestMergeIssue_WorktreeCleanup** — set up a real linked worktree. Verify it's removed after merge.
10. **TestMergeIssue_WorktreeHubWrongBranch** — hub on a different branch in true worktree mode. Verify "park the hub" error.

### issues.db preservation
11. **TestMergeIssue_FFPreservesMainIssuesDB** — mirror `TestMergeIssue_SquashPreservesMainIssuesDB` but with FF strategy. Verify events from both sides survive.

## Approach

- Reuse the existing test patterns (tmpdir, runGit helper, storage.ResetHubRoot cleanup)
- Each test is self-contained with its own repo setup
- Extract a shared helper for the common "init repo + create issue + feature branch" setup to reduce boilerplate

## Acceptance Criteria

- [ ] All 11 tests listed above exist and pass
- [ ] `make test` passes
- [ ] No changes to production code (test-only)
