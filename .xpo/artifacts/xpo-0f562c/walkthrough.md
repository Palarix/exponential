# Walkthrough: Resolve issue IDs in MCP tool responses

## Context

The MCP server exposes tool handlers in `internal/mcpserver/tools.go`. Each handler accepts a user-provided issue ID, performs an operation, and returns a response struct plus a text result. Several handlers were passing the raw user input straight through to responses and broadcasts without resolving it to the canonical form.

## The problem

`GetIssue()` supports prefix/substring matching — a user can pass `d662a2` and it resolves to `xpo-d662a2`. But the `update`, `comment`, `start`, `merge`, `spec`, `walkthrough`, and `artifact` handlers were using the raw input (`in.ID` or `in.IssueID`) in:

1. Response struct fields (returned to MCP clients)
2. `t.broadcast()` calls (sent to WebSocket subscribers)
3. Path strings for artifact storage
4. Arguments to downstream methods

This meant clients received inconsistent IDs depending on what the user typed.

## The fix

The pattern is identical across all handlers — resolve early, use the canonical ID everywhere:

```go
issue, err := c.GetIssue(in.ID)
if err != nil {
    return nil, outType{}, err
}
// From here, use issue.ID instead of in.ID
```

For `spec`, `walkthrough`, and `artifact` (which use `in.IssueID`), the resolved ID is stored in a local `issueID := issue.ID` variable for readability.

## Handlers modified

| Handler | Key change |
|---------|-----------|
| `update` | Added `GetIssue()` before `ValidateUpdatePayload` |
| `comment` | Was already calling `GetIssue()` but discarding the result (`if _, err := ...`); now captures the issue |
| `start` | Added `GetIssue()` before `StartWork()` |
| `merge` | Added `GetIssue()` before `MergeIssue()` |
| `spec` | Added `GetIssue()` after the empty-check, before the operation switch |
| `walkthrough` | Same as `spec` |
| `artifact` | Same as `spec` |

## Why not just fix the downstream methods?

The downstream methods (`UpdateIssue`, `WriteSpec`, etc.) already resolve the ID internally for their own filesystem operations. But they don't return the resolved ID to the caller. Adding a resolve at the handler level is the minimal, non-breaking change — it gives the handler the canonical ID for its response without changing any method signatures.

## Testing

All existing tests pass. The change is transparent when a full canonical ID is provided (since `GetIssue` returns it as-is). The behavioral change only manifests when a short/partial ID is used — responses now contain the full `xpo-` prefixed ID.