# Context Menu Assignee filter: autofocus + consistent styling

## What
The Assignee submenu in `ContextMenu` should autofocus its filter input on open and look/behave like the other picker submenus (Status, Priority, Labels).

## Why
`AssigneePanel` (`web/src/components/ui/ContextMenu.tsx`) is hand-rolled markup rather than built on the shared `Menu` primitives:

1. **Autofocus fails** — it relies on the `autoFocus` attribute. `SubMenu` portals the flyout with `visibility: hidden` and only reveals it in a layout effect after positioning. The browser can't focus an element inside a `visibility: hidden` container at mount, so focus is dropped. `MenuFilter` avoids this by focusing in a `requestAnimationFrame` after the flyout becomes visible.
2. **Styling drift** — custom `px-3` buttons, `text-primary` default text, its own scroll container and empty-state div, versus `MenuItem`'s `px-2 rounded-md text-secondary` with focus/hover states. It also gets no arrow-key navigation, Enter-to-select, or ArrowDown-from-filter behaviour that `Menu` provides.

## How
Rewrite `AssigneePanel` on the same pattern as `LabelPicker`:

```tsx
<Menu onClose={onClose} bare autoFocus={false} maxHeight="18rem">
  <MenuFilter value={filterText} onChange={setFilterText} placeholder="Set assignee..." />
  {issue.assignee && !hasQuery && <MenuItem label="Remove assignee" onClick=... />}
  {people.map(p => <MenuItem label={displayName(p)} icon={<Avatar name={p} size="sm" />} checked={p === issue.assignee} onClick=... />)}
  {people.length === 0 && <MenuLabel>No matching people</MenuLabel>}
</Menu>
```

- `Menu`'s own Escape handling replaces the bespoke `onKeyDown` wrapper. (Behaviour change, approved: Escape with text in the filter now closes the submenu rather than first clearing the filter — matching Labels.)
- Extract the filter logic into a pure, unit-tested helper `filterPeople(people, query)` in `web/src/utils/issues.ts` (next to `collectKnownPeople`), case-insensitive substring match on the full `Name <email>` string, query trimmed, empty/whitespace query returns all.
- `maxHeight="18rem"` on the `Menu` so long people lists scroll (previous list was `max-h-60` = 15rem; the extra ~3rem accounts for the filter row, which now scrolls with the list).

## Out of scope
`CyclePanel` has similar hand-rolled markup — tracked in xpo-70e66a.

## Acceptance Criteria
- [ ] Opening Context Menu → Assignee focuses the filter input immediately (typing filters without clicking).
- [ ] Assignee panel items use `MenuItem` styling identical to Status/Priority/Labels (padding, rounded hover, checkmark for current assignee).
- [ ] ArrowDown from the filter moves into the list; Enter selects; Escape closes the submenu.
- [ ] "Remove assignee" shown only when issue has an assignee and filter is empty.
- [ ] "No matching people" shown when filter matches nobody.
- [ ] `filterPeople` has unit tests; `make test` passes.
