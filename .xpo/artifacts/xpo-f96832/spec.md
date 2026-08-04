# xpo merge with worktree cleanup

## Approach

When `Config.Worktrees` is true, `MergeIssue` skips the `CheckoutBranch(base)` step (hub is already on main), and after a successful merge removes the worktree. The CLI and MCP pre-flight clean check is relaxed to ignore `.xpo/` changes since events accumulate uncommitted under the per-merge cadence.

## Implementation

### 1. New helpers (`internal/exponential/merge.go`)

- `IsWorkingTreeCleanIgnoringXpo()` — `git status --porcelain`, ignores lines where path starts with `.xpo/`
- `HubBranch()` — `git -C <hub> rev-parse --abbrev-ref HEAD` to check hub's current branch from any context

### 2. `MergeIssue()` changes

When `c.Config.Worktrees` is true:
1. Verify hub is on `base` via `HubBranch()`. Error if not.
2. Skip `CheckoutBranch(base)` — already there.
3. Run merge as before (merge, commit, events).
4. After success: `FindWorktreeForBranch(branch)` → `WorktreeRemove(path)`.
5. Always remove worktree, regardless of `DeleteBranch`/`KeepBranch` (worktree is ephemeral; keeping branch ≠ keeping worktree).

When `c.Config.Worktrees` is false: existing checkout-based flow.

### 3. CLI `runMerge()` (`cmd/exponential/merge.go`)

- Add `--no-wt` flag; sets `cfg.Worktrees = false`.
- Clean check: use `IsWorkingTreeCleanIgnoringXpo()` when worktrees enabled, `IsWorkingTreeClean()` otherwise.

### 4. MCP handler (`internal/mcpserver/tools.go`)

- Same clean check logic as CLI.

## Files changed

| File | Change |
|---|---|
| `internal/exponential/merge.go` | Add `IsWorkingTreeCleanIgnoringXpo`, `HubBranch`; worktree path in `MergeIssue` |
| `cmd/exponential/merge.go` | Wire `--no-wt`, adjust clean check |
| `internal/mcpserver/tools.go` | Adjust clean check |
