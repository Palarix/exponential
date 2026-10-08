# Issue detail Cycle popover: shared CyclePicker

## What
Replace the hand-rolled cycle popover in the issue detail sidebar (`web/src/components/IssueDetail/PropertySidebar.tsx`) with the same filterable, keyboard-navigable picker the context menu now uses (xpo-70e66a).

## Why
The sidebar popover has:
- a static "Move to cycle..." header and no filter
- custom `px-3 py-2` buttons
- an inline checkmark SVG plus accent-coloured text for the selected cycle
- status in a separate span
- no arrow-key navigation

It looks and behaves differently from the context menu's Cycle submenu.

## How
1. **Extract `CyclePicker`** into `web/src/components/ui/CyclePicker.tsx`, alongside `StatusPicker`, `PriorityPicker` and `LabelPicker`. The body is today's `CyclePanel` from `ContextMenu.tsx`:
   - props: `{ cycles: Cycle[]; current?: string; onSelect: (cycleId: string) => void; onClose?: () => void }`, where `""` means "No cycle"
   - `Menu bare autoFocus={false} maxHeight="18rem"` + `MenuFilter` + `MenuItem label={cycleLabel(c)} checked=…` + `MenuLabel` empty states, using `filterCycles` and `cycleLabel` from `utils/cycles.ts`
2. **`ContextMenu.tsx`**: delete `CyclePanel` and render `<CyclePicker … onSelect={id => handleAction("UPDATE", { cycle_id: id })} />`. There is no behaviour change in the context menu.
3. **`PropertySidebar.tsx`**: render `<Popover><PopoverPanel><CyclePicker current={issue.cycle_id} onSelect={handleCycleChange} onClose={() => setOpenPopover(null)} /></PopoverPanel></Popover>`, the same wrapper the Labels popover already uses. `handleCycleChange` already treats `""`/null as "No cycle".
4. **`IssueDetail.tsx` keyboard handler**: it currently intercepts ArrowUp/ArrowDown/Enter for any open popover except `labels`. Add `cycle` to that early return so `Menu`'s own navigation isn't fought by the sidebar's `popoverIndex` logic.

The `cycleLabel`/`filterCycles` helpers are already unit-tested. `CyclePicker` is a thin composition of tested primitives; the project has no DOM test setup.

## Out of scope
The sidebar's Status, Estimate, Priority, Parent and Assignee popovers are hand-rolled the same way. They are not touched here.

## Acceptance Criteria
- [ ] Sidebar Cycle popover shows a focused "Move to cycle..." filter; typing filters by number or status.
- [ ] Rows read `Cycle N (status)` in `MenuItem` styling; the checkmark appears only on the issue's cycle.
- [ ] "No cycle" is shown only when the issue has a cycle and the filter is empty; "No matching cycles" when nothing matches; completed cycles are never listed.
- [ ] Arrow keys and Enter work inside the popover without the sidebar handler interfering; Escape closes.
- [ ] Context menu Cycle submenu behaves exactly as before, now via `CyclePicker`.
- [ ] `make test` passes.
