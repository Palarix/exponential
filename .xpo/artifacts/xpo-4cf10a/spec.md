# Milestone effort chips show 0/0 when no issue has an estimate

## What

Milestone story points count an unestimated issue as 1 point, the way the backlog's status group headers do. A milestone with issues never has a points total of 0.

## Why

`MilestoneCounts` adds `iss.Estimate` for each leaf issue, and an unestimated issue has estimate 0. A milestone without estimates gets `points: { total: 0, done: 0 }`, so the pill reads `▲ 0/0`, the summary card `0/0 pts` and the Dashboard `· 0/0 pts`.

The backlog headers (`useBacklogRows.ts`) and parent rows (`rowEstimate` in `milestones-utils.ts`) count `estimate || 1`. Milestones should count the same way.

## How

In `internal/exponential/milestone.go`:

- Add a helper `issuePoints(iss) int` that returns the estimate, or 1 when it is 0.
- Use it in `MilestoneCounts` (open milestones) and in the stats snapshot that milestone complete writes (`stats.Points`).
- The scope rule stays as it is: leaf issues in the milestone, without CANCELED and DUPLICATE. Parents are skipped, so an epic counts as the sum of its children and is not counted twice.

Nothing changes in the frontend. With a non-zero total, the pill, the summary card and the Dashboard already show the right values.

Past milestones keep the points in their stored snapshot. Snapshots written before this change still count unestimated issues as 0. They are not rewritten.

Related: the planned points burndown (xpo-60f745) should use the same rule. Its spec gets a note.

## Acceptance criteria

- [ ] An open milestone with 3 unestimated leaf issues, 1 of them done, reports points 1/3.
- [ ] Mixed: leaves estimated 5 (done) and unestimated (open) report 5/6.
- [ ] An epic with estimated children counts only its children. Its own rolled-up estimate is not added.
- [ ] CANCELED and DUPLICATE issues add no points.
- [ ] Completing a milestone stores the same points in its snapshot.
- [ ] A milestone without issues still reports 0/0.
- [ ] `make test` passes.
