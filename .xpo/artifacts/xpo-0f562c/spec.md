# Spec: Resolve issue IDs in MCP tool responses

## Problem

The MCP handlers for `update`, `spec`, `walkthrough`, and `artifact` return the raw user-provided issue ID in their response structs, broadcast calls, and text results. When a user passes a short/partial ID (e.g. `d662a2`), the response contains that partial string instead of the canonical `xpo-d662a2`.

## Correct Pattern

The `show` and `link` handlers already do this correctly — they call `c.GetIssue(in.ID)` early and use `issue.ID` throughout.

## Fix

For each affected handler, resolve the ID via `c.GetIssue()` before any operations, then use the resolved `issue.ID` in:
1. Response struct fields
2. `t.broadcast()` calls
3. Path construction strings
4. `textResult()` messages
5. Arguments to downstream methods (e.g. `c.UpdateIssue()`)

### Affected handlers

| Handler | Input field | Lines |
|---------|-------------|-------|
| `update` | `in.ID` | 383–404 |
| `spec` | `in.IssueID` | 510–549 |
| `walkthrough` | `in.IssueID` | 551–590 |
| `artifact` | `in.IssueID` | 592–651 |

### Bonus: `comment` and `start`

- `comment` (line 406–422): calls `GetIssue` but discards the resolved ID, still uses `in.ID` in responses.
- `start` (line 462–474): does not resolve ID at all, uses `in.ID` in response and broadcast.

These should also be fixed for consistency.

## Acceptance Criteria

- All six handlers (`update`, `comment`, `start`, `spec`, `walkthrough`, `artifact`) use the canonical resolved ID in all outputs.
- Existing tests pass.
- No behaviour change when a full canonical ID is provided (since `GetIssue` returns it as-is).
