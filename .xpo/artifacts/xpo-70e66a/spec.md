# Context Menu Cycle submenu: filter input + shared Menu primitives

## What
Give the Cycle submenu in the issue context menu a filter input at the top, and rebuild it on the shared `Menu` primitives so it matches Status, Priority, Labels and Assignee.

## Why
`CyclePanel` (`web/src/components/ui/ContextMenu.tsx`) is hand-rolled:
- it has a static "Move to cycle..." header instead of a filter input
- its buttons use custom `px-3` styling, `text-primary` text and an inline status span
- it has no arrow-key navigation or Enter-to-select, and uses a bespoke Escape handler

xpo-690930 fixed the same problem for Assignee; this follows that pattern.

## How
```tsx
<Menu onClose={onClose} bare autoFocus={false} maxHeight="18rem">
  <MenuFilter value={filterText} onChange={setFilterText} placeholder="Move to cycle..." />
  {issue.cycle_id && !hasQuery && <MenuItem label="No cycle" onClick=... />}
  {visible.map(c => <MenuItem key={c.id} label={cycleLabel(c)} checked={c.id === issue.cycle_id} onClick=... />)}
  {visible.length === 0 && <MenuLabel>No matching cycles</MenuLabel>}
</Menu>
```

- **New helpers** in a new file, `web/src/utils/cycles.ts`, with tests in `cycles.test.ts`:
  - `cycleLabel(cycle)` returns `Cycle <number> (<status>)`, e.g. `Cycle 12 (current)`.
  - `filterCycles(cycles, query)`:
    - excludes `completed` cycles (as the panel does today)
    - matches case-insensitively against `cycleLabel`, so `12`, `cycle 12` and `current` all work
    - trims the query; an empty or whitespace-only query returns all non-completed cycles
- **Status goes in the label**, not the `suffix`. The checkmark is reserved for the issue's selected cycle, and `MenuItem` would hide a suffix on the checked row. (User decision.)
- **No cycles configured.** When cycles are disabled or the list is empty, keep the "Cycles not configured" message, rendered as a `MenuLabel` inside a bare `Menu`, so Escape still closes it.
- **Escape** closes the submenu (`Menu`'s handler), the same as Assignee and Labels.

## Out of scope
The cycle popover in `IssueDetail/PropertySidebar.tsx` has the same hand-rolled markup. It is tracked separately.

## Acceptance Criteria
- [ ] The Cycle submenu shows a filter input that is focused on open.
- [ ] Typing filters cycles by number or status; "No matching cycles" is shown when nothing matches.
- [ ] Items use `MenuItem` styling with the label `Cycle N (status)`; a checkmark appears only on the issue's selected cycle.
- [ ] "No cycle" is shown only when the issue has a cycle and the filter is empty.
- [ ] Completed cycles are never listed.
- [ ] ArrowDown from the filter moves into the list; Enter selects; Escape closes.
- [ ] `cycleLabel` and `filterCycles` have unit tests; `make test` passes.
