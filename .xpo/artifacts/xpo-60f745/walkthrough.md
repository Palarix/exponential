# Walkthrough: milestone burn-up by event order

## What was built

The milestone chart is now a **burn-up** with two x-axes:

- **Order** (the default) draws one evenly spaced step per write that changed the milestone's scope or done work.
- **Time** draws one point per UTC day, as the old chart did.

Done work is a solid, filled line. Scope is a dashed line above it. The gap between them is the open work. The y-axis counts what the progress pill counts: issues, or story points.

## Why it ended up a burn-up

The issue asked for a burndown by event order. Tophat on real data changed that in two rounds:

1. **Setup writes.** Milestones are often filled after they become Current, one write per added issue. On an evenly spaced axis those writes made the open line climb before it fell. Fix: writes before any work fold into the first step.
2. **Scope added during the work.** In this project, issues are filed into the Current milestone while it is worked on: an add, then a done, again and again. Milestone 1.6.0 had 28 steps, and the open line zig-zagged 10, 9, 10, 9, 10, 11, … The data was correct, but the chart read as random. A burn-up moves only the scope line for an add, so the done line only rises. We checked it on a second project with four completed milestones (43–72 steps each). The done lines rose steadily, and the plateaus showed real batches of new scope.

The Time axis was a burndown before this issue. It now draws the same burn-up, so the switch changes only the spacing, not what the chart means.

## How the pieces fit together

### Backend: `internal/exponential/milestone.go`

`ComputeMilestoneProgress` sorts the events once and builds two series from them:

- `milestoneDays` replays the events day by day, as before.
- `milestoneSteps` replays them write by write.

Both use `progressReplay`. It holds each issue's milestone, parent, status, estimate and title. Its `scope()` method returns the milestone's scope: leaf issues, not deleted, not CANCELED or DUPLICATE. `tally()` turns a scope into `MilestoneTally`: `scope`, `open`, `completed` and the three point sums. `issuePoints` counts an unestimated issue as 1, so the chart's points match the pill's (xpo-4cf10a).

**Writes.** The event log has no write ID. `groupWrites` treats a run of consecutive events by the same `created_by` and `on_behalf_of`, less than 1 ms apart, as one write. `AppendEvents` stamps a batch from one `time.Now()` and spaces equal times by 1 ns. A milestone complete that moves 8 issues is one write, and one step.

**Step 0.** The replay applies every write that starts at or before `started_at`. That includes the rest of the write that set `started_at`: a milestone complete moves issues into the next milestone and starts it in one write. Next comes the fold. Until the first write that raises the done count or cancels an issue, each write updates step 0 and adds no step. If issues are already done at `started_at`, work has begun and nothing folds. Step 0 keeps `At = started_at` and no changes.

**Later steps.** After each write, the replay compares the scope with the scope before it. If the tally did not change, the write adds no step. Otherwise `changes()` lists one `MilestoneChange` per issue, sorted by ID, and `changeKind` names the kind:

| Before → after | Kind |
|---|---|
| in scope → out, still in the milestone | `canceled` |
| in scope → out, otherwise | `removed` |
| out (CANCELED/DUPLICATE in the milestone) → in | `reopened` |
| out → in, otherwise | `added` |
| open → completed | `done` |
| completed → open | `reopened` |
| same status, other points | `estimated` |

A parent that gets its first child leaves the scope, and the child joins. That write lists a `removed` and an `added` change.

The steps stop at `completed_at`, or at now for an open milestone. `internal/server/milestones.go` returns `steps` next to `days`.

### Frontend

- **`web/src/api/types.ts`:** `MilestoneTally`, `MilestoneDay` and `MilestoneStep` (with `changes`), and `steps` on `MilestoneProgressResponse`.
- **`Milestones/milestone-settings.ts`:** two stored settings on one `useSyncExternalStore`:
  - `useProgressMode`: the pill's issues/points choice, same key as before.
  - `useBurnUpAxis`: `order` or `time`, default `order`.

  A change in one view shows in the others at once. Without `localStorage` (server render, markup tests), the hooks return the defaults.
- **`Milestones/burnup-utils.ts`:** `burnUpPoints(progress, axis, mode)` returns the points to draw: `scope`, `open` and `done`, plus the step or day of each point for the hover. In issues mode it drops a step whose counts equal the previous drawn point. An estimate-only step would draw a flat segment.
- **`Milestones/MilestoneRow.tsx`:**
  - `BurnUp` draws the filled done line under the dashed scope line, with the points evenly spaced. With `hover`, each point owns a transparent band halfway to its neighbours. Resting the pointer on a band shows a marker line and a tooltip. The tooltip shows the step's time, "Done X of Y" and up to 10 changes, then "and N more". A Time point shows its date. The tooltip is shifted by the point's own fraction, so it stays inside the chart at both ends.
  - `BurnUpWithLegend` adds the first and last dates, the "Done / Scope" legend, "N steps" in Order mode, and the Order | Time radios.
  - `PastBurnUp` shows a completed milestone's chart when Order has at least two drawn steps. In Time mode with fewer than two days, it shows "Completed within one day" and the switch, so Order stays one click away.
  - `MilestoneSummaryCard`, the pill's hover card, uses the stored axis and shows no switch.
- **`Dashboard/CurrentMilestoneSection.tsx`** and **`Milestones/Milestones.tsx`** use the new components and the shared pill hook.

## Key decisions

- **Burn-up, on both axes.** See "Why it ended up a burn-up".
- **One step per write.** Per-event steps would turn a milestone complete into a staircase.
- **The setup folds into step 0.** Scope added once work began is still a step: that is the scope creep the chart must show.
- **An estimate change is a step.** It changes the scope in points.
- **`steps` next to `days`, both with points.** The client switches axis and unit without a second request.
- **At most 10 changes in the hover.** A milestone complete can move dozens of issues.
- **"N steps" counts every drawn point**, step 0 included.

## Non-obvious things

- **Clock skew merges writes.** The storage layer stamps a new event 1 ns after the newest event when the clock is behind it. This happens with a clock stepping back, or with merged events from a machine whose clock runs ahead. Writes in that gap group into one step. It is rare with synced clocks. The demo generator, though, writes events hours into the future, so demo data cannot tophat this feature. Tophat used copies of real `issues.db` files.
- **`StartedAt` and the event time can differ.** The `started_at` payload is the write's `time.Now()`. The storage layer can stamp the event later. Step 0 therefore includes the whole write that set `started_at`, not only the events up to that time.
- **A milestone where nothing is ever done** draws one flat point: all its writes fold into step 0.
- Found during tophat, out of scope: stopping several boards at once corrupted the shared `instances.json`, and `TestHandleListInstances` reads that real file. Filed as xpo-fb37da.

## Acceptance criteria

- [x] `GET /api/milestones/{ref}/progress` returns `steps` next to `days`. Evidence: `internal/server/milestones_test.go` checks `prog.Steps`.
- [x] The first step is the state when work began, with the fold rule and the "already done" exception. Evidence: `TestSteps_SetupFoldsIntoTheFirstStep`, `TestSteps_CancelBeginsTheWork`, `TestSteps_IssuesMovedInWithTheStartAreTheFirstStep`, and the tests that start with done issues.
- [x] A milestone with nothing done or canceled has only the first step. Evidence: `TestSteps_SetupFoldsIntoTheFirstStep`.
- [x] Events of one write form one step. Evidence: `TestSteps_OneStepPerWriteWithItsChanges`, `TestSteps_MilestoneCompleteIsOneStep`, `TestSteps_SeparateActorsAreSeparateWrites`.
- [x] A write that changes nothing adds no step. Evidence: `TestSteps_NoOpWritesAddNoStep`.
- [x] Steps and days carry the six counts. Evidence: `TestDays_CarryPoints`, `TestComputeMilestoneProgress_CountsLeavesAndBurnsDown`, `countsOf` in the step tests.
- [x] Unestimated issues count as 1 point. Evidence: `TestSteps_EstimateChangesArePointSteps` (5 → none is 1).
- [x] Each step lists its changes with kind. Evidence: `TestSteps_ReopenAddedRemovedCanceled`, `TestSteps_ParentSwapListsRemovedAndAdded`.
- [x] An estimate change is an `estimated` step. Evidence: `TestSteps_EstimateChangesArePointSteps`.
- [x] The chart is a burn-up with a "Done / Scope" legend. Evidence: `MilestoneRow.test.tsx` "draws the burn-up in story points in points mode" (done-line path), and the legend test in `CurrentMilestoneSection.dom.test.tsx`.
- [x] Issues mode skips unchanged steps, and "N steps" counts drawn steps. Evidence: `burnup-utils.test.ts`, and "defaults to Order and counts the drawn steps…" (3 steps of 4).
- [x] Order spaces steps evenly, and the hover shows time, "Done X of Y", changes and "Start". Time hover shows the date. Evidence: "shows a step's changes on hover", "shows the date and the counts on hover in Time mode".
- [x] The toggle is on the Dashboard and past-milestone charts. Evidence: the Dashboard axis tests, and "shows the burn-up of a milestone completed within one day in Order mode".
- [x] The hover card uses the stored axis and shows no toggle. Evidence: "breaks the scope down by status…" asserts no `Burn-up axis`.
- [x] Order is the default, and the choice is stored. Evidence: "defaults to Order…", "switches to Time and keeps the choice", "uses the stored axis".
- [x] Points mode draws story points everywhere, the Dashboard included. Evidence: "draws every step in story points when the pill shows points" (Dashboard), and the summary-card points test.
- [x] A past milestone completed within one day shows its burn-up in Order mode. Evidence: the `Milestones.test.tsx` one-day test. Tophat: the second project's milestones, completed in 2–5 days.
- [x] In Time mode, `PastBurnUp` keeps the two-day rule and shows the note and switch. Evidence: `MilestoneRow.dom.test.tsx` "keeps the Time rule…".
- [x] An Order chart shows the first and last dates and "N steps". Evidence: the Dashboard tests.
- [x] Go tests cover batches, no-op writes, reopen, scope added and removed, cancel, estimates, points and the fold. Evidence: `internal/exponential/milestone_steps_test.go`.
- [x] Frontend tests cover the drawing, the toggle, the default, the stored choice, points mode, skipped steps and the hover. Evidence: the tests above. `make test` passes: 1723 frontend tests, Go tests, lint and typecheck.
