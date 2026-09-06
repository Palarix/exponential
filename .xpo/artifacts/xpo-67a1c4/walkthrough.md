# Walkthrough: Fix `computeBranchStats` for worktree uncommitted changes

## What was built

Fixed `computeBranchStats` in `internal/exponential/branches.go` so that uncommitted changes in worktree directories are detected and reported, even when the hub checkout is on a different branch (e.g. `main`).

## How the pieces fit together

The branch stats pipeline works like this:

1. `applyBranchInference` iterates all issues, finds matching local/remote branches, and calls `computeBranchStats` for each.
2. `computeBranchStats` counts commits ahead of base. If commits > 0, it diffs against base. If commits == 0, it checks for uncommitted changes via `fillUncommittedStats`.
3. The frontend's `PropertySidebar` shows the merge view button when `commits > 0 || has_uncommitted`.

The bug: step 2 only called `fillUncommittedStats` when `branch == CurrentBranch()`. In worktree mode, the server runs from the hub (on `main`), so this condition was never true for issue branches — their uncommitted changes were invisible.

## The fix

**`fillUncommittedStats(stats, dir)`** — added a `dir` parameter. A local `gitCmd` helper prepends `-C <dir>` to all git commands when `dir` is non-empty. When empty, behavior is identical to before (commands run in the CWD). This avoids duplicating the function.

**`computeBranchStats`** — added a third branch after the existing `CurrentBranch()` check:

```go
} else if wtPath, ok := FindWorktreeForBranch(branch); ok {
    fillUncommittedStats(stats, wtPath)
}
```

`FindWorktreeForBranch` (already existed in `git.go`) returns the worktree path for a branch by scanning `git worktree list`. If the branch has a worktree, we run the diff commands there.

## Key decisions

- Reused `FindWorktreeForBranch` rather than adding worktree detection logic — it already does exactly what's needed and is used elsewhere (merge, start).
- Kept the `CurrentBranch()` path unchanged for non-worktree (checkout) mode — zero risk of regression.
