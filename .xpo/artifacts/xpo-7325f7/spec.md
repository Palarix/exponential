# Spec: Make xpo start idempotent when worktree already exists

## What

When `xpo start <ID>` is called and the caller is already inside the worktree for that issue, return success without recreating the worktree or re-running hooks.

## Why

External orchestration tools (Elysium) create the worktree via `xpo start`, then launch a coding agent inside it. The agent later calls `xpo start` as part of the xpo workflow skill. Currently this errors with "already in progress." The agent needs a no-op success path.

## How

### 1. Early idempotent check in `StartWork()`

After fetching the issue and computing `candidateBranch`, before the existing status switch:

**Worktree mode** (`c.Config.Worktrees && CheckGitRepo()`):
- Call `FindWorktreeForBranch(candidateBranch)` to get the worktree path.
- If found, check whether cwd is inside that worktree path (using `filepath.EvalSymlinks` + prefix comparison with separator boundary).
- If cwd is inside the worktree:
  - Terminal status (DONE, CANCELED, DUPLICATE) → error as usual.
  - Not DOING → transition to DOING via `UpdateIssue`.
  - Return `(candidateBranch, wtPath, msgs, nil)` — no worktree creation, no setup hook.

**Branch mode** (`CheckGitRepo() && !useWorktrees`):
- If the candidate branch is the currently checked out branch:
  - Same status handling as above.
  - Return `(candidateBranch, "", msgs, nil)`.

### 2. Existing paths unchanged

- `force=true` retains destructive takeover semantics.
- The status switch and worktree/branch creation logic remain unchanged for non-idempotent cases.

### 3. Move `candidateBranch` computation

`candidateBranch` must be computed before the status switch (currently after it) so the idempotent check can use it.

## Acceptance Criteria

- [ ] Calling `StartWork` from inside the issue's worktree returns success with the existing worktree path.
- [ ] Status transitions to DOING if it was PLANNED and cwd is inside the worktree.
- [ ] Terminal status still errors even from inside the worktree.
- [ ] No worktree setup hook runs on the idempotent path.
- [ ] `force=true` still does destructive takeover.
- [ ] Branch mode: calling `StartWork` when the candidate branch is already checked out returns success.
- [ ] All existing tests pass unchanged.
- [ ] New tests cover the idempotent worktree path and the idempotent branch path.