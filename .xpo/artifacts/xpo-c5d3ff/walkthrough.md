# Walkthrough: Filter menu counts, the new Milestone menu, and nested submenus

## What was built and why

The issue reported two faults in Issues → Filter → Milestone: every count showed 0, and the order and labels were wrong. Review widened the work in two steps. The separators were too faint to see. Then the user compared five layouts on a temporary tophat page and chose submenus for Upcoming and Past. No menu in the app had nested a submenu before, so the menu primitives needed work too.

## How the pieces fit

### Counts (`filters.ts`, `FilterMenu.tsx`, `Backlog.tsx`)

The Backlog gave `FilterButton` the list the filters had already narrowed. With Current checked, only issues in Current remained, so every other milestone counted 0. The same cause hit other dimensions: Status counted 0 for unchecked values, and Labels, Assignee and Epic lost rows.

- `facetIssues(issues, dim, filters, board, scope)` applies every filter except `dim`'s own. A count now tells you what checking that row gives you.
- `FilterMenu` calls it once per dimension, so it needs the list before filters. `Backlog.tsx` now splits out `tabIssues` (the tab's statuses plus the search) for this, and `filteredIssues` builds on it.
- My Issues, Inbox and Milestones already passed an unfiltered list, so they need no change. Their counts do change: a filter in one dimension now narrows the others, as the user chose.

### The Milestone menu (`filters.ts`, `FilterMenu.tsx`)

- `milestoneFilterOptions` returns a flat list. `slot` marks Current and Next, and `group` marks Upcoming and Past.
- `FilterMenu` renders rows without a `group`, a divider, then one `SubMenu` per group that has rows.
- The flat list is deliberate: search needs all rows at once. With text in the search box, `FilterMenu` ignores the groups.
- `board.upcoming` is already nearest first and `board.past` newest first. The Milestones view reverses `upcoming` for display, but the menu needs no reversal.
- `filterOptionMatches` checks the label and the slot. The labels no longer contain "Current" or "Next", and users can still type them.

### Separators (`Menu.tsx`)

`MenuDivider` used `--color-border-subtle` (`#1e1e1e`) on a `#1c1c1c` surface. It now uses `--color-tint-border`, the wash of the text color from xpo-aaf8b1, so it shows on any surface. `MenuFilter` had its own copy of the line and now renders `MenuDivider`. This changes every menu in the app.

### Nested submenus (`Menu.tsx`, `SubMenu.tsx`, `popover-utils.ts`)

`Menu` already passes a `SubMenu` through at any depth. These parts assumed one open flyout:

- **Outside clicks.** `FilterMenu` checked the first flyout, so a click in the third level closed the menu. It now uses `closest()`.
- **Which flyout is mine.** Each `Menu` has a `useId` and passes it as `_ownerId`. `SubMenu` writes it to `data-submenu-owner`, and `Menu` looks up its flyout by that ID.
- **A stale open index.** `Menu` stores the open submenu by row position. A search replaces the rows, so `Menu` clears the index during render when it no longer points at a submenu. While the index stays set, `Menu` ignores most keys.
- **Direction.** Each flyout records its side in `data-submenu-side`, and a nested flyout opens on its parent's side. `left-start` is a new placement, and it flips right when the left has no room.

### Moves toward a flyout (`Menu.tsx`, `menu-utils.ts`)

Upcoming and Past always sit low in the second level. A straight move from Milestone to Upcoming crossed the Epic row, and the old code switched to any submenu row at once.

- `movingTowardFlyout` returns true for a move toward the flyout's side, when the sideways distance is at least a third of the vertical distance.
- A submenu row now waits 300ms on such a move. A move along the rows still switches at once.
- `_onFlyoutEnter` cancels the wait when the mouse reaches the flyout. Without this, the timer fires anyway and switches away.

This changes how every menu with submenus responds to the mouse.

## Non-obvious things

- **The mouse trail has two rules, and each one fixed a real failure.** The first version of the hover fix did not work in Chrome.
  - Positions expire after 150ms (`TRAIL_MS`). The mouse usually arrives at a row from the other direction, then pauses. Old positions made the next move read as "moving away".
  - `mouseover` records a position too. A row's `mouseenter` fires before the `mousemove` of the same step, so `mousemove` alone misses the step that entered the row.
- **Dividers must be direct children of `Menu`.** `Menu` indexes its children with `Children.forEach`, which flattens arrays but not fragments.
- **Filter and milestone totals can differ.** The filter counts every issue in a milestone, and the Milestones view's total appears to leave out epics. On the demo board, 0.3.0 showed 4 against 3, with one epic in it. The filter count matches what the filter lists.
- **The safe triangle's first condition is nearly always true.** `isInsideSafeTriangle` passes the mouse position as a triangle corner, which makes the check pass for almost any position. This was there before and this issue leaves it alone. It is why `movingTowardFlyout` makes the real decision.
- **`SubMenu.dom.test.tsx` supplies its own layout.** jsdom measures everything as zero, so the file fakes the positions and event times. These are a model of the browser, and they do not prove real browser behaviour.

## Found on the way: xpo-8fc8ba

`make frontend` failed on `main`, because a test fixture from xpo-397365 lacked `time`. `make test` passed, because it does not run `tsc`. The one-line fix ships in this branch. `make test` still does not type check.

## Acceptance criteria

- [x] Milestone counts show the issues in each milestone, whatever is checked in the Milestone filter. Evidence: `facetIssues` tests; Chrome on the demo board.
- [x] Status, Assignee, Worked by, Priority, Labels and Epic ignore their own selection in their counts and rows. Evidence: the "clears every kind of dimension" test; Labels checked in Chrome.
- [x] A selection in one dimension narrows the counts of the other dimensions, in all four views. Evidence: Label = Bug in Chrome matched the API on Issues. The other three views share `FilterMenu`; the user tried the build and approved it.
- [x] On a Backlog tab, the counts include only the tab's statuses and the current search. Evidence: `tabIssues` in `Backlog.tsx`; the `facetIssues` scope test.
- [x] The Milestone menu shows No milestone, Current, Next, a separator, then the Upcoming and Past submenus. Evidence: `FilterMenu.dom.test.tsx`; Chrome.
- [x] The Upcoming submenu lists nearest to farthest. The Past submenu lists newest to oldest. Evidence: `filters.test.ts`, `FilterMenu.dom.test.tsx`.
- [x] Each milestone row shows the name first. Current and Next show a `SlotBadge` after the name. Evidence: screenshots in Chrome.
- [x] A click on a row in a submenu checks the row, and the menu stays open. Evidence: `FilterMenu.dom.test.tsx`; Chrome at 1400px and 900px.
- [x] A submenu row shows the number of its checked rows. Evidence: `FilterMenu.dom.test.tsx`; Chrome.
- [x] Typing in the search box shows matches from all groups as a flat list. Typing "current" or "next" finds those rows. Evidence: `FilterMenu.dom.test.tsx`, `filters.test.ts`.
- [x] Typing closes an open third-level submenu, and arrow keys then work on the search results. Evidence: `FilterMenu.dom.test.tsx` covers the close and that it does not reopen. The arrow keys follow from the cleared index, and no test presses them on search results.
- [x] Right arrow opens a submenu, left arrow closes it, and Escape closes one level at a time. Evidence: `FilterMenu.dom.test.tsx`.
- [x] A straight mouse move from Milestone to Upcoming keeps the Milestone flyout open. Evidence: scripted mouse in Chrome; `SubMenu.dom.test.tsx`; confirmed by the user.
- [x] A vertical mouse move along the top-level Filter rows switches the flyout at once. Evidence: Chrome; `SubMenu.dom.test.tsx`.
- [x] The third level opens on the same side as the second level, and does not overlap the second level's rows. Evidence: flyout positions in Chrome; `SubMenu.dom.test.tsx`.
- [x] `MenuDivider` and `MenuFilter` use `--color-tint-border`, and the line is visible on the menu surface. Evidence: `MenuSeparators.test.tsx`; computed style and screenshots in Chrome.
- [x] `filters.test.ts` covers `facetIssues` and the new `milestoneFilterOptions` output.
- [x] `make test` passes: 1614 frontend tests and the Go suite. `tsc` and `make frontend` also pass.

Seven deliberate breaks of `Menu` and `SubMenu` each failed the matching test in `SubMenu.dom.test.tsx`.
