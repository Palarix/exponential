# Spec: serialize event writes to issues.db

## What

Every write to `.xpo/issues.db` holds an exclusive cross-process file lock, `.xpo/events.lock`. Each appended line goes out in a single `Write` call. Archive re-reads the log while it holds the lock, so it can't drop events that arrived after the preview.

## Why

Two agents (two MCP servers, or the CLI and the web server) that write at the same time can silently lose events:

- `AppendEventCollapsed` reads the log, rebuilds the tail, and replaces the file. An append that lands between the read and the rename disappears. Two concurrent collapses also share the same `issues.db.tmp` path and overwrite each other's temp file.
- `AppendEvent` writes the JSON and the `"\n"` as two separate calls, so another process's write can land between them and corrupt both lines.
- `ArchiveEvents` truncates `issues.db` in place (`O_TRUNC`) and rewrites it from an event list that `maintenance` computed earlier, during the preview. Any event appended since that preview is lost, and a reader that runs mid-rewrite sees a truncated file.

## How

### 1. The lock (`internal/storage/lock.go`, new)

```go
// withEventsLock runs fn while holding an exclusive lock on .xpo/events.lock.
func withEventsLock(fn func() error) error
```

- Uses `gofrs/flock`, like `WithGitLock`. `flock(2)` locks belong to the open file description, so two goroutines in the same process that each call `flock.New` also exclude each other. The goroutine test relies on this.
- Timeout is 10s and the retry delay is 5ms, because event writes take milliseconds. If the lock times out, return the error `another xpo process is writing issues.db — try again in a moment`.
- The lock is not reentrant. Only the three exported write functions take it, and none of them calls another.

### 2. Callers

| Function | Change |
|---|---|
| `AppendEvent` | Take the lock. Marshal the event, append `'\n'` to the same byte slice, and write it with one `Write` call. |
| `AppendEventCollapsed` | Take the lock around the whole read, rebuild, and rewrite. |
| `ArchiveEvents` | New signature: `ArchiveEvents(issueIDs map[string]bool) (ArchiveResult, error)`. Under the lock: re-read the log, replay it (`applyStateEvent`), and **re-check eligibility** — an ID is archived only if the issue is still deleted or in a terminal status (`model.IsTerminal`); otherwise it is skipped. Then back up, append that issue's events to `archive.db`, and rewrite `issues.db` via temp file + rename (`rewriteFile`, no `O_TRUNC`). `ArchiveResult` reports `Archived` and `Skipped` issue IDs. |

`rewriteFile` stays the same apart from checking write errors. It only runs under the lock, so the shared `.tmp` name is safe.

### 3. Lock order with `git.lock`

**Always take `git.lock` before `events.lock`, never the other way round.** Merge holds `git.lock` and calls `recordMerge`, which appends events. Nothing in `storage` ever takes `git.lock`. I'll add this to the doc comments on `WithGitLock` and `withEventsLock`.

### 4. `.gitignore` / doctor

Add `.xpo/events.lock` to the ignore entries that `setup` writes and to the `requiredIgnores` list that `doctor` checks, next to `.xpo/git.lock`.

### 5. Torn tail in readers

Readers don't take the lock. Collapse and archive swap the file in with a rename, so a reader always sees a whole file; but a reader can still catch an `AppendEvent` mid-write. `ReadEvents` therefore skips a **final line that has no `\n` terminator and fails to parse** — a torn append. A complete JSON object without a trailing newline (hand-edited file) is still read, and an invalid *terminated* line still returns an error as today. `ReadEvents` now reads the file whole and splits on `\n` instead of using `bufio.Scanner`, which also removes the scanner's 64 KB line limit.

### 6. Archive caller

`ArchiveStats` gains `IssueIDs map[string]bool` (the IDs the preview chose) and drops the now-unused `ActiveEvents`/`ArchivedEvents` lists. `PerformArchive` passes it and returns the result; `cmd/exponential/archive.go` reports skipped issues ("N skipped — changed since preview").

## Out of scope

- **The branch-mode `unionLines` rewrite in merge** (`merge.go:128`) is also a read-modify-write, but branch mode is being removed in xpo-863802, so I'm leaving it alone.

## Acceptance criteria

- [ ] `AppendEvent`, `AppendEventCollapsed` and `ArchiveEvents` all hold `.xpo/events.lock` while they write.
- [ ] `AppendEvent` issues exactly one `Write` per event.
- [ ] Test: N goroutines run a mix of `AppendEvent` and `AppendEventCollapsed` at the same time (CREATE events, which never collapse, with unique IDs). Afterwards every event is present exactly once and every line parses. Run it under `-race`.
- [ ] Test: a subprocess (the test binary re-invoked through a helper-process env var) appends while the parent collapses. No events are lost. This proves the lock works across processes, not just between goroutines.
- [ ] Test: an event appended after the archive preview is computed survives `ArchiveEvents`.
- [ ] Test: an issue that left DONE after the preview is skipped and stays in `issues.db` with all its events; the result lists it as skipped.
- [ ] Test: `ReadEvents` ignores a torn (unterminated, unparseable) final line, keeps a valid unterminated one, and still errors on an invalid terminated line.
- [ ] The archive rewrite uses temp file + rename (no `O_TRUNC`).
- [ ] Lock order with `git.lock` is documented.
- [ ] `.xpo/events.lock` is gitignored by setup and checked by doctor.
- [ ] `make test` passes.

## Decisions (confirmed with user)

1. Torn tail: `ReadEvents` skips an unterminated final line (option b).
2. Archive API: pass issue IDs and re-check eligibility under the lock (option C over "keep event lists" — fragile content matching, orphaned updates — and "IDs without re-check" — could archive a reopened issue).
