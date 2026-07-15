# MCP link tool allows self-links and duplicates

## Problem

The MCP `link` tool resolves source and target IDs but does not check:
1. Whether source and target are the same issue (self-link).
2. Whether an identical link (same target + kind) already exists on the source issue.

## Fix

In the `link` tool handler (`internal/mcpserver/tools.go`), after resolving both IDs:
1. Reject if `src.ID == tgt.ID` with an error.
2. Check `src.Dependencies` for an existing entry with the same `TargetID` and `Kind`. Reject if found.

## Acceptance Criteria

- [ ] Linking an issue to itself returns an error.
- [ ] Adding a duplicate link (same target and kind) returns an error.
- [ ] Different link kinds to the same target are allowed (e.g. `blocks` and `relates_to`).
- [ ] Tests pass.
