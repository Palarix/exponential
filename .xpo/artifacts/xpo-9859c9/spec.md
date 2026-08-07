# Spec: Alt parent/unparent in Active and Backlog views

## Problem

The Alt-modifier parent/unparent mechanism (xpo-cd0f62) does not work reliably in Active and Backlog views. Three root causes:

1. **Nest target evaluation only runs in `handleDndOver`** — this event fires when the hover target changes, not on every pointer move. If the user presses Alt while already hovering a target, the nest highlight never appears because `handleDndOver` already ran without Alt. `handleDndMove` (which fires continuously) shows indicator bars but never sets the nest target.

2. **Ghost parent expanded-redirect is broken** — `handleDndMove` checks `expandedNodes.has(id)` for the "redirect below expanded parent" logic. Ghost parents auto-expand via `isGhostParent` flag in `useBacklogRows`, but are NOT in `expandedNodes`. So the redirect doesn't fire, causing the indicator to appear between the ghost parent and its first child instead of after the children tree.

3. **No discoverable unparent path** — hovering over ANY depth-0 issue with Alt sets a nest target (would parent, not unparent). The only exception is hovering over the own parent (`alreadyChild` check). Users can't easily discover this.

## Solution

### A. Move nest target evaluation into `handleDndMove`

Since `handleDndMove` fires on every pointer move during drag, the nest target will respond immediately to Alt being pressed/released.

- **Remove** the nest target evaluation block from `handleDndOver` (lines 633-651)
- **Add** the evaluation to `handleDndMove`, before the position calculation:
  - If Alt held AND target is depth 0 AND has no parent_id AND not descendant AND not already-child → set nest target, clear indicator
  - Otherwise → clear nest target
- **Extract** `isDescendant` to a shared `useCallback` (currently inline in `handleDndOver`)
- **Add** Alt to the cross-group bypass in both `handleDndOver` and `handleDndMove` (alongside Ctrl/Meta) — Alt operations should work across status groups

### B. Fix ghost parent expanded-redirect

In `handleDndMove`'s expanded-parent redirect (line 757-773), change:
```ts
expandedNodes.has(overRow.issue.id)
```
to:
```ts
(expandedNodes.has(overRow.issue.id) || overRow.isGhostParent)
```

### C. Add "Remove from parent" to context menu

Add a menu item to the right-click context menu in `ContextMenu.tsx`:
- Only visible when `issue.parent_id` is set
- Positioned between the Cycle item and the Delete separator
- Action: `handleAction("UPDATE", { parent_id: "" })`
- This gives a discoverable, non-DnD way to unparent in any view

## Acceptance criteria

- [ ] Alt+hover over a top-level issue during drag shows nest ring in Active and Backlog views
- [ ] Alt pressed mid-drag (not held from start) updates visual feedback immediately
- [ ] Dragging near a ghost parent's children shows correct indicator placement (after children, not between ghost and first child)
- [ ] Right-click → "Remove from parent" appears for child issues and clears parent relationship
- [ ] All existing DnD behaviors in "All Issues" view remain unchanged
- [ ] `make test` passes
