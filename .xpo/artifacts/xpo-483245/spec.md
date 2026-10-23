# Releases: progress pill switches between issues and story points

## What
A click on a release row's progress pill switches it between issues done of total (`3/6`, with the progress ring) and story points done of total (`▲ 5/13`, without the ring). The switch applies to every release row at once and is saved, like the backlog's status headers (`#` ⇄ ▲). The status breakdown, which a click opened before, now opens as a hover card on the pill. This works for open and past releases. The release pill sits on a raised surface, so the button stands apart from the static count pills on issue and epic rows.

## Why
The pill counts issues, and issues differ in size. Story points show how big a release is, and how much of it is done, without expanding it. Switching the existing pill keeps the row as compact as it is now. A second pill would crowd it.

The plain indicator pill looks inset: it's darker than the row. That suits static counts, but the release pill is now a toggle with a hover card, and an inset pill doesn't look clickable. Raising it shows that it's a control. It also sets it apart from the epics' child-progress pills in the expanded list.

## How

### Server
The server computes the points next to the counts, using the same scope rule:

- **Scope:** the same issues the counts use: leaf issues in the release, without CANCELED and DUPLICATE. Parent issues are skipped, so an epic's points don't count twice next to its children's. Unestimated issues add 0. Done points are the points of the issues in a completed status (the ones `done` counts).
- **Open releases:** `ReleaseCounts` adds up `estimate` for the total and the done points while it counts.
- **Past releases:** `CompleteRelease` writes `points: {total, done}` into the `ReleaseStats` snapshot, next to `counts`, so the points survive `xpo archive` the way the counts do. `ReleaseCounts` reads them from the snapshot.
- **API:** `ReleaseResponse` and the CLI's `ReleaseSummary` get `points: {total, done}`. The field is left out when a past release's snapshot has no points, because it was completed before this change. The web `Release` type gets `points?: { total: number; done: number }`.

### Web
- `ReleaseProgress` takes a `mode: "issues" | "points"` and an `onToggleMode`.
  - **Issues mode:** looks as it does now: the `SubProgress` ring and `3/6`.
  - **Points mode:** the backlog's filled triangle icon and `5/13`, with no ring. The triangle takes the ring's place.
  - A click calls `onToggleMode`. The aria-label says what the pill shows and what a click does, e.g. "3 of 6 issues done. Click for story points".
- **Raised pill:** `indicator-pill.ts` gets `raisedIndicatorPill`: the same shape on `surface-2` with the `border-elevated` edge. On hover the edge goes to `border-overlay` and the text to primary. `ReleaseProgress` uses it. Issue and epic rows keep `indicatorPill`, and so does the No release count, which isn't a button.
- **Breakdown hover card:** while the pointer rests on the pill (`createHoverIntent`, 250 ms, like the issue hover card), the `ReleaseSummaryCard` opens below it (`bottom-end`). A click closes it. It's no longer a click popover.
- **The saved setting:** `ReleasesBoard` keeps the mode in `localStorage` (`exponential-releases-show-points`). `ReleaseStack` takes it as props and passes it to every row. `ReleaseRowBody` gets `progressMode` and `onToggleProgressMode`.
- **Summary card header:** `3/6 done · 5/13 pts`. The points part is left out when the release has no `points`.

## Edge cases
- No estimates in a release: points mode shows `▲ 0/0`.
- A past release with no `points` (completed before this change): in points mode its row shows no pill. Issues mode is unchanged.
- Placeholders have no rows, so they need nothing.

## Acceptance criteria
- [ ] Clicking the progress pill switches all release rows between issues done/total (with the ring) and points done/total (triangle, no ring)
- [ ] The choice is saved across reloads
- [ ] Points cover in-scope leaf issues only (no parents, no CANCELED/DUPLICATE). Unestimated issues count as 0
- [ ] Completing a release stores the points in its snapshot, and past rows show them
- [ ] A past release with no stored points shows no pill in points mode
- [ ] Hovering the pill opens the status breakdown card
- [ ] The release progress pill is raised (`surface-2`, elevated border). Issue and epic pills are unchanged
- [ ] `points` is in the server release response and the CLI JSON release summary
- [ ] Go tests cover `ReleaseCounts` points (open, snapshot, no snapshot points) and `CompleteRelease` writing them. Web tests cover both pill modes, the hidden pill and the raised surface
- [ ] `make test` passes
