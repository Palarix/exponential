# Spec: `xpo merge` auto-checkout main in branch mode

## What

`xpo merge` fails when the global config has `worktrees: true` but the issue was
started with `mode: "branch"`. The merge command reads the global config to decide
worktree vs branch mode, ignoring how the issue was actually started.

## Why

When `xpo start` is called with `mode: "branch"`, no worktree is created — the user
works directly on a branch in the main checkout. But `xpo merge` checks
`c.Config.Worktrees` (global config) and takes the worktree path, which expects the
hub to already be on main. In branch mode the checkout IS on the feature branch, so
the merge rejects with a confusing error.

## How

In `internal/exponential/merge.go`, after computing `useWorktrees` from the config
(line 57), add a runtime check: if `useWorktrees` is true but
`FindWorktreeForBranch(branch)` returns false, set `useWorktrees = false`. This makes
the merge fall through to the branch mode path which calls `CheckoutBranch(base)`
automatically.

```go
useWorktrees := c.Config.Worktrees && CheckGitRepo()
if useWorktrees {
    if _, ok := FindWorktreeForBranch(branch); !ok {
        useWorktrees = false
    }
}
```

No other changes needed — the branch mode path already handles everything correctly.

## Acceptance Criteria

- [ ] `xpo merge` succeeds from a feature branch when the issue was started with
      `mode: "branch"`, even when global config has `worktrees: true`
- [ ] Worktree mode merges are unaffected
- [ ] `make test` passes
