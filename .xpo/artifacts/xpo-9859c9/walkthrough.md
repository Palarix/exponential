# Walkthrough: Alt parent/unparent in Active and Backlog views

## Problem

The Alt-modifier DnD mechanism for parenting/unparenting issues (introduced in xpo-cd0f62) didn't work reliably in Active and Backlog views. Three root causes:

1. Nest target evaluation lived in `handleDndOver`, which only fires on hover target changes — pressing Alt mid-drag had no effect
2. Ghost parent expanded-redirect was broken because ghost parents aren't in `expandedNodes`
3. Unparenting was non-discoverable — required hovering over the own parent with Alt

## Changes

### `web/src/components/Backlog/Backlog.tsx`

#### Extracted `isDescendant` to shared callback

Previously `isDescendant` was defined inline inside `handleDndOver`. Extracted it to a `useCallback` so both `handleDndOver` and `handleDndMove` can reference it.

```ts
const isDescendant = useCallback(
  (parentId: string, childId: string): boolean => {
    for (const i of issues) {
      if (i.parent_id === parentId) {
        if (i.id === childId) return true;
        if (isDescendant(i.id, childId)) return true;
      }
    }
    return false;
  },
  [issues],
);
```

#### Simplified `handleDndOver`

Removed the entire nest target evaluation block (the `if (modifiersRef.current.alt) { ... }` block that checked `!overRow.issue.parent_id`, `isDescendant`, and `alreadyChild`). This handler now only manages:
- Group header drop targets (`dropGroupStatus`)
- Self-check (clear all states)
- Cross-group guard (now bypassed by Alt in addition to Ctrl/Meta)

The key change to the cross-group guard:
```ts
// Before:
!(modifiersRef.current.meta || modifiersRef.current.ctrl)
// After:
!(modifiersRef.current.meta || modifiersRef.current.ctrl || modifiersRef.current.alt)
```

#### Enhanced `handleDndMove`

Added nest target evaluation at the top of the function, after `overRow` validation but before position calculation. Since `handleDndMove` fires on every pointer move during drag, the nest target now responds immediately when the user presses or releases Alt:

```ts
if (modifiersRef.current.alt && overRow.depth === 0 && !overRow.issue.parent_id) {
  const alreadyChild = draggedParentId === overRaw;
  if (!alreadyChild && !isDescendant(draggedId, overRaw)) {
    setDropNestTargetId(overRaw);
    setDropIndicator(null);
    setDropGroupStatus(null);
    return;
  }
}
setDropNestTargetId(null);
```

When setting the nest target, also clears `dropGroupStatus` to prevent both the group highlight and nest ring from showing simultaneously.

Fixed the ghost parent expanded-redirect. Ghost parents auto-expand their children via `isGhostParent` in `useBacklogRows`, but weren't in `expandedNodes`. The redirect now checks both:

```ts
// Before:
expandedNodes.has(overRow.issue.id)
// After:
(expandedNodes.has(overRow.issue.id) || overRow.isGhostParent)
```

Also added Alt to the cross-group bypass (same as in `handleDndOver`) and clears `dropGroupStatus` when showing the Alt indicator, in case the cross-group guard had previously set it.

### `web/src/components/ui/ContextMenu.tsx`

Added "Remove from parent" menu item:
- Imported `Unlink` icon from lucide-react
- New button rendered between the Cycle menu item and the Delete separator
- Only visible when `issue.parent_id` is set (the issue is a child)
- Calls `handleAction("UPDATE", { parent_id: "" })` to clear the parent relationship
- Uses the same styling as other menu items for visual consistency

## How it works now

- **Alt + hover over top-level issue during drag**: Shows nest ring (parent under that issue) — responsive to Alt press/release mid-drag
- **Alt + drop on depth-0 row**: Unparents the dragged issue (clears `parent_id`)
- **Alt + drop on child row**: Makes the dragged issue a sibling (sets same `parent_id`)
- **Right-click any child issue → "Remove from parent"**: Clears `parent_id` without DnD
- **Ghost parents**: Indicator placement now correctly redirects past their children
- **Cross-group**: Alt bypasses the cross-group guard, allowing parent operations across status groups
