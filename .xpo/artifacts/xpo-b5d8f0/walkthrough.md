# Walkthrough: Add `--json` output to `show` command

## What changed

Two files:

### `internal/jsonio/output.go`

Added two fields to `ShowOutput`:
- `Children []IssueSummary` (`json:"children,omitempty"`) — sub-issues of the shown issue
- `Archived bool` (`json:"archived,omitempty"`) — whether the issue was found in the archive

Both use `omitempty`, so the MCP server's output is unchanged (it never sets these fields).

### `cmd/exponential/show.go`

Added `showJSONFlag` and `--json` flag registration. When set, the command builds a
`jsonio.ShowOutput` from the `FindIssue` result (which returns the issue, its children,
and an archived flag), converts children via `jsonio.ToIssueSummary()`, and encodes to
stdout with pretty-printed JSON.

The CLI uses `FindIssue` (not `GetIssue`) for the JSON path, matching the terminal output
path. This provides richer data than the MCP tool — children and archive status — which
is valuable for scripting use cases.

## Design decision

Extended `ShowOutput` rather than creating a separate CLI-specific type. The `omitempty`
tags ensure backward compatibility: MCP clients see no difference, CLI consumers get the
extra fields when present.
