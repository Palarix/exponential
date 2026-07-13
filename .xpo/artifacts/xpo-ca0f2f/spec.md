# Drive Trail Recording

## Overview

Instrument `DriveIssue()` to emit a per-run JSONL trail file at `.xpo/artifacts/<issue-id>/drive-<run-id>.jsonl`. The trail captures the full execution narrative — role handoffs, verdicts, retries, test results, token/cost spend — without bloating the issue event log.

The issue event log receives a single `ARTIFACT` pointer event per run. Renderers (recap, replay, spectator) consume trail files directly.

## Trail File Location

```
.xpo/artifacts/<issue-id>/drive-<run-id>.jsonl
```

`run-id` is a short random hex string (8 chars), generated once at drive start. Same format as issue IDs but without the `xpo-` prefix.

## Trail Event Schema

Every line is a self-contained JSON object:

```json
{
  "run_id": "a1b2c3d4",
  "seq": 0,
  "event": "drive_start",
  "ts": "2026-07-13T10:00:00.000Z",
  "data": { ... }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `run_id` | string | 8-char hex, constant across the run |
| `seq` | int | Monotonically increasing, 0-indexed |
| `event` | string | Event type (see below) |
| `ts` | string | RFC3339 timestamp with milliseconds |
| `data` | object | Event-type-specific payload |

## Event Types

### `drive_start`

Emitted once at the beginning of the run, after pre-flight passes and before any work begins.

```json
{
  "issue_id": "xpo-abc123",
  "title": "Fix projection ordering",
  "supervisor": "claude",
  "supervisor_model": "opus",
  "coder": "claude",
  "max_retries": 3,
  "branch": "xpo-abc123-fix-projection-ordering",
  "timeout": "30m",
  "test_cmd": "make test"
}
```

### `spec_eval`

Emitted after each spec evaluation round (including the initial evaluation).

```json
{
  "round": 1,
  "ready": false,
  "revised": true,
  "tokens_in": 4200,
  "tokens_out": 1800,
  "cost_usd": 0.12
}
```

### `plan`

Emitted after the supervisor plans implementation context.

```json
{
  "files": ["internal/storage/read.go", "internal/model/types.go"],
  "test_cmd": "make test",
  "tokens_in": 3100,
  "tokens_out": 900,
  "cost_usd": 0.08
}
```

### `attempt_start`

Emitted when a coder attempt begins.

```json
{
  "attempt": 1,
  "has_feedback": false
}
```

For retries:

```json
{
  "attempt": 2,
  "has_feedback": true,
  "feedback_preview": "AC-3 not met: projection must handle..."
}
```

`feedback_preview` is truncated to 200 chars. The full feedback is in the preceding `verdict` event.

### `test_result`

Emitted after tests run (skipped if no test command configured).

```json
{
  "attempt": 1,
  "passed": false,
  "exit_code": 1,
  "summary": "3 failures in TestProjection",
  "duration_ms": 4200
}
```

`summary` is the last 5 lines of test output, truncated to 500 chars. Full test output is not stored (it can be enormous and has no replay value).

### `verdict`

Emitted after the supervisor reviews each attempt.

```json
{
  "attempt": 1,
  "passed": false,
  "feedback": "AC-3 not met: the projection does not handle out-of-order events. The test covers sequential replay but the spec requires idempotent application regardless of event ordering.",
  "tokens_in": 8500,
  "tokens_out": 600,
  "cost_usd": 0.15
}
```

`feedback` is the full reviewer feedback (only present when `passed` is false). This is the dramatic beat — don't truncate it.

### `coder_done`

Emitted when the coder agent finishes (regardless of test/review outcome).

```json
{
  "attempt": 1,
  "tokens_in": 22000,
  "tokens_out": 8500,
  "cost_usd": 0.58,
  "duration_ms": 45000
}
```

### `drive_end`

Emitted once at the end of the run.

```json
{
  "outcome": "merged",
  "attempts": 2,
  "commit": "a1b2c3d",
  "tokens_in": 84000,
  "tokens_out": 12000,
  "cost_usd": 1.42,
  "wall_time_ms": 512000
}
```

`outcome` is one of: `merged`, `blocked`, `error`, `timeout`, `no-merge` (when `--no-merge` flag was used).

## Implementation

### New types: `internal/exponential/trail.go`

A new file containing:

- `TrailEvent` struct matching the schema above
- `TrailWriter` struct that holds `run_id`, a `seq` counter, the open file handle, and a `mu sync.Mutex`
- `NewTrailWriter(artifactsDir, issueID string) (*TrailWriter, error)` — generates run ID, creates the file
- `(tw *TrailWriter) Emit(event string, data any) error` — JSON-marshals and appends one line, increments seq
- `(tw *TrailWriter) Close() error` — closes the file handle

`TrailWriter` is goroutine-safe (mutex around writes) but single-writer in practice. The file is opened with `O_APPEND|O_CREATE|O_WRONLY` so partial writes from a crash still produce valid JSONL up to the last complete line.

### Instrumentation points in `drive.go`

All instrumentation goes through `TrailWriter.Emit()`. The trail writer is created early (after pre-flight, before `StartWork`) and closed in a defer.

| Location (current line) | Event | Notes |
|---|---|---|
| After line 576 (pre-flight passes) | `drive_start` | After branch creation, before spec eval |
| After line 611 / 629 (spec eval) | `spec_eval` | Once per evaluation round; capture stats delta from `supExec.Stats()` |
| After line 663 (planning done) | `plan` | Includes file list and resolved test command |
| Line 672 (attempt loop start) | `attempt_start` | Before coder prompt |
| After line 688 (coder returns) | `coder_done` | Stats delta from `coderExec.Stats()` |
| After line 703 (tests run) | `test_result` | Only if test command is set |
| After line 717 / 722 (review done) | `verdict` | Pass or fail with feedback |
| Line 790-793 (final stats) | `drive_end` | After merge, with totals |

### Stats deltas

`AgentStats` is cumulative. To get per-event token/cost deltas, snapshot `Stats()` before and after each agent call and subtract. Add a helper:

```go
func statsDelta(before, after AgentStats) AgentStats {
    return AgentStats{
        Calls:     after.Calls - before.Calls,
        Duration:  after.Duration - before.Duration,
        TokensIn:  after.TokensIn - before.TokensIn,
        TokensOut: after.TokensOut - before.TokensOut,
        CostUSD:   after.CostUSD - before.CostUSD,
    }
}
```

### ARTIFACT pointer event

After the trail is complete and before merge, emit a standard `ARTIFACT` event to the issue log:

```go
c.RecordArtifact(issue.ID, "drive-trail", trailFilename, "created")
```

This is the only touch point with `issues.db`. The artifact event payload includes `artifact_type: "drive-trail"` so renderers can discover trail files via the event log without scanning the filesystem.

### Failure modes

- If trail file creation fails (disk full, permissions), log a warning and continue the drive — trail recording is observability, not critical path. Use a `nil`-safe `TrailWriter` where `Emit()` on nil is a no-op.
- If the drive crashes mid-run, the JSONL is valid up to the last complete line. No `drive_end` event means the run was interrupted — renderers should handle this gracefully.
- If `drive_end` is missing, renderers can infer outcome from the last event present (e.g., last event is `verdict` with `passed: true` but no `drive_end` → crashed during walkthrough/merge).

## Acceptance Criteria

1. A successful `xpo drive` run produces a trail file at `.xpo/artifacts/<issue-id>/drive-<run-id>.jsonl`
2. The trail contains at minimum: `drive_start`, one `attempt_start`, one `coder_done`, one `verdict`, and `drive_end`
3. A failed run (blocked after max retries) produces a trail with `drive_end` having `outcome: "blocked"`
4. An interrupted run produces valid JSONL up to the last complete event (no `drive_end`)
5. Token/cost figures in trail events are non-zero and consistent with the final summary printed to stdout
6. The issue event log receives exactly one `ARTIFACT` event per run pointing to the trail file
7. Trail recording failure does not prevent the drive from completing
8. `xpo artifact show <issue-id>` lists the trail file(s)
9. The trail file is committed to git along with other artifacts (on the feature branch before merge)

## Out of Scope

- Renderers (recap, replay, spectator) — separate issues
- Trail file migration/schema versioning — defer until we actually need to change the schema
- Compression — JSONL files for a single run will be small (5-50 KB); not worth optimizing yet