# Walkthrough: Release filter in Issues, and open releases link to it

## What was built and why

The story started as "Current/Next tabs in Issues". During review we went the way labels already work instead. A **Release** filter narrows any view to a release, and clicking an open release in the Releases view opens Issues with that filter on. Saved release tabs come later from custom views (xpo-6178dc). The epic spec (xpo-4c92d4, decision 2) records this change of direction. It also gave the open-release actions their own story (xpo-88f633), limited the detail page to past releases (xpo-e6ff99), and added a Dashboard section (xpo-574cf5).

While trying it out we found that filters were saved per tab. A release focus set on All Issues was gone on Completed. So the Issues tabs now share one filter set, and each tab applies it within its own statuses.

## How the pieces fit

### The filter (`web/src/components/Backlog/filters.ts`)

- `BacklogFilters.releases: string[]` holds **refs**, not IDs: `@current`, `@next`, `none` and `#7`.
  - A specific release is stored by number because numbers are never reused and survive a label edit.
  - Current and Next are stored as slots, so a saved filter follows them when a release completes.
- `resolveReleaseRef(ref, board)` turns a ref into a release ID, `""` for `none`, or `null` for "matches nothing" (unknown number, placeholder slot, board not loaded yet).
- `matchesFilters(issue, filters, board?)` resolves the refs **every time it runs**; nothing is cached. The board comes from `useReleases()`, so after a release completes, the refetch alone moves `@current` along.
- A reopened issue in a past release isn't in `@current`, because its `release_id` is the past release's ID. This needs no special case.
- `releaseFilterOptions` builds the menu: Current · name, Next · name, No release, upcoming, then past releases, with counts. `releaseRefName` names a ref in the chip.

### Shared filters and tab scope

- The Backlog keeps one key, `exponential-backlog-filters`. This replaces the per-tab keys of xpo-0c0556. Old per-tab entries are just ignored, with no migration.
- A tab is a fixed status scope. `filtersForScope(filters, tabStatuses)` keeps only the selected statuses inside that scope. If none are inside, the Status filter is dropped for that tab, so a filter never empties a tab by conflicting with it. Only Status can conflict; the other dimensions apply everywhere.
- The Backlog matches against `tabFilters`, the scoped copy. The empty state and the key for nodes collapsed while filtering use it too, so they reflect what the tab really applies.
- `FilterButton`, `FilterMenu` and `FilterChips` take an optional `scope`. Only the Backlog passes it, so the other views behave as before.
  - `outOfScope` disables out-of-scope status options. They keep their checkmark, and the chip's ✕ clears them.
  - `dimensionDisabled` disables the whole Status entry on a one-status tab (Backlog).
  - The chip greys out-of-scope values. If none apply, the whole chip is greyed instead, so they aren't dimmed twice.
- View settings stay per tab: flat/nested layout, empty groups, done ghosts and expanded groups.

### The link (`Releases.tsx`, `releases-utils.ts`, `backlog-intent.ts`)

- `releaseLink(release)` decides where a click goes:
  - A past release opens `{ detail }`.
  - Current and Next open `{ filter: "@current" | "@next" }`.
  - An upcoming release opens `{ filter: "#N" }`.
- `NavIntent` gained an optional `tab`. `intentTarget` merges the intent's filters into the shared set and picks the tab. The release link opens All Issues, because on Active the release's Backlog and Done issues would be out of view.

### Polish from review

- `FilterChips` starts with a text **Clear filters** button. The trailing ✕ icon looked like a second chip ✕, so it is gone.
- The status bar in `Layout.tsx` has no side padding any more, so it lines up with the main card on both sides.

### Bug found on the way: xpo-a71a45

`Menu.handleItemMouseEnter` opened a submenu on hover without checking `disabled`, so the disabled Status entry still opened. Click and keyboard already skipped disabled items. `opensOnHover({ isSubMenu, navigable })` in `menu-utils.ts` now controls both hover paths, the direct one and the delayed safe-triangle one.

## Non-obvious things

- The filter menu's counts are computed over the issues the tab shows, so on Completed the release counts include only completed issues. The other dimensions already worked this way.
- Without the releases board (first load), `@current`, `@next` and `#N` match nothing for a moment, and `none` works.
- My Issues keeps per-tab filters on purpose. Its tabs (Assigned, Created) are not status scopes.

## Acceptance criteria

- [x] The Filter menu has a Release dimension with Current, Next, No release and each release. Evidence: `releaseFilterOptions` tests; tried on the demo board.
- [x] `@current` and `@next` follow Current and Next as releases complete or are reordered. Evidence: the "follows Current when it changes" test; refs resolve on every `matchesFilters` call.
- [x] `none` shows issues without a release; `#N` shows that release's issues; a deleted number matches nothing. Evidence: tests in `filters.test.ts`.
- [x] Clicking Current, Next or an upcoming release opens Issues on All Issues with that filter; past releases open their detail page. Evidence: `releaseLink` and `intentTarget` tests; tried on the demo board.
- [x] Filters hold across All, Active, Backlog and Completed. Evidence: a single `exponential-backlog-filters` key; confirmed by the user on the demo board.
- [x] Status options outside a tab's scope are disabled in the menu (the whole entry on Backlog), greyed in the chip, and ignored on that tab; a conflicting Status filter never empties a tab. Evidence: `filtersForScope`, `outOfScope`, `dimensionDisabled` and `opensOnHover` tests; confirmed by the user.
- [x] The chip row starts with a text "Clear filters" button, with no second ✕. Evidence: confirmed by the user on the demo board.
- [x] The bottom status bar is flush with the main card on both sides. Evidence: confirmed by the user.
- [x] `make test` passes: lint, 1398 frontend tests and the Go suite.
