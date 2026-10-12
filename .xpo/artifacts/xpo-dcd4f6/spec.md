# Spec: Backlog row assignee cell becomes an assignee picker button

## What
In `BacklogIssueRow`, the assignee avatar becomes a button that opens `AssigneePicker`, like the priority, status, labels and estimate cells. When nobody is assigned, it shows a muted dashed-circle placeholder instead of nothing.

## Why
- Setting "No assignee" makes the avatar disappear, which shifts the date column. There is also no way to assign from the row.
- Every other property cell in the row is already a picker trigger.

## How
1. **`BacklogIssueRow.tsx`**
   - Add `"assignee"` to `PopoverType`.
   - Replace `{issue.assignee && <Avatar …/>}` with a cell structured like the priority and status cells: a `relative shrink-0` wrapper div with the anchor ref when `popoverType === "assignee"`, and `onClick={e => e.stopPropagation()}`.
   - The wrapper holds a `<button>` styled like the priority and status triggers (`w-6 h-6 -m-1 … rounded hover:bg-white/10`).
     - **Assigned:** the button contains `<Avatar name={issue.assignee} size="sm" />`.
     - **Unassigned:** it contains `<Avatar size="sm" />` (the existing dashed placeholder) wrapped in `opacity-50`, so the placeholder reads as muted.
   - Clicking toggles the popover. The popover is `Popover → PopoverPanel → AssigneePicker`, with `people={knownPeople}`, `current={issue.assignee}`, `onSelect={p => onQuickAssignee(issue.id, p)}` and `onClose={onClosePopover}`.
   - New props: `knownPeople: string[]` and `onQuickAssignee(issueId, person | null)`.
   - The button gets `aria-label` (the assignee's short name, or "No assignee", plus ", change assignee") and spreads `popoverTriggerProps("assignee", popoverType)`. That helper moved from `IssueDetail/popover-trigger.ts` to `utils/popover-trigger.ts` so both features share it; its test moved to `utils/popover-trigger.test.ts`.
2. **`Backlog.tsx`**
   - Widen the `openPopover` / `handleOpenPopover` type unions with `"assignee"`.
   - Compute `knownPeople = useMemo(() => collectKnownPeople(issues, contributors), …)` once and pass it to the rows. It's a stable reference, so the memoized rows don't re-render because of it.
   - Add `handleQuickAssignee`: `setOpenPopover(null)`, then `updateIssue(id, { assignee: person ?? "" })` with a toast ("Assigned to Name" / "Assignee removed").
   - **Keyboard:** add an `a` row shortcut that opens the assignee popover on the focused issue row, matching s/l/e/p, plus a help entry `["a", "Set assignee"]`.

## Testing
`Backlog/BacklogIssueRow.test.tsx` renders the row with `renderMarkup` inside a `DndContext`:
- **Unassigned:** the trigger has `aria-label="No assignee, change assignee"`, `aria-haspopup="menu"` and `aria-expanded="false"`, and contains the dashed placeholder.
- **Assigned:** the trigger is labeled with the person's short name and has no placeholder.
- **Open state:** rendering it portals `Popover`, which fails server-side. So it is covered by `utils/popover-trigger.test.ts`, which the row reuses.

## Out of scope
- **Board cards:** `BoardCard` renders the avatar the same way (`issue.assignee && <Avatar/>`). It is filed as follow-up xpo-5afd82 (your decision).

## Acceptance criteria
- [ ] Unassigned rows show a muted dashed-circle avatar in the assignee column, and the column stays aligned.
- [ ] Clicking the avatar, assigned or not, opens `AssigneePicker` anchored to it.
- [ ] Picking a person or "No assignee" updates the issue and closes the popover.
- [ ] (Decision: confirmed by you) `a` on a focused row opens the assignee picker, and it is listed in keyboard help.
- [ ] The trigger exposes `aria-haspopup="menu"`, `aria-expanded` and an accessible name.
- [ ] Tests cover the placeholder and trigger markup via `renderMarkup`.
- [ ] `make test` passes.
