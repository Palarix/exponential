# Release issue rows show labels, estimate and assignee

## What changed
Expanded releases in the Releases view used to list issues as ID, status, parent breadcrumb, title and a sub-issue pill pushed to the far right. Labels, estimate and assignee were only in the hover card, so planning a release meant hovering every row. The rows now follow the backlog's layout:

`ID · Status · [Parent ›] Title · [sub-issue pill] · <spacer> · Labels · Estimate · Assignee`

The view also drops its `max-w-7xl` column and goes full width, as the backlog does, to make room.

## How it fits together
- **`IssueRow`** (`web/src/components/Releases/ReleaseIssues.tsx`) is the only row component. Open releases wrap it in `DraggableIssueRow` and past releases use it directly, so both get the new columns. The title no longer takes `flex-1`. The pill follows it directly, and a `flex-1` spacer pushes the three new columns right.
- **Display pieces are shared, the row is not.** Labels use `OverflowLabels`, the estimate uses `EstimateBadge` in the backlog's `w-10` right-aligned box, and the assignee uses `Avatar size="sm"`, faded to 50% when empty. `BacklogIssueRow` itself is not reused: it is built around the backlog's popovers, keyboard navigation and tree.
- **Read-only on purpose.** There are no buttons, popovers or `stopPropagation` around the new columns. A click anywhere on the row opens the issue, and a drag can start anywhere on it, exactly as before. The only button in the row is still `CopyableId`.
- **`data-backlog-row`.** `OverflowLabels` finds its row with `closest("[data-backlog-row]")` and fits the labels to 35% of that row's width, showing `+N` for the rest. Without the attribute it would show every label uncapped. The Releases view's DOM is separate from the backlog's, so the backlog's keyboard navigation (which queries this attribute inside `[data-context-issue]`) is unaffected.
- **`rowEstimate(issue, all)`** (`releases-utils.ts`) reproduces the backlog's parent estimate. A parent shows its open child points, or its done child points once it is closed. A child with no estimate counts as 1. An issue without children shows its own estimate. It is a separate function rather than an extension of `childProgress`, because the hover card also uses `childProgress` and needs only counts.
- **Full width.** `Releases.tsx` passes the padding as `contentClassName` instead of `maxWidth="7xl"` + `innerClassName`.

## Decisions
- No "+ label" hover hint on rows without labels: in the backlog it is the trigger for the label picker, and these rows have no picker.
- Cycle, created date and the branch, chat, attachment and release indicators stay backlog-only. The goal was who does what and how big it is, not a full copy of the backlog row.

## Acceptance criteria
- [x] Release issue rows show labels, estimate and assignee on the far right, in that order, using `OverflowLabels`, `EstimateBadge` and `Avatar`. Test: "shows issue rows in the backlog's order: ID, status, title, then labels, estimate and assignee, no priority".
- [x] They are read-only: no pickers, the row click opens the issue. Test: "shows labels, estimate and assignee read-only: no pickers in the row" (the only `<button>` is the copyable ID; no `aria-haspopup`).
- [x] A parent's estimate shows its remaining child points, as in the backlog. Tests: `rowEstimate` unit tests (own estimate, open parent = 6 remaining, closed parent = 3 done), plus the Courier app row showing 3.
- [x] The sub-issue progress pill sits right after the title. Test: "puts a parent's child progress right after its title, before its remaining child points" (title < pill < spacer < estimate).
- [x] The Releases view is full width. `maxWidth` removed; checked by the user on the demo board.
- [x] Unchanged: click opens the issue, hover card, drag-and-drop between open releases, past releases read-only. The existing tests "keeps the issues of open releases draggable" and "lists what a past release delivered, read-only" still pass; the user tried the rows on the demo board.
- [x] Tests cover the new columns, their order, the parent estimate, and the pill position; `make test` passes (1554 frontend tests, Go tests, lint).
