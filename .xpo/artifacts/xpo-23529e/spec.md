# Spec: Add SearchInput to Timeline TopBar

## What

Add the shared `SearchInput` component to the Timeline view's TopBar center slot, enabling users to filter displayed timeline events by title or issue ID.

## Why

The Timeline is the only major view with a TopBar that lacks a search input. Users scanning the timeline for a specific issue's events currently have to scroll through everything manually.

## How

### 1. Add `filterBySearch` to `timeline-utils.ts`

Pure function: takes entries and a query string, returns entries matching lowercased query against `issue_id` and `issue_title`. Both fields use optional chaining — `issue_id` may be undefined for unlinked commits.

### 2. Add search state to `Timeline`

Add `const [search, setSearch] = useState("")` alongside the existing filter state. Chain `filterBySearch` into the `entries` useMemo after type/person filters.

### 3. Wire SearchInput into `HeaderBar`

Pass `search` and `onSearchChange` as props to `HeaderBar`. Render `SearchInput` in the TopBar's `center` slot with placeholder "Search timeline...".

### 4. Pagination uses `rawEntries.length`

Both pagination controls (`isLast` and "Load more" visibility) use `rawEntries.length` instead of `entries.length`, so the "Load more" button remains visible when search narrows results.

The "No matching activity" empty state shows a "Load more" button via `EmptyState`'s `actionLabel`/`onAction` props when `rawEntries.length >= limit`, so users can fetch older events to find matches.

### 5. Tests

Tests for `filterBySearch` in `timeline-utils.test.ts`:
- Empty query returns all entries
- Whitespace-only query returns all entries
- Partial title match
- Issue ID match
- Case-insensitive matching
- No-match returns empty
- Missing `issue_title` handled
- Missing `issue_id` (unlinked commits) handled without crash

## Acceptance criteria

- [x] `SearchInput` appears in Timeline TopBar center slot
- [x] Typing filters displayed events by issue title (case-insensitive substring match)
- [x] Typing filters displayed events by issue ID (case-insensitive substring match)
- [x] `/` focuses the search input, `Escape` clears and blurs
- [x] Empty search shows all events (respecting existing type/person filters)
- [x] Existing event-type and person filters continue to work alongside search
- [x] Unlinked commits (no issue_id) don't crash when searching
- [x] "Load more" remains accessible when search narrows or empties results
- [x] Tests pass
