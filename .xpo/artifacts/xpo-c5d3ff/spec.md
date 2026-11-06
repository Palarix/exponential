# Filter menu: correct counts, and a new Milestone menu

## What

1. Each filter dimension counts issues as if its own selection were empty. The selections in the other dimensions still narrow the counts.
2. The Milestone submenu shows No milestone, Current and Next, a separator, then an **Upcoming** submenu and a **Past** submenu. Each milestone row shows its name. Current and Next show a `SlotBadge` after the name.
3. Menu separators are visible on every surface. This change applies to all menus in the app.
4. The menu primitives support a submenu inside a submenu. No menu used this before.
5. A move toward an open flyout no longer switches to a submenu row on the way. This change applies to all menus with submenus.

## Why

`Backlog.tsx:1515` gives `FilterButton` the list `filteredIssues`. The active filters have already narrowed that list. If Current is checked, only issues in Current remain, so every other milestone counts 0.

The same cause affects other dimensions in the Backlog:
- Status, Priority and Worked by count 0 for each unchecked value.
- Labels and Assignee lose their rows, because these rows come from the narrowed list.
- Epic loses the selected epic's row, because the epic itself is not its own child.

My Issues, Inbox and Milestones have the opposite fault. They give the menu an unfiltered list, so a filter in one dimension never narrows the counts of another.

Menu separators use `--color-border-subtle` (`#1e1e1e`). The filter menu's surface measures `#1c1c1c` in the browser, so the line is only 2 steps away from its surface and almost invisible.

A flat Milestone list needs three group separators, and the list grows with every completed milestone. The user compared five layouts on a tophat page and chose the submenus.

Upcoming and Past are always low in the second level. In Chrome, a straight mouse move from Milestone to Upcoming crossed the Epic row and replaced the Milestone flyout with the Epic flyout. The safe triangle at `Menu.tsx:202` protected only rows that are not submenus.

The second level opens left of the top-level menu. In Chrome, the third level opened right, because it had room there. It overlapped the second level, and on a short window it can cover the top-level menu.

## How

### Counts

- Add `facetIssues(issues, dim, filters, board, scope)` to `web/src/components/Backlog/filters.ts`. It applies `filtersForScope` when `scope` is set, clears `dim` with `setSelected`, then runs `matchesFilters`.
- In `FilterMenu.tsx`, call `computeFilterOptions` with the `facetIssues` result for each dimension. Add `filters` and `scope` to the `useMemo` dependencies.
- In `Backlog.tsx`, split out the list before filters: the tab's statuses plus the search. `filteredIssues` applies `matchesFilters` to that list. `FilterButton` receives that list.
- My Issues, Inbox and Milestones already pass an unfiltered list, so they need no change.

### Milestone menu

- Change `milestoneFilterOptions` to return a flat list of `{ value, label, count, slot?, group? }`.
- `label` is the milestone name. `slot` is `current` or `next`. `group` is `upcoming` or `past`.
- `board.upcoming` is already nearest first, and `board.past` is already newest first. The menu uses both as they are.
- In `FilterMenu.tsx`, render the rows without a `group`, a `MenuDivider`, then one `SubMenu` for each group that has rows.
- A `SubMenu` row shows how many of its rows are checked, in the style of the top-level Filter rows.
- When the search box has text, `FilterMenu` renders all matching rows as one flat list.
- `SlotBadge` comes from `MilestoneRow.tsx`, through the `content` prop of `MenuItem`.

### Nested submenus

- `FilterMenu.tsx`: the outside-click check uses `closest("[data-submenu-flyout]")`, so a click in any open flyout counts as inside.
- `Menu.tsx`: each `Menu` gets an ID from `useId`, and passes it to its children as `_ownerId`.
- `SubMenu.tsx`: the flyout sets `data-submenu-owner` to `_ownerId`.
- `Menu.tsx`: the safe triangle reads the flyout with this menu's own ID, not the first flyout in the document.
- `Menu.tsx`: when the open index no longer points at a submenu, `Menu` clears it during render. A search replaces the rows, so the index goes stale.
- `popover-utils.ts`: add the placement `left-start`. It flips to the right when the left has no room.
- `SubMenu.tsx`: each flyout records its side in `data-submenu-side`. A nested flyout reads its parent's side and opens on that side.

### Mouse moves toward a flyout

- `menu-utils.ts`: add `movingTowardFlyout(from, to, flyoutIsRight)`. It returns true when the mouse moves toward the flyout's side, and the sideways distance is at least a third of the vertical distance.
- `Menu.tsx`: keep up to 5 mouse positions from the last 150ms. `from` is the oldest position, so one small movement does not decide the result.
- `Menu.tsx`: record positions on `mouseover` and on `mousemove`. A row's `mouseenter` fires before the `mousemove` of the same step, so `mousemove` alone misses the step that entered the row.
- `Menu.tsx`: a submenu row now waits like other rows when the mouse moves toward the flyout. A move along the rows still switches at once.
- `Menu.tsx`: pass `_onFlyoutEnter` to children. It cancels the pending switch.
- `SubMenu.tsx`: call `_onFlyoutEnter` when the mouse enters the flyout.

### Menu separators

- In `web/src/components/ui/Menu.tsx`, change `MenuDivider` to `--color-tint-border`.
- Change `MenuFilter` to render `MenuDivider`, in place of its own copy of the line.

## Decisions

- **Upcoming and Past are submenus, not flat groups.** The user chose this on the tophat page, over a merged flat list, a "Show older" row and a Past-only submenu. The cost is one more step to reach an upcoming milestone.
- **No milestone, Current and Next stay at the top, as one group.** These three need one click, and a separator between them adds nothing.
- **Other dimensions still narrow the counts, in all views.** The user chose this. A count then tells the user how many issues they get when they check that row.
- **The fix goes in `FilterMenu`, not in each view.** All four views get the same counts. My Issues, Inbox and Milestones change: their counts now respond to the other dimensions.
- **The search box searches all rows, including rows in closed submenus, and it keeps matching "Current" and "Next".** The labels no longer contain these words, so the search also checks `slot`.
- **Separators use `--color-tint-border`, not a lighter fixed gray.** The user asked for the surface-mixed color. The token is a wash of the text color (xpo-aaf8b1), so it stays visible on any menu surface. The token `--color-border-subtle` does not change, because other content dividers use it.
- **The direction of the mouse decides, not a delay on every submenu row.** A delay on all rows would slow each move along the Filter menu by 300ms. An earlier draft of this spec kept the instant switch. The Chrome test showed that layout E needs this change.
- **Positions expire after 150ms.** A position from before a pause belongs to an earlier move. The first version of this fix kept them, and an arrival from the left read as a move away from a left-side flyout.
- **A nested flyout keeps its parent's side.** A menu that goes left, then right, overlaps itself.

## Edge cases

- **MEDIUM:** A checked row in a closed submenu is out of sight. The submenu row's count and the filter chip show the selection.
- **MEDIUM:** A selected Label or Assignee row disappears when another dimension leaves it with 0 issues. These rows come from the issues in the list. The Backlog does this today, and this issue does not change it. The chip still shows the selection.
- **MEDIUM:** A slow move toward a flyout that stays on one row switches after 300ms. The safe triangle already works this way for other rows.
- **LOW:** A group with no rows has no submenu. With neither group, the menu also has no separator.
- **LOW:** A window too narrow for the third level on the left opens it on the right.
- **LOW:** A placeholder Current or Next has no ID, so it counts 0. The row shows `#N` and the badge.
- **LOW:** Before the board loads, the menu shows only No milestone.
- **LOW:** Filter refs (`@current`, `@next`, `#N`, `none`) and chip names (`milestoneRefName`) do not change. The chip still reads "Current".
- **LOW:** The `MenuFilter` line now has `role="separator"`, like every other `MenuDivider`.

## Acceptance criteria

- [ ] Milestone counts show the issues in each milestone, whatever is checked in the Milestone filter.
- [ ] Status, Assignee, Worked by, Priority, Labels and Epic ignore their own selection in their counts and rows.
- [ ] A selection in one dimension narrows the counts of the other dimensions, in all four views.
- [ ] On a Backlog tab, the counts include only the tab's statuses and the current search.
- [ ] The Milestone menu shows No milestone, Current, Next, a separator, then the Upcoming and Past submenus.
- [ ] The Upcoming submenu lists nearest to farthest. The Past submenu lists newest to oldest.
- [ ] Each milestone row shows the name first. Current and Next show a `SlotBadge` after the name.
- [ ] A click on a row in a submenu checks the row, and the menu stays open.
- [ ] A submenu row shows the number of its checked rows.
- [ ] Typing in the search box shows matches from all groups as a flat list. Typing "current" or "next" finds those rows.
- [ ] Typing closes an open third-level submenu, and arrow keys then work on the search results.
- [ ] Right arrow opens a submenu, left arrow closes it, and Escape closes one level at a time.
- [ ] A straight mouse move from Milestone to Upcoming keeps the Milestone flyout open.
- [ ] A vertical mouse move along the top-level Filter rows switches the flyout at once.
- [ ] The third level opens on the same side as the second level, and does not overlap the second level's rows.
- [ ] `MenuDivider` and `MenuFilter` use `--color-tint-border`, and the line is visible on the menu surface.
- [ ] `filters.test.ts` covers `facetIssues` and the new `milestoneFilterOptions` output.
- [ ] `make test` passes.
