import { useState, useEffect, useMemo, useCallback } from "react";
import { fetchUser, type User } from "../../api/client";
import type { Issue } from "../../api/types";
import {
  Avatar,
  BranchBadge,
  StatusIcon,
  PriorityIcon,
  CopyableId,
  LabelBadge,
  EmptyState,
  TopBar,
  ContextMenu,
  FilterButton,
  CountBadge,
} from "../ui";
import { User as UserIcon } from "lucide-react";
import { extractEmail, formatShortDate } from "../../utils/format";
import { isEditableTarget } from "../../utils/keyboard";
import { useAllLabels } from "../../hooks/useLabels";
import { type BacklogFilters, hasActiveFilters, matchesFilters } from "../Backlog/filters";
import { useKeyboardHandler } from "../../keyboard";
import { Tabs } from "../ui/Tabs";

export type MyIssuesTab = "assigned" | "created";

const TAB_CONFIGS: Record<MyIssuesTab, { label: string }> = {
  assigned: { label: "Assigned to Me" },
  created: { label: "Created by Me" },
};

function applyFilters(issues: Issue[], filters: BacklogFilters): Issue[] {
  return issues.filter((i) => matchesFilters(i, filters));
}

interface MyIssuesProps {
  issues: Issue[];
  onIssueClick: (issue: Issue) => void;
  activeTab: MyIssuesTab;
  onTabChange: (tab: MyIssuesTab) => void;
  filters: BacklogFilters;
  onFiltersChange: (filters: BacklogFilters) => void;
  onRefresh?: () => void;
  contributors?: string[];
  onConfigLabelsChange?: (labels: Record<string, string>) => void;
  patchIssue?: (issueId: string, patch: Partial<Issue>) => void;
}

export default function MyIssues({
  issues,
  onIssueClick,
  activeTab,
  onTabChange,
  filters,
  onFiltersChange,
  onRefresh,
  contributors = [],
  onConfigLabelsChange,
  patchIssue,
}: MyIssuesProps) {
  const [user, setUser] = useState<User | null>(null);
  const [contextMenu, setContextMenu] = useState<{ issueId: string; x: number; y: number } | null>(null);

  useEffect(() => {
    fetchUser()
      .then(setUser)
      .catch(() => {});
  }, []);

  const handleKeyDown = useCallback((e: KeyboardEvent) => {
    if (isEditableTarget(e)) return;
    if (e.metaKey || e.ctrlKey) return;
  }, []);

  useKeyboardHandler({
    scope: "my-issues",
    priority: "view",
    handler: handleKeyDown,
    shortcuts: [],
  });

  const allKnownLabels = useAllLabels(issues);
  const userEmail = user?.email?.toLowerCase().trim() || "";

  const { assigned, created } = useMemo(() => {
    if (!userEmail) return { assigned: [] as Issue[], created: [] as Issue[] };
    const byDate = (a: Issue, b: Issue) =>
      new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime();
    return {
      assigned: issues
        .filter((i) => i.assignee && extractEmail(i.assignee) === userEmail)
        .sort(byDate),
      created: issues
        .filter((i) => extractEmail(i.created_by) === userEmail)
        .sort(byDate),
    };
  }, [issues, userEmail]);

  const tabIssues = activeTab === "assigned" ? assigned : created;
  const filtered = useMemo(
    () => (hasActiveFilters(filters) ? applyFilters(tabIssues, filters) : tabIssues),
    [tabIssues, filters],
  );

  return (
    <div className="flex flex-col h-full">
      {/* Header */}
      <TopBar
        left={
          <Tabs
            items={TAB_CONFIGS}
            activeId={activeTab}
            onChange={onTabChange}
            keyboardNavigationEnabled={!contextMenu}
          />
        }
        right={
          <div className="flex items-center gap-2">
            <FilterButton issues={tabIssues} filters={filters} onFiltersChange={onFiltersChange} />
            <CountBadge count={filtered.length} />
          </div>
        }
      />

      {/* List */}
      {filtered.length === 0 ? (
        <EmptyState
          icon={<UserIcon className="w-16 h-16" />}
          title={
            activeTab === "assigned"
              ? "No issues assigned to you"
              : "No issues created by you"
          }
          description={
            hasActiveFilters(filters)
              ? "Try adjusting your filters."
              : activeTab === "assigned"
                ? "Issues assigned to you will appear here."
                : "Issues you've created will appear here."
          }
        />
      ) : (
        <div className="flex-1 overflow-y-auto">
          {filtered.map((issue) => (
            <div
              key={issue.id}
              onClick={() => onIssueClick(issue)}
              onContextMenu={(e) => { e.preventDefault(); setContextMenu({ issueId: issue.id, x: e.clientX, y: e.clientY }); }}
              className="flex items-center gap-3 px-5 h-10 border-b border-[var(--color-border-subtle)] cursor-pointer transition-colors duration-[var(--duration-fast)] hover:bg-[var(--color-hover-surface)] group"
            >
              <PriorityIcon priority={issue.priority || 0} size={16} />
              <CopyableId
                id={issue.id}
                className="text-xs w-24 text-left shrink-0 truncate tabular-nums"
              />
              <StatusIcon
                status={issue.status}
                size={14}
                isInferred={issue.is_inferred}
              />
              <span className="text-sm truncate min-w-0 text-[var(--color-text-primary)]">
                {issue.title}
              </span>
              {issue.branch_stats && (
                <BranchBadge stats={issue.branch_stats} />
              )}
              <div className="flex-1" />
              {issue.labels?.map((label) => (
                <LabelBadge key={label} label={label} />
              ))}
              {issue.assignee && <Avatar name={issue.assignee} size="sm" />}
              <span className="text-xs text-[var(--color-text-muted)] tabular-nums shrink-0 w-16 text-right">
                {formatShortDate(issue.updated_at)}
              </span>
            </div>
          ))}
        </div>
      )}

      {contextMenu && onRefresh && (() => {
        const ctxIssue = issues.find((i) => i.id === contextMenu.issueId);
        if (!ctxIssue) return null;
        return (
          <ContextMenu
            issue={ctxIssue}
            issues={issues}
            x={contextMenu.x}
            y={contextMenu.y}
            onClose={() => setContextMenu(null)}
            onRefresh={onRefresh}
            allLabels={allKnownLabels}
            contributors={contributors}
            onConfigLabelsChange={onConfigLabelsChange}
            patchIssue={patchIssue}
          />
        );
      })()}
    </div>
  );
}
