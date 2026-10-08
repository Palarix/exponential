# Walkthrough: Issue detail Cycle popover uses shared CyclePicker

## What was built
The cycle popover in the issue detail sidebar now uses a shared `CyclePicker` component. The context menu's Cycle submenu uses the same component. Both surfaces now have the same filter, labels, selection indicator and keyboard navigation.

## How the pieces fit
- **`web/src/components/ui/CyclePicker.tsx` (new).** This is the former `CyclePanel` from `ContextMenu.tsx`, made standalone so it no longer depends on `Issue` or the context menu's action callback.
  - Props: `{ cycles, current?, onSelect(cycleId), onClose? }`. `onSelect("")` means "No cycle".
  - It is built from `Menu bare autoFocus={false}`, `MenuFilter`, `MenuItem` and `MenuLabel`, using `filterCycles` and `cycleLabel` from `utils/cycles.ts` (both added in xpo-70e66a).
  - After a selection it calls `onSelect` and then `onClose`, the same as `StatusPicker` and `PriorityPicker`.
  - It is exported from `ui/index.ts` next to the other pickers.
- **`ContextMenu.tsx`.** `CyclePanel` is deleted, and the Cycle `SubMenu` renders `CyclePicker`, mapping `onSelect` to `handleAction("UPDATE", { cycle_id })`. There is no behaviour change.
- **`PropertySidebar.tsx`.** The hand-rolled popover body (header, buttons, inline checkmark SVG) is replaced with `Popover` → `PopoverPanel` → `CyclePicker`. The Labels popover uses the same wrapper, and `Board.tsx` already shows the shared pickers inside `Popover` the same way. The existing `handleCycleChange` already treats an empty or null value as "No cycle", so it is passed straight through.
- **`IssueDetail.tsx`.** The sidebar keyboard handler drives the hand-rolled popovers (Status, Estimate, Priority) through `popoverIndex` and calls `preventDefault` on ArrowUp, ArrowDown and Enter for any open popover. It already skipped `labels`; it now also skips `cycle`, so `Menu`'s own navigation isn't overridden.

## Why it matters
Before this change the context menu and the sidebar each had their own copy of the cycle UI, and the two had drifted apart: the sidebar had no filter, a different checkmark and different padding. With one component, future changes only need to be made once.

## Gotcha for future migrations
Any sidebar popover moved onto a `Menu`-based picker must be added to the early return in `IssueDetail.tsx`'s `handleKeyboard` (`if (openPopover === "labels" || openPopover === "cycle") return;`). Otherwise the `popoverIndex` logic swallows the arrow keys. This applies to Status, Estimate, Priority, Assignee and Parent, which are still hand-rolled, if they are migrated later.

## Acceptance Criteria
- [x] The sidebar Cycle popover shows a focused "Move to cycle..." filter; typing filters by number or status. Evidence: `CyclePicker` uses `MenuFilter` and `filterCycles`; the user verified it.
- [x] Rows read `Cycle N (status)` in `MenuItem` styling; the checkmark appears only on the issue's cycle. Evidence: `cycleLabel`, `checked={c.id === current}`; the user verified it.
- [x] "No cycle" is shown only when the issue has a cycle and the filter is empty; "No matching cycles" when nothing matches; completed cycles are never listed. Evidence: `current && !hasQuery` guard; `MenuLabel` empty state; `filterCycles` tests.
- [x] Arrow keys and Enter work inside the popover without interference; Escape closes. Evidence: `cycle` added to the `handleKeyboard` early return; `Menu` and `Popover` both close on Escape.
- [x] The context menu Cycle submenu behaves as before via `CyclePicker`. Evidence: same component body; the user verified it.
- [x] `make test` passes. Evidence: exit 0, 415 frontend tests.
