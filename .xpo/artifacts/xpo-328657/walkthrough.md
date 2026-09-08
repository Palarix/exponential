# Walkthrough: Fix low-priority test assertion quality issues

## What changed

Four fixes addressing weak test assertions identified by the external audit.

### 1. DoneCountLimited assertion tightened (`list_test.go`)

Added `doneCount == 0` check before the existing `> 3` check. Previously, returning zero DONE issues would pass — now it fails, verifying that recent DONE issues are actually included.

### 2. Force-worktree existence check (`start_test.go`)

Added `os.Stat(wtPath2)` after force takeover to verify the recreated worktree directory exists on disk, not just that a non-empty path was returned.

### 3. Archive fallback test (`local_transport_test.go`)

`TestFindIssue_ArchivedFallback` — creates an issue, archives all events via `storage.ArchiveEvents`, then calls `FindIssue`. Verifies the issue is found with `archived=true`. This covers the `local_transport.go:109-117` archive fallback path that was previously untested.

### 4. Deduplicated GetIssue tests (`local_transport_test.go`)

Replaced `TestGetIssue_Found`, `TestGetIssue_NotFound`, `TestGetIssue_ShortID` (which duplicated `resolve_test.go`) with:
- `TestGetIssue_FieldsPreserved` — verifies title, status, and estimate survive the round-trip (not just ID matching)
- `TestGetIssue_AmbiguousShortID` — verifies ambiguous prefix returns the correct error (complements `resolve_test.go`'s ambiguous test with a different entry point)
