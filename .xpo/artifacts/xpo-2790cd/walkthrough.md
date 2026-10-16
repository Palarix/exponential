## What changed
`Menu` closes itself after the user activates any item that isn't a checkbox. That's right for most items, which do their job and finish, but wrong for an item whose `onClick` switches its own menu to a next step: closing unmounts the component that holds that step's state. Two items did this:

- **Label picker, `Create "…"`:** `handleSelect` set `creatingLabel` to show the "Pick a color" step, and the menu then closed the picker. A label is only created once a color is picked, so nothing happened. This came back when `LabelPicker` moved onto `Menu` (xpo-9a7907).
- **Issue context menu, Delete:** it set `confirmDelete` to show the confirmation, and the menu then called `closeAll`. Only ⌘⌫ / Ctrl⌫ worked, because the shortcut doesn't go through the menu.

## How it works
- `MenuItem` takes a `keepOpen` prop. Its doc comment says when to use it: activating the item opens a next step inside this menu.
- `Menu` collects child props into `ChildMeta` while rendering, and now also records `keepOpen` there.
- The close decision is a pure helper, `closesOnActivate({ isCheckbox, keepOpen })` in `menu-utils.ts`. It returns true only for a plain item. `activateItem` calls it, and click, Enter/Space and letter shortcuts all go through `activateItem`, so all three are fixed.
- `LabelPicker` sets `keepOpen` on the Create row, and `ContextMenu` sets it on Delete.

Both the context menu's Labels submenu and the issue sidebar's Labels popover render `LabelPicker`, which owns its own `Menu`, so one prop fixes both.

## Decisions
- **An opt-in prop rather than inferring it.** The menu can't tell whether an `onClick` moves to a next step. Making it explicit at the call site keeps the default (close) right for the many single-step items.
- **A pure helper for the rule.** The web tests have no DOM (they render static markup), so a menu can't be clicked in a test. Moving the rule into `menu-utils.ts` makes it unit-testable next to `nextIndex` and `matchShortcut`.
- **The chat menus are left alone.** `ChatTitleBar` and `ChatHistory` close their menu on purpose and open a separate `ConfirmDialog`, which is correct.

## Follow-ups
Polish that came up during review, filed separately:
- xpo-e35f16: create labels in one step without a color, and manage colors and built-in labels in the Labels view. It includes making `config.UpdateLabel` save colors for labels that aren't in the config.
- xpo-33c92b: a shared `DeleteIssueDialog`, built on `ConfirmDialog`, for the context menu and the sidebar.

## Acceptance criteria
- [x] Clicking `Create "foo"` shows the color step; picking a color creates the label and applies it to the issue: confirmed by the user on the tophat board.
- [x] Enter on the focused Create row does the same: Enter goes through `activateItem`, the same path as click; confirmed by the user.
- [x] Works from both the context menu and issue detail view: both render `LabelPicker`; confirmed by the user.
- [x] Clicking or pressing Enter on Delete in the issue context menu shows the delete confirmation: `keepOpen` on the Delete item; confirmed by the user.
- [x] Other menu items still close the menu on activate: `closesOnActivate({ isCheckbox: false })` returns true (tested).
- [x] Regression test covers the close-on-activate rule that caused the bug: `closesOnActivate` tests in `web/src/components/ui/Menu.test.ts`. `make test` passes (1157 frontend tests, plus lint and Go tests).
