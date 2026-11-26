# Spec: Fail clearly when the hub is a bare repository or has no `.xpo`

## What

Make xpo stop with a clear error in two cases where it now reads an empty board:

1. The hub has no `.xpo` directory.
2. The git common directory is a bare repository, so there is no primary checkout to use as the hub.

## Why

`storage.HubRoot()` (xpo-580061) returns `filepath.Dir(git-common-dir)`. For a worktree of a bare repository, the common directory is the bare repository (for example `mirror.git`). Its parent directory is not a checkout and has no `.xpo`. Then:

- `ReadEvents` treats the missing `issues.db` as "no events". `list` returns an empty board and no error.
- `withEventsLock` cannot create `.xpo/events.lock`, so every write fails with a lock error that does not name the cause.

The CLI guard in `cmd/exponential/main.go` checks `.xpo` in the current directory, not on the hub. A worktree has a tracked `.xpo/config.yaml`, so the guard passes.

Exponential Personal needs a primary checkout as the hub. Support for shared or hosted workspaces is out of scope. This issue only replaces the silent failure with an error.

## How

### Storage (`internal/storage/root.go`)

- Add two sentinel errors:
  - `ErrBareRepository`: "xpo needs a primary checkout: this is a worktree of a bare repository (<common dir>), which xpo does not support".
  - `ErrNoXpoDir`: "no .xpo directory at <hub>: run xpo from an Exponential project, or run 'xpo init'".
  The returned errors wrap the sentinels and include the path, so `errors.Is` works.
- `HubRoot()` keeps its signature. Inside the same `sync.Once`, it runs `git --git-dir=<common dir> rev-parse --is-bare-repository`. When the result is `true`, it records the bare state. `HubRoot()` still returns `filepath.Dir(common dir)`.
- Add `CheckHub() error`:
  - returns the `ErrBareRepository` error when the common directory is bare,
  - returns the `ErrNoXpoDir` error when `XpoDir()` does not exist,
  - returns `nil` otherwise.
- `ResetHubRoot()` also clears the bare state.

### Storage choke points

Call `CheckHub()` first in:

- `ReadEvents`, `ValidateEvents`, `ReadArchivedEvents` (`read.go`),
- `CheckEventOrder` (`event_order.go`),
- `withEventsLock` (`lock.go`). Every event write goes through it.

A missing `issues.db` inside an existing `.xpo` stays "no events". A fresh `xpo init` creates `.xpo` before it creates `issues.db`.

### CLI guard (`cmd/exponential/main.go`)

- In `PersistentPreRunE`, call `storage.CheckHub()` before the existing `.xpo` check, for every command except the exception list.
- On `ErrBareRepository`, return the error. With `--json`, the existing JSON error path reports it.
- Keep the existing "not an Exponential project yet" message for a missing `.xpo` in the current directory.
- Exempt `mcp` from the new check. The MCP server keeps running, and each tool call returns the storage error. A server that exits at startup hides the reason from the agent.

## Decisions

- **Refuse, not fall back to the worktree.** Using the worktree root as the hub puts the issue log on the branch, and `start` and `merge` break. That belongs to a future shared-workspace product.
- **No hub override (flag or environment variable).** An override that points outside the repository is the out-of-repository `.xpo` idea, which is deferred.
- **Check in storage, not only in the CLI.** The MCP server and the board server both read through storage. One check covers all entry points.
- **The bare check belongs to Personal's hub discovery, not to storage.** In Studio (milestone 1.8.0), each project gets its `.xpo` path from the server, and its clone of the origin can be bare. When xpo-5433a8 replaces the `storage` globals with a `Project` value, move the bare check into the constructor that Personal uses. The `ErrNoXpoDir` check can stay with the project.
- **`HubRoot()` keeps returning a path.** Changing it to return an error touches 23 call sites. The storage choke points stop all reads and writes before a call site uses the path.

## Edge cases

- MEDIUM: tests that create a temp directory without `.xpo` and expect an empty read now get `ErrNoXpoDir`. Fix each test to create `.xpo`, or assert the error when the test is about a missing `.xpo`.
- LOW: no git (not a repository). `HubRoot()` returns `.`. `CheckHub()` then only checks `./.xpo`.
- LOW: `git rev-parse --is-bare-repository` fails. Treat the result as "not bare".
- LOW: artifact reads (`spec`, `walkthrough`) join paths under `XpoDir()` directly. They already return a file-not-found error, not empty content. No change.

## Acceptance criteria

- [ ] In a worktree of a bare repository, `CheckHub()` returns an error that matches `ErrBareRepository` and names the common directory.
- [ ] In that worktree, `ReadEvents` and `AppendEvent` return the same error.
- [ ] In that worktree, `xpo list` exits non-zero with the bare-repository message. `xpo list --json` reports it as a JSON error.
- [ ] When the hub has no `.xpo`, `ReadEvents` returns an error that matches `ErrNoXpoDir`, not an empty list.
- [ ] When `.xpo` exists but `issues.db` does not, `ReadEvents` returns no events and no error.
- [ ] A normal checkout and a linked worktree of a normal repository behave as before.
- [ ] `make test` passes.
