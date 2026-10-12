# Walkthrough: Board card assignee avatar becomes an assignee picker button

## What was wrong
`BoardCard` rendered `{issue.assignee && <Avatar/>}`. Unassigned cards showed nothing, and you couldn't assign from a card. This is the Board counterpart of xpo-dcd4f6, which did the same for backlog rows.

## What changed

### `BoardCard.tsx`
- **Content:** `BoardCardContent` takes an optional `assignee: { open, onOpen }`.
  - **When given** (sortable cards), the avatar slot is a `<button data-assignee-trigger>` with the same look as the backlog row trigger. It is named "Name, change assignee" or "No assignee, change assignee", and spreads `popoverTriggerProps("assignee", …)`.
  - **When absent** (the `BoardCard` drag overlay), the same visual renders without a button, because the overlay is just a picture of the card.
- **`AssigneeAvatar`:** renders the person's avatar, or `<Avatar size="sm" />` (the dashed placeholder) at 50% opacity.
- **Click handling:** the trigger stops click propagation so the card doesn't open the issue. A click never starts a drag, because the board's `PointerSensor` has a 3px activation distance.
- **Props:** `SortableBoardCard` gains `assigneeOpen` and `onAssigneeOpen`. `BoardColumn` forwards `openAssigneeId` and `onAssigneeOpen(issue)`.

### `Board.tsx`
- **Picker:** `knownPeople` is memoized and passed to `AssigneePicker` in the board's popover block. Selecting goes through the existing `handleQuickUpdate`, which closes the popover and saves.
- **Toggle:** clicking an open card's trigger closes the popover. This works because `Popover`'s outside-click handler ignores its anchor.
- **Keyboard:** the `a` shortcut opens the picker on the focused card and is listed in keyboard help, following the backlog decision in xpo-dcd4f6.

### Related fixes on this branch (found during tophat)
- **xpo-bfca2d (popover anchoring):** the s/e/l popovers were anchored to `boardRef`, so they appeared in the board's top-left corner.
  - Every board popover now goes through `openCardPopover(issueId, type)`. It sets one shared `popoverAnchorRef` from `cardPopoverAnchorSelector`: the card for status, estimate and labels, the avatar trigger for assignee, with the board as a fallback.
  - Clicks and shortcuts share this single path, so they can't drift apart.
- **xpo-93efdd (keyboard focus):** keyboard navigation indexed the full `COLUMNS` list. With Backlog hidden (the default), `j` focused cards in an invisible column. That showed no focus ring, sent shortcuts to the wrong issue, and made the anchor lookup fall back to the board corner.
  - `board-nav.ts` adds `navigableColumnIds`, which keeps only visible and expanded columns.
  - `stepColumn` moves to the next non-empty column for ArrowLeft/ArrowRight.
  - `resolveFocusColumn` keeps focus in the current column if it has cards, otherwise falls back to the first non-empty one.
  - `focusedIssueId` and the key handler work on that resolved column. `j` and `k` commit the fallback to state so later presses stay in that column.

## Testing
- **`Board/BoardCard.test.tsx`** renders with `renderMarkup` inside `DndContext` + `SortableContext`. It covers the unassigned placeholder trigger, the assigned trigger name, `aria-expanded="true"` from `assigneeOpen`, and the overlay rendering no trigger. The card's popover lives in `Board`, so the open state renders fine server-side.
- **`Board/board-popover.test.ts`** covers the anchor selector for each popover type.
- **`Board/board-nav.test.ts`** covers filtering hidden and collapsed columns, stepping across empty columns and stopping at the edges, and the fallback when the focused column is empty or missing.

I checked it in the browser after a hard reload:
- Clicking a card's avatar opens the picker under it with the current assignee checked, and doesn't open the issue.
- With Backlog hidden, `j` shows the focus ring on a Planned card and `→` moves to In Progress.
- `s` opens the status picker under the focused card.

## Acceptance criteria
- [x] Unassigned cards show a muted dashed placeholder avatar. Covered by the `BoardCard.test.tsx` placeholder test.
- [x] Clicking the avatar, assigned or not, opens `AssigneePicker` anchored to it, and picking an option updates the issue. Checked in the browser; selection goes through `handleQuickUpdate`.
- [x] Clicking the avatar does not open the issue or start a drag. `stopPropagation` handles the click and the 3px drag activation distance handles the drag; checked in the browser and in your tophat.
- [x] `a` on a focused card opens the assignee picker, anchored to that card's avatar, and is listed in keyboard help. Implemented through `openCardPopover`; the anchor is covered by `board-popover.test.ts`, and you checked it in tophat.
- [x] The trigger has `aria-haspopup="menu"`, `aria-expanded` and an accessible name. Asserted in `BoardCard.test.tsx`.
- [x] Tests cover the placeholder and trigger markup via `renderMarkup`, in `BoardCard.test.tsx`.
- [x] `make test` passes: 638 tests and lint, and `tsc -b` is clean.
