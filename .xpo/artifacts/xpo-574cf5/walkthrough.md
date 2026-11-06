# Walkthrough: Current milestone section on the Dashboard

## What was built

The Dashboard has a new "Current Milestone" section directly under Composition. It shows the name and goal of the Current milestone, the progress in issues and story points, the target date with the time bar, and the burndown. A click on the name opens All Issues with the `@current` filter.

The burndown now draws from the first day of a milestone. This change also applies to the hover card in the Milestones view.

## How the pieces fit together

- `Dashboard/CurrentMilestoneSection.tsx` holds two components. The default export reads `useMilestones` and decides whether the section shows. `CurrentMilestoneCard` reads `useMilestoneProgress` and renders the content.
- The card mounts only when the section shows. So the progress request never runs for a placeholder, and no milestone ref has to be invented while the board is loading.
- `Dashboard.tsx` only renders `<CurrentMilestoneSection />`. The component reads its own data, so the hook order stays correct with the early return of an empty Dashboard.
- `milestones-utils.ts` gains `showsOnDashboard`, and receives `todayISO` and `milestoneNameClass`.
- `MilestoneRow.tsx` gains `BurndownWithLegend`, extracted from `PastBurndown`, and now exports `MilestoneDate`.

## Key decisions

1. **The section is under Composition.** The first version sat between Pulse and Composition. In review, Composition then read as the composition of the milestone. Composition describes the whole board, so it now comes first. Pulse stays the first section.
2. **The chart draws from the first day.** The first spec showed "Not enough days yet." with fewer than 2 points. In review, the user did not want to wait days for a chart. The server already sends one point from `started_at`, so `Burndown` repeats that point and draws flat lines. No backend change was needed.
3. **The fix is in `Burndown`, not in the section.** The hover card in the Milestones view gets the same behaviour, and its message is gone.
4. **`PastBurndown` keeps its guard of 2 or more days.** A milestone completed within one day still shows no burndown in its expanded row.
5. **The progress is static text.** The `MilestoneProgress` pill has a hover card that would repeat the section.
6. **No status breakdown and no slot badge.** The issue does not ask for the breakdown, and the section title already says "Current".

## Non-obvious details

- `milestoneNameClass` moved to `milestones-utils.ts` because the `react-refresh/only-export-components` lint rule does not allow a plain function export from a component file.
- `BurndownWithLegend` shows the end date only with 2 or more points, so a first-day chart does not show the same date twice.
- On the first day, the time bar is empty because no time has passed. This is correct.
- The section's position in `Dashboard.tsx` has no automated test. The position was checked in the browser only.
- Headless Chrome with `--dump-dom` or `--screenshot` captures the board while lazy views still show "Loading...". To check the UI headlessly, drive Chrome over the DevTools protocol and wait for the content.

## Acceptance criteria

- [x] The Dashboard shows a "Current Milestone" section directly under Composition when Current is saved and has scope. Evidence: headless Chrome on a demo board; the user tried the build.
- [x] Pulse stays the first section of the Dashboard, and Composition stays directly under Pulse. Evidence: section order in headless Chrome: Pulse, Composition, Current Milestone, Workload, Active Epics, Recent Activity.
- [x] The section shows the milestone name, and the goal if it has one. Evidence: `CurrentMilestoneSection.dom.test.tsx`; headless Chrome.
- [x] The section shows the scope as done of total issues, and as done of total story points if the milestone has points. Evidence: `CurrentMilestoneSection.dom.test.tsx`; headless Chrome showed `3/8 done · 9/28 pts`.
- [x] With a target date, the section shows the date and the time bar. Without one, it shows neither. Evidence: `CurrentMilestoneSection.dom.test.tsx`.
- [x] The target date uses the warning colour when it has passed with open scope. Evidence: `CurrentMilestoneSection.dom.test.tsx`, which also covers the case with all scope done.
- [x] A started milestone shows the burndown with its dates and legend, from its first day. Evidence: `CurrentMilestoneSection.dom.test.tsx` for several days; headless Chrome for the first day.
- [x] On its first day, the burndown draws flat lines for open and scope, and shows the date once. Evidence: `MilestoneRow.test.tsx`; `CurrentMilestoneSection.dom.test.tsx`; headless Chrome.
- [x] The burndown in the hover card of an open milestone in the Milestones view also draws from the first day. Evidence: `MilestoneRow.test.tsx`.
- [x] The text "Not enough days yet." no longer exists. Evidence: the branch is removed from `MilestoneSummaryCard`; tests assert its absence.
- [x] A milestone that was never started shows no burndown in the section. Evidence: `CurrentMilestoneSection.dom.test.tsx`.
- [x] A click on the name opens Issues on All Issues with the `@current` milestone filter. Evidence: `CurrentMilestoneSection.dom.test.tsx`; headless Chrome showed the "Milestone: Current" chip and the 8 issues.
- [x] The section does not show while Current is a placeholder or has no scope. Evidence: `milestones-utils.test.ts`; `CurrentMilestoneSection.dom.test.tsx`.
- [x] The section does not show while the milestones request is pending or has failed. Evidence: `CurrentMilestoneSection.dom.test.tsx`.
- [x] The section collapses like the other sections and keeps its state in localStorage. Evidence: `CurrentMilestoneSection.dom.test.tsx`.
- [x] `make test` passes. Evidence: exit 0, 1634 frontend tests.
