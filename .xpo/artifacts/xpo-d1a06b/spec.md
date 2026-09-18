## Spec: Standardize JSON error output across all CLI --json commands

## What

Every CLI command with a `--json` flag must emit `jsonio.ErrorOutput` JSON to stdout
(via `exitJSONError`) on failure instead of plain text, and exit non-zero.

## Why

In `--json` mode, callers (scripts, MCP bridges, agents) parse stdout as JSON. Plain-text
errors break that contract. `comment.go` was fixed in xpo-aa4529; this issue covers the
remaining 11 command files with 25+ error paths.

## Scope

| File | Flag | Handler | Error paths | Notes |
|------|------|---------|-------------|-------|
| `add.go` | `addJSONFlag` | `Run` | 6 | Includes duplicate-check warning |
| `update.go` | `updateJSONFlag` | `Run` | 6 | Includes "no changes" informational |
| `update.go` | `doneJSONFlag` | `Run` | 1 | |
| `update.go` | `plannedJSONFlag` | `Run` | 1 | |
| `list.go` | `listJSONFlag` | `Run` | 1 | Error before flag check |
| `show.go` | `showJSONFlag` | `Run` | 1 | Error before flag check |
| `comments.go` | `commentsJSONFlag` | `RunE` | 1 | Returns error to cobra |
| `history.go` | `historyJSON` | `Run` | 3 | Errors before flag check |
| `rationale.go` | `rationaleJSONFlag` | `Run` | 1 | Currently writes to stderr |
| `inbox.go` | `inboxJSONFlag` | `RunE` | 2 | `--clear` ignores `--json` |
| `blocked.go` | `blockedJSONFlag` | `Run` | 2 | Two modes (list + mark) |
| `pulse.go` | `pulseJSON` | `RunE` | 1 | |

## How

### Pattern

For each error path in a `--json` command, wrap with a flag check:

**`Run` commands** (currently `fmt.Printf("Error: ...") + os.Exit(1)`):
```go
// Before:
if err != nil {
    fmt.Printf("Error: %v\n", err)
    os.Exit(1)
}

// After:
if err != nil {
    if xxxJSONFlag {
        exitJSONError(err)
    }
    fmt.Printf("Error: %v\n", err)
    os.Exit(1)
}
```

**`RunE` commands** (currently `return fmt.Errorf(...)`):
```go
// Before:
if err != nil {
    return fmt.Errorf("issue %s not found: %w", id, err)
}

// After:
if err != nil {
    if xxxJSONFlag {
        exitJSONError(fmt.Errorf("issue %s not found: %w", id, err))
    }
    return fmt.Errorf("issue %s not found: %w", id, err)
}
```

### Per-file notes

**`add.go`** — The duplicate-check warning (styled text + `os.Exit(0)`) should emit
an `ErrorOutput` with a descriptive message and exit non-zero in JSON mode. A duplicate
warning that exits 0 with styled text is not parseable; in JSON mode it's an error.

**`update.go`** — The "no changes" path (`UpdatePayloadEmpty`) currently prints and
returns silently. In JSON mode, emit `ErrorOutput` with "no changes in payload" and
exit non-zero — an empty update is a caller error.

**`update.go` done/planned** — Error fires before the JSON flag check; add the flag
check before the error.

**`show.go`** — The `showIssue` function is called from multiple places. Pass the JSON
flag through so it can conditionally emit JSON errors.

**`inbox.go`** — When `--clear` and `--json` are both set: emit JSON for both success
and error. Success: `{"cleared": true}` or similar (but that's a new output type —
out of scope; just ensure errors are JSON). For now, only wrap the error path.

**`rationale.go`** — Currently writes error to stderr; in JSON mode, use `exitJSONError`
(writes to stdout) instead.

### Non-goals

- No new output types (except `ErrorOutput` which already exists).
- No handler type changes (`Run` → `RunE` or vice versa).
- No changes to non-JSON paths.
- No changes to `comment.go` (already done).

## Acceptance Criteria

- All `--json` CLI commands emit `jsonio.ErrorOutput` JSON on failure
- All `--json` CLI commands exit non-zero on failure
- Non-JSON error paths are unchanged
- `make test` passes
