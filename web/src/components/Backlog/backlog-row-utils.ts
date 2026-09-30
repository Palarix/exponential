import type { HierarchyMode } from "./useBacklogRows";

/**
 * Which indicator a backlog row shows in its tree-toggle slot:
 * - toggle:   clickable chevron (parent with visible children)
 * - expanded: static rotated chevron (ghost parent, always expanded)
 * - none:     blank spacer (no children, children all hidden from this view,
 *             or flat mode)
 */
export type NodeIndicator = "toggle" | "expanded" | "none";

export function nodeIndicator({
  hasChildren,
  hasVisibleChildren,
  isGhostParent,
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
  return isGhostParent ? "expanded" : "toggle";
}
