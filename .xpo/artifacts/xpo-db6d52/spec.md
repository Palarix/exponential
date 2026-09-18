## Spec: Resolve all CLI JSON input IDs to canonical issue IDs

## What

Add `client.GetIssue(id)` resolution before operations in all affected CLI `--json` paths,
and use the returned `issue.ID` in both operations and output structs — matching how the
MCP handlers work.

## Why

CLI JSON paths currently echo the caller-supplied identifier (short hashes, prefixless IDs,
partial matches) in their output. The MCP handlers always resolve to canonical IDs first.
Scripts and agents consuming CLI JSON output get noncanonical IDs that may not round-trip.

## Affected commands

| Command | File | Issue |
|---------|------|-------|
| `update --json` | `cmd/exponential/update.go` | Uses raw `args[0]` in output |
| `done --json` | `cmd/exponential/update.go` | Uses raw `args[0]` in output |
| `planned --json` | `cmd/exponential/update.go` | Uses raw `args[0]` in output |
| `start --json` | `cmd/exponential/update.go` | Uses raw `input.ID` in output |
| `comment --json` | `cmd/exponential/comment.go` | Calls GetIssue but discards result |
| `blocked [id] --json` | `cmd/exponential/blocked.go` | Uses raw `args[0]` in output |

## How

For each affected path, add (or fix) the resolve step:

```go
issue, err := client.GetIssue(rawID)
if err != nil {
    exitJSONError(err)
}
// Use issue.ID for all subsequent operations and output
```

### update --json (`update.go`)

Currently at line ~59: `client.UpdateIssue(id, payload, "update")` with raw `id`.
Add `client.GetIssue(id)` before the operation, use `issue.ID` in both `UpdateIssue`
call and `UpdateOutput`.

### done --json (`update.go`)

Currently: `client.UpdateIssue(args[0], ...)` and `UpdateOutput{ID: args[0]}`.
Add resolve, use `issue.ID`.

### planned --json (`update.go`)

Same pattern as `done`.

### start --json (`update.go`)

Currently: `client.StartWork(input.ID, ...)` and `StartOutput{ID: input.ID}`.
Add `client.GetIssue(input.ID)` resolve, use `issue.ID` in both.

### comment --json (`comment.go`)

Currently calls `client.GetIssue(issueID)` but assigns to `_`. Change to capture
the result and use `issue.ID` throughout.

### blocked [id] --json (`blocked.go`)

Currently: `client.UpdateIssue(args[0], ...)` and `UpdateOutput{ID: args[0]}`.
Add resolve, use `issue.ID`.

## Acceptance Criteria

1. All six JSON CLI commands resolve IDs through GetIssue before operating
2. Output contains canonical issue ID regardless of input format
3. CLI JSON output matches MCP output for the same resolved issue
4. `make test` passes
