## What was built

Added empty state messaging to the Backlog component's filtered views so users see helpful feedback instead of a blank area when no issues match a tab or filter combination.

## How it works

A single conditional gate was added to `web/src/components/Backlog/Backlog.tsx`, inserted between the toolbar and the rows/DndContext section. When `filteredIssues.length === 0` (but the project does have issues — the existing `issues.length === 0` check higher up already handles the truly-empty project case), the component renders an `EmptyState` instead of the row list.

The messaging distinguishes two scenarios:

1. **No issues match the tab's status filter** (no user-applied filters or search active):
   - Backlog: "No issues in the backlog" / "Issues with Backlog status will appear here."
   - Active: "No active issues" / "Issues that are Planned, In Progress, or Blocked will appear here."
   - Done: "No completed issues" / "Completed, canceled, and duplicate issues will appear here."

2. **User-applied filters or search exclude everything** (`hasActiveFilters(filters) || search`):
   - "No matching issues" / "Try adjusting your filters or search."

The "All Issues" tab is unaffected — when `issues.length === 0` the existing empty state at line 1267 fires first; when issues exist but all are filtered out, the new gate catches it with a generic fallback.

## Key decisions

- **Reused `EmptyState` component** with the same dashed-rectangle SVG icon already used for the "No issues yet" state, keeping visual consistency.
- **Followed the `MyIssues.tsx` pattern** (line 186) — same conditional structure with `hasActiveFilters()` to switch between filter-aware and default messaging.
- **Wrapped DndContext, context menu, and modal** inside a fragment within the ternary so the empty state fully replaces the interactive row area — no orphaned DnD listeners or context menus when the list is empty.
