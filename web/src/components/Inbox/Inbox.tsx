import { useState, useMemo, useEffect, useCallback, useRef, lazy, Suspense } from "react";
import { isEditableTarget } from "../../utils/keyboard";
import { fetchUser, type User } from "../../api/client";
import type { InboxItem, Issue } from "../../api/client";
import {
  StatusIcon,
  LabelBadge,
  BellIcon,
  CheckCircleIcon,
} from "../ui";
import { shortName, formatRelativeTime } from "../../utils/format";
import FilterMenu from "../Backlog/FilterMenu";
import { type BacklogFilters, hasActiveFilters } from "../Backlog/filters";

const IssueDetail = lazy(() => import("../IssueDetail/IssueDetail"));

const STATUS_LABEL: Record<string, string> = {
  BACKLOG: "moved to Backlog",
  PLANNED: "marked as Planned",
  DOING: "started working",
  BLOCKED: "marked as Blocked",
  DONE: "completed",
};

interface IssueGroup {
  issueId: string;
  issueTitle: string;
  latestAt: number;
  hasUnread: boolean;
  events: InboxItem[];
}

function groupByIssue(items: InboxItem[], lastReadTime: number): IssueGroup[] {
  const map = new Map<string, IssueGroup>();
  for (const item of items) {
    let group = map.get(item.issue_id);
    if (!group) {
      group = {
        issueId: item.issue_id,
        issueTitle: item.issue_title || item.issue_id,
        latestAt: 0,
        hasUnread: false,
        events: [],
      };
      map.set(item.issue_id, group);
    }
    const t = new Date(item.created_at).getTime();
    if (t > group.latestAt) group.latestAt = t;
    if (t > lastReadTime) group.hasUnread = true;
    group.events.push(item);
  }
  return Array.from(map.values()).sort((a, b) => b.latestAt - a.latestAt);
}

function extractEmail(identity: string): string {
  return identity.match(/<([^>]+)>/)?.[1]?.toLowerCase().trim() || "";
}

function agentLabel(evt: InboxItem, userEmail: string): string | null {
  if (!evt.on_behalf_of) return null;
  const principalEmail = extractEmail(evt.on_behalf_of);
  if (principalEmail && principalEmail === userEmail) {
    return `${shortName(evt.created_by)} (You)`;
  }
  return shortName(evt.created_by);
}

function buildChangeSummary(events: InboxItem[], userEmail: string): string[] {
  const parts: string[] = [];
  let commentCount = 0;
  const statusChanges: string[] = [];
  let assigneeChange: string | null = null;
  let wasMerged = false;
  let wasCreated = false;
  let otherUpdates = 0;
  let actor: string | null = null;

  for (const evt of events) {
    if (!actor) actor = agentLabel(evt, userEmail);
    const p = evt.payload || {};
    switch (evt.type) {
      case "COMMENT":
        commentCount++;
        break;
      case "CREATE":
        wasCreated = true;
        break;
      case "MERGE":
        wasMerged = true;
        break;
      case "UPDATE":
        if (p.status) {
          const label = STATUS_LABEL[String(p.status)] ?? `moved to ${String(p.status)}`;
          if (!statusChanges.includes(label)) statusChanges.push(label);
        } else if (p.assignee !== undefined) {
          const a = String(p.assignee);
          assigneeChange = a ? `reassigned to ${shortName(a)}` : "assignee removed";
        } else {
          otherUpdates++;
        }
        break;
    }
  }

  if (actor) parts.push(actor);
  if (wasCreated) parts.push("Issue created");
  for (const s of statusChanges) parts.push(`Status ${s}`);
  if (wasMerged) parts.push("Merged");
  if (assigneeChange) parts.push(assigneeChange.charAt(0).toUpperCase() + assigneeChange.slice(1));
  if (commentCount > 0) parts.push(`${commentCount} new comment${commentCount !== 1 ? "s" : ""}`);
  if (otherUpdates > 0) parts.push(`${otherUpdates} other update${otherUpdates !== 1 ? "s" : ""}`);
  return parts;
}

function applyFilters(groups: IssueGroup[], issues: Issue[], filters: BacklogFilters): IssueGroup[] {
  return groups.filter(group => {
    const issue = issues.find(i => i.id === group.issueId);
    if (!issue) return true;
    if (filters.statuses.length > 0 && !filters.statuses.includes(issue.status)) return false;
    if (filters.labels.length > 0 && !filters.labels.some(l => issue.labels?.includes(l))) return false;
    if (filters.assignees.length > 0) {
      const match = issue.assignee
        ? filters.assignees.includes(issue.assignee)
        : filters.assignees.includes("__unassigned__");
      if (!match) return false;
    }
    if (filters.priorities.length > 0 && !filters.priorities.includes(issue.priority || 0)) return false;
    if (filters.epicId && issue.parent_id !== filters.epicId) return false;
    return true;
  });
}

// ── Main component ────────────────────────────────────────────────

const GROUPS_PER_PAGE = 30;

interface InboxProps {
  items: InboxItem[];
  lastRead: string;
  issues: Issue[];
  onMarkAllRead: () => void;
  onRefresh: () => void;
  prefix: string;
  contributors: string[];
  onConfigLabelsChange: (labels: Record<string, string>) => void;
  filters: BacklogFilters;
  onFiltersChange: (filters: BacklogFilters) => void;
}

export default function Inbox({
  items,
  lastRead,
  issues,
  onMarkAllRead,
  onRefresh,
  prefix,
  contributors,
  onConfigLabelsChange,
  filters,
  onFiltersChange,
}: InboxProps) {
  const [user, setUser] = useState<User | null>(null);
  const [visibleCount, setVisibleCount] = useState(GROUPS_PER_PAGE);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [focusedIndex, setFocusedIndex] = useState(-1);
  const [keyboardNav, setKeyboardNav] = useState(false);
  const [showFilterMenu, setShowFilterMenu] = useState(false);
  const filterBtnRef = useRef<HTMLButtonElement>(null);

  useEffect(() => {
    fetchUser().then(setUser).catch(() => {});
  }, []);

  const lastReadTime = lastRead ? new Date(lastRead).getTime() : 0;

  const groups = useMemo(() => groupByIssue(items, lastReadTime), [items, lastReadTime]);
  const unreadGroups = useMemo(() => groups.filter(g => g.hasUnread), [groups]);
  const filteredGroups = useMemo(
    () => hasActiveFilters(filters) ? applyFilters(unreadGroups, issues, filters) : unreadGroups,
    [unreadGroups, issues, filters],
  );
  const notificationIssues = useMemo(() => {
    const ids = new Set(unreadGroups.map(g => g.issueId));
    return issues.filter(i => ids.has(i.id));
  }, [unreadGroups, issues]);
  const visibleGroups = filteredGroups.slice(0, visibleCount);
  const hasMore = visibleCount < filteredGroups.length;

  const selectedIssue = selectedId ? issues.find(i => i.id === selectedId) ?? null : null;
  const selectedGroup = selectedId ? groups.find(g => g.issueId === selectedId) ?? null : null;

  const issueIds = useMemo(() => visibleGroups.map(g => g.issueId), [visibleGroups]);
  const selectedNavIndex = selectedId ? issueIds.indexOf(selectedId) : -1;

  const selectGroup = useCallback((issueId: string) => {
    setSelectedId(issueId);
  }, []);

  const handleMarkAllRead = useCallback(() => {
    setSelectedId(null);
    setFocusedIndex(-1);
    onMarkAllRead();
  }, [onMarkAllRead]);

  const navigateIssue = useCallback((direction: "prev" | "next") => {
    const idx = selectedNavIndex + (direction === "prev" ? -1 : 1);
    if (idx >= 0 && idx < issueIds.length) {
      setSelectedId(issueIds[idx]);
      setFocusedIndex(idx);
    }
  }, [selectedNavIndex, issueIds]);

  // Auto-select first group if nothing is selected
  useEffect(() => {
    if (!selectedId && visibleGroups.length > 0) {
      setSelectedId(visibleGroups[0].issueId);
      setFocusedIndex(0);
    }
  }, [visibleGroups, selectedId]);

  // Scroll focused card into view
  useEffect(() => {
    if (!keyboardNav || focusedIndex < 0) return;
    const el = document.querySelector(`[data-inbox-group="${issueIds[focusedIndex]}"]`);
    el?.scrollIntoView({ block: "nearest" });
  }, [focusedIndex, keyboardNav, issueIds]);

  // Keyboard navigation
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (isEditableTarget(e)) return;
      if (e.metaKey || e.ctrlKey) return;

      if (e.key === "f") {
        e.preventDefault();
        setShowFilterMenu(v => !v);
        return;
      }
      if (e.key === "ArrowDown" || e.key === "j") {
        e.preventDefault();
        setKeyboardNav(true);
        setFocusedIndex(i => {
          const next = Math.min(i + 1, visibleGroups.length - 1);
          setSelectedId(visibleGroups[next]?.issueId ?? null);
          return next;
        });
        return;
      }
      if (e.key === "ArrowUp" || e.key === "k") {
        e.preventDefault();
        setKeyboardNav(true);
        setFocusedIndex(i => {
          const next = Math.max(i - 1, 0);
          setSelectedId(visibleGroups[next]?.issueId ?? null);
          return next;
        });
        return;
      }
      if (e.key === "r") {
        e.preventDefault();
        handleMarkAllRead();
        return;
      }
    };
    document.addEventListener("keydown", handler);
    return () => document.removeEventListener("keydown", handler);
  }, [visibleGroups, handleMarkAllRead]);

  const userEmail = user?.email?.toLowerCase().trim() || "";

  const changeSummary = useMemo(() => {
    if (!selectedGroup) return [];
    const unreadEvents = selectedGroup.events.filter(
      e => new Date(e.created_at).getTime() > lastReadTime
    );
    return buildChangeSummary(unreadEvents.length > 0 ? unreadEvents : selectedGroup.events, userEmail);
  }, [selectedGroup, lastReadTime, userEmail]);

  return (
    <div className="flex h-full">
      {/* Left panel — notification cards */}
      <div className="w-80 xl:w-96 shrink-0 flex flex-col border-r border-[var(--color-border-subtle)]">
        {/* Header */}
        <div className="flex items-center gap-3 pl-4 pr-3 h-11 border-b border-[var(--color-border-subtle)] shrink-0">
          <span className="text-sm font-medium text-[var(--color-text-primary)]">
            Notifications
          </span>
          <div className="flex-1" />
          <div className="relative">
            <button
              ref={filterBtnRef}
              onClick={() => setShowFilterMenu(v => !v)}
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
                issues={notificationIssues}
                filters={filters}
                onChange={onFiltersChange}
                anchorRef={filterBtnRef}
                onClose={() => setShowFilterMenu(false)}
              />
            )}
          </div>
          {unreadGroups.length > 0 && (
            <button
              onClick={handleMarkAllRead}
              className="flex items-center gap-1.5 px-2 py-1 rounded-[var(--radius-sm)] text-xs text-[var(--color-text-secondary)] bg-[var(--color-surface-1)] border border-[var(--color-border-default)] hover:bg-[var(--color-hover-surface)] hover:text-[var(--color-text-primary)] transition-colors"
              title="Mark all as read (r)"
            >
              <CheckCircleIcon className="w-3.5 h-3.5" />
              Mark all read
            </button>
          )}
        </div>

        {/* Card list */}
        <div className="flex-1 overflow-y-auto">
          {filteredGroups.length === 0 ? (
            <div className="flex flex-col items-center justify-center h-full gap-3 px-6">
              <BellIcon className="w-10 h-10 text-[var(--color-text-muted)] opacity-20" />
              <div className="text-center">
                <p className="text-sm font-medium text-[var(--color-text-primary)]">
                  {hasActiveFilters(filters) ? "No matching notifications" : "All caught up"}
                </p>
                <p className="text-xs text-[var(--color-text-muted)] mt-1">
                  {hasActiveFilters(filters) ? "Try adjusting your filters." : "No new notifications."}
                </p>
              </div>
            </div>
          ) : (
            <>
              {visibleGroups.map((group, gi) => {
                const issue = issues.find(it => it.id === group.issueId);
                const isSelected = group.issueId === selectedId;
                const isFocused = keyboardNav && gi === focusedIndex;

                return (
                  <div
                    key={group.issueId}
                    data-inbox-group={group.issueId}
                    onClick={() => {
                      selectGroup(group.issueId);
                      setFocusedIndex(gi);
                      setKeyboardNav(false);
                    }}
                    className={`
                      px-4 py-2.5 border-b border-[var(--color-border-subtle)] cursor-pointer transition-colors duration-[var(--duration-fast)]
                      ${isSelected || isFocused
                        ? "bg-[var(--color-surface-2)]"
                        : "hover:bg-[var(--color-hover-surface)]"
                      }
                    `.trim().replace(/\s+/g, " ")}
                  >
                    {/* Top row: status + issue ID … time */}
                    <div className="flex items-center gap-2 min-w-0">
                      {issue && <StatusIcon status={issue.status} size={13} isInferred={issue.is_inferred} />}
                      <span className="font-mono text-xs text-[var(--color-text-muted)] shrink-0">
                        {group.issueId}
                      </span>
                      <span className="ml-auto text-xs tabular-nums text-[var(--color-text-muted)] shrink-0">
                        {formatRelativeTime(new Date(group.latestAt).toISOString())}
                      </span>
                    </div>
                    {/* Title */}
                    <div className={`text-sm truncate mt-1 ${isSelected ? "text-[var(--color-text-primary)] font-medium" : "text-[var(--color-text-primary)]"}`}>
                      {group.issueTitle}
                    </div>
                    {/* Labels */}
                    {issue?.labels && issue.labels.length > 0 && (
                      <div className="flex items-center gap-2 mt-1.5">
                        {issue.labels.map(label => (
                          <LabelBadge key={label} label={label} />
                        ))}
                      </div>
                    )}
                  </div>
                );
              })}
              {hasMore && (
                <div className="flex justify-center py-3">
                  <button
                    onClick={() => setVisibleCount(c => c + GROUPS_PER_PAGE)}
                    className="text-xs text-[var(--color-text-muted)] hover:text-[var(--color-text-secondary)] transition-colors"
                  >
                    Load more ({filteredGroups.length - visibleCount} remaining)
                  </button>
                </div>
              )}
            </>
          )}
        </div>
      </div>

      {/* Right panel — change summary + issue detail */}
      <div className="flex-1 flex flex-col min-w-0 overflow-hidden">
        {selectedIssue ? (
          <div className="flex-1 overflow-hidden">
            <Suspense fallback={null}>
              <IssueDetail
                issue={selectedIssue}
                issues={issues}
                currentIndex={selectedNavIndex}
                totalCount={issueIds.length}
                onClose={() => setSelectedId(null)}
                onNavigate={navigateIssue}
                onRefresh={onRefresh}
                prefix={prefix}
                contributors={contributors}
                onConfigLabelsChange={onConfigLabelsChange}
                banner={changeSummary.length > 0 ? (
                  <div className="flex items-center gap-2 px-5 py-2 border-b border-[var(--color-border-subtle)] bg-[var(--color-surface-2)] shrink-0">
                    <svg className="w-4 h-4 text-[var(--color-accent-primary)] shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
                      <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z" />
                    </svg>
                    <span className="text-xs text-[var(--color-text-secondary)]">
                      {changeSummary.join(" · ")}
                    </span>
                  </div>
                ) : undefined}
              />
            </Suspense>
          </div>
        ) : (
          <div className="flex flex-col items-center justify-center h-full gap-3">
            <BellIcon className="w-12 h-12 text-[var(--color-text-muted)] opacity-20" />
            <p className="text-sm text-[var(--color-text-muted)]">
              Select a notification to view details
            </p>
          </div>
        )}
      </div>
    </div>
  );
}
