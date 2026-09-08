# Walkthrough: Replace Timeline tabs with event-type filter menu

## What changed

The Timeline view's three-button tab strip (All / Issues / Commits) was replaced with a multi-select checklist filter popover, and several previously unrendered event types were added.

**Single file changed:** `web/src/components/Timeline/Timeline.tsx`

## How it works

### Event categorization

A new `categorizeEntry()` function maps each `TimelineEntry` to one of six categories:

| Category | Matches |
|----------|---------|
| Issues | CREATE and non-terminal UPDATE events |
| Closed | UPDATE events where `payload.status` is DONE, CANCELED, or DUPLICATE |
| Comments | COMMENT events |
| Merges | MERGE events |
| Artifacts | ARTIFACT events (specs, walkthroughs, uploaded artifacts) |
| Commits | Entries with `kind === "commit"` |

The "Closed" category uses `TERMINAL_STATUSES` (a `Set` of the three terminal status strings) to distinguish close events from regular issue updates.

### Filter state

The old `filter: KindFilter` (single-select enum passed to the API as `kind`) was replaced with `enabledTypes: Set<EventCategory>` — all six categories enabled by default. Filtering moved from server-side (the `kind` query parameter) to client-side: `fetchTimeline(limit)` always fetches everything, then `entries` is derived by filtering `rawEntries` through `enabledTypes` before applying the existing person filter. This makes toggling filters instant with no network round-trip.

### EventTypeFilter component

A new inline component renders via `createPortal` as a fixed-position popover anchored right-aligned below the filter button. It contains:

- A header row with **All** / **None** shortcut buttons
- A separator
- Six checkbox rows, one per category, each with an icon and label

The popover closes on outside click or Escape. Checkbox styling matches the `SubMenu` pattern from `Backlog/FilterMenu.tsx`.

### HeaderBar changes

- `TopBar.left` now shows a "Timeline" breadcrumb label (matching Dashboard's "Overview", Inbox's "Notifications")
- `TopBar.right` holds the filter button (28×28 funnel icon with accent dot indicator when filters are active) and the existing `PersonFilter`
- The tab strip is completely removed

### New event rendering

**ARTIFACT events** — Previously fell through `resolveIconKey` and returned `null`, making them invisible. Now handled with a `Paperclip` icon and description format: "{name} {action} {filename} on {issue}".

**CANCELED/DUPLICATE statuses** — Previously fell through to the `"status-backlog"` default icon. Now have dedicated `ActIconKey` entries (`"status-canceled"` with `Ban` icon, `"status-duplicate"` with `Copy` icon) and correct verbs in `describeIssueEvent` ("canceled" / "marked as Duplicate").

### Empty state

When filters produce zero matching entries (but raw data exists), an `EmptyState` with "No matching activity" is shown instead of a blank page. The header bar remains visible so the user can adjust filters.

## Key decisions

- **Client-side filtering over server-side**: The dataset is small (100–200 entries at a time) and the UX benefit of instant toggling outweighs fetching less data. The `kind` parameter is still available on the API if needed in the future.
- **Single-panel popover** (not the two-panel dimension menu from Backlog): Timeline filtering has only one dimension (event type), so the sub-menu pattern would add unnecessary complexity.
- **Six categories** rather than the original two (issue_event/commit): Provides meaningful granularity — "what closed today" and "what specs were written" are common questions the old tabs couldn't answer.
