# Spec: Board card assignee avatar becomes an assignee picker button

## What
On Board cards, the assignee avatar becomes a button that opens `AssigneePicker`. When the card is unassigned, it shows a muted dashed placeholder. This mirrors xpo-dcd4f6 for backlog rows.

## Why
`BoardCard` renders `{issue.assignee && <Avatar/>}`. Unassigned cards show nothing, and nobody can be assigned from a card.

## How
1. **`BoardCard.tsx`**
   - `BoardCardContent` takes an optional `assignee?: { open: boolean; onOpen: () => void }`.
     - **When given**, which is the case for `SortableBoardCard`, the avatar slot renders a `<button data-assignee-trigger>`. It uses the same `w-6 h-6 -m-1 … rounded hover:bg-white/10` treatment as the backlog row, the same `aria-label` ("Name, change assignee" / "No assignee, change assignee") and `popoverTriggerProps("assignee", …)`.
     - **When absent**, as in the `BoardCard` drag overlay, it renders a static avatar or placeholder. The overlay is a picture of the card, not something you interact with.
   - **Placeholder:** `<Avatar size="sm" />` (no name) in an `opacity-50` wrapper, the same as the backlog row.
   - **Click handling:** the trigger calls `stopPropagation` on `onClick` so the card doesn't open the issue. Dragging is unaffected: the board's `PointerSensor` has a 3px activation distance, so a click never starts a drag.
   - `SortableBoardCard` gets two new props, `assigneeOpen` and `onAssigneeOpen`, and passes them through.
2. **`BoardColumn.tsx`**: forwards `openAssigneeId` and `onAssigneeOpen(issue)` to the cards.
3. **`Board.tsx`**
   - `openPopover.type` gains `"assignee"`.
   - `knownPeople = useMemo(collectKnownPeople(issues, contributors))`.
   - **Anchor:** `openCardPopover(issueId, "assignee")` points the shared `popoverAnchorRef` at `[data-board-card="<id>"] [data-assignee-trigger]`, falling back to `boardRef` (see xpo-bfca2d below). Click and keyboard share this one path.
   - **Rendering:** `Popover anchorRef={assigneeAnchorRef}` → `PopoverPanel` → `AssigneePicker`, with `onSelect → handleQuickUpdate(id, { assignee: p ?? "" })`.
   - **Keyboard:** an `a` shortcut on the focused card opens the assignee picker, next to s/l/e, with a help entry `["a", "Set assignee"]`. This follows the decision you made for backlog rows in xpo-dcd4f6.

## Related fixes on this branch (found during tophat)
- **xpo-bfca2d:** s/e/l popovers were anchored to `boardRef`, the board's top-left corner. All board popovers now share one `popoverAnchorRef` set by `openCardPopover(issueId, type)`. The anchor comes from `cardPopoverAnchorSelector` in `board-popover.ts`: the card itself, or the card's avatar trigger for the assignee picker, falling back to the board.
- **xpo-93efdd:** keyboard focus indexed the full `COLUMNS` list, so with Backlog hidden, `j` focused cards in the invisible Backlog column. Navigation now walks `navigableColumnIds` (visible and expanded columns) and steps with `stepColumn`. `resolveFocusColumn` falls back to the first non-empty column. The helpers are in `board-nav.ts`.

## Testing
`Board/BoardCard.test.tsx` uses `renderMarkup` inside `DndContext` + `SortableContext`:
- **Unassigned card:** a trigger named "No assignee, change assignee", with `aria-haspopup="menu"`, `aria-expanded="false"` and the dashed placeholder.
- **Assigned card:** the trigger is named after the person.
- **`assigneeOpen`:** produces `aria-expanded="true"`. The popover itself lives in `Board`, not in the card, so this renders fine server-side.
- **`BoardCard` overlay:** renders no button, only the avatar or placeholder.
- **Helpers:** `board-popover.test.ts` covers the anchor selector, and `board-nav.test.ts` covers the column navigation helpers.

## Acceptance criteria
- [ ] Unassigned cards show a muted dashed placeholder avatar.
- [ ] Clicking the avatar, assigned or not, opens `AssigneePicker` anchored to it, and picking an option updates the issue.
- [ ] Clicking the avatar does not open the issue or start a drag.
- [ ] `a` on a focused card opens the assignee picker, anchored to that card's avatar, and is listed in keyboard help.
- [ ] The trigger has `aria-haspopup="menu"`, `aria-expanded` and an accessible name.
- [ ] Tests cover the placeholder and trigger markup via `renderMarkup`.
- [ ] `make test` passes.
