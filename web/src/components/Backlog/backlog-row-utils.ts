import type { HierarchyMode } from "./useBacklogRows";

/**
 * Which indicator a backlog row shows in its tree-toggle slot:
 * - toggle:   clickable chevron (parent with visible children, ghost or real)
 * - none:     blank spacer (no children, children all hidden from this view,
 *             or flat mode)
 */
export type NodeIndicator = "toggle" | "none";

export function nodeIndicator({
  hasChildren,
  hasVisibleChildren,
  hierarchyMode,
}: {
  hasChildren: boolean;
  hasVisibleChildren: boolean;
  isGhostParent?: boolean;
  hierarchyMode: HierarchyMode;
}): NodeIndicator {
  if (!hasChildren || !hasVisibleChildren || hierarchyMode !== "nested") {
    return "none";
  }
  return "toggle";
}

/**
 * Parents whose children render. Collapse state is shared per issue ID, so a
 * ghost parent and its real row collapse together. While a search or filter
 * is active the persisted state is ignored (so matches are never hidden) and
 * only the transient, filter-scoped collapsed set applies.
 */
export function expandedNodeIds({
  parentIds,
  persistedCollapsed,
  transientCollapsed,
  filtering,
}: {
  parentIds: Iterable<string>;
  persistedCollapsed: Set<string>;
  transientCollapsed: Set<string>;
  filtering: boolean;
}): Set<string> {
  const collapsed = filtering ? transientCollapsed : persistedCollapsed;
  const expanded = new Set(parentIds);
  for (const id of collapsed) expanded.delete(id);
  return expanded;
}
