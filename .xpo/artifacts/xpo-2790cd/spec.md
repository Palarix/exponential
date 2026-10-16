## What
Some menu items take the user to a next step inside their own menu instead of finishing. Activating one of these items should keep the menu open so that step can show. Today the menu closes, and two flows break:
- **Label picker:** `Create "foo"` should show the "Pick a color" step. Instead the popover closes and no label is created.
- **Issue context menu:** **Delete** should show its confirmation ("Delete this issue?" or the sub-issues choice). Instead the menu closes and nothing happens. Only ⌘⌫ / Ctrl⌫ works, because it skips the menu.

## Why
`Menu.activateItem` (`web/src/components/ui/Menu.tsx:132-133`) calls `onClose()` after the `onClick` of any item that isn't a checkbox. Both items set local state for their next step in `onClick`, and then `onClose` unmounts the component that owns that state.
- The context menu (`ContextMenu.tsx:183`) and the issue detail sidebar (`PropertySidebar.tsx:473`) both render `LabelPicker`, which owns its own `Menu`, so one fix to the picker covers both.
- The chat menus (`ChatTitleBar`, `ChatHistory`) are not affected: they close the menu on purpose and open a separate `ConfirmDialog`.

## How
- `MenuItem` gets an optional `keepOpen?: boolean` prop: "activating this item moves to a next step inside the menu, so don't close."
- `Menu` stores it in `ChildMeta`. The close decision moves into a pure helper, `closesOnActivate({ isCheckbox, keepOpen })` in `menu-utils.ts`, which returns `!isCheckbox && !keepOpen`. `activateItem` uses it.
- `LabelPicker` sets `keepOpen` on the Create row.
- `ContextMenu` sets `keepOpen` on the Delete item.
- Click, Enter and letter shortcuts all go through `activateItem`, so they all get the fix.

## Tests
The web tests run without a DOM: they render static markup, effects don't run, and state can't change. So:
- `Menu.test.ts`: unit tests for `closesOnActivate`:
  - a plain item closes the menu
  - a checkbox item keeps it open
  - an item with `keepOpen` keeps it open
- Both flows checked by hand.

## Follow-ups
- Labels: remove the color step and manage colors and built-ins in the Labels view (filed separately).
- Delete issue: a shared, polished confirmation dialog (filed separately).

## Acceptance criteria
- [ ] Clicking `Create "foo"` shows the color step; picking a color creates the label and applies it to the issue
- [ ] Enter on the focused Create row does the same
- [ ] Works from both the context menu and issue detail view
- [ ] Clicking or pressing Enter on Delete in the issue context menu shows the delete confirmation (and the sub-issue choice when the issue has children)
- [ ] Other menu items still close the menu on activate
- [ ] Regression test covers the close-on-activate rule that caused the bug
