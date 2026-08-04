# xpo start with worktrees

## Approach

Modify `StartWork()` to create a git worktree instead of checking out in the primary checkout. The worktree flag is driven by `Config.Worktrees` (default `true`), overridable per-invocation via `--no-wt` on the CLI.

## Implementation Plan

### 1. Config fields (`internal/config/config.go`)

Add to `Config` struct:
```go
Worktrees     bool   `mapstructure:"worktrees" yaml:"worktrees"`
WorktreeSetup string `mapstructure:"worktree_setup" yaml:"worktree_setup"`
```

Default in `LoadConfig()`: `v.SetDefault("worktrees", true)`.

### 2. Git worktree helpers (`internal/exponential/git.go`)

```go
func WorktreeAdd(path, branch, base string) error
func WorktreeAddExisting(path, branch string) error
func WorktreeRemove(path string) error
func WorktreeList() ([]WorktreeEntry, error)
func FindWorktreeForBranch(branch string) (string, bool)
func WorktreeDir(branch string) string  // returns .xpo/worktrees/<branch>
```

`WorktreeList` parses `git worktree list --porcelain` which outputs blocks like:
```
worktree /path/to/main
HEAD abc1234
branch refs/heads/main

worktree /path/to/feature
HEAD def5678
branch refs/heads/feature-branch
```

### 3. StartWork changes (`internal/exponential/start.go`)

New signature:
```go
func (c *Client) StartWork(id string, force bool) (branchName, worktreePath string, msgs []string, err error)
```

Added `worktreePath` return value. When `c.Config.Worktrees` is true and in a git repo:
1. Compute `wtPath = WorktreeDir(branchName)` 
2. If `force` and worktree exists for branch: `WorktreeRemove` first
3. If branch exists locally: `WorktreeAddExisting(wtPath, branchName)`
4. Else: `WorktreeAdd(wtPath, branchName, base)`
5. Ensure `.xpo/worktrees/` is in `.gitignore`
6. Run `worktree_setup` hook if configured
7. Return `worktreePath` as the absolute path

When `c.Config.Worktrees` is false: existing checkout-based behavior, `worktreePath` empty.

### 4. Gitignore helper (`internal/exponential/setup.go`)

Add `".xpo/worktrees/"` to the `ignoreEntries` list in `InitProject()`. Also add a standalone `EnsureGitignoreEntry(entry)` function that `StartWork` calls on first worktree creation.

### 5. CLI flag (`cmd/exponential/update.go`)

Add `--no-wt` flag to `startCmd`. In the handler, set `cfg.Worktrees = false` before calling `StartWork`. Print the worktree path after success messages.

### 6. MCP tool (`internal/mcpserver/tools.go`)

Add `WorktreePath` to `startOut`. Update handler to pass through the returned path.

### 7. Worktree setup hook

After worktree creation, if `c.Config.WorktreeSetup != ""`:
```go
cmd := exec.Command("sh", "-c", c.Config.WorktreeSetup)
cmd.Dir = absWorktreePath
cmd.Run()
```

## Files changed

| File | Change |
|---|---|
| `internal/config/config.go` | Add `Worktrees`, `WorktreeSetup` fields + default |
| `internal/exponential/git.go` | Add worktree helper functions |
| `internal/exponential/start.go` | Worktree creation logic, setup hook |
| `internal/exponential/setup.go` | Add `.xpo/worktrees/` to gitignore entries, `EnsureGitignoreEntry` |
| `cmd/exponential/update.go` | Wire `--no-wt` flag, print worktree path |
| `internal/mcpserver/tools.go` | Add `worktree_path` to `startOut`, update handler |
