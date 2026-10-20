# Spec: Releases — Release filter in Issues, and open releases link to it

> **Read the spec of the parent epic xpo-4c92d4 first. It is authoritative** (see decision 2). The backend comes from xpo-f89e55. Read its spec for the `release` refs.

## What
1. A **Release** filter dimension in the Filter menu, with Current, Next, No release and every release. Current and Next resolve each time the filter runs.
2. Clicking an open release (Current, Next, Upcoming) in the Releases view opens Issues on the All Issues tab with that release filter, the way clicking a label in the Labels view does (`Labels.tsx:24`). Past releases keep opening their detail page (xpo-e6ff99).
3. **The Issues tabs share one set of filters.** A tab is a fixed status scope; the shared filters layer on top. Status options outside a tab's scope are disabled in the menu, greyed in the chip, and ignored on that tab.

## Why
You can focus on one release without the rest of the backlog, in the view that already has ordering, multi-select and shortcuts. The focus holds as you switch between All, Active, Backlog and Completed. Saved filters never need editing when a release completes. Custom views (xpo-6178dc) will later save "Release: Current" as a tab.

## Decisions copied from the epic
- Open releases have no detail page; they link to Issues with the release filter.
- The display name is the label, or `#7` without one (`Release.name`).
- A reopened issue in a RELEASED release is not in `@current`.

## Decisions for this story
1. **The filter is a structured dimension, not text.** Issues filters are a `BacklogFilters` object that the Filter menu edits. A new `releases: string[]` field holds refs: `@current`, `@next`, `none` and `#7`. Several values match any of them, like labels.
2. **Specific releases are stored as `#N`**, not as a label or `rel-` ID. Numbers are never reused and survive label edits. A deleted or unknown number matches nothing.
3. **`matchesFilters` takes the releases board as an optional third argument** and resolves `@current`/`@next`/`#N` to release IDs on every call. Without a board (still loading), those match nothing; `none` still works. A placeholder slot has no ID and matches nothing. Every view that calls `matchesFilters` passes the board (Issues, My Issues, Inbox, Releases).
4. **Menu options:** Current, Next, No release, then Upcoming releases, then Past releases. Current and Next read `Current · 1.5` and `Next · 1.6`. Counts are per resolved release, like the other dimensions.
5. **Chip:** `Release: Current, 1.6`. An unknown number shows as `#7`.
6. **The link from the Releases view opens Issues on All Issues** and adds the release filter to the shared filters, as the label link does. `NavIntent` gets an optional `tab`. On another tab (say Active) the release's Backlog and Done issues would be out of view.
7. **The link uses the slot ref for Current and Next** (`@current`, `@next`) and `#N` for Upcoming. A filter opened from Current then keeps following Current. Upcoming releases have no slot ref.
8. **Placeholders have no row in the Releases view**, so they never link.

### Shared filters across Issues tabs
9. **One filter set for all four Issues tabs**, saved under `exponential-backlog-filters`. This replaces the per-tab keys of xpo-0c0556. Old per-tab saved filters are dropped; there is no migration.
10. **A tab is a fixed status scope** (All: every status; Active: Planned, Doing, Blocked; Backlog: Backlog; Completed: Done, Canceled, Duplicate). Only the Status dimension can conflict with it; every other dimension applies on every tab.
11. **Effective status filter on a tab** = the selected statuses inside the tab's scope. If none of the selected statuses are in scope, the Status filter is ignored on that tab, and the tab shows all its issues. A filter never empties a tab by conflicting with it.
12. **Out-of-scope statuses are disabled in the menu.** They keep their checkmark and show the tooltip "Not on this tab", but can't be toggled; the chip's ✕ clears them. A tab whose scope has a single status (Backlog) disables the whole Status entry, because narrowing it does nothing. The Status chip greys the values that don't apply, with the same tooltip; when none apply, the whole chip is greyed.

| Tab | Status menu |
|---|---|
| All Issues | all 7 statuses |
| Active | Planned, Doing, Blocked; the rest disabled |
| Backlog | Status entry disabled |
| Completed | Done, Canceled, Duplicate; the rest disabled |

13. **View settings stay per tab:** layout (flat/nested), empty groups, done ghosts and expanded groups.
14. **My Issues is out of scope.** Its tabs (Assigned, Created) are not status scopes, and it keeps per-tab filters.
15. **The chip row starts with a text "Clear filters" button**, then the chips. Each chip keeps its own ✕. The old trailing ✕ icon for clearing everything looked like a second chip ✕, so it is gone. This applies everywhere `FilterChips` is used (Issues, My Issues, Inbox, Releases).
16. **The bottom status bar is flush with the main card**, left and right (no side padding in `Layout.tsx`). "Clear filters" lines up with the card's left border, and the version text with its right border.

## Flow
1. Tests first in `filters.test.ts`: `releases` in `matchesFilters` (each ref, multi-value, no board, placeholder, deleted number, reopened issue in a RELEASED release vs `@current`), `hasActiveFilters`, `getSelected`/`setSelected`, `parseStoredFilters` filling `releases` for old saved filters, and the chip label.
2. Tests first for the link ref of a release (`@current`, `@next`, `#N`) and for routing (open → Issues, past → detail).
3. Add `releases` to `BacklogFilters`, `EMPTY_FILTERS`, `hasActiveFilters`, `FilterDimension`, `getSelected`, `setSelected` and `matchesFilters` in `web/src/components/Backlog/filters.ts`.
4. Add the Release dimension to `FilterMenu.tsx` and the chip to `FilterChips.tsx`. Both read `useReleases()`.
5. Pass the board to `matchesFilters` in Backlog, MyIssues, Inbox and Releases.
6. Add `tab` to `NavIntent` (`web/src/app/contexts.ts`). Backlog switches to that tab and merges the intent's filters into the shared filters.
7. In `Releases.tsx`, open releases call `navigate("backlog", { tab: "all", filters: { releases: [ref] } })`. Past releases keep `setReleaseNumber`.
8. Tests first for `filtersForScope(filters, statuses)` (in-scope statuses kept, out-of-scope dropped, all out of scope → Status ignored, other dimensions untouched), `outOfScope` and `dimensionDisabled`.
9. Backlog: one `exponential-backlog-filters` key; apply `filtersForScope(filters, tab statuses)` before `matchesFilters`.
10. `FilterMenu` and `FilterChips` take an optional `scope` (the tab's statuses). The menu disables out-of-scope status options and, for a single-status scope, the Status entry; the chip greys out-of-scope values. Only the Backlog passes it.
11. `FilterChips`: a text "Clear filters" button first, then the chips; drop the trailing clear-all icon.
12. `Layout.tsx`: drop the status bar's side padding.
13. Run `make test`.

## Edge cases
- **MEDIUM:** A saved filter has `@current` and Current is completed. The filter shows the new Current after the refetch, with no edit.
- **MEDIUM:** Status = Done is set on Completed, then the user switches to Active. Active ignores the Status filter and shows all its issues; the chip is greyed and its ✕ clears it.
- **LOW:** A saved filter names a deleted release (`#7`). It matches nothing; the chip shows `#7`.
- **LOW:** The releases query has not loaded yet. `@current`, `@next` and `#N` match nothing until it loads.
- **LOW:** The Issues view already has other filters. The link adds the release filter to them, as the label link does.

## Acceptance criteria
- [ ] The Filter menu has a Release dimension with Current, Next, No release and each release.
- [ ] `@current` and `@next` follow Current and Next as releases complete or are reordered.
- [ ] `none` shows issues without a release; `#N` shows that release's issues; a deleted number matches nothing.
- [ ] Clicking Current, Next or an Upcoming release in the Releases view opens Issues on All Issues with that release filter. Past releases open their detail page.
- [ ] Filters hold across All, Active, Backlog and Completed.
- [ ] Status options outside a tab's scope are disabled in the menu (the whole entry on Backlog), greyed in the chip, and ignored on that tab; a conflicting Status filter never empties a tab.
- [ ] The chip row starts with a text "Clear filters" button; there is no second ✕ after the chips.
- [ ] The bottom status bar is flush with the main card on both sides.
- [ ] `make test` passes.
