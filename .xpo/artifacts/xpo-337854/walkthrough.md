# Walkthrough: Board column sorting fix

## What changed

**File:** `web/src/components/Board/Board.tsx` — `buildContainers` function (lines 47-60)

## The bug

The Board's `buildContainers` function sorted ALL issues together in one `sortGroup(issues, 'manual')` call, then split the sorted list into per-status columns. The `sortGroup` comparator with `'manual'` has a special case: when both issues are DONE, it compares by `updated_at`; otherwise it compares by `sort_order`. When sorting a mixed bag of DONE and non-DONE issues together, this creates a non-transitive comparator — the relative ordering of DONE issues among themselves could change depending on which non-DONE issues were interleaved during the sort. The result: unpredictable within-column ordering that differed from the Backlog view.

The Backlog avoids this entirely by grouping issues by status *before* sorting, then sorting within each group using a consistent comparator.

## The fix

`buildContainers` now follows the same pattern as the Backlog:

1. **Group first** — iterate issues into per-status arrays via a `Map<string, Issue[]>`.
2. **Sort per-column** — for each column, call `sortGroup` with the appropriate key:
   - `'updated'` for the DONE column (sort by recency)
   - `'manual'` for all other columns (sort by `sort_order` fractional indexing key)
3. **Extract IDs** — map the sorted issues to their IDs for the containers state.

Since each `sortGroup` call now operates on issues of a single status, the comparator is fully consistent — the DONE-vs-nonDONE special case never triggers. The within-column ordering now matches the Backlog exactly.

## What to watch for

- **Drag-and-drop reordering** still works because `buildContainers` is only called to initialize and sync state from props; during drag operations, container state is managed independently via `setContainers`.
- The fix does not add sort key selection to the Board UI — the Board always uses manual ordering. If sort key support is wanted later, `buildContainers` would need a `sortKey` parameter wired through from `App.tsx`.
