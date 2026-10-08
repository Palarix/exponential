# Walkthrough: Add `SearchInput` to My Issues TopBar

## What was built
My Issues now has the shared `SearchInput` (from xpo-ef6602) in its TopBar center slot. It filters the active tab (Assigned or Created) by title, ID or label. This was the last list view in epic xpo-570a07 without the TopBar search.

## How the pieces fit
1. **`matchesSearch(issue, query)` (`Backlog/filters.ts`)**
   - It's a pure predicate placed next to `matchesFilters`, because the two are always used together.
   - It trims and lowercases the query, and an empty query matches everything.
   - Otherwise it matches a substring of the title, ID or any label.
   - The parameter type is structural (`{ id, title, labels? }`), so it can be tested without a full `Issue`.
2. **Backlog** previously had this check written inline in its `filteredIssues` filter. It now calls `matchesSearch`.
   - The local `query` is now `search.trim().toLowerCase()`. That keeps `filtering` and `filterKey`, which drive the collapse state while searching, consistent with what the predicate considers active.
3. **My Issues**
   - Holds `search` as local state, so it survives switching tabs.
   - The `filtered` memo runs `matchesSearch && matchesFilters` only when a search or filter is active. Otherwise it returns `tabIssues` directly.
   - The count badge and the empty-state hint ("Try adjusting your filters or search.") both come from `filtered` and `searching`.
   - The now-unused local `applyFilters` was removed.
4. **Keyboard**: `SearchInput` owns `/` and `Esc`. My Issues passes `keyboardNavigationEnabled={!contextMenu}`, the same gate its `Tabs` use, so `/` doesn't fire while the context menu is open.

## Decisions
- **Labels are searched** even though the issue said "title or ID". One shared predicate keeps Backlog and My Issues consistent. Approved during review.
- **Backlog's query is trimmed now.** This is a small behavior change: before, `"bug "` matched nothing; now it matches "bug". It emerged during implementation and was approved during review.
- The match is a substring, not fuzzy, and the search isn't persisted. This is the same as Labels (xpo-c6043f).

## Acceptance criteria
- [x] The My Issues TopBar shows `SearchInput` in the center slot with placeholder "Filter issues...". Evidence: `MyIssues.tsx` `center={<SearchInput … />}`.
- [x] Typing filters the active tab by title, ID or label, case-insensitively. Evidence: `matchesSearch` tests for title, ID, label and case.
- [x] `/` focuses the search box and `Esc` clears it and blurs; `/` is disabled while the context menu is open. Evidence: `keyboardNavigationEnabled={!contextMenu}`; tested by the user.
- [x] The count badge and empty state reflect the search. Evidence: both are derived from `filtered` and `searching`; tested by the user.
- [x] Backlog search behaves the same after migrating. Evidence: same predicate semantics plus trimming; the user checked it.
- [x] `matchesSearch` has unit tests. Evidence: 7 tests in `filters.test.ts`, written first.
- [x] `make test` passes. Evidence: lint, 429 frontend tests and the Go suite are green.
