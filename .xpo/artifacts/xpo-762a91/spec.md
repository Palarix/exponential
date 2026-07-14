# Prompt Before Discarding New Issue

## Problem

When a user is composing a new issue and closes the modal (Escape, backdrop click, X button, or Cancel), any content they've entered is silently lost. There is no confirmation step.

## Requirements

1. **Guard all close paths** — Escape key, backdrop click, X button, and Cancel button must all go through the same guard logic.
2. **Only prompt when there's content** — if both `title` and `description` are empty (trimmed), close immediately without prompting.
3. **Confirmation dialog** — when the form has content, show a small inline confirmation with:
   - Message: "Discard this issue?"
   - **Discard** button (destructive style) — clears form state and closes the modal.
   - **Cancel** button (ghost/secondary style) — dismisses the confirmation and returns focus to the form.
4. **No draft saving** — that is a separate feature (xpo-758398). This story only adds a discard guard.

## Design Decisions

### Where to put the guard

The guard lives in `NewIssueModal.tsx`, not in the generic `Modal` component. `NewIssueModal` already owns the form state needed to decide whether content exists, and the confirmation UI is specific to this form. The `Modal` component's `onClose` prop receives a wrapped handler that checks for content before closing.

### Confirmation UI

Render the confirmation inline within the modal (overlay the form content area) rather than spawning a second nested modal. This avoids z-index stacking issues and matches the confirmation pattern already used in `ContextMenu.tsx` (inline state toggle with confirm/cancel buttons).

### Escape key during confirmation

If the confirmation is showing and the user presses Escape again, treat it as "Cancel" (dismiss confirmation, keep editing) — not as a second discard attempt. This prevents accidental double-Escape from discarding content.

## Implementation Outline

### `NewIssueModal.tsx`

1. Add a `hasContent` derived value: `title.trim() !== "" || description.trim() !== ""`.
2. Add a `confirmDiscard` boolean state (default `false`).
3. Create a `handleClose` function:
   - If `!hasContent` → reset form state and call `onClose()`.
   - If `hasContent` → set `confirmDiscard = true`.
4. Create a `handleDiscard` function: reset all form state, set `confirmDiscard = false`, call `onClose()`.
5. Create a `handleCancelDiscard` function: set `confirmDiscard = false`.
6. Pass `handleClose` as the `onClose` prop to the `<Modal>` component (guards Escape, backdrop, X button).
7. Wire the existing Cancel button to `handleClose` instead of raw `onClose`.
8. When `confirmDiscard` is `true`, render an overlay inside the modal body with the confirmation message and two buttons (Discard / Cancel), styled consistently with the app's design system (CSS variables, same button patterns as elsewhere).

### `Modal.tsx`

No changes needed. The `onClose` callback it receives will already contain the guard logic.

## Acceptance Criteria

- [ ] Pressing Escape with an empty form closes the modal immediately.
- [ ] Pressing Escape with content in title or description shows the discard confirmation.
- [ ] Clicking "Discard" in the confirmation clears all form fields and closes the modal.
- [ ] Clicking "Cancel" in the confirmation dismisses it and returns to the form.
- [ ] Pressing Escape while the confirmation is showing dismisses the confirmation (does not discard).
- [ ] Backdrop click and X button with content also show the confirmation.
- [ ] Backdrop click and X button with an empty form close immediately.
- [ ] After discarding, reopening the modal shows a clean empty form.
