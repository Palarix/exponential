# Replace Timeline tabs with event-type filter menu

## What
Replace the three-button tab strip (All / Issues / Commits) in the Timeline view with a filter button that opens a checklist popover, matching the filter pattern used in Backlog and MyIssues.

## Why
- Tabs imply exclusive single-selection; event filtering is naturally multi-select
- The current `KindFilter` only supports two coarse categories (`issue_event` | `commit`); finer event types (comments, merges, artifacts, closures) are lumped together or not rendered at all
- A checklist popover scales to more event types without crowding the top bar
- Consistent with the established filter pattern across the app

## Event type categories

Derived from `TimelineEntry.kind`, `event_type`, and `payload.status`:

| Category | Key | Matches | Icon |
|----------|-----|---------|------|
| Issues | `issues` | kind=`issue_event`, event_type CREATE or UPDATE (non-terminal status + field changes) | `Plus` |
| Closed | `closed` | kind=`issue_event`, event_type=UPDATE with status in DONE, CANCELED, DUPLICATE | `Check` |
| Comments | `comments` | kind=`issue_event`, event_type=COMMENT | `MessageSquareMore` |
| Merges | `merges` | kind=`issue_event`, event_type=MERGE | `GitMerge` |
| Artifacts | `artifacts` | kind=`issue_event`, event_type=ARTIFACT | `Paperclip` |
| Commits | `commits` | kind=`commit` | `GitCommitVertical` |

## How

### 1. Add missing event rendering to Timeline

**ARTIFACT events:** `resolveIconKey` and `describeIssueEvent` have no ARTIFACT case.
- Add `"artifact"` to `ActIconKey`, with `Paperclip` icon
- ARTIFACT case in `resolveIconKey` → `"artifact"`
- ARTIFACT case in `describeIssueEvent` → `"{name} {action} {filename} on"`
- Entry in `CIRCLE_STYLE` and `SHARED_ICONS`

**CANCELED/DUPLICATE statuses:** `resolveIconKey` maps only DONE/DOING/BLOCKED/PLANNED/BACKLOG. Add:
- `"status-canceled"` and `"status-duplicate"` to `ActIconKey`
- Map CANCELED → `Ban` icon, DUPLICATE → `Copy` icon (or reuse existing)
- Corresponding verbs in `describeIssueEvent`: "canceled" / "marked as Duplicate"
- Entries in `CIRCLE_STYLE` and `SHARED_ICONS`

### 2. Replace filter state
- Remove: `KindFilter` type, `FILTER_OPTIONS` constant, `filter`/`setFilter` state
- Add: `EventCategory` type, `EVENT_CATEGORIES` constant array, `enabledTypes`/`setEnabledTypes` state as `Set<EventCategory>` — all enabled by default
- Always call `fetchTimeline(limit)` with no `kind` param (fetch everything, filter client-side). Removes re-fetch on filter change.

### 3. Client-side filtering
Categorize each entry:
- kind=`commit` → `"commits"`
- event_type=COMMENT → `"comments"`
- event_type=MERGE → `"merges"`
- event_type=ARTIFACT → `"artifacts"`
- event_type=UPDATE with payload.status in DONE/CANCELED/DUPLICATE → `"closed"`
- everything else (CREATE, other UPDATE) → `"issues"`

Apply `enabledTypes` filter after fetch, before the existing `person` filter.

### 4. EventTypeFilter popover (inline component)
Single-panel checklist popover anchored to the filter button:
- Header row with **All** / **None** text buttons
- Separator
- One row per category: checkbox + icon + label
- Close on outside click or Escape

### 5. HeaderBar changes
- Remove tab strip from `TopBar.left`
- Add filter button + PersonFilter to `TopBar.right`
- Filter button: 28×28px, funnel SVG (same as Backlog), `<Tooltip content="Filter">`
- Accent dot indicator when not all types selected

## Acceptance criteria
- [ ] Tab strip removed from Timeline header
- [ ] Filter button with funnel icon in `TopBar.right`
- [ ] Clicking opens checklist popover with 6 event categories
- [ ] All / None shortcuts work
- [ ] Multiple types toggled independently
- [ ] Indicator dot when filter active (not all types selected)
- [ ] Client-side filtering (no re-fetch on toggle)
- [ ] ARTIFACT events render in timeline (spec added, walkthrough written, etc.)
- [ ] CANCELED/DUPLICATE status transitions render with correct icons/verbs
- [ ] Person filter works alongside event-type filter
- [ ] Escape closes popover
- [ ] `make test` passes
