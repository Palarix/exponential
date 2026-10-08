# Add `SearchInput` to My Issues TopBar

## What
Put the shared `SearchInput` (from xpo-ef6602) in the My Issues `TopBar` `center` slot. It filters the active tab's issues (Assigned or Created).

## Why
This is part of the WebUI refactor epic xpo-570a07. It's the last list view without the TopBar search, after Backlog, Dependencies, Timeline and Labels (xpo-c6043f).

## How
- Add a pure `matchesSearch(issue, query)` to `web/src/components/Backlog/filters.ts`, next to `matchesFilters`.
  - It trims and lowercases the query and returns true for an empty query.
  - Otherwise it matches a substring of the title, ID or any label.
  - These are the same semantics as Backlog's current inline check.
- Migrate Backlog's inline title/ID/label check to `matchesSearch`. Behavior stays the same; it removes duplication.
- In `MyIssues.tsx`:
  - Add a `search` state.
  - `filtered` = tab issues → `matchesSearch` → `matchesFilters`.
  - `center={<SearchInput value={search} onChange={setSearch} placeholder="Filter issues..." keyboardNavigationEnabled={!contextMenu} />}`.
  - `CountBadge` reflects the searched count, because it already uses `filtered`.
- Empty state: when a search or filter is active, the description reads "Try adjusting your filters or search." (Backlog's wording).

## Acceptance criteria
- [ ] The My Issues TopBar shows `SearchInput` in the center slot with placeholder "Filter issues..."
- [ ] Typing filters the active tab by title, ID or label, case-insensitively
- [ ] `/` focuses the search box and `Esc` clears it and blurs; `/` is disabled while the context menu is open
- [ ] The count badge and empty state reflect the search
- [ ] Backlog search behaves the same after migrating to `matchesSearch`
- [ ] `matchesSearch` has unit tests (empty or whitespace query, title, ID, label, case, no match)
- [ ] `make test` passes

## Decisions
1. **Labels are searched too**, though the issue says "title or ID". This keeps it consistent with Backlog through a single shared helper. Revisit if label matches turn out to be noisy.
2. The search is a substring match, not fuzzy, and isn't persisted. It survives switching tabs within the view. This matches the Labels decision.
3. The placeholder is "Filter issues...", the same as Backlog.
