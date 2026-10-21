# Dialogs: one frame for every dialog

## What
Replace `Modal`, `ModalFooter` and `DialogPanel` with one `Dialog` frame in `ui/Dialog.tsx`: a header, a scrollable body and a footer, with 16px (`p-4`) padding on every edge. Move every dialog onto it.

## Why
Today there are three frames: `Modal`, `DialogPanel` and a hand-built one in MergeView. Their padding differs (footer buttons sit 12px from the bottom but 20px from the right), and so do their titles (16px `h2` vs 14px `h3`), backdrops (`black/50` vs `black/30`) and z-index (`z-50` vs `z-[60]`).

## How

### `ui/Dialog.tsx`
- **`DialogOverlay`** (exists): the portal, backdrop and Escape. It also takes over what `Modal` does today:
  - Tab stays inside the dialog.
  - On open, focus goes to the first focusable element unless something inside the dialog already has focus (for example `autoFocus`).
  - On close, focus goes back to the element that opened it.
  - The page behind does not scroll. A stacked dialog restores the previous value when it closes, so the page stays locked under the dialog below.
  - Keeps `enabled`, so a dialog stacked on top can take over Escape and Tab.
  - One backdrop, `black/40`, on `DIALOG_LAYER`.
- **`Dialog`** (replaces `DialogPanel`): the card.
  - `id`, `role` (`dialog` | `alertdialog`), `describedBy`.
  - `title` (14px semibold `h2`) and optional `icon`. Optional `headerActions`. With `onClose`, a close button in the header.
  - `size`: `sm` 384, `md` 448 (default, today's `DialogPanel`), `lg` 512, `xl` 672, `2xl` 860. Max height 85vh. Only the body scrolls.
  - `footerLeft`: defaults to the esc hint. `footer`: the buttons, right-aligned. With no `footer` and no `footerLeft`, there is no footer.

### Look: B (picked on the tophat)
- No dividers. The header, body and footer share the card's `surface-3`.
- 16px on every outer edge, and 16px between header, body and footer (no doubled padding): header `p-4 pb-0`, body `p-4`, footer `p-4 pt-0`.
- Rejected: A (dividers above and below, with the footer on `surface-2`) and C (a header divider only, with the footer on `surface-2`).

### Migration
| Dialog | Today | Change |
|---|---|---|
| `ConfirmDialog` (all callers) | `DialogPanel` | `Dialog` with `role="alertdialog"`. Callers keep the same API. |
| Edit label, New/Edit release | `DialogPanel` | `Dialog` |
| New issue / Edit draft | `Modal` 2xl, buttons inside the body | `Dialog size="2xl"` with a close button. The buttons move into `footer`, and "Discard draft" into `footerLeft`. |
| Unable to merge (MergeView) | `Modal` xl | `Dialog size="xl"` with a close button and no footer |
| Confirm merge (MergeView) | hand-built overlay | `Dialog size="xl"` with the merge icon and a close button. The "Delete branch" checkbox goes in `footerLeft`. |
| Update sub-issues? (Backlog, PropertySidebar) | `Modal` xl, raw buttons in the body | `Dialog size="xl"`. `footerLeft`: Abort (ghost). `footer`: Just this issue (secondary), Update all (primary). |
| Move sub-issues too? (ReleaseAssign) | `Modal` xl, raw buttons in the body | Same as above, with Move all |

`Modal.tsx` and `ModalFooter` are deleted, and `index.ts` exports `Dialog` and `DialogOverlay`.

Not in scope: the command palette, keyboard help and the release picker. They are search-first overlays and stay as they are.

## Acceptance criteria
- [ ] One `Dialog` frame in `ui/Dialog.tsx`; `Modal`, `ModalFooter` and `DialogPanel` are gone.
- [ ] Every dialog in the migration table renders through `Dialog` + `DialogOverlay`.
- [ ] Look B: no dividers, one surface, and 16px padding, so the footer buttons sit 16px from the bottom and the right.
- [ ] Only the body scrolls when the content is taller than 85vh. The header and footer stay visible.
- [ ] Focus trap, initial focus, focus return and scroll lock work for every dialog, including a confirmation stacked on New issue or Edit label.
- [ ] Escape closes only the top dialog.
- [ ] Unit tests cover the header (title, icon, close button), the footer (esc hint by default, `footerLeft`, no footer without either), and the sizes.
- [ ] The tophat page is deleted before the commit, and `make test` passes.

## Decisions
1. **Title size**: 14px semibold everywhere.
2. **Esc hint**: shown on the footer's left wherever `footerLeft` is not set, including New issue and the sub-issue prompts.
3. **Backdrop**: `black/40` for every dialog.
4. **Look**: B, picked on the tophat.
