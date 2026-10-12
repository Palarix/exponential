# Walkthrough: Board keyboard focus only visits visible columns

## Cause
Board keyboard navigation stored `focusCol` as an index into the full `COLUMNS` list, and ArrowLeft/ArrowRight walked `COLUMNS` too. The board only renders `visibleColumns`, which drops `hiddenColumns`. Backlog is usually hidden, so the default `focusCol = 0` pointed at Backlog. As a result:
- `j` focused an invisible card, so no focus ring appeared.
- `e`, `s`, `l`, `a` and Enter acted on that hidden issue.
- Popover anchoring fell back to the board corner (see xpo-bfca2d).

## Fix
Fixed on the xpo-5afd82 branch and merged with it. The new pure helpers are in `Board/board-nav.ts`:
- `navigableColumnIds(columns, hidden, collapsed)` keeps the ids that are neither hidden nor collapsed, in board order.
- `stepColumn(colIds, from, dir, containers)` returns the next column in `dir` that has cards, or `from` when there is none.
- `resolveFocusColumn(colIds, focusCol, containers)` returns `focusCol` while it has cards, otherwise the first non-empty column, or -1 for an empty board.

In `Board.tsx`:
- `navColIds` is memoized from the hidden and collapsed sets, and `navFocusCol` is resolved on every render.
- `focusedIssueId` reads from that resolved column and clamps the card index.
- `j` and `k` commit the resolved column to state. ArrowLeft/ArrowRight use `stepColumn`.
- `focusCol` is now an index into the navigable columns. Collapsing or hiding a column can't strand focus, because the resolve step re-runs on every render.

## Acceptance criteria
- [x] With Backlog hidden, `j` focuses the first card of the first visible non-empty column, with the focus ring. Checked in the browser after a hard reload.
- [x] ArrowLeft/ArrowRight never land on a hidden or collapsed column. Only `navigableColumnIds` are walked; covered by `board-nav.test.ts` and checked in the browser.
- [x] There are unit tests for the column-navigation helpers: `board-nav.test.ts` has 6 tests.
