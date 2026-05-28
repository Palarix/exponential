import { useMemo } from "react";
import type { Issue } from "../../api/client";
import { sortGroup } from "../../utils/sort";
import type { SortKey } from "../../utils/sort";
import type { Tab } from "./Backlog";

const TAB_CONFIGS: Record<Tab, { statuses: string[] }> = {
  all: { statuses: ["BACKLOG", "PLANNED", "DOING", "BLOCKED", "DONE"] },
  active: { statuses: ["PLANNED", "DOING", "BLOCKED"] },
  backlog: { statuses: ["BACKLOG"] },
};

export type RowItem =
  | {
      kind: "group";
      status: string;
      label: string;
      count: number;
      isEmpty: boolean;
    }
  | {
      kind: "issue";
      issue: Issue;
      depth: number;
      hasChildren: boolean;
      childDone: number;
      childTotal: number;
      parentBreadcrumb?: string;
      isGhostParent?: boolean;
    };

const STATUS_META: Record<string, { label: string }> = {
  BACKLOG: { label: "Backlog" },
  PLANNED: { label: "Planned" },
  DOING: { label: "In Progress" },
  BLOCKED: { label: "Blocked" },
  DONE: { label: "Done" },
};

export function useBacklogRows(
  issues: Issue[],
  filteredIssues: Issue[],
  activeTab: Tab,
  expandedGroups: Set<string>,
  expandedNodes: Set<string>,
  sortKey: SortKey,
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
    const isAllTab = activeTab === "all";

    for (const status of visibleStatuses) {
      const groupIssues = filteredIssues.filter((i) => i.status === status);
      if (groupIssues.length === 0 && !isAllTab) continue;

      result.push({
        kind: "group",
        status,
        label: STATUS_META[status]?.label || status,
        count: groupIssues.length,
        isEmpty: groupIssues.length === 0,
      });

      if (expandedGroups.has(status) && groupIssues.length > 0) {
        const groupIssueIds = new Set(groupIssues.map((i) => i.id));
        const effectiveSortKey = status === "DONE" ? ("updated" as const) : sortKey;

        if (isAllTab) {
          const topLevel = sortGroup(
            groupIssues.filter((i) => !i.parent_id || !groupIssueIds.has(i.parent_id)),
            effectiveSortKey,
          );
          const addTree = (issue: Issue, depth: number, breadcrumb?: string) => {
            const allChildren = childrenByParent.get(issue.id) || [];
            const doneCount = allChildren.filter((c) => c.status === "DONE").length;
            result.push({
              kind: "issue",
              issue,
              depth,
              hasChildren: allChildren.length > 0,
              childDone: doneCount,
              childTotal: allChildren.length,
              parentBreadcrumb: breadcrumb,
            });
            const visibleChildren = sortGroup(
              allChildren.filter((c) => groupIssueIds.has(c.id)),
              effectiveSortKey,
            );
            if (visibleChildren.length > 0 && expandedNodes.has(issue.id)) {
              for (const child of visibleChildren) addTree(child, depth + 1);
            }
          };
          for (const issue of topLevel) {
            const parent = issue.parent_id ? issues.find((i) => i.id === issue.parent_id) : null;
            addTree(issue, 0, parent ? parent.title : undefined);
          }
        } else {
          const topLevel = sortGroup(
            groupIssues.filter((i) => !i.parent_id),
            effectiveSortKey,
          );
          const orphanedChildren = groupIssues.filter(
            (i) => i.parent_id && !groupIssueIds.has(i.parent_id),
          );
          const ghostParentIds = new Set(orphanedChildren.map((i) => i.parent_id!));

          const addTree = (issue: Issue, depth: number, isGhost?: boolean) => {
            const allChildren = childrenByParent.get(issue.id) || [];
            const doneCount = allChildren.filter((c) => c.status === "DONE").length;
            result.push({
              kind: "issue",
              issue,
              depth,
              hasChildren: allChildren.length > 0,
              childDone: doneCount,
              childTotal: allChildren.length,
              isGhostParent: isGhost,
            });
            const visibleChildren = sortGroup(
              allChildren.filter((c) => groupIssueIds.has(c.id)),
              effectiveSortKey,
            );
            if (visibleChildren.length > 0 && (isGhost || expandedNodes.has(issue.id))) {
              for (const child of visibleChildren) addTree(child, depth + 1);
            }
          };

          for (const issue of topLevel) addTree(issue, 0);
          for (const parentId of ghostParentIds) {
            const parent = issues.find((i) => i.id === parentId);
            if (parent) addTree(parent, 0, true);
          }
        }
      }
    }
    return result;
  }, [visibleStatuses, filteredIssues, activeTab, expandedGroups, expandedNodes, issues, childrenByParent, sortKey]);
}
