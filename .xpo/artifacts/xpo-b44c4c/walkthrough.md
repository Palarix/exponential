# Walkthrough: Expand All / Collapse All Buttons

## What changed

A single file was modified: `web/src/components/Backlog/Backlog.tsx`.

### 1. New imports (line 37)

Added `ChevronsUpDown` and `ChevronsDownUp` from `lucide-react`. These are the same icons used in the MergeView's diff expand/collapse controls, keeping the iconography consistent across the app.

### 2. Two new callbacks (after `toggleNode`)

**`expandAllNodes`** — To expand all parent nodes, we simply clear the collapsed set in localStorage by writing `"[]"`. Since `expandedNodes` is computed as "all parent IDs minus collapsed IDs," an empty collapsed set means everything is expanded. Then we bump `nodeToggleCount` to trigger the `useMemo` recomputation.

**`collapseAllNodes`** — The inverse: we write *all* parent IDs (the keys of `childrenByParent`) into the collapsed set in localStorage, then bump the counter. This makes `expandedNodes` empty, collapsing every parent node.

Both callbacks follow the same localStorage + counter pattern that `toggleNode` already uses, so no new state mechanism was introduced.

### 3. Toolbar buttons (between Filter and Sort)

Two icon-only buttons are rendered inside the toolbar's `ml-auto` div, between the Filter dropdown and the Sort dropdown. They are wrapped in a conditional `{childrenByParent.size > 0 && (...)}` so they only appear when there are actually issues with children in the current view.

Each button:
- Uses `h-6 w-6` sizing matching the toolbar's visual density
- Has a `title` attribute for tooltip on hover ("Expand all" / "Collapse all")
- Follows the same muted-text-with-hover style as the Filter and Sort buttons

### How it connects to existing code

The key insight is that the backlog already tracks collapsed nodes in localStorage under `exponential-backlog-nodes-collapsed`, and derives the `expandedNodes` set via a `useMemo` that depends on `nodeToggleCount`. The expand/collapse all feature simply writes to that same localStorage key in bulk (either clearing it or filling it with all parent IDs) and bumps the same counter. No new state variables, no new storage keys, no changes to `useBacklogRows` or the rendering logic.