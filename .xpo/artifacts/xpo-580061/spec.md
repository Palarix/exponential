# Hub-Rooted Storage Layer

## Approach

Introduce a `HubRoot()` function that discovers the primary checkout (the "hub") from any git context — worktree or primary. All `.xpo/` path construction routes through this, so storage reads/writes always target the hub's copy of `issues.db` and `artifacts/`.

### Discovery mechanism

```
git rev-parse --path-format=absolute --git-common-dir
```

- In a worktree: returns the primary checkout's `.git` directory (e.g. `/repo/.git`)
- In a primary checkout: returns `.git` (relative) or the absolute `.git` path
- Parent of the result is the hub root

Cache the result per-process (it never changes during a run).

### Architecture

**Storage package** (`internal/storage/`): Add `SetRoot(dir)` / `root()`. Called once at startup by CLI main and MCP server init. All functions use `root()` instead of hardcoded `".xpo"`. Default fallback: `".xpo"` (backward compatible for tests and non-git contexts).

**Exponential package** (`internal/exponential/`): Add `HubRoot()` and `XpoDir()` to `git.go`. Use `XpoDir()` in `artifactDir()`, `WithGitLock()`, `GitCommit()`, and the auto-commit `git add` calls in `add.go`, `comment.go`, `update.go`, `delete.go`.

**Server package** (`internal/server/`): `WatchDB()` and `statDB()` resolve path via storage root.

**Collapse** (`internal/storage/collapse.go`): `readCommittedBytes()` uses `git -C <hub> show HEAD:.xpo/issues.db` so it reads the hub's committed state, not the worktree's.

## Callsites to update (production code only)

| File | Line(s) | Current | Change |
|---|---|---|---|
| `storage/storage.go` | 12 | `filepath.Join(".xpo", "issues.db")` | `filepath.Join(root(), "issues.db")` |
| `storage/read.go` | 13, 36 | `filepath.Join(".xpo", "issues.db")` / `"archive.db"` | via `root()` |
| `storage/archive.go` | 16-17 | `filepath.Join(".xpo", ...)` | via `root()` |
| `storage/collapse.go` | 15, 34 | `"HEAD:.xpo/issues.db"`, `filepath.Join(".xpo", ...)` | `git -C <hub>`, via `root()` |
| `exponential/client.go` | 16-17 | `"git", "add", ".xpo/issues.db"` | via `XpoDir()` |
| `exponential/artifact.go` | 36 | `filepath.Join(".xpo", "artifacts", ...)` | via `XpoDir()` |
| `exponential/gitlock.go` | 22 | `filepath.Join(".xpo", "git.lock")` | via `XpoDir()` |
| `exponential/add.go` | 112 | `"git", "add", ".xpo/issues.db"` | via `XpoDir()` |
| `exponential/comment.go` | 40 | same | via `XpoDir()` |
| `exponential/update.go` | 184 | same | via `XpoDir()` |
| `exponential/delete.go` | 78 | same | via `XpoDir()` |
| `exponential/merge.go` | 135 | same | via `XpoDir()` |
| `server/server.go` | 190 | `filepath.Join(".xpo", "issues.db")` | via storage root |
| `server/watch.go` | 19 | `filepath.Join(".xpo", "issues.db")` | via storage root |
| `mcpserver/tools.go` | 544, 590, 648, 661, 672 | `fmt.Sprintf(".xpo/artifacts/...")` | via `XpoDir()` |

## Out of scope for this story

- Config path resolution (`.xpo/config.yaml`) — config is loaded once at startup from cwd, which will still be the hub in the worktree model
- `init` / `setup.go` — always runs in the hub by definition
- `demo.go` — operates on a target directory, not the current repo
- Test files — tests create their own temp `.xpo/` dirs; no routing needed

## Acceptance Criteria

- [ ] `HubRoot()` returns the primary checkout path from any worktree
- [ ] `HubRoot()` returns cwd when not in a worktree (backward compatible)
- [ ] `storage.ReadEvents()` reads from `<hub>/.xpo/issues.db`
- [ ] `appendEvent()` writes to `<hub>/.xpo/issues.db`
- [ ] All git staging commands (`git add`) use hub-rooted paths
- [ ] Artifact read/write uses hub-rooted paths
- [ ] Git lock uses hub-rooted path
- [ ] Server file watcher uses hub-rooted path
- [ ] Existing tests pass without modification (fallback to ".xpo")
