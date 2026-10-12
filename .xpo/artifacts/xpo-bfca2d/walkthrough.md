# Walkthrough: Board s/e/l popovers anchor to the focused card

## Cause
`Board.tsx` rendered the status, estimate and labels popovers with `<Popover anchorRef={boardRef}>`, so `computePopoverPosition` placed them against the whole board container: its top-left corner.

## Fix
Fixed on the xpo-5afd82 branch and merged with it.
- **One path for every popover:** `openCardPopover(issueId, type)` replaces the separate `setOpenPopover({ … })` calls in the keyboard handler and the assignee click handler. It sets one shared `popoverAnchorRef`, then opens the popover.
- **Choosing the anchor:** `cardPopoverAnchorSelector(issueId, type)` in `Board/board-popover.ts` returns `[data-board-card="<id>"]`, or the card's `[data-assignee-trigger]` for the assignee picker. The ref is the element that selector finds inside the board, with `boardRef` only as a fallback.
- **Rendering:** all four popovers render with `anchorRef={popoverAnchorRef}`.

The visible symptom also depended on xpo-93efdd. Keyboard focus could point at a card in the hidden Backlog column, which has no element to anchor to, so the fallback kicked in.

## Acceptance criteria
- [x] `s`, `e` and `l` open their popovers anchored to the focused card. Checked in the browser (`s` on a focused In Progress card opens under it) and in your tophat; the selector is unit-tested in `board-popover.test.ts`.
- [x] `a` still anchors to the card's avatar. The assignee case is covered in `board-popover.test.ts`.
