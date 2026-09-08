# Walkthrough: `xpo merge` auto-checkout main in branch mode

## Context

`xpo start` supports two modes: **worktree** (default when `worktrees: true` in config)
creates an isolated git worktree, and **branch** (via `--mode branch` or MCP `mode:
"branch"`) creates a regular branch in the main checkout. The `xpo merge` command
decides which mode to use based solely on the global config — it never checked how the
issue was actually started.

When an issue was started in branch mode but the global config had `worktrees: true`,
merge took the worktree path, called `HubBranch()`, found the checkout on the feature
branch (not main), and errored with "park the hub on the default branch before merging."

## What changed

**`internal/exponential/merge.go`** — added a runtime fallback after the config-based
`useWorktrees` decision (line 57):

```go
useWorktrees := c.Config.Worktrees && CheckGitRepo()
if useWorktrees {
    if _, ok := FindWorktreeForBranch(branch); !ok {
        useWorktrees = false
    }
}
```

If the config says worktrees but no worktree exists for this branch, `useWorktrees`
drops to false. The existing branch mode path then calls `CheckoutBranch(base)` to
switch to main before merging — no manual checkout needed.

## How the pieces fit

- **Worktree mode** (worktree exists): unchanged — `HubBranch()` check still runs,
  merge executes from the hub, worktree is cleaned up afterward.
- **Branch mode** (no worktree): `FindWorktreeForBranch` returns false, falls through
  to `CheckoutBranch(base)`, merge runs in the main checkout. The `issues.db` union
  merge and squash/ff commit paths all work correctly in this mode (already did before).
- **`FindWorktreeForBranch`** runs `git worktree list` — lightweight, already used
  later in the same function for cleanup (line 169). No new dependencies.

## Key decisions

- **Runtime detection over stored metadata**: instead of recording the mode at `start`
  time and reading it back at merge time, we detect it by checking for an actual
  worktree. This is simpler, requires no schema changes, and handles edge cases
  (worktree manually removed, config changed between start and merge).
