# Terminal statuses: CANCELED and DUPLICATE

## What
Add CANCELED and DUPLICATE as terminal statuses alongside DONE, using a single status dimension instead of a separate resolution field.

## Why
A single status dimension is simpler to filter, display, and query. The existing UI patterns (status popover, DnD, board columns, backlog groups) are data-driven and pick up new statuses naturally.

## Status model

**Terminal statuses:** DONE, CANCELED, DUPLICATE
**Active statuses:** BACKLOG, PLANNED, DOING, BLOCKED

Any active status can transition to any terminal status. Any terminal status can transition back to any active status (reopen).

### Helpers
- `IsTerminal(s)` — true for DONE, CANCELED, DUPLICATE
- `IsCompleted(s)` — true for DONE only (shipped work, for metrics)

## Icons

- DONE: filled circle with checkmark (green) — existing custom SVG
- CANCELED: Lucide `CircleMinus` (medium gray)
- DUPLICATE: Lucide `CirclePercent` (medium gray)

## Ordering

DONE appears before CANCELED and DUPLICATE in all status lists, columns, and groups.

## Semantic decisions

| Area | Behavior |
|------|----------|
| Blocker checks | All terminal statuses unblock |
| Auto-close parent | Triggers when all children are terminal; parent always goes to DONE |
| Merge | Always sets DONE |
| Start rejection | Rejects all terminal statuses |
| Velocity/burndown | Only counts DONE (IsCompleted) |
| Epic progress | "Delivered" = IsCompleted; "remaining" excludes IsTerminal |
| Cycle rollover | Skips all terminal |
| Staleness/bug-age | Excludes all terminal |

## Frontend features

### "Show empty groups" toggle
View Options dropdown includes a "Show empty groups" toggle under the Layout heading, persisted per tab in localStorage. Defaults to on for "All Issues" and "Done" tabs, off for others.

### Board View Options
Settings2 dropdown in the Board header with column visibility toggles. Each column can be shown/hidden independently, persisted in localStorage. Issue count reflects only visible columns.

## Acceptance criteria
- [x] CANCELED and DUPLICATE appear in status popovers
- [x] Status icons render correctly (Lucide CircleMinus / CirclePercent, medium gray)
- [x] Board shows CANCELED/DUPLICATE columns with visibility toggles
- [x] Done tab shows all terminal statuses
- [x] "Show empty groups" toggle in View Options
- [x] Auto-close parent triggers when all children are terminal
- [x] Blockers unblock on any terminal status
- [x] `xpo start` rejects all terminal statuses
- [x] Velocity/burndown only counts DONE
- [x] `xpo merge` always sets DONE
- [x] MCP `update` accepts CANCELED/DUPLICATE as status values
- [x] Board issue count matches visible columns
- [x] Board View Options dropdown with column toggles