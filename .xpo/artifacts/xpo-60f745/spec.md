# Spec: milestone burn-up by event order

## What

The milestone chart becomes a **burn-up**: done work rises toward the scope, drawn as a dashed line, and the gap between them is the open work. It gets a second x-axis: **Order**. Order is the default. It draws one evenly spaced step for each write that changes the milestone's scope or done work. **Time** draws one point per day, as the daily burndown did. A toggle switches between the two, and this browser keeps the choice. The y-axis follows the progress pill: issues or story points.

## Why

Agents close many issues within hours. A daily chart then shows one cliff, or nothing for a milestone done in one day. Order shows every step of progress, whatever its speed. The timestamp stays available on hover.

A burndown (open work falling) does not survive the Order axis. In this project and in another one tried during review, issues are filed into the Current milestone while it is worked on: an add, then a done, over and over. On an evenly spaced axis the open line then zig-zags and reads as random. In a burn-up an add moves only the scope line, and the done line only rises (a reopen is the one dip). Plateaus read as what they are: a batch of new scope before work went on.

## Acceptance criteria

- [ ] `GET /api/milestones/{ref}/progress` returns `steps` next to `days`.
- [ ] The first step is the milestone's state when work began: at `started_at`, plus every later write up to the first one that raises the done count or cancels an issue of the scope. If issues are already done at `started_at`, work has begun and nothing folds. The server adds one step for each write from there on that changes a count or a point sum.
- [ ] A milestone where nothing is done or canceled yet has only the first step: its scope so far.
- [ ] Events of one write form one step. A milestone complete that moves 8 issues is one step.
- [ ] A write that changes nothing in the milestone adds no step.
- [ ] Each step and each day carries issue counts and story points: `scope`, `open`, `completed`, `scope_points`, `open_points`, `completed_points`.
- [ ] Point sums count an unestimated issue as 1, as `MilestoneCounts` does (`issuePoints`, xpo-4cf10a).
- [ ] Each step lists its changes: issue ID, title and kind (`done`, `reopened`, `added`, `removed`, `canceled`, `estimated`).
- [ ] An estimate change on an issue in the milestone is a step with kind `estimated`.
- [ ] The chart is a burn-up on both axes: a solid, filled done line under a dashed scope line. The legend reads "Done" and "Scope".
- [ ] In issues mode, the chart skips steps whose issue counts did not change. "N steps" counts the steps the chart draws.
- [ ] In Order mode, the chart spaces steps evenly. Hovering a step shows its time, "Done X of Y" and its changes, at most 10, then "and N more". The first step shows "Start". In Time mode, the hover shows the date and "Done X of Y".
- [ ] The toggle (Order / Time) appears on the Dashboard chart and the expanded past-milestone chart, as a small segmented control right of the legend.
- [ ] The hover-card chart in the Milestones view uses the stored axis. It shows no toggle.
- [ ] Order is the default. This browser keeps the choice.
- [ ] With the pill in points mode, all burn-up charts draw story points. The Dashboard chart uses the same stored pill setting.
- [ ] A past milestone completed within one day shows its burn-up in Order mode.
- [ ] In Time mode, `PastBurnUp` keeps the old rule: no chart with fewer than two days. It shows "Completed within one day" and the toggle instead, so Order stays one click away.
- [ ] Under an Order chart: the first and last dates at the ends, and "N steps".
- [ ] Go tests cover step building: batches, no-op writes, reopen, scope added and removed, cancel, estimate changes, points, and the setup folded into the first step.
- [ ] Frontend tests cover the burn-up drawing, the toggle, the default, the stored choice, the points mode, skipped steps in issues mode and the hover content.

## Flow

### Backend (`internal/exponential/milestone.go`)

1. Track the estimate in the replay state. Read it from `CreatePayload.Estimate` and `UpdatePayload.Estimate`.
2. Track the title in the replay state, so a change shows the title at that time.
3. Add `MilestoneStep`: `At`, the six counts, and `Changes []MilestoneChange` (`IssueID`, `Title`, `Kind`).
4. Add `ScopePoints`, `OpenPoints` and `CompletedPoints` to `MilestoneDay`.
5. Extract the count of one point into a helper. Days and steps both use it.
6. In `ComputeMilestoneProgress`, replay all events up to `started_at`, and the rest of the write that set it. Unless that state has done issues, keep folding writes into it until the first write that raises the done count or has a `canceled` change. Emit that state as step 0, at `started_at`, with no changes.
7. Group the later events into writes (see Decisions). After each write, compare the milestone's issues with their state before the write.
8. Emit a step when a count or a point sum changed. Record one change per issue whose membership, state or estimate changed.
9. Stop at `completed_at` (or now), as the daily chart does.
10. Add `Steps` to `MilestoneProgress`, and `steps` to the response in `internal/server/milestones.go`.

### Frontend

1. Add `MilestoneStep` and `MilestoneChange` to `web/src/api/types.ts`. Add the point fields to `MilestoneDay`.
2. Move the pill's `PROGRESS_MODE_KEY` read into a shared hook. `Milestones.tsx` and the Dashboard section both use it.
3. Add a stored `BurnUpAxis` setting (`order` | `time`, default `order`) with its own hook.
4. Rename the chart to `BurnUp`, and make it take generic points: `{ scope, open, done }[]` plus the step or day of each point for the hover. It draws `done` filled, and `scope` dashed.
5. In issues mode, drop the steps whose counts equal the previous drawn step.
6. Add hover to `BurnUp`. In Order mode, the tooltip shows the step time, the done count and its changes. In Time mode, it shows the date and the done count.
7. `BurnUpWithLegend` gets the Order / Time segmented control right of the legend, and the footer ("N steps" in Order mode).
8. `CurrentMilestoneSection`, `MilestoneSummaryCard` and `PastBurnUp` choose steps or days from the stored axis, and counts or points from the stored pill mode.
9. `PastBurnUp` shows the chart in Order mode when there are at least two drawn steps.

## Decisions

- **A burn-up, not a burndown, on both axes.** Scope added while a milestone is worked on made the open line zig-zag on the Order axis. A burn-up moves only the scope line for an add. Both axes draw it, so the toggle changes only the spacing, not what the chart means.
- **One step per write, not per event.** A batch is one action of one actor. Per-event steps would turn a milestone complete into a staircase.
- **A write is a run of consecutive events by the same actor, less than 1 ms apart.** The event log has no write ID. `AppendEvents` stamps a batch from one `time.Now()`, and `stampInOrder` spaces equal times by 1 ns. Separate writes by a human or an agent are far more than 1 ms apart.
- **The setup folds into the first step.** Milestones are often filled after they become Current: one write per added issue or estimate. Those writes would be a run of steps before any work. Work begins with the first issue done or canceled, or at the start when issues are already done. Scope added after that is still a step up: that is scope creep the chart must show. A milestone in which nothing is ever done draws one flat point.
- **An estimate change is a step (kind `estimated`).** It changes the scope in points: scope creep that the points chart must show.
- **Unestimated issues count as 1 point** (`issuePoints`, from xpo-4cf10a). The chart's points then match the pill's. Removing an estimate (5 → none) is a change to 1, not to 0.
- **Issues mode skips steps that change only points.** A flat segment would add no information to an issue-count chart.
- **`steps` next to `days`, not instead of it.** Time mode still needs `days`.
- **Points in the API for both axes.** The client switches between counts and points without a second request.
- **The pill setting stays one setting.** The Dashboard reads the same key as the Milestones view. A user who counts in points sees points everywhere.
- **No toggle on the small hover-card chart.** The card is a preview, and a toggle there would compete with the pill click.
- **The toggle is a small Order | Time segmented control right of the legend.**
- **The hover lists at most 10 changes.** A milestone complete can move dozens of issues; the rest are counted.
- **"N steps" counts every drawn point, the first step included**, as "at least two drawn steps" does.

## Edge cases

- **HIGH:** none. This changes only reads and display.
- **MEDIUM:** a milestone with hundreds of steps. The chart draws all of them; the line gets dense but stays correct. No thinning in this issue.
- **MEDIUM:** a parent gets its first child during the milestone. The parent leaves the scope (only issues without children count), and the child joins. That write is one step with a `removed` and an `added` change.
- **LOW:** an issue moves to another milestone and back. Each move is a step.
- **LOW:** a placeholder or never-started milestone returns empty `steps` and `days`, as before.

## Assumptions

- The 1 ms grouping rule matches every multi-event write in today's code: milestone complete, milestone delete and moves with children.
- Clocks are roughly in sync. Writes made while the clock is behind the newest event in the log are stamped 1 ns apart and can group into one step.
- Archived events stay part of the replay, as before, so past milestones keep their steps.
