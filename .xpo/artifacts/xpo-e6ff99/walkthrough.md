# Walkthrough: Releases — past releases expand in place instead of a detail page

## What was built and why
This issue started as a read-only detail page for past releases: a header, final stats, a burndown and the delivered issues. We built it and tophatted it on the demo board. It showed nothing the Releases view didn't already show, because the expanded row already lists the issues and the progress popover already breaks down the stats. So we dropped the page.

A past release is now read where it already is:

- **Clicking a past release's title toggles its expansion,** like the chevron.
- **The expanded row is the release's record,** top to bottom: the burndown from `started_at` to `released_at`, then the delivered issues. Later, retrospective notes (xpo-f5e7db) go at the very bottom.

A dedicated page comes back only if it tells the release's story: scope changes, timing, release notes. The epic spec (xpo-4c92d4, decision 2 and Out of scope) records this.

## How the pieces fit together

### Routing: the release page is gone
- `views.ts` no longer parses or formats `#/releases/N`. `#/releases/7` falls through to the plain Releases view.
- `Route.releaseNumber` and `AppNav.releaseNumber`/`setReleaseNumber` are removed from `views.ts`, `app/contexts.ts` and `App.tsx`.
- `ReleaseDetail.tsx` and `findRelease` are deleted.
- There is no fallback for old links. The route was never released, and we don't keep backward compatibility for unreleased surfaces.

### Click targets: `releaseLink` → `releaseFilter`
- `releaseLink` returned `{detail}` or `{filter}`. It is now `releaseFilter(release): string | null`: `@current`, `@next` or `#N` for an open release, and null for a past one.
- `Releases()` navigates to Issues only when there is a filter.
- `PastReleases` no longer takes `onOpen`. It passes `onOpen={() => onToggle(r.number)}` to `ReleaseRowBody`, so the title and the chevron do the same thing.
- In `ReleaseRowBody`, a past release's title button carries `aria-expanded` and drops the hover underline, since it is a toggle and not a link. An open release's title is unchanged.

### The burndown in the expanded row: `PastBurndown`
`PastBurndown` lives in `ReleaseRow.tsx`, next to `Burndown`, which it reuses at 96px tall. It adds:
- the first and last day under the chart;
- an Open/Scope legend between them.

It renders nothing when:
- the release was never Current (no `started_at`), or
- the release has fewer than two days of data (started and shipped the same day).

This keeps the expanded row lean, with no "no burndown" message.

`PastReleases` renders it above the issue list, and only while nothing narrows the lists (`!narrowing`). While searching or filtering, the user is looking for issues, so the chart would only push the matches down.

The data comes from `useReleaseProgress('#N')`, the same query the popover uses. For a RELEASED release, the server already computes the days up to `released_at`, so no backend change was needed.

### The popover is unchanged
`ReleaseSummaryCard` still shows the burndown only for OPEN releases. For a past release the burndown would be a duplicate of the expanded row, so its popover shows the final stats (the snapshot counts) only.

### Shared issue list: `ReleaseIssues.tsx`
`ReleaseIssueList` and `IssueRow` moved out of `Releases.tsx` into `ReleaseIssues.tsx` while the detail page needed them. We kept the move after the page was dropped, because it slims `Releases.tsx`.
- The list's `readOnly` flag is replaced by two props:
  - `renderRow` swaps in draggable rows; read-only rows are the default.
  - `empty` sets the empty message.
- `DraggableIssueRow` stays in `Releases.tsx`, because it depends on that file's drag data types.

## Decisions made during review
- **Drop the page** after the first tophat ("really weak"). Read past releases in place.
- **The burndown goes in the expanded row,** not the popover, above the delivered issues.
- **Retrospective notes go at the very bottom** of the expanded row. xpo-f5e7db's title and description were updated to match.
- **The chart height went from 56px to 96px** after tophat ("quite squished").

## Acceptance criteria
- [x] Clicking a past release's title expands and collapses it; there is no detail page.
  - Evidence: `ReleaseRow.test.tsx`, "makes a past release's title a toggle for its issues, not a link".
  - Evidence: `ReleaseDetail.tsx` and the route are deleted.
  - Evidence: tophatted on the demo board.
- [x] `#/releases/N` opens the Releases list.
  - Evidence: `views.test.ts`, "opens the Releases list for any release link: releases have no page".
- [x] An expanded past release shows its burndown above the delivered issues; never-Current releases show none.
  - Evidence: `Releases.test.tsx`, "an expanded past release", which has three tests: the burndown appears above the issues, a never-Current release has none, and the burndown hides under a search.
  - Evidence: tophatted with demo release 1.0, which was backdated to run Oct 12 → Oct 20 with scope added on Oct 19.
- [x] Open releases still link to Issues with the release filter.
  - Evidence: `releases-utils.test.ts`, "releaseFilter".
  - Evidence: `ReleaseRow.test.tsx`, "keeps an open release's title a link".
- [x] `make test` passes: lint, 1402 frontend tests and the Go tests.
