# Walkthrough: Context Menu Cycle submenu filter input + shared Menu primitives

## What was built
The Cycle submenu in the issue context menu (`CyclePanel` in `web/src/components/ui/ContextMenu.tsx`) now has a filter input. It is rebuilt on the shared `Menu` / `MenuFilter` / `MenuItem` / `MenuLabel` primitives, completing the consistency pass that xpo-690930 started for Assignee. All six property submenus (Status, Priority, Assignee, Labels, Estimate, Cycle) now share styling and keyboard behaviour.

## How the pieces fit
```tsx
<Menu onClose={onClose} bare autoFocus={false} maxHeight="18rem">
  <MenuFilter … placeholder="Move to cycle..." />
  {issue.cycle_id && !hasQuery && <MenuItem label="No cycle" … />}
  {visible.map(c => <MenuItem label={cycleLabel(c)} checked={c.id === issue.cycle_id} … />)}
  {visible.length === 0 && <MenuLabel>No matching cycles</MenuLabel>}
</Menu>
```
- `MenuFilter` focuses on the next animation frame, after `SubMenu` has made the flyout visible. The `autoFocus` attribute would not work here: the flyout is still `visibility: hidden` when the input mounts, so the browser drops the focus. See the xpo-690930 walkthrough.
- `bare` and `autoFocus={false}` follow the same pattern as `LabelPicker` and the Assignee panel.
- `Menu` provides arrow-key navigation, Enter-to-select and Escape-to-close, which replace the old bespoke `onKeyDown` wrapper.
- The `useState`/`useMemo` hooks sit before the "Cycles not configured" early return, so hook order is stable.

## New helpers: `web/src/utils/cycles.ts`
- `cycleLabel(cycle)` returns `Cycle <n> (<status>)`. It is both the display text and the text the filter matches, so what you see is what you can search.
- `filterCycles(cycles, query)` always drops `completed` cycles, then does a case-insensitive substring match on `cycleLabel` with a trimmed query. An empty query returns all open cycles.

Both are pure and covered by 8 tests in `cycles.test.ts`. They are intended for reuse by xpo-f16b80, the issue detail sidebar's cycle popover.

## Decisions
- **Status goes in the label, not in `MenuItem`'s `suffix`.** The user's rule is that the checkmark is always a selection indicator. `MenuItem` hides the suffix on a checked row, so a suffix would mean the selected cycle loses its status. Putting status in the label keeps it visible on every row, and the checkmark means only "this issue is in this cycle".
- **"Cycles not configured" is a `MenuLabel` inside a bare `Menu`**, not a raw `div`, so Escape still closes the submenu through `Menu`'s handler.
- **`maxHeight="18rem"`**, matching the Assignee panel.
- **The now-unused `Check` import from lucide was removed.**

## Acceptance Criteria
- [x] The Cycle submenu shows a filter input that is focused on open. Evidence: `MenuFilter` rAF focus; the user verified it in the app.
- [x] Typing filters cycles by number or status; "No matching cycles" is shown when nothing matches. Evidence: `filterCycles` tests for number, label and status; `MenuLabel` empty state.
- [x] Items use `MenuItem` styling with the label `Cycle N (status)`; a checkmark appears only on the selected cycle. Evidence: `cycleLabel` test; `checked={c.id === issue.cycle_id}`; the user verified it.
- [x] "No cycle" is shown only when the issue has a cycle and the filter is empty. Evidence: `issue.cycle_id && !hasQuery` guard.
- [x] Completed cycles are never listed. Evidence: test "never returns completed cycles even when they match".
- [x] ArrowDown from the filter moves into the list; Enter selects; Escape closes. Evidence: `Menu` keyboard handler, the same as Labels and Assignee.
- [x] `cycleLabel` and `filterCycles` have unit tests; `make test` passes. Evidence: 8 new tests; `make test` exit 0 (415 frontend tests).
