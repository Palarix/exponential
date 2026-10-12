# Walkthrough: backlog row assignee cell becomes an assignee picker button

## What was wrong
`BacklogIssueRow` rendered the assignee avatar only when an assignee was set (`{issue.assignee && <Avatar/>}`). After choosing "No assignee", which became easy with xpo-c8eaf4, the cell vanished and the date column shifted left. The avatar also wasn't interactive, while every other property cell in the row (priority, status, labels, estimate) opens a picker.

## What changed

### `BacklogIssueRow.tsx`
- **The cell:** the assignee cell copies the structure of the priority and status cells.
  - The wrapper is a `relative shrink-0` div. It takes the shared `popoverAnchorRef` only when `popoverType === "assignee"`, and calls `stopPropagation` so a click doesn't also open the issue.
  - Inside is a `w-6 h-6 -m-1` hover button.
- **What it shows:** `<Avatar name=… />` when assigned. When unassigned, it shows `<Avatar size="sm" />` with no name, the existing dashed-circle placeholder, wrapped in `opacity-50` so it reads as muted. The cell always takes the same space, so the column stays aligned.
- **The popover:** `Popover → PopoverPanel → AssigneePicker` (from xpo-c8eaf4), with `people={knownPeople}`, `current={issue.assignee}` and `onSelect → onQuickAssignee`.
- **Accessibility:** the trigger is named "Name, change assignee" or "No assignee, change assignee", so a screen reader hears both the current value and what the button does. It spreads `popoverTriggerProps("assignee", popoverType)` for `aria-haspopup` and `aria-expanded`.
- **Props and types:** `PopoverType` gains `"assignee"`. New props are `knownPeople` and `onQuickAssignee`.

### `Backlog.tsx`
- **People list:** `knownPeople = useMemo(collectKnownPeople(issues, contributors))` is computed once and passed to every row. The rows are `memo`ized, and the stable array reference keeps them from re-rendering because of it.
- **Handler:** `handleQuickAssignee(issueId, person | null)` closes the popover, calls `updateIssue({ assignee: person ?? "" })`, and shows a toast: "Assigned to Name" or "Assignee removed". `AssigneePicker` already skips `onSelect` when the choice is unchanged, so re-picking the current value doesn't trigger a save.
- **Shortcut:** the `a` row shortcut opens the assignee popover on the focused issue row, next to s/l/e/p, and is listed in keyboard help as "Set assignee".

### `utils/popover-trigger.ts` (moved)
`popoverTriggerProps` moved here from `IssueDetail/` because two features now use it. Its unit test moved with it to `utils/popover-trigger.test.ts`.

## Testing
`Backlog/BacklogIssueRow.test.tsx` renders the row with `renderMarkup` inside a `DndContext`, because the row uses dnd-kit hooks. It checks:
- **Unassigned:** the trigger's name, `aria-haspopup`, `aria-expanded="false"` and the dashed placeholder.
- **Assigned:** the trigger is named after the person and has no placeholder.

The open state can't be rendered server-side, because `Popover` portals into `document.body`. It is covered by the shared `popoverTriggerProps` test instead.

## Acceptance criteria
- [x] Unassigned rows show a muted dashed-circle avatar in the assignee column, and the column stays aligned. The placeholder test asserts the dashed placeholder; alignment was checked in your tophat.
- [x] Clicking the avatar, assigned or not, opens `AssigneePicker` anchored to it. The trigger toggles `onOpenPopover(rowIndex, "assignee")`; checked in tophat.
- [x] Picking a person or "No assignee" updates the issue and closes the popover, via `handleQuickAssignee`; checked in tophat.
- [x] `a` on a focused row opens the assignee picker and is listed in keyboard help. It was added to the backlog keyboard handler and shortcut list (your call); checked in tophat.
- [x] The trigger exposes `aria-haspopup="menu"`, `aria-expanded` and an accessible name. Asserted in `BacklogIssueRow.test.tsx`; the expanded state is covered by `popover-trigger.test.ts`.
- [x] Tests cover the placeholder and trigger markup via `renderMarkup`, in `BacklogIssueRow.test.tsx`.
- [x] `make test` passes: 626 tests, and `tsc -b` is clean.

## Follow-ups
- xpo-5afd82: the same treatment for Board cards.
