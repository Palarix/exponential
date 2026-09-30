# Walkthrough: serialize event writes to issues.db

## The problem

`.xpo/issues.db` is an append-only JSON-lines log that several processes write at once: every agent's MCP server, the CLI, and the web server. Nothing coordinated those writes:

- **Collapse is read-modify-write.** `AppendEventCollapsed` reads the log, rebuilds the uncommitted tail in memory, and replaces the file through `issues.db.tmp` + rename. An append from another process that lands between the read and the rename is overwritten. Two concurrent collapses also share the same `.tmp` path.
- **Append took two writes.** `AppendEvent` wrote the JSON, then `"\n"`, so another process's line could land between them.
- **Archive used stale data and truncated in place.** `ArchiveEvents` rewrote `issues.db` with `O_TRUNC` from event lists computed during the `xpo archive` preview, before the confirmation prompt. Anything written after the preview was lost, and a reader that ran mid-rewrite saw a truncated file.

`git.lock` didn't help: it only wraps git operations in start and merge.

## The fix

### One lock for every writer: `internal/storage/lock.go`

`withEventsLock(fn)` takes an exclusive `gofrs/flock` lock on `.xpo/events.lock`, with the same pattern as `WithGitLock` but a 10s timeout and 5ms retry, because event writes take milliseconds. `flock(2)` locks belong to the open file description and each call opens its own, so goroutines in the same process exclude each other as well. The in-process concurrency test relies on this.

The three exported writers are the only callers. Each wraps an unexported function that does the real work (`appendEventCollapsed`, `archiveEvents`). The lock is not reentrant, so nothing called inside the lock may call an exported writer.

**Lock order:** always `git.lock` → `events.lock`. Merge holds `git.lock` and records its MERGE/DONE events inside it. `storage` never takes `git.lock`, so no cycle is possible. This is written on both `withEventsLock` and `WithGitLock`.

### Writers

- `AppendEvent` marshals the event, appends `'\n'` to the same slice, and writes it with one `Write` under the lock. Marshalling happens before the lock is taken to keep the critical section short.
- `AppendEventCollapsed` holds the lock across the whole read → rebuild → rewrite. `rewriteFile` now checks every write error and removes the temp file on failure. Its doc comment states that callers must hold the lock, because the `.tmp` path is shared.
- `ArchiveEvents(issueIDs map[string]bool) (ArchiveResult, error)` replaces the event-list API. Under the lock it:
  1. re-reads the log and replays it with the existing `applyStateEvent`;
  2. keeps only the IDs that are still deleted or terminal (`model.IsTerminal`), and reports the rest as `Skipped`;
  3. moves **every** event of the kept issues to `archive.db`, including comments written after the preview, so an issue's history never ends up split across the two files;
  4. writes the backup, then replaces `issues.db` through `rewriteFile` (temp file + rename, no truncation).

Why IDs with a re-check (option C in the review discussion):
- Passing event lists would need content matching, which breaks as soon as a collapse changes an event. It would also leave orphaned updates behind in `issues.db`.
- Passing IDs without the re-check could archive an issue that was reopened between the preview and the confirmation.

With C, the preview can only overstate what gets archived, never understate it. `xpo archive` prints the skipped issues.

`ArchiveStats` now carries `IssueIDs`. Its `ActiveEvents`/`ArchivedEvents` fields had no remaining users and were removed.

### Readers stay lock-free

Collapse and archive swap in a whole file by rename, so the only thing a reader can observe is an append halfway through its write. `ReadEvents` now reads the file whole and splits on `\n`. A **final** line with no newline that fails to parse is treated as that in-progress append and skipped. Two details:

- A final line that is a valid event but has no trailing newline (a hand-edited file) is still read. A prefix of a JSON object never parses, so the "fails to parse" condition catches exactly the torn-write case.
- Any other invalid line still returns an error, as before. Dropping `bufio.Scanner` also removes its 64 KB-per-line limit.

`ValidateEvents` (doctor) and `ReadArchivedEvents` are unchanged. Doctor is diagnostic, and a torn tail there only shows up for a moment.

### Housekeeping

`.xpo/events.lock` is added next to `.xpo/git.lock` in setup's gitignore entries, doctor's required ignores, and this repo's `.gitignore`.

## Out of scope

The branch-mode `unionLines` rewrite in `merge.go` is also a read-modify-write, but branch mode is being removed in xpo-863802.

## Acceptance criteria

- [x] `AppendEvent`, `AppendEventCollapsed` and `ArchiveEvents` all hold `.xpo/events.lock` while they write. *Evidence:* each is a `withEventsLock` wrapper; `TestAppendEvent_WaitsForEventsLock` shows `AppendEvent` blocking while another holder has the lock.
- [x] `AppendEvent` issues exactly one `Write` per event. *Evidence:* `storage.go`, where the newline is appended to the marshalled slice before the single `f.Write(line)`.
- [x] Concurrent goroutines appending and collapsing lose nothing, and every line parses, under `-race`. *Evidence:* `TestConcurrentAppendAndCollapse_NoLostEvents`, which failed on 3 of 3 runs before the fix and passes on `-race -count=5` after it.
- [x] A second process appending while the parent collapses loses nothing. *Evidence:* `TestCrossProcessAppendAndCollapse_NoLostEvents` (re-invokes the test binary through `TestHelperProcessAppend`), which failed on 3 of 3 runs before the fix and passes after.
- [x] An event appended after the archive preview survives. *Evidence:* `TestArchiveEvents_KeepsEventsAppendedAfterPreview`; also `TestArchiveEvents_MovesLateEventsOfArchivedIssue`.
- [x] An issue that left DONE after the preview is skipped, stays in `issues.db` with all its events, and is listed as skipped. *Evidence:* `TestArchiveEvents_SkipsIssueReopenedAfterPreview`.
- [x] `ReadEvents` ignores a torn final line, keeps a valid unterminated one, and still errors on an invalid terminated line. *Evidence:* `TestReadEvents_IgnoresTornFinalLine`, `TestReadEvents_KeepsValidUnterminatedFinalLine`, `TestReadEvents_ErrorsOnMalformedLineBeforeTail`, `TestReadEvents_MalformedJSON`.
- [x] The archive rewrite uses temp file + rename. *Evidence:* `TestArchiveEvents_ReplacesFileAtomically` (inode changes).
- [x] Lock order with `git.lock` is documented. *Evidence:* doc comments on `withEventsLock` and `WithGitLock`.
- [x] `.xpo/events.lock` is gitignored by setup and checked by doctor. *Evidence:* `setup.go` and `doctor.go` ignore lists.
- [x] `make test` passes. *Evidence:* exit 0 (lint, 369 frontend tests, all Go packages).
