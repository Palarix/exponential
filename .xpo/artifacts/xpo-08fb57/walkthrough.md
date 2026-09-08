# Walkthrough: Backlog `{`/`}` expand/collapse all parents

## Summary
Added `}` and `{` keyboard shortcuts to expand and collapse all parent issue tree nodes in the Backlog view. These complement the existing `[`/`]` tab-cycling shortcuts — same physical keys, shifted for a broader action (bulk tree expand/collapse vs. tab cycling).

## Changes

### `web/src/components/Backlog/Backlog.tsx`
- Added stable refs (`expandAllNodesRef`, `collapseAllNodesRef`) for the existing `expandAllNodes` and `collapseAllNodes` functions, following the same pattern used by all other keyboard-accessible functions in this component.
- Wired `}` and `{` key handlers into the existing `useEffect` keydown listener, placed after the `metaKey`/`ctrlKey` guard and before the `[`/`]` tab-cycling block. The handlers call the existing node expand/collapse functions which manage the `exponential-backlog-nodes-collapsed` localStorage key and bump `nodeToggleCount` to trigger a `useMemo` recomputation of `expandedNodes`.

### `web/src/components/KeyboardHelp/KeyboardHelp.tsx`
- Added two entries to the BACKLOG shortcut group: `}` "Expand all parents" and `{` "Collapse all parents".

## Key decisions
- **Scope limited to parent issue tree nodes** — status group expand/collapse was considered but dropped to keep the feature focused. The `[`/`]` family controls navigation-level shortcuts, not status-group state.
- **Ghost parents are not collapsible** — they exist as contextual breadcrumbs for active children; collapsing them would hide active issues behind done issues.
- **Reuses existing functions** — `expandAllNodes` and `collapseAllNodes` already existed but had no keyboard binding. No new state management was needed.
