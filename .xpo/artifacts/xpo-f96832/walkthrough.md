# Walkthrough: xpo merge with worktree cleanup

## What changed

`xpo merge` now supports the hub-and-spoke worktree model. When `Config.Worktrees` is true, it skips the `CheckoutBranch(base)` step (the hub is already on main), and after a successful merge it removes the worktree for the merged branch.

## The worktree path in `MergeIssue` (merge.go)

The key change is a conditional at the top of the `WithGitLock` block:

```go
if useWorktrees {
    current := HubBranch()
    if current != base {
        return fmt.Errorf("hub checkout is on %s, not %s — ...")
    }
} else {
    if err := CheckoutBranch(base); err != nil { ... }
}
```

Under the classic flow, `MergeIssue` checks out main before merging. Under the worktree flow, it verifies the hub IS on main (it should be by convention) and skips the checkout entirely. This is the key safety improvement — no branch switching means no risk of trampling uncommitted work.

After a successful merge and commit, the worktree path cleans up:

```go
if useWorktrees {
    if wtPath, ok := FindWorktreeForBranch(branch); ok {
        if err := WorktreeRemove(wtPath); err != nil {
            result.Messages = append(result.Messages, "Warning: ...")
        } else {
            result.Messages = append(result.Messages, "Removed worktree ...")
        }
    }
}
```

Worktree cleanup is non-fatal. If removal fails (e.g. the directory is in use), the merge still succeeds — it's a warning, not a rollback. The cleanup always runs regardless of `--keep-branch` / `--delete-branch` flags, because worktrees are ephemeral working directories, not the branches themselves.

## New helpers

**`IsWorkingTreeCleanIgnoringXpo()`** — Like `IsWorkingTreeClean()` but skips files under `.xpo/`. Under the hub model with per-merge event cadence, `.xpo/issues.db` has uncommitted events between merges — that's by design, not dirt. The function parses `git status --porcelain` output and ignores any line whose path starts with `.xpo/`.

**`HubBranch()`** — Runs `git -C <hub> rev-parse --abbrev-ref HEAD` to get the hub's current branch from any context (hub or worktree). Used to verify the hub is on the default branch before merging.

## CLI changes (cmd/exponential/merge.go)

Added `--no-wt` flag (sets `cfg.Worktrees = false`). The pre-flight clean check now branches:

- Worktrees enabled: `IsWorkingTreeCleanIgnoringXpo()` — allows `.xpo/issues.db` changes
- Worktrees disabled: `IsWorkingTreeClean()` — original strict check

## MCP handler changes (mcpserver/tools.go)

Same branching logic for the clean check. The `clientFor(req)` call was moved earlier in the function so `c.Config.Worktrees` is available before the strategy parsing.

## Merge failure behavior

On merge failure (e.g. conflicts), the worktree is preserved. The user can resolve conflicts manually in the hub's working tree, then re-run the merge. This matches the original behavior where a failed merge leaves the working tree in a conflicted state for manual resolution.
