# Expand All / Collapse All Buttons for Backlog

## Overview

Add icon-only expand-all and collapse-all buttons to the backlog toolbar. These control the expansion state of **issue nodes** (parents with children), not status group headers.

## Current State

- Issue node expand/collapse is tracked via a `collapsed` set persisted in `localStorage` under key `exponential-backlog-nodes-collapsed`.
- By default all parent nodes are expanded; collapsed ones are explicitly tracked.
- The `expandedNodes` set is derived in a `useMemo` from `childrenByParent` keys minus the collapsed set.
- Re-renders are triggered by incrementing `nodeToggleCount`.

## Design

### Buttons

Two icon-only buttons placed in the toolbar's right-side `ml-auto` div, between the Filter button and the Sort dropdown:

| Button | Icon (lucide-react) | Tooltip | Action |
|---|---|---|---|
| Expand All | `ChevronsDownUp` | "Expand all" | Clear the collapsed set → all parents expand |
| Collapse All | `ChevronsUpDown` | "Collapse all" | Set collapsed = all `childrenByParent` keys → all parents collapse |

The buttons use the same icon-only style as other toolbar buttons (`h-6 w-6 rounded-[var(--radius-sm)]` with muted text and hover states). They are only rendered when `childrenByParent.size > 0` (i.e., there are issues with children in the current view).

### Implementation

1. Import `ChevronsDownUp` and `ChevronsUpDown` from `lucide-react` in `Backlog.tsx`.
2. Add `expandAllNodes` callback: clears localStorage key and increments counter.
3. Add `collapseAllNodes` callback: sets localStorage key to all `childrenByParent` keys and increments counter.
4. Render two icon-only buttons in the toolbar between Filter and Sort, conditionally visible when parent issues exist.

## Acceptance Criteria

- [ ] Expand-all button expands all collapsed parent issues
- [ ] Collapse-all button collapses all expanded parent issues
- [ ] Buttons only appear when there are issues with children
- [ ] State persists across page reloads (via localStorage)
- [ ] Buttons use existing toolbar styling (icon-only, muted text, hover effect)
- [ ] Tooltips on hover indicate the action