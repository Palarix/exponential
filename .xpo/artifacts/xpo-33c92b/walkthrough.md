## What changed

Deleting an issue used to show one of two hand-built popovers: one in `ContextMenu.tsx`, one inline in `PropertySidebar.tsx`. They were near-copies, styled ad hoc, and the context-menu one appeared where the menu had been. Both now open one shared dialog, `ui/DeleteIssueDialog.tsx`, which is built on `ConfirmDialog`. Review turned up related polish, which shipped in the same change: a better `ConfirmDialog` layout, a success toast, and red menu items that stay red on hover.

## How the pieces fit

### `ConfirmDialog` — layout for every confirm dialog
- `ConfirmDialog` keeps the overlay: the portal, the backdrop and the Escape shortcut. The visible card moved into a portal-free `ConfirmDialogPanel`. The test suite renders to static HTML, which can't render portals, so the split is what makes the layout testable.
- The panel shows the title, then an optional new `body` slot, then the description.
- The buttons sit in a footer bar on `surface-2` with an `esc` "to cancel" hint on the left.
- The panel is `max-w-md`, up from `max-w-sm`.
- Every existing confirm dialog (draft delete, the discard prompts in the new-issue dialog, chat delete, chat history delete) now uses a `secondary` Cancel instead of `ghost`. The new-issue modal's own footer Cancel is not a confirm dialog and is unchanged.

### `DeleteIssueDialog`
- Props: `{ issue: {id, title, status}, subIssueCount, onDelete(cascade), onCancel }`.
- The title is "Delete issue?". The `body` is a card with the status icon, the ID in mono and the title, so you can see which issue goes.
- Without sub-issues the description is "This can't be undone." and the buttons are Cancel · **Delete**.
- With sub-issues the description is "It has N sub-issues. Keep them as top-level issues, or delete them too?", with a singular form for one sub-issue. The buttons are Cancel · Keep sub-issues · **Delete all**.
- Keep sub-issues is soft red (`secondary` with `tone="danger"`), because it still deletes the parent. Delete all is solid red.
- It takes `subIssueCount` rather than a boolean so the text can state the number.

### Callers
- **ContextMenu:** `confirmDelete` is now a boolean, set by the Delete item (which already has `keepOpen`, from xpo-2790cd) or by ⌘⌫ / Ctrl⌫. While it is set, the component returns only the dialog, so the menu unmounts and `menuRef` is null. That keeps the menu's outside-mousedown handler from closing the dialog when you click inside it. Cancel closes the whole context menu instead of reopening it.
- **PropertySidebar:** the "Delete issue" button opens the dialog. Confirming closes the dialog and deletes. On success the issue view closes, as before.

### Success toast
- `useDeleteIssue` adds an `onSuccess` that shows `deleteIssueToast(vars)`: "Deleted <id>", or "Deleted <id> and its sub-issues".
- It is set on the hook, not on the `mutate` call. TanStack Query skips callbacks passed to `mutate` once the calling component unmounts, and the context menu unmounts right after confirming. The hook's own callbacks still run.

### Destructive menu items
- `MenuItem`'s class logic moved into a pure, tested `menuItemClass` in `menu-utils.ts`.
- When a destructive item is hovered or keyboard-focused, it keeps `--color-error` text on an `--color-error-bg` tint. Previously it switched to the neutral highlight with primary text.

## Decisions made in review
- The dialog design was picked on a temporary `#/tophat` page, now deleted. The options were an icon-tile card, an issue-as-heading layout with a list of the sub-issues, and choice cards with a single Delete button. The pick was the card layout without the icon column.
- After seeing it, the user asked for:
  - the issue card first, then the text, then the buttons;
  - Keep sub-issues to look destructive (soft red), not neutral;
  - Cancel in the secondary style across all confirm dialogs;
  - the footer bar on every `ConfirmDialog`, for consistency.

## Acceptance criteria
- [x] The context menu (click, Enter, ⌘⌫) and the issue sidebar open the same `DeleteIssueDialog`. Both render `<DeleteIssueDialog issue={issue} subIssueCount={…}>` (`ContextMenu.tsx`, `PropertySidebar.tsx`). The user tried both on the tophat board.
- [x] The dialog names the issue (status, ID, title) in a card. Covered by the `DeleteIssueDialog.test.tsx` test "names the issue in a card".
- [x] Both variants work. `DeleteIssueDialog.test.tsx` checks that Delete sends `cascade=false`, Keep sub-issues sends `false` and Delete all sends `true`. The user also deleted issues on the demo board.
- [x] Escape and a backdrop click cancel without deleting. `ConfirmDialog` wires both to `onCancel`, and a test asserts that `onCancel` is passed through and that Cancel never calls `onDelete`.
- [x] A toast confirms the delete. `deleteIssueToast` is tested in `mutation-options.test.ts`, and the user confirmed the toast on the board.
- [x] Every `ConfirmDialog` shows the footer bar with the esc hint and a secondary Cancel. `ConfirmDialog.test.tsx` checks the footer, and the four other call sites now use `variant="secondary"`.
- [x] Destructive menu items stay red on hover and keyboard focus. Covered by the `menuItemClass` tests in `Menu.test.ts`.
- [x] The old hand-built popovers are removed. The `"choose"`/`"confirm"` state and markup are gone from both files, and the sidebar test asserts that no inline confirmation renders.

`make test` passes: lint and 1171 frontend tests, plus the Go suite.
