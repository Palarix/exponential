# Worktree-Based Git Operations

## Problem

`xpo start` and `xpo merge` mutate the primary checkout's working tree via `git checkout`. When multiple sessions (agents or humans) work concurrently on the same repo, one session's `start` switches the checkout out from under another session's uncommitted work. The `WithGitLock` serializes operations but doesn't prevent the fundamental hazard -- it ensures one git command at a time, not that the checkout is safe to switch.

**Motivating incident:** Two Claude sessions ran in parallel on the same repo. The issues.db event stream handled concurrency correctly, but `xpo start` would have stomped the first session's uncommitted work if the primary checkout hadn't happened to be idle on main. Timing luck, not design.

## Design: Hub-and-Spoke Worktrees

### Hub (primary checkout)

The primary checkout stays parked on `main` permanently. It is the integration point:

- **Merge target**: `xpo merge` always runs here (already on main, no checkout needed)
- **Event authority**: All `issues.db` mutations route to the hub's copy via hub-rooted storage. The board watches the hub's file and reflects real-time state.
- **Clean by convention**: No feature work happens here; only merges and issue events.

### Spokes (worktrees)

Each `xpo start` creates a git worktree for the story branch. Code changes happen exclusively in worktrees.

- **Placement**: `.xpo/worktrees/<branch-name>/` (inside the repo, gitignored). Follows the Claude Code convention (`.claude/worktrees/`).
- **Lifecycle**: Created by `start`, removed by `merge` (or manually via `git worktree remove`).
- **Build state**: A configurable `worktree_setup` hook runs after worktree creation to restore build artifacts (dependency install, asset import, compilation).

## Detailed Command Changes

### `xpo start <id>` (default: worktree)

1. Validate status, transition to DOING (unchanged)
2. Construct branch name: `<id>-<slugified-title>` (unchanged)
3. **New**: `git worktree add .xpo/worktrees/<branch> -b <branch> <base>`
   - If branch already exists locally: `git worktree add .xpo/worktrees/<branch> <branch>`
   - Add `.xpo/worktrees/` to `.gitignore` if not present
4. **New**: Run `worktree_setup` hook (if configured) in the new worktree directory
5. **New**: Return `worktree_path` in CLI output and MCP `startOut` response

**`--no-wt` flag**: Falls back to current `git checkout -b` behavior in the primary checkout.

**`--force` + worktree constraint**: Git prohibits a branch being checked out in two worktrees. `start --force` removes the existing worktree for that branch (`git worktree remove --force`), then creates a fresh one. This matches "force takeover" semantics -- clean slate for the new owner.

### `xpo merge <id>` (default: worktree-aware)

1. Resolve the issue's branch and locate its worktree (via `git worktree list`)
2. Merge runs in the hub -- already on main, no checkout switch needed
3. On success:
   - Commit accumulated `issues.db` events in the merge commit (per-merge cadence, unchanged from today)
   - `git worktree remove <path>` + `git worktree prune`
   - Delete branch (local + remote) as before
4. On failure: `git merge --abort` (unchanged), worktree preserved for manual conflict resolution

**`--no-wt` flag**: Falls back to current behavior (checkout main, merge, checkout back).

### Hub-Rooted Storage

The storage layer gains a "hub root" concept:

- **Discovery**: From any worktree, run `git rev-parse --path-format=absolute --git-common-dir`. The parent of the returned path is the hub root.
- **Read/write routing**: `storage.ReadEvents()` and `appendEvent()` use `<hub-root>/.xpo/issues.db` instead of relative `.xpo/issues.db`.
- **Effect**: MCP servers running in worktrees still read/write the hub's event log. The board (watching the hub) reflects all changes in real time. Feature branches carry only code.

### Event Commit Cadence

Per-merge (unchanged from today). Events accumulate uncommitted on the hub's working tree and get committed as part of the merge commit. This keeps main's history clean -- each merge commit bundles code changes + the issue lifecycle events that led to them.

### Configuration

New fields in `.xpo/config.yaml`:

```yaml
worktrees: true              # default true; set false to disable globally
worktree_setup: "make deps"  # shell command run in new worktrees (optional)
```

The `--no-wt` CLI flag overrides `worktrees: true` per-invocation. When `worktrees: false`, `start` and `merge` behave as they do today.

## MCP Tool Changes

### `start` tool

Add `worktree_path` to `startOut`:

```go
type startOut struct {
    ID           string   `json:"id"`
    Branch       string   `json:"branch"`
    WorktreePath string   `json:"worktree_path,omitempty"`
    Messages     []string `json:"messages"`
}
```

When worktrees are enabled, `worktree_path` is the absolute path to the created worktree. The calling agent uses this to set its working directory.

### `merge` tool

No schema change needed. Worktree cleanup is an internal implementation detail.

## File-Level Changes

| File | Change |
|---|---|
| `internal/exponential/git.go` | Add `WorktreeAdd()`, `WorktreeRemove()`, `WorktreeList()`, `FindWorktreeForBranch()`, `HubRoot()` |
| `internal/exponential/start.go` | Default to `WorktreeAdd`; `--no-wt` path preserves current behavior |
| `internal/exponential/merge.go` | Assert hub is on main; after merge: `WorktreeRemove` + prune |
| `internal/exponential/setup.go` | Add `.xpo/worktrees/` to `.gitignore` entries |
| `internal/storage/storage.go` | Route reads/writes through `HubRoot()` instead of relative cwd |
| `internal/config/config.go` | Add `Worktrees bool`, `WorktreeSetup string` fields |
| `internal/mcpserver/tools.go` | Add `worktree_path` to `startOut`; populate from `StartWork` return |
| `cmd/exponential/update.go` | Wire `--no-wt` flag on `startCmd` |
| `cmd/exponential/merge.go` | Wire `--no-wt` flag on `mergeCmd` |

## Acceptance Criteria

- [ ] `xpo start <id>` creates a worktree at `.xpo/worktrees/<branch>/` and returns the path
- [ ] `xpo merge <id>` runs the merge on the hub (main), removes the worktree, prunes
- [ ] MCP `start` tool returns `worktree_path` in its response
- [ ] All `issues.db` reads/writes route through the hub root, even from worktree MCP servers
- [ ] The board reflects real-time issue state when agents work in worktrees
- [ ] `--no-wt` flag on both commands preserves current checkout-based behavior
- [ ] `worktrees: false` in config disables worktrees globally
- [ ] `worktree_setup` hook runs in new worktrees when configured
- [ ] `start --force` removes existing worktree before creating a fresh one
- [ ] `.xpo/worktrees/` is added to `.gitignore` by `init` and on first worktree creation
- [ ] Two concurrent `xpo start` calls for different issues succeed without interfering
- [ ] Merge cleans up worktree even when `--delete-branch` is not set

## Dependencies

- **xpo-c762f7** (gitattributes `merge=union`): Still needed for `--no-wt` fallback and repos not using worktrees. Should be done first or in parallel.

## Out of Scope

- Checkout-less merges via `git merge-tree` / `git commit-tree` (considered; the hub model is simpler and keeps checked-out copies in sync)
- Remote worktrees or worktrees on different machines
- Automatic push of worktree branches to remote (agents do this themselves when ready)
