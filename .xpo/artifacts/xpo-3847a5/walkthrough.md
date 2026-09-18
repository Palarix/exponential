# Walkthrough: Refactor `add --json` to use shared MCP input schema

## What changed

One file: `cmd/exponential/add.go`.

The `--json` path already accepted the correct input schema (`AddInput`) via the
`inputs` backward-compat shim. This change completes the alignment by:

1. Importing `jsonio` directly instead of the `inputs` shim
2. Adding `client.ValidateCreatePayload()` for full validation parity with the MCP tool
3. Emitting structured `jsonio.AddOutput` JSON instead of plain text

## How the pieces fit

The CLI `--json` path now mirrors the MCP `add` handler exactly:

```
stdin → jsonio.DecodeStrict → AddInput.ToCreatePayload() → ValidateCreatePayload → AddIssue → AddOutput JSON
```

The MCP path does the same sequence, just with the SDK handling deserialization and
the response wrapping.

## Key decisions

**Removed manual estimate validation.** The old code had a standalone
`config.ValidateEstimate()` call. `ValidateCreatePayload` already calls this (plus
title length, description size, assignee format, parent/dep resolution, priority range,
and cycle validation), so the manual call was redundant.

**Kept duplicate check.** The `--force` / duplicate-check logic is CLI-specific — the
MCP tool doesn't have it. It stays as-is since it's outside the scope of schema alignment.

**No CLI-level tests added.** The project has no cobra command test harness, and all
underlying pieces (`DecodeStrict`, `ToCreatePayload`, `ValidateCreatePayload`, `AddOutput`)
are well-tested at the `jsonio`, `inputs`, and `mcpserver` layers. The CLI wiring is
three lines of glue.

## Output format

Before (plain text):
```
Created xpo-abc123: My issue title
  Labels: bug, feature
  Assignee: someone
```

After (structured JSON):
```json
{
  "id": "xpo-abc123",
  "title": "My issue title",
  "status": "BACKLOG"
}
```

This matches the `jsonio.AddOutput` schema returned by the MCP `add` tool.