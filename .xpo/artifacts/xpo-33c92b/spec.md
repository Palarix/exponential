## What
Replace the hand-built delete confirmations in `ContextMenu.tsx` and `PropertySidebar.tsx` with one shared `ui/DeleteIssueDialog.tsx` built on `ConfirmDialog`, and give `ConfirmDialog` a more deliberate layout.

## Why
The two copies drift and look unpolished. The bare `ConfirmDialog` (title, text, buttons) had no visual hierarchy, and "Delete issue?" alone didn't say which issue.

## How

### ConfirmDialog (all confirm dialogs)
- Body: `px-5 pt-5 pb-4` with the title, a new optional `body` slot (e.g. a card naming what is affected), then the description.
- Footer bar: `border-t` on `surface-2`, `px-5 py-3`. On the left, an `esc` key hint with "to cancel". On the right, the buttons.
- The chat, draft and discard dialogs get the new footer too; their props are unchanged.
- Every confirm dialog's Cancel button is `secondary` (no border), not `ghost`.

### DeleteIssueDialog
`DeleteIssueDialog({ issue, subIssueCount, onDelete(cascade: boolean), onCancel })`. `issue` is `{ id, title, status }`.

- Title: "Delete issue?".
- Body: under the title and above the description, a card on `surface-2` showing the issue's status icon, its ID in mono, and its title (truncated). The card says which issue it is.

| Variant | Description | Buttons |
|---|---|---|
| No sub-issues | "This can't be undone." | Cancel (secondary, no border) · **Delete** (danger) → `onDelete(false)` |
| Sub-issues | "It has N sub-issue(s). Keep them as top-level issues, or delete them too?" | Cancel (secondary, no border) · Keep sub-issues (secondary, `tone="danger"`: soft red) → `onDelete(false)` · **Delete all** (danger) → `onDelete(true)` |

Escape and a backdrop click call `onCancel` (from `ConfirmDialog`). It is exported from `ui/index.ts`.

**Success toast:** `useDeleteIssue` shows a toast once the delete succeeds, for every caller: "Deleted <id>", or "Deleted <id> and its sub-issues" when `cascade` is set. The message comes from a pure `deleteIssueToast` in `mutation-options.ts`. Errors still show the existing error toast.

**Menu destructive items** (`MenuItem destructive`): hovered or keyboard-focused, they keep their red text and get a red tint (`--color-error-bg`) instead of the neutral highlight with primary text. The class logic moves to a pure `menuItemClass` in `menu-utils.ts` so it can be tested.

**ContextMenu:** `confirmDelete` becomes a boolean. The Delete item (click/Enter, `keepOpen`) and ⌘⌫/Ctrl⌫ set it. While it is set, the component renders only the dialog in place of the menu, so the outside-mousedown handler, which needs `menuRef`, can't close it. `onDelete` closes the menu and deletes. `onCancel` closes the whole context menu.

**PropertySidebar:** `confirmDelete` becomes a boolean. The "Delete issue" button stays and opens the dialog. `onDelete` closes the dialog and deletes, navigating away on success as before. `onCancel` closes the dialog.

## Decisions
- **Design chosen on a tophat page:** an issue card plus a footer bar with the esc hint, and no icon column. A choice-card variant and an issue-as-heading variant were rejected.
- **Issue first, then the text, then the choice**: the card sits between the title and the description (chosen in review).
- **The footer bar and a secondary Cancel apply to every `ConfirmDialog`**, so confirm dialogs stay consistent across the app (chosen in review).
- **Cancel from the context menu closes the menu too**, rather than going back to it. A menu that reappears after a modal feels odd.
- **Both delete actions read as destructive**: Keep sub-issues still deletes the parent, so it is soft red (secondary, danger tone), while Delete all stays solid red (chosen in review).
- Buttons sit in one row with the destructive action rightmost, like the other confirm dialogs.

## Tests (written first)
- `ui/DeleteIssueDialog.test.tsx`:
  - Both variants render the right title, description and buttons.
  - The `body` card names the issue (ID, title and status).
  - Delete, Keep sub-issues and Delete all call `onDelete` with the right `cascade`.
  - Cancel, Escape and a backdrop click call `onCancel` and never `onDelete`.
- `api/mutation-options.test.ts`: `deleteIssueToast` names the issue, and mentions sub-issues only for a cascade.
- `ui/Menu.test.ts`: `menuItemClass` keeps a destructive item red with a red tint when focused and on hover; other items are unchanged.
- `PropertySidebar.test.tsx`: the "Delete issue" button is there, and no inline confirmation is rendered.

## Acceptance criteria
- [ ] The context menu (click, Enter, ⌘⌫) and the issue sidebar open the same `DeleteIssueDialog`
- [ ] The dialog names the issue (status, ID, title) in a card
- [ ] Both variants work: deleting without sub-issues; with sub-issues, keeping them or deleting them too
- [ ] Escape and a backdrop click cancel without deleting
- [ ] A toast confirms the delete
- [ ] Every `ConfirmDialog` shows the footer bar with the esc hint and a secondary Cancel
- [ ] Destructive menu items stay red on hover and keyboard focus
- [ ] The old hand-built popovers are removed
