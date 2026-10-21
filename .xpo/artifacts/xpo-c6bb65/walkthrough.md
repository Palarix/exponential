# Walkthrough: one frame for every dialog

## What changed
The board had three dialog frames: `Modal`, `DialogPanel`, and a hand-built one in MergeView. Each had its own padding, title size, backdrop and z-index. All dialogs now use one frame in `web/src/components/ui/Dialog.tsx`, which has two parts:

- **`DialogOverlay`**: the portal, the backdrop and the keyboard handling.
- **`Dialog`**: the card itself.

`Modal.tsx`, `ModalFooter` and `DialogPanel` are deleted.

## How the pieces fit
- **`DialogOverlay`** is mounted only while the dialog is open (`{open && <DialogOverlay>}`), so its mount and unmount effect does the open and close work:
  - It remembers the element that had focus, locks `body` scroll, and on the next frame focuses the first focusable element. It skips that step when something inside the dialog already has focus, so `autoFocus` wins.
  - On unmount it restores the *previous* `overflow` value rather than clearing it. A confirmation stacked on New issue therefore leaves the page locked when it closes. `Modal` used to reset `overflow` to `""` here.
  - Escape and Tab are `overlay`-priority shortcuts in the keyboard registry. A Tab binding without a shift modifier also matches Shift+Tab, so one handler covers both directions. The `enabled` prop turns both off while a second dialog is stacked on top; New issue and Edit label pass `enabled={!confirm}`.
- **`Dialog`** is a flex column capped at 85vh. The header and footer are `shrink-0`; the body is `flex-1 min-h-0 overflow-y-auto`, so only the body scrolls. Each part carries `data-slot="header|body|footer"`, which the tests use to find them.
- **Spacing (look B)**: header `p-4 pb-0`, body `p-4`, footer `p-4 pt-0`. That gives 16px on every outer edge and 16px between the parts, without doubling. Ghost buttons placed in `footerLeft` (Abort, Delete label, Discard draft) use `-ml-3`, so their text lines up with the title.
- **Footer**: `footerLeft` defaults to the esc hint. With neither `footer` nor `footerLeft`, the footer is not rendered at all (used by "Unable to merge").

## Migration notes
- **`ConfirmDialog`** keeps its API. The title moved into the header, so the body and description lost their old `mt-*` offsets.
- **New issue**: the overlay replaced `Modal`'s `isOpen` prop with an early `return null` after all hooks. The ⌘Enter handler stays on the body wrapper, and the confirmations are rendered next to the overlay instead of inside the card.
- **The sub-issue prompts** (Backlog, PropertySidebar, ReleaseAssign) used to put raw `<button>`s at the end of the body. Their IIFE now returns the whole `DialogOverlay`, so the footer callbacks can use `targetLabel`. In ReleaseAssign, `MoveChildrenBody` became `MoveChildrenPanel`, which renders the full `Dialog`, so its test can check that the choices sit in the footer.
- **Confirm merge** gained Escape and the focus trap, which the hand-built frame never had. It uses `Button` with `loading`.

## Decisions
- Look B (no dividers, one surface) was picked over A (dividers, footer on `surface-2`) and C (header divider only) on a temporary `#/tophat` page, which was deleted before commit.
- Titles are 14px semibold everywhere, the esc hint shows wherever `footerLeft` is unset, and the backdrop is `black/40`.
- The command palette, keyboard help and the release picker are search-first overlays and stay as they are.

## Acceptance criteria
- [x] One `Dialog` frame in `ui/Dialog.tsx`; `Modal`, `ModalFooter` and `DialogPanel` are gone. Evidence: `Modal.tsx` is deleted, and a grep for `\bModal\b|ModalFooter|DialogPanel\b` matches only `ConfirmDialogPanel`.
- [x] Every dialog in the migration table renders through `Dialog` + `DialogOverlay`. Evidence: ConfirmDialog, EditLabelDialog, NewReleaseDialog, NewIssueModal, MergeView (×2), Backlog, PropertySidebar, ReleaseAssign.
- [x] Look B: no dividers, one surface, 16px padding. Evidence: the `Dialog.test.tsx` tests "orders header, body and footer, with no dividers on one surface" and "pads 16px on every edge without doubling it between slots".
- [x] Only the body scrolls inside 85vh. Evidence: the test "scrolls only the body, inside an 85vh cap", and the long sample on the tophat.
- [x] Focus trap, initial focus, focus return and scroll lock, including stacked dialogs. Evidence: `DialogOverlay` effect and Tab binding; the tophat's "stacked confirm" live check; the user's review on the demo board.
- [x] Escape closes only the top dialog. Evidence: `enabled={!confirm}` on New issue and Edit label; the overlay registry runs the most recently registered binding.
- [x] Unit tests cover the header, the footer and the sizes. Evidence: 16 tests in `Dialog.test.tsx`, plus the updated `ConfirmDialog.test.tsx` and `ReleaseAssign.test.tsx`.
- [x] The tophat page is deleted, and `make test` passes. Evidence: `Tophat.tsx` is removed and `main.tsx` is restored; `make test` exits 0 (1,539 frontend tests).
