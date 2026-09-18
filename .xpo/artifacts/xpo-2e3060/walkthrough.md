# Walkthrough: Add `--json` output to `list` command

## What changed

One file: `cmd/exponential/list.go`.

## How it works

Added a `listJSONFlag` bool variable and `--json` flag registration. When set, the command
builds a `jsonio.ListOutput` by converting each filtered `*model.Issue` via
`jsonio.ToIssueSummary()`, then encodes it to stdout with `json.NewEncoder` + `SetIndent("", "  ")`.

The JSON output path runs *before* the "No issues found" check, so empty results produce
`{"issues": []}` — correct for machine-readable output. All existing filter flags
(`--status`, `--label`, `--assignee`, `--mine`, `--since`, `--before`, `--cycle`,
`--archived`, `--all`, `--match`, `--parent`) continue to work because filtering happens
in `client.ListIssues(opts)` before the output format branch.

Follows the same pattern as `rationale --json` and `pulse --json` (pretty-printed JSON
to stdout with early return).

## Key decisions

- **JSON before empty check**: scripts should get a valid JSON envelope even when no issues
  match, not a text message they'd have to parse.
- **Pretty-printed**: consistent with other `--json` commands in the CLI.
