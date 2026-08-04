# Walkthrough: xpo start with worktrees

## What changed

`xpo start <id>` now creates a git worktree by default instead of checking out a branch in the primary checkout. This keeps the hub (primary checkout) parked on `main` so concurrent sessions can't trample each other's work.

## The worktree flow in `start.go`

The signature gained a `worktreePath` return value:

```go
func (c *Client) StartWork(id string, force bool) (branchName, worktreePath string, msgs []string, err error)
```

When `c.Config.Worktrees` is `true` and we're in a git repo, `StartWork`:

1. Computes `wtPath = WorktreeDir(branchName)` which resolves to `.xpo/worktrees/<branch>` under the hub root
2. Acquires `WithGitLock` (same lock as before — prevents concurrent git mutations)
3. If `force` and a worktree already exists for the branch: `WorktreeRemove` removes it first
4. Creates the worktree: `git worktree add -b <branch> <path> <base>` (new branch) or `git worktree add <path> <branch>` (existing branch)
5. Ensures `.xpo/worktrees/` is in `.gitignore` via `EnsureGitignoreEntry`
6. Runs `worktree_setup` hook if configured (e.g. `make deps`)
7. Returns the absolute path as `worktreePath`

When `c.Config.Worktrees` is `false`, the existing checkout-based flow runs unchanged.

## Git worktree helpers in `git.go`

Six new functions:

- `WorktreeDir(branch)` — canonical path: `storage.XpoDir() + "/worktrees/" + branch`
- `WorktreeAdd(path, branch, base)` — `git worktree add -b <branch> <path> <base>`
- `WorktreeAddExisting(path, branch)` — `git worktree add <path> <branch>` for existing branches
- `WorktreeRemove(path)` — `git worktree remove --force <path>` + `git worktree prune`
- `WorktreeList()` — parses `git worktree list --porcelain` into `[]WorktreeEntry{Path, Branch}`
- `FindWorktreeForBranch(branch)` — linear scan of `WorktreeList()` results

## Config changes

Two new fields in `Config`:

```yaml
worktrees: true              # default; set false to disable
worktree_setup: "make deps"  # optional shell command run in new worktrees
```

The default `true` is set via `v.SetDefault("worktrees", true)` in `LoadConfig()`. The CLI `--no-wt` flag overrides it by setting `cfg.Worktrees = false` before creating the client.

## Callers updated

Four callers of `StartWork` needed the new signature:

- `cmd/exponential/update.go` — CLI handler; gained `--no-wt` flag
- `internal/mcpserver/tools.go` — MCP handler; passes through `wtPath` in `startOut.WorktreePath`
- `internal/exponential/drive.go` — automated drive; ignores `worktreePath` (uses `_`)
- `internal/server/handlers.go` — web API handler; ignores `worktreePath` (uses `_`)

## Test changes

Existing tests: `Worktrees: false` in test config preserves their checkout-based assertions.

New test `TestStartWork_Worktree_CreatesWorktree`: sets `Worktrees: true`, creates a git repo, calls `StartWork`, and verifies:
- Worktree directory exists on disk
- Branch name is correct
- Primary checkout stays on `main` (not switched)
- `FindWorktreeForBranch` finds the worktree at the expected path

All tests call `storage.ResetHubRoot()` in setup and cleanup because `HubRoot()` caches per-process and tests change directories.
