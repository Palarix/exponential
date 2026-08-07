# Spec: Guard parent changes behind Alt modifier during DnD

## Summary

Normal drag-and-drop should only reorder (change `sort_order` and optionally `status`). Parent relationship changes (both parenting and unparenting) require holding Alt. This prevents accidental unparenting and eliminates the bug where reordering orphaned children in "All Issues" clears their parent.

## Current behavior

- **Alt-drop on an issue**: Nests the dragged issue under the target (sets `parent_id`). ✓ Already correct.
- **Drop at depth 0**: Clears `parent_id` if the issue had one. ✗ Bug — accidental unparenting.
- **Drop at depth > 0**: Sets `parent_id` to the target row's parent. ✗ Can accidentally reparent.

## New behavior

### Without Alt (normal DnD)

- **Reorder within same depth**: Only update `sort_order`. Parent preserved.
- **Cross-group drop**: Update `status` and `sort_order`. Parent preserved.
- **Child dragged above/below its parent's tree**: Drop indicator stays pinned to valid positions within the parent's children (depth 1). The child cannot be visually placed at depth 0 without Alt.

### With Alt held

- **Alt-drop on an issue**: Nest under target (set `parent_id`). Same as current.
- **Alt-drop at depth 0**: Unparent (clear `parent_id`). New explicit gesture.
- **Alt-drop at depth > 0**: Reparent to target row's parent. New explicit gesture.

### Visual indication

When dragging a child issue:
- Without Alt: drop indicators only appear at valid child positions (depth 1 under the parent). If dragged far from the parent, no indicator appears (or indicator stays at last valid position).
- When Alt is pressed mid-drag: indicator expands to show depth-0 positions, signaling that a parent change is now possible.

## Implementation

### `performDrop` changes

1. Remove the `if (dragged?.parent_id) update.parent_id = ""` line in the depth-0 branch.
2. Remove the `update.parent_id = targetParentId` line in the depth > 0 branch.
3. Only set `parent_id` when Alt was held during the drop (already tracked for nest drops).

### Drop indicator constraints

In `handleDndOver` / collision detection:
- When dragging a child issue without Alt, constrain drop targets to rows that are siblings of the dragged issue (same `parent_id`) or the parent's own row (for position within the tree).
- When Alt is held, allow all positions (current behavior).

## Acceptance Criteria

- [ ] Normal reorder of a child never changes `parent_id`.
- [ ] Alt-drop at depth 0 unparents (clears `parent_id`).
- [ ] Alt-drop on an issue nests (sets `parent_id`) — existing behavior preserved.
- [ ] Drop indicator respects constraints: child can only be placed among siblings without Alt.
- [ ] Works correctly on All Issues, Active, and Backlog tabs.
- [ ] `make build` and `make test` pass.
