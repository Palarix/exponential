# Walkthrough: Fail clearly when the hub is a bare repository or has no `.xpo`

## What was built

xpo finds its data through the "hub": the primary checkout that holds `.xpo/`. `storage.HubRoot()` takes the parent directory of `git-common-dir` as the hub. That works for a normal checkout and its linked worktrees, because the common directory is `<checkout>/.git`.

For a worktree of a bare repository, the common directory is the bare repository itself, for example `mirror.git`. Its parent is not a checkout and has no `.xpo`. Before this change:

- `ReadEvents` treated the missing `issues.db` as "no events". Every read returned an empty board, with no error.
- `withEventsLock` could not create `.xpo/events.lock`. Every write failed with a lock error that did not name the cause.
- The CLI guard checked `.xpo` in the current directory. A worktree has a tracked `.xpo/config.yaml`, so the guard passed.

Now xpo stops with a clear error in both bad cases. Exponential Personal needs a primary checkout. Shared and hosted workspaces belong to Exponential Studio (milestone 1.8.0).

## How the pieces fit together

### `internal/storage/root.go`

- `HubRoot()` keeps its signature and its `sync.Once` cache. Inside the same `Once`, it runs `git --git-dir=<common dir> rev-parse --is-bare-repository`. When git answers `true`, it stores the common directory in `bareCommon`.
- `CheckHub()` returns:
  - `ErrBareRepository`, wrapped with the bare repository path, when `bareCommon` is set;
  - `ErrNoXpoDir`, wrapped with the hub path, when `XpoDir()` is not a directory;
  - `nil` otherwise.
- `ResetHubRoot()` also clears `bareCommon`, so tests can switch directories.

`rev-parse --is-bare-repository` inside the worktree answers `false`, because the worktree has a working tree. The command must run against the common directory with `--git-dir`. A failed git call counts as "not bare".

### Storage choke points

These functions call `CheckHub()` first:

| Function | File | Covers |
|---|---|---|
| `ReadEvents` | `read.go` | every board read, `list`, `show`, projections |
| `ValidateEvents` | `read.go` | doctor |
| `ReadArchivedEvents` | `read.go` | archived reads |
| `CheckEventOrder` | `event_order.go` | doctor |
| `withEventsLock` | `lock.go` | every event write |

An existing `.xpo` without `issues.db` still reads as "no events". `xpo init` creates `.xpo` before the first event, so a new project is not an error.

### CLI guard (`cmd/exponential/main.go`)

`PersistentPreRunE` calls `storage.CheckHub()` after the exception list. On `ErrBareRepository` it returns the error, so every command fails the same way on stderr, or as a JSON error with `--json`. `mcp` is exempt. The MCP server keeps running, and each tool call returns the storage error. A server that exits at startup hides the reason from the agent.

The guard reacts only to `ErrBareRepository`. For a missing `.xpo`, the existing "not an Exponential project yet" message stays.

## Key decisions

- **Refuse, not fall back to the worktree.** With the worktree as the hub, the issue log travels on the branch. Then `start` nests worktrees, `merge` merges into the feature branch, and `merge=union` puts concurrent edits to one issue in the wrong order. Each fix added a caveat, so shared workspaces go to Studio.
- **No hub override.** A flag or `XPO_HUB` that points outside the repository is the out-of-repository `.xpo` idea, which is deferred.
- **Check in storage, not only in the CLI.** The CLI, the MCP server and the board server all read through storage. One check covers all three.
- **`HubRoot()` still returns a path.** An error return touches 23 call sites. The choke points stop every read and write before a call site uses the path.
- **Studio note.** A Studio project gets its `.xpo` path from the server, and its clone of the origin can be bare. When xpo-5433a8 replaces the `storage` globals with a `Project` value, the bare check moves into the constructor that Personal uses. `ErrNoXpoDir` can stay with the project.

## Non-obvious details

- Without the CLI guard, `list` already failed through storage, but it printed the error on stdout, by its own convention. The guard makes the failure the same for every command.
- No existing test relied on a missing `.xpo` reading as empty. No test needed changes.

## Acceptance criteria

- [x] In a worktree of a bare repository, `CheckHub()` returns an error that matches `ErrBareRepository` and names the common directory. Evidence: `TestCheckHub_BareRepositoryWorktree` in `internal/storage/hubcheck_test.go`.
- [x] In that worktree, `ReadEvents` and `AppendEvent` return the same error. Evidence: `TestBareRepositoryWorktree_ReadsAndWritesFail` also covers `ReadArchivedEvents`, `ValidateEvents` and `CheckEventOrder`.
- [x] In that worktree, `xpo list` exits non-zero with the bare-repository message. `xpo list --json` reports it as a JSON error. Evidence: `TestBareRepoWorktree_ListFails` and `TestBareRepoWorktree_ListJSONFails` in `cmd/exponential/bare_repo_test.go`, run against the built binary.
- [x] When the hub has no `.xpo`, `ReadEvents` returns an error that matches `ErrNoXpoDir`, not an empty list. Evidence: `TestCheckHub_MissingXpoDir`.
- [x] When `.xpo` exists but `issues.db` does not, `ReadEvents` returns no events and no error. Evidence: `TestCheckHub_XpoDirWithoutIssuesDB`.
- [x] A normal checkout and a linked worktree of a normal repository behave as before. Evidence: `TestCheckHub_LinkedWorktreeOfNormalRepo`, the existing `HubRoot` tests, and the full Go suite.
- [x] `make test` passes. Evidence: lint, 1777 frontend tests and the Go suite passed, exit 0.
