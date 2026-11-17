# Walkthrough: milestone points count an unestimated issue as 1

## What changed

`MilestoneCounts` and the completion snapshot now count an unestimated issue as 1 story point. A milestone with issues never shows `▲ 0/0` in points mode. The change is server-side only. The pill, the summary card and the Dashboard already render a non-zero total correctly.

## Why this approach

The first spec proposed a frontend fallback: show the issue count when the points total is 0. The user rejected it. A milestone with issues cannot have 0 points under the rule the backlog already uses: an unestimated issue is 1 point (`estimate || 1` in `useBacklogRows.ts` and `rowEstimate`). Milestones now count the same way, so the fix belongs in the count, not in the display.

The other two parts of the backlog rule were already in place. `MilestoneCounts` counts only leaf issues, so an epic counts as the sum of its children and is never counted twice.

## Code

`internal/exponential/milestone.go`:

- New helper `issuePoints(iss)`. It returns the estimate, or 1 when the estimate is 0.
- `MilestoneCounts` (open milestones) adds `issuePoints` for the total and the done points.
- The milestone complete handler stores the same sums in `MilestoneStats.Points`.
- The `MilestoneCounts` doc comment names the rule.

## Tests

- `TestMilestoneCounts_UnestimatedIssuesCountOnePoint` (new): three unestimated issues with one done give 1/3. A milestone without issues gives 0/0.
- `TestCompleteMilestone_SnapshotCountsUnestimatedIssuesAsOnePoint` (new): done issues estimated 5 and unestimated give a 6/6 snapshot.
- `TestMilestoneCounts_SumsPointsOfTheScope`: the expected total goes from 3/8 to 3/9. The case also checks that the epic, the CANCELED issue and an issue in another milestone add no points.
- `TestMilestonesAPI_Lifecycle`: one unestimated issue in Current now gives 1 point, not 0.

## Other changes

- The issue description now states the counting rule instead of the fallback.
- The xpo-60f745 spec (points burndown) gets the same rule in its AC and Decisions, so the burndown matches the pill.
- CHANGELOG: the unreleased Milestones entry now says unestimated issues count as 1 point, and lists xpo-4cf10a.

## Not changed

Milestones completed before this change keep their stored snapshot. In that snapshot, unestimated issues still count as 0. No migration rewrites them.

## Acceptance criteria

- [x] An open milestone with 3 unestimated leaf issues, 1 of them done, reports points 1/3. `TestMilestoneCounts_UnestimatedIssuesCountOnePoint`. On the dev board, milestone "Polish" showed 1/3.
- [x] Mixed: leaves estimated 5 (done) and unestimated (open) report 5/6. Covered by `TestMilestoneCounts_SumsPointsOfTheScope` (3/9 with one unestimated open issue). On the dev board, "1.0 Launch" showed 1/12: 10 estimated points and 2 unestimated issues, 1 done.
- [x] An epic with estimated children counts only its children. `TestMilestoneCounts_SumsPointsOfTheScope`. On the dev board, "1.1 Mood" showed 0/19, the children's sum, not the epic's 22.
- [x] CANCELED and DUPLICATE issues add no points. `TestMilestoneCounts_SumsPointsOfTheScope` and `TestCompleteMilestone_MovesOpenIssuesToNextAndKeepsDone`.
- [x] Completing a milestone stores the same points in its snapshot. `TestCompleteMilestone_SnapshotCountsUnestimatedIssuesAsOnePoint`.
- [x] A milestone without issues still reports 0/0. `TestMilestoneCounts_UnestimatedIssuesCountOnePoint`.
- [x] `make test` passes: lint, 1709 frontend tests, Go tests.
