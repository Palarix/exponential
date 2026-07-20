# Spec: Board view column sorting consistency

## Problem

`buildContainers` in `web/src/components/Board/Board.tsx` sorts all issues as a single flat list via `sortGroup(issues, 'manual')` before splitting into status columns. The comparator is inconsistent when mixing DONE and non-DONE issues, producing non-deterministic within-column ordering that diverges from the Backlog view.

## Requirements

- Within each status column, the Board must show issues in the same order as the Backlog's default (manual) sort.
- Non-DONE columns: sorted by `sort_order` (fractional indexing key).
- DONE column: sorted by `updated_at` descending (most recent first).
- Each column is sorted independently — no cross-status interference.

## Design

Rewrite `buildContainers` to:

1. Group issues by status into per-column arrays.
2. Sort each column independently using `sortGroup`:
   - DONE → `sortGroup(columnIssues, 'updated')`
   - All others → `sortGroup(columnIssues, 'manual')`
3. Extract IDs from the sorted arrays.

## Files changed

- `web/src/components/Board/Board.tsx` — `buildContainers` function (lines 47-53)

## Acceptance criteria

- [ ] Board column ordering matches Backlog ordering for all statuses
- [ ] DONE column sorted by recency (`updated_at` descending)
- [ ] Non-DONE columns sorted by `sort_order`
- [ ] Drag-and-drop reordering within/across columns still works
- [ ] All existing tests pass
