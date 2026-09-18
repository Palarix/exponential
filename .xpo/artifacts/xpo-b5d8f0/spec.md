# Spec: Add `--json` output to `show` command

## What

Add a `--json` flag to `cmd/exponential/show.go`. When set, output the issue as a
`jsonio.ShowOutput` JSON object. Extend `ShowOutput` with `Children` and `Archived`
fields to capture the richer data available from the CLI's `FindIssue` call.

## Why

Part of the unified JSON I/O epic (xpo-a4a271). The CLI `show` command uses `FindIssue`
which returns children and an archived flag — data the MCP `show` tool doesn't surface
because it uses `GetIssue`. The JSON output should include this richer data since it's
already available.

## Acceptance Criteria

1. `xpo show <id> --json` emits valid JSON matching extended `jsonio.ShowOutput` shape
2. `Children` field included as `[]IssueSummary` (omitted when empty)
3. `Archived` field included as `bool` (omitted when false)
4. MCP server behavior unchanged (it doesn't set these fields)
5. `make test` passes

## Flow

1. Add `Children []IssueSummary` and `Archived bool` fields to `jsonio.ShowOutput` with
   `omitempty` tags.
2. Add `--json` flag to `show.go`.
3. In the `showIssue` function: if `--json`, build `jsonio.ShowOutput` from the issue
   (reusing the same field mapping as the MCP handler), add children via
   `jsonio.ToIssueSummary()`, set `Archived`, encode to stdout, return early.

## Decisions

1. **Extend `ShowOutput`, not a separate type** — adding `omitempty` fields doesn't break
   the MCP contract (fields are absent when not set). One type serves both transports.
2. **Use `FindIssue` for JSON path** — consistent with the terminal output path. The extra
   data (children, archive status) is valuable for scripting.
