# Walkthrough: Guard parent changes behind Alt modifier during DnD

## The problem

Drag-and-drop in the backlog could accidentally change parent relationships. Reordering a child in "All Issues" view (where orphaned children appear at depth 0) would silently clear `parent_id`. Dropping a top-level issue among another parent's children would silently reparent it.

## Design decision

Instead of complex conditional logic to detect when a parent change is "intentional" vs "accidental", we use a UX guard: **Alt is required for any parent change**. This mirrors the existing pattern where Cmd/Ctrl is required for cross-group precise positioning.

- Normal drag: reorder only (`sort_order`, optionally `status`)
- Alt + drop on issue: nest (set `parent_id`)
- Alt + drop at depth 0: unparent (clear `parent_id`)

## Implementation

### `performDrop` (Backlog.tsx)

The function gained an `altHeld: boolean` parameter, captured from `modifiersRef.current.alt` at drop time in `handleDndEnd`.

In the depth > 0 branch, `update.parent_id = targetParentId` is now behind `if (altHeld)`.
In the depth 0 branch, `update.parent_id = ""` is now behind `if (altHeld)`.

The Alt-drop-on-issue nest gesture (scenario A) was already Alt-only and is unchanged.

### Drop indicator constraints (handleDndMove)

Refactored from a chain of nested conditionals into three focused helpers:

**`findAfterTree(parentIdx)`**: Given a parent's row index, returns the indicator position after its entire child subtree — either `{ rowIndex, position: "above" }` on the next depth-0 row, or `{ rowIndex, position: "below" }` on the last child if no depth-0 row follows.

**`isSiblingPosition(draggedParentId, targetRow, position)`**: Pure predicate. Returns true when the target position doesn't require a parent change:
- Child issue: target must be a sibling (same `parent_id`) or "below" on its own parent row
- Top-level issue: target must be at depth 0

**`handleDndMove`** reads top-to-bottom:
1. Group header → indicator above first issue
2. Self-row or cross-group without Cmd/Ctrl → suppress
3. Compute raw above/below from pointer position
4. No-op suppression: adjacent to drag ghost (would be identity reorder)
5. Expanded parent redirect: "below" an expanded parent redirects into children (if Alt held or own parent) or past the tree (via `findAfterTree`)
6. Alt held → any position is valid
7. Without Alt → `isSiblingPosition` check; if false and target is a child row, redirect via `findAfterTree` on the target's parent; otherwise suppress

### Documentation updates (expo-landing repo)

- `keyboard-shortcuts.md`: Added "Alt + drag to top level → unparent" row and a note explaining that normal DnD never changes parent relationships.
- `parent-sub-issues.md`: Added "Remove from parent" section with CLI (`xpo update <id> --parent ""`) and web board (Alt-drag to top level) instructions, plus a note about the safety constraint.
