import { useState, useEffect, useMemo, useRef, useCallback } from "react";
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
  UserIcon,
} from "../ui";
import { formatShortDate } from "../../utils/format";
import { isEditableTarget } from "../../utils/keyboard";
import FilterMenu from "../Backlog/FilterMenu";
import { type BacklogFilters, hasActiveFilters } from "../Backlog/filters";

export type MyIssuesTab = "assigned" | "created";

const TAB_CONFIGS: Record<MyIssuesTab, { label: string }> = {
  assigned: { label: "Assigned to Me" },
  created: { label: "Created by Me" },
};

function extractEmail(identity: string): string {
  return identity.match(/<([^>]+)>/)?.[1]?.toLowerCase().trim() || "";
}

function applyFilters(issues: Issue[], filters: BacklogFilters): Issue[] {
  return issues.filter((i) => {
    if (filters.statuses.length > 0 && !filters.statuses.includes(i.status))
      return false;
    if (
      filters.labels.length > 0 &&
      !filters.labels.some((l) => i.labels?.includes(l))
    )
      return false;
    if (filters.assignees.length > 0) {
      const match = i.assignee
        ? filters.assignees.includes(i.assignee)
        : filters.assignees.includes("__unassigned__");
      if (!match) return false;
    }
    if (
      filters.priorities.length > 0 &&
      !filters.priorities.includes(i.priority || 0)
    )
      return false;
    if (filters.epicId && i.parent_id !== filters.epicId) return false;
    return true;
  });
}

interface MyIssuesProps {
  issues: Issue[];
  onIssueClick: (issue: Issue) => void;
  activeTab: MyIssuesTab;
  onTabChange: (tab: MyIssuesTab) => void;
  filters: BacklogFilters;
  onFiltersChange: (filters: BacklogFilters) => void;
}

export default function MyIssues({
  issues,
  onIssueClick,
  activeTab,
  onTabChange,
  filters,
  onFiltersChange,
}: MyIssuesProps) {
  const [user, setUser] = useState<User | null>(null);
  const [showFilterMenu, setShowFilterMenu] = useState(false);
  const showFilterMenuRef = useRef(showFilterMenu);
  showFilterMenuRef.current = showFilterMenu;
  const filterBtnRef = useRef<HTMLButtonElement>(null);

  useEffect(() => {
    fetchUser()
      .then(setUser)
      .catch(() => {});
  }, []);

  const handleKeyDown = useCallback((e: KeyboardEvent) => {
    if (isEditableTarget(e)) return;
    if (e.metaKey || e.ctrlKey) return;
    if (e.key === "f") {
      e.preventDefault();
      setShowFilterMenu((v) => !v);
      return;
    }
    if (showFilterMenuRef.current && e.key === "Escape") {
      e.preventDefault();
      setShowFilterMenu(false);
    }
  }, []);

  useEffect(() => {
    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [handleKeyDown]);

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
      <div className="flex items-center gap-4 px-5 h-11 border-b border-[var(--color-border-subtle)] shrink-0">
        {(Object.entries(TAB_CONFIGS) as [MyIssuesTab, { label: string }][]).map(
          ([id, config]) => (
            <button
              key={id}
              onClick={() => onTabChange(id)}
              className={`text-sm font-medium h-full border-b-2 -mb-px transition-colors duration-[var(--duration-fast)] ${
                activeTab === id
                  ? "text-[var(--color-text-primary)] border-[var(--color-text-primary)]"
                  : "text-[var(--color-text-muted)] border-transparent hover:text-[var(--color-text-secondary)]"
              }`}
            >
              {config.label}
            </button>
          ),
        )}
        <div className="ml-auto flex items-center gap-3">
          <div className="relative">
            <button
              ref={filterBtnRef}
              onClick={() => setShowFilterMenu((v) => !v)}
              className="flex items-center gap-1 h-6 px-2 rounded-[var(--radius-sm)] text-xs text-[var(--color-text-muted)] hover:text-[var(--color-text-secondary)] hover:bg-[var(--color-hover-surface)] transition-colors relative"
            >
              <svg
                className="w-4 h-4"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                strokeWidth={2}
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  d="M12 3c2.755 0 5.455.232 8.083.678.533.09.917.556.917 1.096v1.044a2.25 2.25 0 01-.659 1.591l-5.432 5.432a2.25 2.25 0 00-.659 1.591v2.927a2.25 2.25 0 01-1.244 2.013L9.75 21v-6.568a2.25 2.25 0 00-.659-1.591L3.659 7.409A2.25 2.25 0 013 5.818V4.774c0-.54.384-1.006.917-1.096A48.32 48.32 0 0112 3z"
                />
              </svg>
              Filter
              {hasActiveFilters(filters) && (
                <span className="absolute -top-0.5 -right-0.5 w-2 h-2 rounded-full bg-[var(--color-accent-primary)]" />
              )}
            </button>
            {showFilterMenu && (
              <FilterMenu
                issues={tabIssues}
                filters={filters}
                onChange={onFiltersChange}
                anchorRef={filterBtnRef}
                onClose={() => setShowFilterMenu(false)}
              />
            )}
          </div>
          <span className="text-xs text-[var(--color-text-muted)] tabular-nums">
            {filtered.length} issue{filtered.length !== 1 ? "s" : ""}
          </span>
        </div>
      </div>

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
    </div>
  );
}
