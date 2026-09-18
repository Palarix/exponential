# Walkthrough: Add SearchInput to Timeline TopBar

## What was built

A search input in the Timeline view's TopBar that filters displayed events by issue title or issue ID. This follows the same integration pattern established in Backlog (xpo-ef6602) and Dependencies.

## How the pieces fit together

### Filtering logic — `filterBySearch` in `timeline-utils.ts`

A pure function that takes an array of `TimelineEntry` and a query string. It trims and lowercases the query, then filters entries where `issue_id` or `issue_title` contains the query as a substring. Both fields use optional chaining — the server omits `issue_id` for commits not linked to an issue, so a bare `.toLowerCase()` call would crash at runtime.

The function slots into the existing `entries` useMemo chain: type filter → person filter → search filter. Empty/whitespace queries skip filtering entirely.

### UI wiring — `Timeline.tsx`

- Added `search` state (`useState("")`) in the `Timeline` component.
- Passed `search` and `onSearchChange` to the private `HeaderBar` sub-component.
- `HeaderBar` renders `<SearchInput>` in the TopBar's `center` slot (previously unused) with placeholder "Search timeline...". No `keyboardNavigationEnabled` override is needed — Timeline's popovers (EventTypeFilter, PersonFilter) close on outside click or Escape before the `/` shortcut fires.

### Pagination fix

The original code used `entries.length` for two pagination decisions: whether the last day group's timeline line terminates (`isLast`) and whether to show "Load more". After adding search filtering, `entries` can be much smaller than `rawEntries`, which would incorrectly hide the pagination control. Both checks now use `rawEntries.length`.

Additionally, the "No matching activity" empty state (which returns early before the "Load more" button renders) now passes `actionLabel="Load more"` and `onAction={handleLoadMore}` to `EmptyState` when `rawEntries.length >= limit`, so users can still fetch older events when the current batch has no matches.

## Key decisions

- **Match fields**: Only `issue_id` and `issue_title` are matched, per the issue description. Commit messages, authors, and branch names are not searched — this keeps the behavior focused and consistent with how Backlog/Dependencies search works (matching issue-level fields only).
- **No `keyboardNavigationEnabled` prop**: Timeline's overlays (EventTypeFilter, PersonFilter) already register their own Escape handlers at `overlay` priority and close before the search shortcut's `control` priority fires, so no coordination is needed.

## Acceptance criteria

- [x] `SearchInput` appears in Timeline TopBar center slot
- [x] Typing filters displayed events by issue title (case-insensitive substring match)
- [x] Typing filters displayed events by issue ID (case-insensitive substring match)
- [x] `/` focuses the search input, `Escape` clears and blurs
- [x] Empty search shows all events (respecting existing type/person filters)
- [x] Existing event-type and person filters continue to work alongside search
- [x] Unlinked commits (no issue_id) don't crash when searching
- [x] "Load more" remains accessible when search narrows or empties results
- [x] Tests pass — 365 frontend tests (8 new), Go suite, lint all green
