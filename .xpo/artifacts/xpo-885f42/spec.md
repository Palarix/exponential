## Spec: Add --json input/output to link command

## What

Add a `--json` flag to the `link` CLI command that reads `jsonio.LinkToolInput` from
stdin and emits `jsonio.LinkOutput` on success / `jsonio.ErrorOutput` on failure.

## Why

The `link` command is one of the few mutation commands without `--json` support, breaking
the unified JSON I/O contract.

## How

Single file change: `cmd/exponential/link.go`.

### Flow (--json path)

```
stdin → jsonio.DecodeStrict → LinkToolInput → validate → client.GetIssue (both)
→ self-link check → duplicate check → client.UpdateIssue → LinkOutput JSON
```

### Steps

1. Add `linkJSONFlag bool` and register `--json` flag in `init()`.
2. Add a `--json` branch at the top of `Run`, before the positional-arg path:
   - Read stdin via `readStdinExplicit()`
   - Decode into `jsonio.LinkToolInput` via `jsonio.DecodeStrict`
   - Validate: source/target non-empty, type normalizes
   - Resolve both issues via `client.GetIssue`
   - Self-link check (MCP parity)
   - Duplicate-link check (MCP parity)
   - Build and apply the dependency update
   - Emit `jsonio.LinkOutput{Source, Target, Kind}` as indented JSON
   - All errors use `exitJSONError`
3. Add imports: `jsonio`, `encoding/json`.
4. Add the JSON flag check to the existing error paths in the non-JSON branch
   (for the centralized error handling from xpo-d1a06b).

### Non-goals

- Backporting self-link / duplicate-link checks to the non-JSON path (separate concern).
- Changing the positional-arg interface.

## Acceptance Criteria

- `echo '{"source":"x","target":"y","type":"blocks"}' | xpo link --json` works
- Output matches `jsonio.LinkOutput` schema
- Errors match `jsonio.ErrorOutput` schema
- `make test` passes
