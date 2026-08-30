# Spec: Squash/FF merge overwrites main's issues.db when worktrees disabled

## What
When `worktrees: false`, `MergeIssue` with squash or ff-only strategy silently drops `issues.db` events that exist on `main` but not on the feature branch.

## Why
The `merge=union` gitattribute driver only runs during 3-way merges (`--no-ff`). Squash and ff-only merges apply the branch's diff directly, replacing main's `issues.db`. In worktree mode the hub stays on `main` so this never happens — all `issues.db` writes target the hub. In non-worktree mode the feature branch carries a diverged copy.

## How

### Approach
In `MergeIssue` (`internal/exponential/merge.go`), for non-worktree squash/ff merges:

1. **Before** `runGitMerge`: read main's `issues.db` content into memory
2. **After** a successful merge: read the post-merge `issues.db` (now the branch's version)
3. Perform a line-based union of both versions — keep all lines from main, then append any lines from the branch version not already present
4. Write the unioned content back to `issues.db`
5. Continue with the existing MERGE/DONE event appending

This simulates the `merge=union` behavior that git skips for squash/ff.

### Union function
Add a `unionMergeFile` helper that:
- Reads lines from both byte slices
- Builds a set from the base (main) lines
- Appends branch-only lines
- Returns the combined content

### Scope
- Only applies when `!useWorktrees` AND strategy is squash or ff-only
- `--no-ff` is unaffected (merge=union works correctly there)

## Acceptance Criteria
- [ ] Squash merge with `worktrees: false` preserves events from both main and the feature branch
- [ ] FF merge with `worktrees: false` preserves events from both main and the feature branch
- [ ] `--no-ff` merge behavior is unchanged
- [ ] Worktree-enabled merge behavior is unchanged
- [ ] New test case covers the diverged `issues.db` scenario
