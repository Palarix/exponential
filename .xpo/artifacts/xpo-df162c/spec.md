# Release issue rows show labels, estimate and assignee

## What
Issue rows in the Releases view (`IssueRow` in `web/src/components/Releases/ReleaseIssues.tsx`) get the backlog's right-hand columns: **Labels, Estimate, Assignee**, read-only. The sub-issue pill moves next to the title. The Releases view goes full width.

## Why
Today labels, estimate and assignee are only in the hover card. Planning a release means knowing who does what and how big it is, at a glance.

## How
**Row layout** (left → right), matching `BacklogIssueRow`:

`ID · Status · [Parent ›] Title · [sub-issue pill] · <spacer> · Labels · Estimate · Assignee`

- Title block: the title no longer takes `flex-1`; it shrinks (`min-w-0 truncate`) and the pill follows it directly, `shrink-0`. A `flex-1` spacer comes after the pill, as in the backlog.
- **Labels**: `<OverflowLabels labels={issue.labels} />` when there are labels; nothing otherwise (the backlog's "+ label" hover hint is for its picker, so it is left out). The row carries `data-backlog-row`, the hook `OverflowLabels` measures to fit labels to 35% of the row width.
- **Estimate**: `<EstimateBadge>` in a `w-10 justify-end` box, as in the backlog. For parents the value is the backlog's: remaining child points (points done if the parent is terminal), each child counting its estimate or 1. A new `rowEstimate(issue, all)` in `releases-utils.ts` computes it.
- **Assignee**: `<Avatar name={issue.assignee} size="sm" />` (its `title` names the assignee); unassigned shows the empty avatar at 50% opacity with the title "No assignee", as in the backlog. No button: nothing to click.
- No popovers, buttons or `stopPropagation`: a click anywhere still opens the issue, and dragging still works from anywhere on the row.
- Out of scope: cycle, created date, branch/chat/attachment indicators, release chip.

**Full width**: drop `maxWidth="7xl"` from the Releases `ViewContainer` (the backlog has none), keeping the `px-6 pt-5 pb-10` padding on the content.

**Docs**: update `IssueRow`'s doc comment ("compact for the half-width lists" is stale).

## Acceptance criteria
- [ ] Release issue rows show labels, estimate and assignee on the far right, in that order, using `OverflowLabels`, `EstimateBadge` and `Avatar`.
- [ ] They are read-only: no pickers, the row click opens the issue.
- [ ] A parent's estimate shows its remaining child points, as in the backlog.
- [ ] The sub-issue progress pill sits right after the title.
- [ ] The Releases view is full width.
- [ ] Unchanged: click opens the issue, hover card, drag-and-drop between open releases, past releases read-only.
- [ ] Tests cover the new columns, their order, the parent estimate, and the pill position; `make test` passes.
