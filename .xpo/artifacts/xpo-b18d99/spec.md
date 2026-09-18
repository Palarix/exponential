## Spec: Add jsonio.PulseOutput wrapper for pulse --json

## What

Add a `PulseOutput` type alias in `jsonio` and update `pulse.go` to use it.

## How

Two files:

### 1. `internal/jsonio/output.go`

Add a type alias:
```go
type PulseOutput = server.PulseMetrics
```

This adds a `jsonio → server` dependency. No cycle exists (`server` does not import `jsonio`).

### 2. `cmd/exponential/pulse.go`

Change `enc.Encode(m)` to `enc.Encode(jsonio.PulseOutput(m))`. Since it's a type alias,
this is a no-op cast — the JSON output is identical.

## Non-goals

- No new MCP pulse tool.
- No duplication of PulseMetrics or its 9 sub-types.

## Acceptance Criteria

- `pulse --json` emits a `jsonio.PulseOutput` type
- JSON output unchanged
- `make test` passes
