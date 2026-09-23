# Walkthrough: Make xpo start idempotent when worktree already exists

## What changed

Two files changed: `internal/exponential/start.go` and `internal/exponential/start_test.go`.

`StartWork()` now has two early-return idempotent paths that fire before the existing status switch, so agents already inside a worktree (or on the correct branch) can call `xpo start` without error.

## How it works

### Worktree mode idempotent path

Before any status gate, `StartWork` computes the candidate branch and calls `FindWorktreeForBranch`. If a worktree exists for that branch and the process's cwd is inside it (checked via `cwdInsidePath`, which resolves symlinks and does a separator-safe prefix comparison), the function:

1. Errors if the issue is in a terminal state (DONE, CANCELED, DUPLICATE).
2. Transitions to DOING if not already there.
3. Returns the existing worktree path and branch — no worktree creation, no setup hook.

The `c.Config.Worktrees` flag gates the check instead of `CheckGitRepo()`, because `CheckGitRepo()` only looks for `.git` in cwd and fails from subdirectories inside a worktree. `FindWorktreeForBranch` uses `git worktree list` which works from any depth.

### Branch mode idempotent path

If worktrees are disabled and `CurrentBranch()` matches the candidate branch, the same status handling applies and the function returns early with a "Resuming on current branch" message.

### What doesn't change

`force=true` bypasses both idempotent paths and retains its destructive takeover semantics. The existing status switch, worktree creation, branch creation, and setup hook logic are untouched.

## Key decisions

- **Exact branch match via `FindWorktreeForBranch`:** The idempotent check matches on the full computed branch name (`<issueID>-<slugified-title>`), not a prefix match on the issue ID. This is correct for the target scenario where the same `xpo start` call created the worktree, so the branch name is deterministic.
- **`cwdInsidePath` with symlink resolution:** macOS temp dirs go through `/private/var` symlinks; `filepath.EvalSymlinks` on both paths ensures the prefix comparison works in tests and on real systems.

## Acceptance criteria

- [x] Calling `StartWork` from inside the issue's worktree returns success with the existing worktree path — `TestStartWork_Worktree_Idempotent_AlreadyDoing`
- [x] Status transitions to DOING if it was PLANNED and cwd is inside the worktree — `TestStartWork_Worktree_Idempotent_PlannedTransitions`
- [x] Terminal status still errors even from inside the worktree — `TestStartWork_Worktree_Idempotent_TerminalErrors`
- [x] No worktree setup hook runs on the idempotent path — verified in `AlreadyDoing` test via marker file removal
- [x] `force=true` still does destructive takeover — `TestStartWork_Worktree_Idempotent_ForceStillDestroys`
- [x] Branch mode: calling `StartWork` when the candidate branch is already checked out returns success — `TestStartWork_Branch_Idempotent_AlreadyCheckedOut`
- [x] Works from subdirectories inside the worktree — `TestStartWork_Worktree_Idempotent_Subdirectory`
- [x] All existing tests pass unchanged