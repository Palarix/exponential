# Spec: Current milestone section on the Dashboard

## What

The Dashboard gets a "Current Milestone" section. The section shows the name and goal of the Current milestone, its progress, its target date with the time bar, and its burndown from `started_at`. A click on the name opens Issues with the `@current` milestone filter.

## Why

The Dashboard is the overview of the board, and it says nothing about the work in focus. Epic xpo-4c92d4 (Decision 2) places the Current burndown on the Dashboard. The epic also has this acceptance criterion: "The Dashboard shows the Current milestone's progress and burndown."

## Acceptance criteria

- [ ] The Dashboard shows a "Current Milestone" section directly under Composition when Current is saved and has scope.
- [ ] Pulse stays the first section of the Dashboard, and Composition stays directly under Pulse.
- [ ] The section shows the milestone name, and the goal if it has one.
- [ ] The section shows the scope as done of total issues, and as done of total story points if the milestone has points.
- [ ] With a target date, the section shows the date and the time bar. Without one, it shows neither.
- [ ] The target date uses the warning colour when it has passed with open scope.
- [ ] A started milestone shows the burndown with its dates and legend, from its first day.
- [ ] On its first day, the burndown draws flat lines for open and scope, and shows the date once.
- [ ] The burndown in the hover card of an open milestone in the Milestones view also draws from the first day.
- [ ] The text "Not enough days yet." no longer exists.
- [ ] A milestone that was never started shows no burndown in the section.
- [ ] A click on the name opens Issues on All Issues with the `@current` milestone filter.
- [ ] The section does not show while Current is a placeholder or has no scope.
- [ ] The section does not show while the milestones request is pending or has failed.
- [ ] The section collapses like the other sections and keeps its state in localStorage.
- [ ] `make test` passes.

## Flow

1. Write the tests first: `milestones-utils.test.ts` for `showsOnDashboard`, `MilestoneRow.test.tsx` for a one-day burndown, and `CurrentMilestoneSection.dom.test.tsx` for the section.
2. Add `showsOnDashboard(milestone)` to `Milestones/milestones-utils.ts`. It returns false for a placeholder and for `total === 0`.
3. Move `todayISO` from `Milestones/Milestones.tsx` to `milestones-utils.ts` and export it.
4. Move `milestoneNameClass` from `Milestones/MilestoneRow.tsx` to `milestones-utils.ts` and export it. The `react-refresh/only-export-components` lint rule does not allow a plain function export from a component file.
5. Export `MilestoneDate` from `MilestoneRow.tsx`.
6. Change `Burndown` to return null only for an empty `days`, and to draw one day as a flat line.
7. Remove the "Not enough days yet." branch from `MilestoneSummaryCard`.
8. Extract `BurndownWithLegend({ days })` from `PastBurndown` in `MilestoneRow.tsx`. It renders the chart, the dates and the legend.
9. Make `BurndownWithLegend` show the end date only when `days` has 2 or more points.
10. Make `PastBurndown` use `BurndownWithLegend`, and keep its guard of 2 or more days.
11. Add `Dashboard/CurrentMilestoneSection.tsx` with a default export that reads `useMilestones`, and hides the section with `showsOnDashboard`.
12. Add a `CurrentMilestoneCard({ milestone, onOpen })` to the same file. It reads `useMilestoneProgress`, so the request runs only when the section shows.
13. Make the name a button that calls `onOpen`. Use `MilestoneTitle` with its `name` prop and `badge={false}`.
14. Make `onOpen` call `milestoneFilter` and `navigate("backlog", { tab: "all", filters: { milestones: [filter] } })`, as `Milestones.tsx` does.
15. Render `CurrentMilestoneSection` in `Dashboard.tsx` directly under Composition, with the storage key `exponential-dashboard-milestone-open`.

## Layout

One `Card` in the section body, full width, in this order:

```
1.6.0 – Better agent chat                    (ring) 5/12 done · 8/20 pts
Target date                                                       Nov 20
▓▓▓▓▓▓▓▓▓▓▓▓▓░░░░░░░░░░░░░░░░░
[ burndown chart, 96 px high ]
Oct 27                ─ Open   ┄ Scope                            Nov 6
```

## Decisions

1. **Directly under Composition, not between Pulse and Composition (user decision).** Pulse is always the first section. Composition describes the whole board. Under the milestone section, Composition reads as the composition of the milestone.
2. **A separate component, not more code in `Dashboard.tsx`.** The component reads its own data, so the hook order stays correct with the early return of an empty Dashboard. The file also stays smaller.
3. **Reuse from `MilestoneRow.tsx`, not `Cycles.tsx`.** `Burndown`, `TimeBar`, `MilestoneTitle` and `MilestoneDate` already take a `Milestone` or `MilestoneDay[]`. `Cycles.tsx` has no shared chart for this data.
4. **A static progress text, not the `MilestoneProgress` pill.** The pill's hover card repeats the section, and the section has room for both issues and points.
5. **No breakdown by status.** The issue does not ask for it, and the hover card in the Milestones view already has it.
6. **No slot badge.** The section title already says "Current".
7. **No link for a placeholder.** The section does not show for a placeholder, so `@current` always names a saved milestone here.
8. **A chart from the first day, not a wait for a second point (user decision).** The server sends one point from the day a milestone starts. `Burndown` draws that point as a flat line, so open and scope show at once. The fix is in `Burndown`, so the hover card in the Milestones view gets it too.
9. **`PastBurndown` keeps its guard.** A milestone completed within one day still shows no burndown in its expanded row. Its story is the delivered issues, and the flat line adds nothing there.

## Edge cases

- **MEDIUM:** Current completes while the Dashboard is open. The board data refreshes, and the section shows the new Current, or hides.
- **MEDIUM:** All scope is done, but Current is still open. The section shows 12/12 done, and the date does not show as overdue.
- **MEDIUM:** Current was never started and has a target date. The time bar starts from `created_at`, as it does in the Milestones view. No burndown shows.
- **LOW:** The progress request is pending. The card shows no chart until the days arrive.
- **LOW:** Current has no label. The name shows as `#4`.
- **LOW:** Current has no goal. The name shows alone, with no dash.
- **LOW:** The Dashboard has no issues. The empty state shows, and the section does not render.

## Assumptions

- Open milestones always send `points` when they have estimates. If `points` is missing, the section shows only the issue count.
- The existing query invalidation keeps `useMilestones` and `useMilestoneProgress` up to date on the Dashboard, as it does in the Milestones view.

## Out of scope

- A section for Next or Upcoming milestones.
- Forecasts on the burndown (epic spec, Out of scope).
- Burndown points within a day. The server keeps daily points.
- Backend changes. `GET /api/milestones` and `GET /api/milestones/{id}/progress` already return all the data.
