## What

Add empty state messaging to the Backlog component's filtered views (Backlog, Active, Done tabs) so users see helpful feedback instead of a blank area when no issues match.

## Why

Currently `Backlog.tsx` only shows an `EmptyState` when `issues.length === 0` (the entire project is empty). When a specific tab has no matching issues — either because no issues have that status, or because user-applied filters exclude everything — the view renders an empty area with just group headers (or nothing if `showEmptyGroups` is off). This is confusing.

## How

Insert a `filteredIssues.length === 0` check between the toolbar and the rows section in `Backlog.tsx`. This mirrors the pattern already used in `MyIssues.tsx` (line 186).

Two cases to distinguish:

1. **No issues match the tab's status filter** (no active filters applied) — show a tab-specific message:
   - Backlog tab: "No issues in the backlog" / "Issues with Backlog status will appear here."
   - Active tab: "No active issues" / "Issues that are Planned, In Progress, or Blocked will appear here."
   - Done tab: "No completed issues" / "Completed, canceled, and duplicate issues will appear here."
   - All tab: (already handled by the existing `issues.length === 0` check)

2. **User-applied filters exclude everything** (`hasActiveFilters(filters)` or `search` is non-empty) — show: "No matching issues" / "Try adjusting your filters or search."

Reuse the existing `EmptyState` component. Use an inline SVG icon consistent with the existing empty-state icons (the dashed-rectangle list style already used for the "No issues yet" state).

## Acceptance Criteria

- [ ] Switching to the Backlog tab when no issues are in BACKLOG status shows an empty state instead of blank
- [ ] Switching to the Active tab when no issues are PLANNED/DOING/BLOCKED shows an empty state
- [ ] Switching to the Done tab when no issues are DONE/CANCELED/DUPLICATE shows an empty state
- [ ] When user-applied filters or search exclude all issues on any tab, the empty state says to adjust filters
- [ ] The "All Issues" tab still shows the existing "No issues yet" state when the project is truly empty
- [ ] The `EmptyState` component is reused, no new component needed
- [ ] The "Press C to create an issue" hint remains visible in the empty state
