# Walkthrough: Context Menu Assignee filter autofocus + consistent styling

## What was built
The Assignee submenu in the issue context menu (`web/src/components/ui/ContextMenu.tsx`) was rebuilt on the shared `Menu` primitives (`Menu`, `MenuFilter`, `MenuItem`, `MenuLabel`) instead of hand-rolled markup. The filter now autofocuses when the submenu opens, and the panel looks and behaves like the Status, Priority and Labels submenus.

## Why the autofocus was broken
`SubMenu` renders its flyout in a portal with `visibility: hidden`. A layout effect positions it and only then sets it to `visible`. The old panel used the input's `autoFocus` attribute, which fires on mount, while the container is still hidden. Browsers refuse to focus elements inside `visibility: hidden`, so the focus was silently dropped.

`MenuFilter` handles this by focusing inside `requestAnimationFrame`, which runs after the layout effect has made the flyout visible. Using `MenuFilter` fixes the bug without special handling in `AssigneePanel`.

Lesson for future panels: anything rendered through `SubMenu.renderPanel` should not rely on the `autoFocus` attribute. Use `MenuFilter`, or defer focus by one frame.

## How the pieces fit
```tsx
<Menu onClose={onClose} bare autoFocus={false} maxHeight="18rem">
  <MenuFilter … placeholder="Set assignee..." />
  {issue.assignee && !hasQuery && <MenuItem label="Remove assignee" … />}
  {people.map(p => <MenuItem label={name} icon={<Avatar/>} checked={p === issue.assignee} … />)}
  {people.length === 0 && <MenuLabel>No matching people</MenuLabel>}
</Menu>
```
- `bare` drops `Menu`'s own chrome, because `SubMenu`'s flyout already supplies the surface, border and shadow.
- `autoFocus={false}` stops `Menu` from moving focus to the first item, so the filter input keeps focus. `LabelPicker` does the same.
- `Menu` provides arrow-key navigation, Enter-to-select, ArrowDown from the filter into the list, and Escape-to-close. These replace the old bespoke `onKeyDown` wrapper.
- `checked` on `MenuItem` renders the standard accent checkmark, which replaces the old lucide `Check` plus accent text colour.

## Filtering
The filter logic moved into a pure helper, `filterPeople(people, query)`, in `web/src/utils/issues.ts` next to `collectKnownPeople`:
- the query is trimmed and lower-cased
- an empty or whitespace-only query returns everyone
- it does a substring match on the full `Name <email>` string, so typing an email domain works too

It is covered by 6 unit tests in `issues.test.ts`.

## Decisions
- **Escape closes immediately.** Previously, Escape with text in the filter cleared the text first. It now closes the submenu, the same as Labels. The user approved this.
- **`maxHeight="18rem"`.** The old list was capped at 15rem. The extra ~3rem accounts for the filter row, which now sits inside the scroll container and scrolls with the list.
- **The Cycle submenu was left alone.** It has the same hand-rolled pattern; that work is tracked in xpo-70e66a.

## Acceptance Criteria
- [x] Opening Context Menu → Assignee focuses the filter input immediately. Evidence: it uses `MenuFilter`'s rAF focus; the user verified it in the app.
- [x] Assignee items use `MenuItem` styling identical to Status/Priority/Labels. Evidence: rendered via `MenuItem` with `checked`; the user verified it.
- [x] ArrowDown from the filter moves into the list; Enter selects; Escape closes. Evidence: provided by `Menu`'s keyboard handler, the same as `LabelPicker`.
- [x] "Remove assignee" is shown only when the issue has an assignee and the filter is empty. Evidence: `issue.assignee && !hasQuery` guard.
- [x] "No matching people" is shown when the filter matches nobody. Evidence: `MenuLabel` rendered when `people.length === 0`.
- [x] `filterPeople` has unit tests and `make test` passes. Evidence: 6 new tests; `make test` exit 0 (407 frontend tests passing).
