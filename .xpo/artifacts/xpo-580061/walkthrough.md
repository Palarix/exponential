# Walkthrough: Hub-Rooted Storage Layer

## What problem does this solve?

When xpo gains worktree support (parent epic xpo-f7ea12), agents will work in git worktrees while the primary checkout ("hub") stays on `main`. But `issues.db` and artifacts need to live on the hub so the board sees real-time state and events don't diverge across branches. Every path that reads or writes `.xpo/issues.db` or `.xpo/artifacts/` needed to resolve through the hub, not relative to the current working directory.

## The core addition: `storage/root.go`

This new file contains three exported functions:

```go
func HubRoot() string   // absolute path to the primary checkout
func XpoDir() string    // HubRoot() + "/.xpo"
func ResetHubRoot()     // clears cache (for tests only)
```

`HubRoot()` runs `git rev-parse --path-format=absolute --git-common-dir` once (cached via `sync.Once`). In a normal checkout, `--git-common-dir` returns the `.git` directory and `filepath.Dir()` gives the repo root. In a worktree, `--git-common-dir` returns the *primary* checkout's `.git` directory — so `filepath.Dir()` still gives the hub root. If the command fails (non-git context, old git version), it falls back to `"."` — current directory, preserving backward compatibility.

## What changed in the storage package

Four files had hardcoded `filepath.Join(".xpo", "issues.db")` or similar:

- **`storage.go`** (`AppendEvent`) — writes events
- **`read.go`** (`ReadEvents`, `ReadArchivedEvents`) — reads events
- **`archive.go`** (`ArchiveEvents`) — moves events to archive
- **`collapse.go`** (`AppendEventCollapsed`, `readCommittedBytes`) — reads/rewrites with collapse optimization

All now use `XpoDir()` instead of `".xpo"`. The `readCommittedBytes` function additionally uses `git -C <hub>` so the `git show HEAD:.xpo/issues.db` command targets the hub's HEAD, not the worktree's.

## What changed in the exponential package

Five files had `git add .xpo/issues.db` for auto-commit:
- `add.go`, `comment.go`, `update.go`, `delete.go` — auto-commit after mutations (when `AutoCommit` config is true)
- `client.go` (`GitCommit`) — explicit commit helper used by merge and drive

All now use `git -C <hub> add .xpo/issues.db` and `git -C <hub> commit -m ...`. The `-C` flag changes git's working directory to the hub, so the staging and commit happen in the hub's index and tree. In the current (non-worktree) model, the hub IS the cwd, so this is a no-op behavioral change. In the future worktree model, it ensures commits land on main.

Two other files changed:
- **`artifact.go`** — `artifactDir()` now returns `filepath.Join(storage.XpoDir(), "artifacts", issueID)` instead of `filepath.Join(".xpo", "artifacts", issueID)`. This routes all artifact file I/O through the hub.
- **`gitlock.go`** — `WithGitLock()` acquires the lock at `filepath.Join(storage.XpoDir(), "git.lock")`. This ensures all xpo processes (hub or worktree) compete for the same lock file.

## What changed in the server package

- **`server.go`** (`statDB`) — checks `filepath.Join(storage.XpoDir(), "issues.db")` for modification time
- **`watch.go`** (`WatchDB`) — watches the same hub-rooted path for fsnotify events

Both now resolve through `storage.XpoDir()`. Since the board server runs from the hub, this is currently equivalent, but it's correct by construction rather than by coincidence.

## What changed in the MCP server

**`mcpserver/tools.go`** — the `spec`, `walkthrough`, and `artifact` handlers construct a `path` field in their responses to tell callers where the file lives. These now use `filepath.Join(storage.XpoDir(), "artifacts", issueID, filename)` instead of `fmt.Sprintf(".xpo/artifacts/%s/%s", ...)`. This produces absolute paths that are correct even when the MCP server runs from a worktree.

## Why not dependency injection / SetRoot()?

The codebase uses package-level functions (`storage.ReadEvents()`, `storage.AppendEvent()`) rather than method receivers on a storage struct. Adding a `SetRoot()` init call would require threading it through every entry point (CLI commands, MCP server, test setup). The `sync.Once` lazy discovery is simpler: the first call to `HubRoot()` discovers the hub, all subsequent calls return the cached value. Tests work without setup because non-git contexts fall back to `"."`.

## What to watch for

- **`--path-format=absolute`** requires git 2.31+ (released March 2021). If targeting older git, the fallback to `"."` handles it gracefully but silently — worktree routing won't work on very old git. This is acceptable since worktrees themselves require a modern git.
- **`ResetHubRoot()`** exists for tests that need to change directories between test cases. It resets the `sync.Once` so the next call re-discovers. Don't use it in production code.
