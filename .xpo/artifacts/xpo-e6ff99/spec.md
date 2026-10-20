# Spec: Releases — past releases expand in place (no detail page)

> **Read the spec of the parent epic xpo-4c92d4 first.** The backend comes from xpo-f89e55 and the Releases view from xpo-9a7a77.

## What
Drop the release detail page. A past release is read where it already is, in the Releases view: clicking its title expands it, and the expanded row is the release's record: its burndown, then the issues it delivered. Retrospective notes (xpo-f5e7db) will go at the very bottom, below the issues.

## Why
A first detail page (header, final stats, burndown, issue list) was built and tophatted. It added nothing the Releases view didn't already show: the expanded row lists the issues, the popover breaks down the stats. A dedicated page only earns its place once it tells the release's story (scope changes, timing, release notes). That needs backend work and is revisited if it becomes an issue.

## Decisions
- **No release detail page and no `#/releases/N` route.** The route, `releaseNumber` in the app's nav state, `findRelease` and `ReleaseDetail.tsx` are removed. `#/releases/7` opens plain Releases. No fallback for old links (they were never released).
- **Clicking a past release's title toggles its expansion**, like the chevron, and carries `aria-expanded`. It is not underlined on hover, since it is not a link. Clicking an open release still opens Issues with its release filter (xpo-325f73).
- **`releaseLink` becomes `releaseFilter`:** the filter an open release opens, or null for a past release.
- **The expanded past release shows its burndown above the delivered issues**, from `started_at` to `released_at`, with both dates under the chart and a legend (Open, Scope). A release that was never Current, or was started and released on the same day, shows no burndown and no message: the row stays lean.
- **The burndown hides while the search or filters narrow the lists**, so the matching issues stay in focus.
- **The progress popover is unchanged:** a past release's popover shows its final stats, not the burndown (the expanded row has it).
- **`ReleaseIssueList` and `IssueRow` move to `ReleaseIssues.tsx`** (kept from the first attempt). The list takes `renderRow` for draggable rows and an `empty` message instead of a `readOnly` flag.

## Acceptance criteria
- [ ] Clicking a past release's title expands and collapses it; there is no detail page.
- [ ] `#/releases/N` opens the Releases list.
- [ ] An expanded past release shows its burndown above the delivered issues; never-Current releases show none.
- [ ] Open releases still link to Issues with the release filter.
- [ ] `make test` passes.
