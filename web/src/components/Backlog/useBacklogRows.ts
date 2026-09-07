import { useMemo } from "react";
import type { Issue } from "../../api/client";
import { isTerminal, isCompleted } from "../../constants";
import { sortGroup } from "../../utils/sort";
import type { SortKey } from "../../utils/sort";
import type { Tab } from "./Backlog";

const TAB_CONFIGS: Record<Tab, { statuses: string[] }> = {
  all: { statuses: ["BACKLOG", "PLANNED", "DOING", "BLOCKED", "DONE", "CANCELED", "DUPLICATE"] },
  backlog: { statuses: ["BACKLOG"] },
  active: { statuses: ["PLANNED", "DOING", "BLOCKED"] },
  done: { statuses: ["DONE", "CANCELED", "DUPLICATE"] },
};

export type HierarchyMode = "nested" | "flat";
export type TreeGuide = "pipe" | "tee" | "corner" | "blank";

export type RowItem =
  | {
      kind: "group";
      status: string;
      label: string;
      count: number;
      storyPoints: number;
      isEmpty: boolean;
    }
  | {
      kind: "issue";
      issue: Issue;
      depth: number;
      hasChildren: boolean;
      childDone: number;
      childTotal: number;
      childPointsDone: number;
      childPointsTotal: number;
      parentBreadcrumb?: string;
      isGhostParent?: boolean;
      isGhostChild?: boolean;
      treeGuides: TreeGuide[];
    };

const STATUS_META: Record<string, { label: string }> = {
  BACKLOG: { label: "Backlog" },
  PLANNED: { label: "Planned" },
  DOING: { label: "In Progress" },
  BLOCKED: { label: "Blocked" },
  DONE: { label: "Done" },
  CANCELED: { label: "Canceled" },
  DUPLICATE: { label: "Duplicate" },
};

export function useBacklogRows(
  issues: Issue[],
  filteredIssues: Issue[],
  activeTab: Tab,
  expandedGroups: Set<string>,
  expandedNodes: Set<string>,
  sortKey: SortKey,
  hierarchyMode: HierarchyMode = "nested",
  showEmptyGroups: boolean = false,
  showGhosts: boolean = true,
): RowItem[] {
  const childrenByParent = useMemo(() => {
    const map = new Map<string, Issue[]>();
    for (const issue of issues) {
      if (issue.parent_id) {
        const siblings = map.get(issue.parent_id) || [];
        siblings.push(issue);
        map.set(issue.parent_id, siblings);
      }
    }
    return map;
  }, [issues]);

  const visibleStatuses = TAB_CONFIGS[activeTab].statuses;

  return useMemo(() => {
    const result: RowItem[] = [];

    const issueChildStats = (issue: Issue) => {
      const allChildren = childrenByParent.get(issue.id) || [];
      return {
        allChildren,
        hasChildren: allChildren.length > 0,
        childDone: allChildren.filter((c) => isCompleted(c.status)).length,
        childTotal: allChildren.length,
        childPointsTotal: allChildren.reduce(
          (s, c) => s + (c.estimate || 1),
          0,
        ),
        childPointsDone: allChildren
          .filter((c) => isCompleted(c.status))
          .reduce((s, c) => s + (c.estimate || 1), 0),
      };
    };

    for (const status of visibleStatuses) {
      const groupIssues = filteredIssues.filter((i) => i.status === status);
      if (groupIssues.length === 0 && !showEmptyGroups) continue;

      const groupIssueIds = new Set(groupIssues.map((i) => i.id));
      const storyPoints = groupIssues.reduce((sum, i) => {
        const children = childrenByParent.get(i.id);
        if (children && children.length > 0) {
          const childrenInGroup = children.filter((c) =>
            groupIssueIds.has(c.id),
          );
          if (childrenInGroup.length > 0) return sum;
          const relevant = isTerminal(i.status)
              ? children.filter((c) => isCompleted(c.status))
              : children.filter((c) => !isTerminal(c.status));
          return sum + relevant.reduce((s, c) => s + (c.estimate || 1), 0);
        }
        return sum + (i.estimate || 1);
      }, 0);

      result.push({
        kind: "group",
        status,
        label: STATUS_META[status]?.label || status,
        count: groupIssues.length,
        storyPoints,
        isEmpty: groupIssues.length === 0,
      });

      if (!expandedGroups.has(status) || groupIssues.length === 0) continue;

      const effectiveSortKey =
        isTerminal(status) ? ("updated" as const) : sortKey;

      if (hierarchyMode === "flat") {
        // Group children by parent so siblings stay together.
        // Order: top-level issues in sort order, each followed by its
        // children; then orphan groups (parent in different status)
        // clustered by parent sort position.
        const topLevel: Issue[] = [];
        const childGroups = new Map<string, Issue[]>();
        for (const issue of groupIssues) {
          if (issue.parent_id) {
            const group = childGroups.get(issue.parent_id) || [];
            group.push(issue);
            childGroups.set(issue.parent_id, group);
          } else {
            topLevel.push(issue);
          }
        }

        const sortedTop = sortGroup(topLevel, effectiveSortKey);
        const usedParents = new Set<string>();
        const flatList: { issue: Issue; breadcrumb?: string }[] = [];

        for (const issue of sortedTop) {
          flatList.push({ issue });
          const children = childGroups.get(issue.id);
          if (children) {
            usedParents.add(issue.id);
            for (const child of sortGroup(children, effectiveSortKey)) {
              flatList.push({ issue: child, breadcrumb: issue.title });
            }
          }
        }

        const orphanGroups = [...childGroups.entries()].filter(
          ([pid]) => !usedParents.has(pid),
        );
        orphanGroups.sort((a, b) => {
          const pa = issues.find((i) => i.id === a[0]);
          const pb = issues.find((i) => i.id === b[0]);
          return (pa?.sort_order || "") < (pb?.sort_order || "")
            ? -1
            : (pa?.sort_order || "") > (pb?.sort_order || "")
              ? 1
              : 0;
        });
        for (const [parentId, children] of orphanGroups) {
          const parent = issues.find((i) => i.id === parentId);
          for (const child of sortGroup(children, effectiveSortKey)) {
            flatList.push({ issue: child, breadcrumb: parent?.title });
          }
        }

        for (const { issue, breadcrumb } of flatList) {
          const stats = issueChildStats(issue);
          result.push({
            kind: "issue",
            issue,
            depth: 0,
            hasChildren: stats.hasChildren,
            childDone: stats.childDone,
            childTotal: stats.childTotal,
            childPointsDone: stats.childPointsDone,
            childPointsTotal: stats.childPointsTotal,
            parentBreadcrumb: breadcrumb,
            treeGuides: [],
          });
        }
      } else {
        // Nested mode (all tabs): real issues in their own status group,
        // ghost parents for orphaned children, ghost children for context.
        const topLevel = groupIssues.filter((i) => !i.parent_id);
        const orphanedChildren = groupIssues.filter(
          (i) => i.parent_id && !groupIssueIds.has(i.parent_id),
        );
        const ghostParentIds = new Set(
          orphanedChildren.map((i) => i.parent_id!),
        );

        const ghostParents = [...ghostParentIds]
          .map((id) => issues.find((i) => i.id === id))
          .filter(Boolean) as Issue[];
        const merged = sortGroup(
          [...topLevel, ...ghostParents],
          effectiveSortKey,
        );

        const addTree = (
          issue: Issue,
          depth: number,
          isGhost?: boolean,
          treeGuides?: TreeGuide[],
        ) => {
          const stats = issueChildStats(issue);
          result.push({
            kind: "issue",
            issue,
            depth,
            hasChildren: stats.hasChildren,
            childDone: stats.childDone,
            childTotal: stats.childTotal,
            childPointsDone: stats.childPointsDone,
            childPointsTotal: stats.childPointsTotal,
            isGhostParent: isGhost,
            treeGuides: treeGuides || [],
          });

          // Ghost parents only show their real children (the ones in this
          // group). Real parents show all children — real + ghost for context.
          const allVisual = isGhost
            ? sortGroup(
                stats.allChildren.filter((c) => groupIssueIds.has(c.id)),
                effectiveSortKey,
              ).map((c) => ({ child: c, ghost: false }))
            : sortGroup(stats.allChildren, effectiveSortKey)
                .filter((c) => groupIssueIds.has(c.id) || showGhosts || !isTerminal(c.status))
                .map((c) => ({
                  child: c,
                  ghost: !groupIssueIds.has(c.id),
                }));

          if (
            allVisual.length > 0 &&
            (isGhost || expandedNodes.has(issue.id))
          ) {
            const inherited: TreeGuide[] = (treeGuides || []).map((g) =>
              g === "tee" || g === "pipe" ? "pipe" : "blank",
            );
            allVisual.forEach(({ child, ghost: isGhostChild }, idx) => {
              const isLast = idx === allVisual.length - 1;
              const guides: TreeGuide[] = [
                ...inherited,
                isLast ? "corner" : "tee",
              ];
              if (isGhostChild) {
                const cs = issueChildStats(child);
                result.push({
                  kind: "issue",
                  issue: child,
                  depth: depth + 1,
                  hasChildren: cs.hasChildren,
                  childDone: cs.childDone,
                  childTotal: cs.childTotal,
                  childPointsDone: cs.childPointsDone,
                  childPointsTotal: cs.childPointsTotal,
                  isGhostChild: true,
                  treeGuides: guides,
                });
              } else {
                addTree(child, depth + 1, false, guides);
              }
            });
          }
        };

        for (const issue of merged) {
          addTree(issue, 0, ghostParentIds.has(issue.id));
        }
      }
    }

    return result;
  }, [
    visibleStatuses,
    filteredIssues,
    activeTab,
    expandedGroups,
    expandedNodes,
    issues,
    childrenByParent,
    sortKey,
    hierarchyMode,
    showEmptyGroups,
    showGhosts,
  ]);
}
