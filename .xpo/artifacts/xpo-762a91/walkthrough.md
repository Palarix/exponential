# Walkthrough: Prompt Before Discarding New Issue

## What changed

All changes are in `web/src/components/NewIssueModal/NewIssueModal.tsx`.

### Discard guard on all close paths

Previously, the `<Modal>` component received the raw `onClose` prop, so Escape, backdrop click, the X button, and the Cancel button all closed the modal immediately — silently discarding any content the user had typed.

Now a `handleClose` function wraps `onClose`:

```tsx
const handleClose = () => {
  if (confirmDiscard) { setConfirmDiscard(false); return; }
  if (!hasContent) { resetForm(); onClose(); return; }
  setConfirmDiscard(true);
};
```

- If the confirmation dialog is already showing, Escape dismisses it (returns to the form).
- If `title` and `description` are both empty (trimmed), the modal closes immediately — no prompt needed.
- Otherwise, it sets `confirmDiscard = true` to show the confirmation dialog.

The `<Modal>` and Cancel button both receive `handleClose` instead of `onClose`.

### Confirmation dialog

When `confirmDiscard` is true, a portal renders a small `alertdialog` on top of the new-issue modal. The new-issue form remains visible behind it so the user can re-read their draft before deciding.

The dialog has:
- A title: "Discard this issue?"
- A message: "Your issue draft has unsaved changes."
- Two buttons: **Cancel** (ghost, dismisses the dialog) and **Discard** (danger, resets form and closes modal).

### Keyboard trapping

The parent `Modal` component registers a document-level `keydown` listener for Escape. Without intervention, pressing Escape while the confirmation is showing would also fire the parent modal's handler and potentially leak to other views (e.g. IssueDetail navigation).

To prevent this, a `useEffect` registers a **capture-phase** listener on `document` while `confirmDiscard` is true:

```tsx
useEffect(() => {
  if (!confirmDiscard) return;
  const trap = (e: KeyboardEvent) => {
    if (e.key === "Escape") {
      e.stopImmediatePropagation();
      e.preventDefault();
      setConfirmDiscard(false);
    }
  };
  document.addEventListener("keydown", trap, true); // capture phase
  return () => document.removeEventListener("keydown", trap, true);
}, [confirmDiscard]);
```

Capture phase fires before bubble phase, and `stopImmediatePropagation` prevents any other document-level handlers from seeing the event. This keeps Escape fully contained within the confirmation dialog.

### Form reset consolidation

The state-reset logic that was previously duplicated inline after a successful create was extracted into a `resetForm` helper. Both the create-success path and the discard path call it, keeping the reset in one place.

## What was intentionally left out

Draft saving (persisting the form contents for later) is tracked separately in xpo-758398. This implementation only guards against accidental discard.
