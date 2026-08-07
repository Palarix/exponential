# Spec: Fix sub-issues preview overflow in cascade dialog

## Problems

1. All labels are shown per child row, causing horizontal overflow.
2. Long titles can push the row content to wrap.
3. The child list has no scroll constraint — many children push buttons off-screen.

## Fix

### Labels: cap at 2

Show at most 2 `LabelBadge` components per row. If more exist, append a `+N` text indicator.

### Titles: enforce truncation

The title `<span>` already has `truncate` but lacks a width constraint. Add `flex-1` so it takes remaining space and actually truncates.

### Scrolling

Wrap the child list `<div>` with `max-h-64 overflow-y-auto` so long lists scroll within the modal.

### Apply to both views

Both `Backlog.tsx` and `PropertySidebar.tsx` have the same modal — fix both.

## Acceptance Criteria

- [ ] At most 2 labels shown per child row, with `+N` for extras.
- [ ] Long titles truncate with ellipsis.
- [ ] Child list scrolls when it exceeds ~16rem height.
- [ ] `make build` passes.
