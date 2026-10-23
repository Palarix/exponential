# Walkthrough: Releases — progress pill switches between issues and story points

## What was built and why
A release's progress pill counted only issues, and issues differ in size. Now a click on the pill switches every release row between issues done of total (`3/6`, with the progress ring) and story points done of total (`▲ 5/13`, triangle only). The choice is saved in this browser.

The issue asked for a second pill with the point total. During review we chose to switch the existing pill instead, as the backlog's status headers do (`#` ⇄ ▲). That keeps the row as compact as it was. Review also changed two looks:
- **No ring in points mode.** The triangle takes its place, so the two modes look different at a glance.
- **The release pill is raised.** It's now a control, a toggle with a hover card. The inset indicator pill looked like a static count, and it matched the epics' child-progress pills in the expanded list.

## How the pieces fit

### Server: points are part of a release's scope
`ReleaseCounts` (`internal/exponential/release.go`) used to return `(counts, total, done)`. It now returns a `ReleaseScope` struct with `Counts`, `Total`, `Done` and `Points *model.ReleasePoints`. `ReleaseView` and `ReleaseProgress` embed it, so `v.Counts`, `v.Total` and `v.Done` work as before, and the callers pass the scope on in one piece: `releaseToResponse(r, slot, v.ReleaseScope)`.

Points follow the same rule as the counts. They add up leaf issues in the release, without CANCELED and DUPLICATE. Parents are skipped, so an epic's estimate doesn't count twice next to its children's. Unestimated issues add 0.

**Past releases** read their counts from the snapshot that `CompleteRelease` writes, so they survive `xpo archive`. Points now go into the same snapshot: `ReleaseStats.Points`. Releases completed before this change have no points in their snapshot. `Points` is then nil, and the API leaves the field out. We decided not to rebuild them from `leaf_ids`, because archived issues would be missing from the sum and it would be wrong without saying so. Instead, in points mode those rows show no pill.

`points` is in the web API (`ReleaseResponse`) and the CLI/MCP JSON (`ReleaseSummary`). `jsonio` can't import `exponential`, so `ToReleaseSummary` takes `points` as an extra parameter instead of the scope struct.

### Web
- **`ReleaseProgress`** (`ReleaseRow.tsx`) takes `mode` and `onToggleMode`. It picks `done`/`total` from `release` or `release.points`. Its aria-label says what the pill shows and what a click does.
- **The breakdown** opened on click before. A click now switches the mode, so the `ReleaseSummaryCard` opens on hover instead. It uses `createHoverIntent` (250 ms, like the issue hover card) and a `pointer-events-none` popover. The card header adds `· 5/13 pts` when the release has points.
- **The saved setting** lives in `ReleasesBoard` (`localStorage` key `exponential-releases-show-points`). It goes down as props: `ReleaseStack` → `ReleaseItem`/`PastReleases` → `ReleaseRowBody`. It sits at the top on purpose. The tests render `ReleaseStack` server-side, with no `localStorage`, so `ReleaseStack` stays pure and takes `progressMode` as a prop.
- **`raisedIndicatorPill`** (`ui/indicator-pill.ts`) is the indicator pill as a control. It sits on `surface-2` with the `border-elevated` edge, and on hover the edge goes to `border-overlay` and the text to primary. Only the release progress pill uses it. Issue rows, epic rows and the No release count keep the inset `indicatorPill`.

## Acceptance criteria
- [x] Clicking the progress pill switches all release rows between issues done/total (with the ring) and points done/total (triangle, no ring). Evidence: `ReleaseRow.test.tsx` "shows story points done of total in points mode" and "shows the ring in issues mode and the triangle instead in points mode"; `Releases.test.tsx` "shows every row's progress, open and past, in story points in points mode"; tried on the demo board.
- [x] The choice is saved across reloads. Evidence: `ReleasesBoard` reads and writes `exponential-releases-show-points`; tried on the demo board.
- [x] Points cover in-scope leaf issues only (no parents, no CANCELED/DUPLICATE). Unestimated issues count as 0. Evidence: `TestReleaseCounts_SumsPointsOfTheScope`.
- [x] Completing a release stores the points in its snapshot, and past rows show them. Evidence: `TestCompleteRelease_MovesOpenIssuesToNextAndKeepsDone` checks the snapshot (canceled left out); demo 1.0 completed shows 17/17.
- [x] A past release with no stored points shows no pill in points mode. Evidence: `TestReleaseCounts_PastReleaseReadsPointsFromItsSnapshot`; `ReleaseRow.test.tsx` "hides the pill in points mode for a past release without stored points".
- [x] Hovering the pill opens the status breakdown card. Evidence: `ReleaseProgress` hover intent; tried on the demo board.
- [x] The release progress pill is raised (`surface-2`, elevated border). Issue and epic pills are unchanged. Evidence: "raises the progress pill, a button, off the row"; `IssueRow` and the No release count still use `indicatorPill`.
- [x] `points` is in the server release response and the CLI JSON release summary. Evidence: `TestReleaseResponse_PointsOnlyWhenKnown`; the board API check in `TestReleasesAPI_Lifecycle`; `ToReleaseSummary` takes `points`.
- [x] Go tests cover `ReleaseCounts` points (open, snapshot, no snapshot points) and `CompleteRelease` writing them. Web tests cover both pill modes, the hidden pill and the raised surface. Evidence: the tests named above.
- [x] `make test` passes. Evidence: lint plus 1560 web tests and the Go suite, all green.

## Open for later
Completing a release that isn't Current is still allowed and shows no warning. We left it as it is, to revisit later.
