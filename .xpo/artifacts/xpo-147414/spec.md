# Relationship Links in Issue Details View Broken

## Problem

Two bugs:

1. **Dependency target IDs stored without prefix** — `LinksToDependencies()` in `internal/inputs/inputs.go` stores the user-provided target ID directly without resolving it via `GetIssue()`. When agents pass short IDs (e.g. `d662a2`), the dependency is stored without the `xpo-` prefix. The PropertySidebar can't find the target issue in its lookup (`issues.find(t => t.id === dep.target_id)`), so it falls back to showing the raw (prefixless) ID, and the link points to `#/issues/d662a2` which doesn't match any issue.

2. **No 404 page** — navigating to `#/issues/<invalid-id>` silently shows the Backlog view with no error message.

## Fix 1: Resolve dependency target IDs

`LinksToDependencies` can't resolve IDs itself (it has no access to the client/transport). The fix goes in the callers that create issues with links:

- **MCP `add` tool** (`internal/mcpserver/tools.go`) — after calling `ToCreatePayload`, resolve each dependency's `TargetID` via `c.GetIssue()` before calling `c.AddIssue()`.
- **MCP `update` tool** — same resolution for `ToUpdatePayload` dependencies.

This matches what the `link` tool already does correctly.

Additionally, fix existing broken data: the PropertySidebar should attempt a fuzzy match when an exact `dep.target_id` match fails — try matching by suffix (the hash portion). This handles old events with prefixless IDs without requiring a data migration.

## Fix 2: Issue not found page

When `selectedIssueId` is set but `selectedIssue` is null (no matching issue), render a "not found" state instead of silently falling through to the Backlog view. Show:
- A message like "Issue not found"
- The ID that was searched for
- A link back to the Backlog

## Acceptance Criteria

- [ ] Creating an issue with links via MCP `add` stores full `xpo-` prefixed target IDs.
- [ ] Updating an issue's dependencies via MCP `update` stores full target IDs.
- [ ] PropertySidebar shows correct titles and working links for dependencies (including old prefixless data).
- [ ] Navigating to `#/issues/<invalid-id>` shows a "not found" page with a link back.
