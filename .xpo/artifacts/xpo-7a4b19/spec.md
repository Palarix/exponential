# Trail Watcher — Broadcast Trail Events via SSE

## Overview

Watch `.xpo/artifacts/` for new or appended drive trail files and broadcast each new trail event as a granular SSE message to connected web clients. This bridges the gap between trail recording (xpo-ca0f2f) and the frontend trail viewer (xpo-575923).

Follows the same fsnotify pattern as the existing `WatchDB()` and `WatchGitRefs()` watchers.

## Design

### What to watch

Trail files live at `.xpo/artifacts/<issue-id>/drive-<run-id>.jsonl`. The watcher needs to:

1. Watch `.xpo/artifacts/` for new subdirectories (issue directories created during a drive)
2. Watch each issue subdirectory for new `drive-*.jsonl` files
3. Tail active trail files for appended lines

Since fsnotify on Linux doesn't recursively watch, and trail files are append-only JSONL, the simplest approach is:

- On startup, scan for existing `drive-*.jsonl` files and record their sizes
- Watch `.xpo/artifacts/` for `CREATE` events (new directories or files)
- When a `drive-*.jsonl` file is created or modified, read from the last-known offset to EOF, parse each new line as JSON, and broadcast

### SSE message format

Event name: `drive_event`

Data payload:

```json
{
  "issue_id": "xpo-abc123",
  "trail_file": "drive-a1b2c3d4.jsonl",
  "event": {
    "run_id": "a1b2c3d4",
    "seq": 3,
    "event": "test_result",
    "ts": "2026-07-13T10:05:32.000Z",
    "data": { "attempt": 1, "passed": false, "exit_code": 1 }
  }
}
```

The outer envelope adds `issue_id` and `trail_file` so the frontend can route the event to the correct issue view without parsing the inner event. The inner `event` field is the raw trail event line, passed through verbatim.

### SSE hub extension

The existing `SSEHub` in `sse.go` broadcasts `SSEEvent` structs with `Type` and `IssueID` fields. Extend this:

```go
type SSEEvent struct {
    Type    string      `json:"type"`
    IssueID string      `json:"issue_id"`
    Data    interface{} `json:"data,omitempty"`    // new: arbitrary payload
}
```

The `handleSSE()` handler already JSON-marshals events. Adding a `Data` field is backwards-compatible — existing event types (`issue_updated`, etc.) just have `nil` data.

For `drive_event`, the frontend receives:

```
event: drive_event
data: {"issue_id":"xpo-abc123","trail_file":"drive-a1b2c3d4.jsonl","event":{...}}
```

### Implementation: `watch_trails.go`

New file in `internal/server/`, following the pattern of `watch.go` and `watch_git.go`.

```go
func (s *Server) WatchTrails(ctx context.Context) {
    // 1. Resolve artifacts dir from config
    // 2. Initial scan: find all drive-*.jsonl files, record sizes
    // 3. Set up fsnotify watcher on .xpo/artifacts/
    // 4. On CREATE/WRITE events for drive-*.jsonl files:
    //    a. Read from last-known offset to EOF
    //    b. Parse each line as JSON
    //    c. Extract issue ID from parent directory name
    //    d. Broadcast each line as a drive_event SSE message
    //    e. Update offset tracker
    // 5. Debounce: 50ms (shorter than the 150ms used for issues.db,
    //    because trail events are small and we want low latency)
}
```

**Offset tracking**: a `map[string]int64` mapping trail file paths to their last-read byte offset. On file modification, seek to the offset, read new lines, update. This handles the append-only nature of trail files without re-reading the whole file.

**Issue ID extraction**: parse from the directory name — `.xpo/artifacts/xpo-abc123/drive-xxx.jsonl` → `xpo-abc123`. No need to read the file content for routing.

**Dynamic directory watching**: when a new directory appears in `.xpo/artifacts/`, add it to the fsnotify watch list. This handles the case where a drive creates a new issue artifact directory.

### Lifecycle

`WatchTrails()` is started alongside `WatchDB()` and `WatchGitRefs()` in the server startup path (both `board.go` and `serve.go`). It runs in a goroutine with the server's context for clean shutdown.

### Edge cases

- **Multiple concurrent drives** (future): each drive writes to a different trail file. The watcher handles them independently — each file has its own offset tracker.
- **CLI drives while board is open**: work automatically. The CLI writes the trail file, fsnotify fires, watcher reads and broadcasts.
- **Trail file written before board starts**: on startup scan, we record sizes but don't broadcast historical events. The frontend fetches completed trails via the artifact GET endpoint.
- **Rapid writes**: fsnotify may coalesce multiple appends into one WRITE event. The offset-based reader handles this correctly — it reads all new lines since the last offset, regardless of how many appends happened.

## Acceptance Criteria

1. A `drive_event` SSE message is broadcast within 200ms of a trail event being appended to a trail file
2. Each trail event produces exactly one SSE message (no duplicates, no missed events)
3. CLI-initiated drives (not started from the web UI) produce SSE events when the board is open
4. Multiple concurrent trail files are tracked independently
5. Server startup with existing trail files does not broadcast historical events
6. Server shutdown cleanly stops the watcher (no goroutine leaks)
7. Missing or inaccessible artifacts directory does not crash the server

## Out of Scope

- Frontend rendering (xpo-575923)
- Drive subprocess management (xpo-234823)
- Trail recording itself (xpo-ca0f2f)