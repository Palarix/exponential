# Spec: Fix `computeBranchStats` for worktree uncommitted changes

## What

`computeBranchStats` fails to detect uncommitted changes in worktree branches because `fillUncommittedStats` only runs when `branch == CurrentBranch()`. In worktree mode, the hub checkout is on `main`, so this condition is never true for issue branches.

## Why

Without this fix, worktree branches with uncommitted (but not yet committed) changes show `has_uncommitted: false` and `commits: 0`, which hides the merge view button in the web UI. Users can't review their changes until they commit.

## How

1. **Modify `computeBranchStats`** — when `commits == 0` and the branch is not the current branch, check if the branch has a worktree via `FindWorktreeForBranch`. If a worktree exists, call `fillUncommittedStats` targeting that directory.

2. **Modify `fillUncommittedStats`** — add a `dir` parameter (empty string = current directory, non-empty = use `git -C <dir>`). This avoids duplicating the function while keeping the existing behavior for non-worktree mode.

## Acceptance Criteria

- Worktree branches with uncommitted changes report `has_uncommitted: true`
- The merge view button appears for worktree issues with uncommitted changes
- Non-worktree (checkout) mode behavior is unchanged
- Existing tests continue to pass
