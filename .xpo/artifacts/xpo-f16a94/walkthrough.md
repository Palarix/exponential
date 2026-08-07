# Walkthrough: Cap rendered issue count in Board and Backlog views

## What changed

Two frontend files were modified to limit DOM nodes rendered per group/column.

### 1. `web/src/components/Board/BoardColumn.tsx`

The DONE column already had a render cap pattern: a `DONE_VISIBLE_COUNT = 5` constant, a `showAll` state, a `shouldTruncate` guard, a `visibleIds` slice, and a "+N more" button. The change generalizes this to all columns.

**Added:** `COLUMN_VISIBLE_COUNT = 50` constant.

**Changed:** The `shouldTruncate` logic. Previously it was `isDone && itemIds.length > DONE_VISIBLE_COUNT && !showAll`. Now it computes a `cap` per column (`DONE_VISIBLE_COUNT` for DONE, `COLUMN_VISIBLE_COUNT` for everything else) and uses `itemIds.length > cap && !showAll`. The `visibleIds` and `hiddenCount` derivations use `cap` instead of the hardcoded DONE constant.

Everything else — the `showAll` state, the `SortableContext` receiving `visibleIds`, the "+N more" button, the column header showing `itemIds.length` (full count) — remains unchanged.

### 2. `web/src/components/Backlog/Backlog.tsx`

No render cap existed. Added one per status group section.

**Added:** `GROUP_VISIBLE_COUNT = 100` constant near the top of the file.

**Added:** `showAllGroups` state — a `Set<string>` tracking which status groups the user has expanded past the cap.

**Changed:** The `issueRows.map()` call inside the sections rendering loop. It's now wrapped in an IIFE that:
1. Checks if the group's row count exceeds `GROUP_VISIBLE_COUNT` and the group isn't in `showAllGroups`
2. Slices `issueRows` to the first 100 if capped
3. Renders the sliced rows via the existing `.map()` call
4. Appends a "+N more" button when capped, styled with the same muted-text pattern used elsewhere in the toolbar

The "+N more" button's click handler adds the group's status to `showAllGroups`, which triggers a re-render showing all rows.

### Why the stats stay correct

The render cap is purely a DOM optimization. All counts and aggregations are computed upstream:

- **Board column header** displays `itemIds.length` — the full unsliced array
- **Backlog group header** displays `groupRow.count` and `groupRow.storyPoints`, both computed in `useBacklogRows` from the full `filteredIssues` array before any rendering happens

The cap only affects which items are passed to `.map()` in the JSX. Filtering, sorting, searching, and all statistical aggregations operate on the complete in-memory dataset.

### Why original indices are preserved

The Backlog uses `data-row={i}` indices for keyboard navigation, focus tracking, popover anchoring, and DnD drop indicators. The `index` values come from the position in the full `rows` array (assigned in the sections-building loop at the top of the IIFE). Slicing `issueRows` removes entries from the rendered output but doesn't renumber the remaining ones — each visible row keeps its original index from the `rows` array.