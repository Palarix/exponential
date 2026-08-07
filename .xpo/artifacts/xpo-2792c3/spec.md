# MCP show tool does not reflect backlog sort order

## Root Cause

`SortIssues` in `internal/exponential/projection.go` sorts by `CreatedAt` instead of `SortOrder`. The web UI compensates by re-sorting client-side (via `sort.ts` manual mode), so drag-and-drop appears correct in the browser — but MCP tools and CLI consume the server's ordering directly.

## Fix

Change the two sort comparisons in `SortIssues`:
1. **Root sorting** (line 324-328): compare `SortOrder` lexicographically instead of `CreatedAt`
2. **Child sorting** (line 346): compare `SortOrder` lexicographically instead of `CreatedAt` (keeping the parent-first guard)

`backfillSortOrder` already ensures every issue has a non-empty `SortOrder` after projection, so no edge-case handling is needed.

## Acceptance Criteria

- [ ] `mcp__xpo__list` returns issues in `sort_order` (fractional-index) order
- [ ] Child issues still appear immediately after their parent
- [ ] Issues without explicit sort order (legacy data) sort correctly via backfill
- [ ] Existing tests pass