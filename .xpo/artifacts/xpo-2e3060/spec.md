# Spec: Add `--json` output to `list` command

## What

Add a `--json` flag to `cmd/exponential/list.go`. When set, output the filtered issues
as a `jsonio.ListOutput` JSON envelope instead of the terminal table.

## Why

Part of the unified JSON I/O epic (xpo-a4a271). The `list` command should emit the
same JSON structure that the MCP `list` tool returns, so scripts and agents get a
consistent contract regardless of transport.

## Acceptance Criteria

1. `xpo list --json` emits valid JSON matching `jsonio.ListOutput` shape
2. All existing filter flags work with `--json`
3. `make test` passes

## Flow

1. Add `listJSONFlag bool` variable and `--json` flag registration in `init()`.
2. In the `Run` function, after filtering and before rendering: if `listJSONFlag` is set,
   build a `jsonio.ListOutput` by converting each `*model.Issue` via `jsonio.ToIssueSummary()`,
   encode to stdout with `json.NewEncoder` + `SetIndent("", "  ")`, and return early.
3. Add a test case in `cmd/exponential/` or verify manually.

## Decisions

1. **Use `jsonio.ListOutput` not raw model** — matches the MCP contract exactly.
2. **Pretty-printed JSON** — consistent with `rationale --json` and `pulse --json` which
   use `SetIndent("", "  ")`.
3. **No `--json` input** — `list` is read-only, no mutation payload to accept.
