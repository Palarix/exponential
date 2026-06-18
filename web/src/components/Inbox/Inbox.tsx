import { useState, useMemo, useEffect, type ReactNode } from "react";
import { isEditableTarget } from "../../utils/keyboard";
import Markdown from "react-markdown";
import remarkBreaks from "remark-breaks";
import remarkGfm from "remark-gfm";
import type { InboxItem, Issue } from "../../api/client";
import {
  StatusIcon,
  PlusIcon,
  CommentIcon,
  MergeIcon,
  CheckCircleIcon,
  PlayIcon,
  BlockedIcon,
  ArchiveIcon,
  UserIcon,
} from "../ui";
import { shortName, formatRelativeTime, stripMarkdown } from "../../utils/format";

type ActIconKey =
  | "create" | "comment" | "merge"
  | "status-done" | "status-doing" | "status-blocked" | "status-planned" | "status-backlog"
  | "assign" | "update";

const ICON_COLORS: Partial<Record<ActIconKey, string>> = {
  "status-done": "text-[var(--color-success)]",
  "status-doing": "text-[var(--color-warning)]",
  "status-blocked": "text-[var(--color-error)]",
  "status-planned": "text-[var(--color-text-secondary)]",
};

function ActIcon({ k }: { k: ActIconKey }) {
  const color = ICON_COLORS[k] ?? "text-[var(--color-text-muted)]";
  const cls = `w-4 h-4 shrink-0 ${color}`;

  const icons: Record<ActIconKey, ReactNode> = {
    create: <PlusIcon className={cls} />,
    comment: <CommentIcon className={cls} />,
    merge: <MergeIcon className={cls} />,
    "status-done": <CheckCircleIcon className={cls} />,
    "status-doing": <PlayIcon className={cls} />,
    "status-blocked": <BlockedIcon className={cls} />,
    "status-planned": (
      <svg className={cls} fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.75}>
        <path strokeLinecap="round" strokeLinejoin="round" d="M6.75 3v2.25M17.25 3v2.25M3 18.75V7.5a2.25 2.25 0 012.25-2.25h13.5A2.25 2.25 0 0121 7.5v11.25m-18 0A2.25 2.25 0 005.25 21h13.5A2.25 2.25 0 0021 18.75m-18 0v-7.5A2.25 2.25 0 015.25 9h13.5A2.25 2.25 0 0121 11.25v7.5" />
      </svg>
    ),
    "status-backlog": <ArchiveIcon className={cls} />,
    assign: <UserIcon className={cls} />,
    update: (
      <svg className={cls} fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.75}>
        <path strokeLinecap="round" strokeLinejoin="round" d="M16.862 4.487l1.687-1.688a1.875 1.875 0 112.652 2.652L10.582 16.07a4.5 4.5 0 01-1.897 1.13L6 18l.8-2.685a4.5 4.5 0 011.13-1.897l8.932-8.931z" />
      </svg>
    ),
  };
  return <>{icons[k]}</>;
}

const STATUS_LABEL: Record<string, { verb: string; icon: ActIconKey }> = {
  BACKLOG: { verb: "moved this to Backlog", icon: "status-backlog" },
  PLANNED: { verb: "marked this as Planned", icon: "status-planned" },
  DOING: { verb: "started working on this", icon: "status-doing" },
  BLOCKED: { verb: "marked this as Blocked", icon: "status-blocked" },
  DONE: { verb: "completed this", icon: "status-done" },
};

interface EventDescription {
  icon: ActIconKey;
  sentence: ReactNode;
  detail?: string;
}

function describeItem(item: InboxItem, who: string): EventDescription | null {
  const p = item.payload || {};
  const name = <span className="font-medium text-[var(--color-text-primary)]">{who}</span>;
  switch (item.type) {
    case "CREATE":
      return { icon: "create", sentence: <>{name} created this issue</> };
    case "COMMENT":
      return { icon: "comment", sentence: <>{name} left a comment</>, detail: String(p.text ?? "") };
    case "MERGE": {
      const strategy = p.strategy ? ` via ${String(p.strategy)}` : "";
      return { icon: "merge", sentence: <>{name} merged this{strategy}</> };
    }
    case "UPDATE": {
      if (p.status) {
        const s = STATUS_LABEL[String(p.status)] ?? { verb: `moved this to ${String(p.status)}`, icon: "status-backlog" as ActIconKey };
        return { icon: s.icon, sentence: <>{name} {s.verb}</> };
      }
      if (p.assignee !== undefined) {
        const assignee = String(p.assignee);
        if (!assignee) return { icon: "assign", sentence: <>{name} removed the assignee</> };
        return { icon: "assign", sentence: <>{name} assigned this to <span className="font-medium text-[var(--color-text-primary)]">{shortName(assignee)}</span></> };
      }
      return { icon: "update", sentence: <>{name} updated this issue</> };
    }
    default:
      return null;
  }
}

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

const GROUPS_PER_PAGE = 20;

interface InboxProps {
  items: InboxItem[];
  lastRead: string;
  issues: Issue[];
  onIssueClick?: (issue: Issue) => void;
  onMarkAllRead: () => void;
}

type Tab = "new" | "read";

export default function Inbox({ items, lastRead, issues, onIssueClick, onMarkAllRead }: InboxProps) {
  const [tab, setTab] = useState<Tab>("new");
  const [expandedGroups, setExpandedGroups] = useState<Set<string>>(() => new Set());
  const [expandedComments, setExpandedComments] = useState<Set<string>>(() => new Set());
  const [visibleCount, setVisibleCount] = useState(GROUPS_PER_PAGE);

  const lastReadTime = lastRead ? new Date(lastRead).getTime() : 0;

  const groups = useMemo(() => groupByIssue(items, lastReadTime), [items, lastReadTime]);
  const unreadGroups = useMemo(() => groups.filter(g => g.hasUnread), [groups]);
  const readGroups = useMemo(() => groups.filter(g => !g.hasUnread), [groups]);
  const activeGroups = tab === "new" ? unreadGroups : readGroups;
  const visibleGroups = activeGroups.slice(0, visibleCount);
  const hasMore = visibleCount < activeGroups.length;

  const [focusedIndex, setFocusedIndex] = useState(-1);
  const [keyboardNav, setKeyboardNav] = useState(false);

  const focusedGroup = keyboardNav && focusedIndex >= 0 ? visibleGroups[focusedIndex] : null;

  useEffect(() => {
    if (!keyboardNav || !focusedGroup) return;
    const el = document.querySelector(`[data-inbox-group="${focusedGroup.issueId}"]`);
    el?.scrollIntoView({ block: "nearest" });
  }, [focusedGroup, keyboardNav]);

  useEffect(() => {
    setFocusedIndex(-1);
    setKeyboardNav(false);
  }, [tab]);

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (isEditableTarget(e)) return;
      if (e.metaKey || e.ctrlKey) return;

      if (e.key === "ArrowDown" || e.key === "j") {
        e.preventDefault();
        setKeyboardNav(true);
        setFocusedIndex(i => Math.min(i + 1, visibleGroups.length - 1));
        return;
      }
      if (e.key === "ArrowUp" || e.key === "k") {
        e.preventDefault();
        setKeyboardNav(true);
        setFocusedIndex(i => Math.max(i - 1, 0));
        return;
      }
      if (e.key === "ArrowRight" && focusedGroup) {
        e.preventDefault();
        if (!expandedGroups.has(focusedGroup.issueId)) toggleGroup(focusedGroup.issueId);
        return;
      }
      if (e.key === "ArrowLeft" && focusedGroup) {
        e.preventDefault();
        if (expandedGroups.has(focusedGroup.issueId)) toggleGroup(focusedGroup.issueId);
        return;
      }
      if (e.key === "r") {
        e.preventDefault();
        onMarkAllRead();
        return;
      }
      if (e.key === "Enter" && focusedGroup) {
        e.preventDefault();
        const issue = issues.find(it => it.id === focusedGroup.issueId);
        if (issue) onIssueClick?.(issue);
        return;
      }
    };
    document.addEventListener("keydown", handler);
    return () => document.removeEventListener("keydown", handler);
  }, [visibleGroups, focusedGroup, issues, onIssueClick, expandedGroups]);

  const toggleGroup = (id: string) => {
    setExpandedGroups(prev => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  const toggleComment = (key: string) => {
    setExpandedComments(prev => {
      const next = new Set(prev);
      if (next.has(key)) next.delete(key);
      else next.add(key);
      return next;
    });
  };

  return (
    <div className="flex flex-col h-full">
      {/* Header with tabs */}
      <div className="flex items-center gap-4 px-6 py-3 border-b border-[var(--color-border-subtle)]">
        <button
          onClick={() => { setTab("new"); setVisibleCount(GROUPS_PER_PAGE); }}
          className={`text-sm pb-0.5 border-b-2 transition-colors ${tab === "new" ? "text-[var(--color-text-primary)] font-medium border-[var(--color-accent-primary)]" : "text-[var(--color-text-muted)] border-transparent hover:text-[var(--color-text-secondary)]"}`}
        >
          New{unreadGroups.length > 0 && ` (${unreadGroups.length})`}
        </button>
        <button
          onClick={() => { setTab("read"); setVisibleCount(GROUPS_PER_PAGE); }}
          className={`text-sm pb-0.5 border-b-2 transition-colors ${tab === "read" ? "text-[var(--color-text-primary)] font-medium border-[var(--color-accent-primary)]" : "text-[var(--color-text-muted)] border-transparent hover:text-[var(--color-text-secondary)]"}`}
        >
          Read
        </button>
        <div className="flex-1" />
        {tab === "new" && unreadGroups.length > 0 && (
          <button
            onClick={onMarkAllRead}
            className="px-3 py-1.5 text-xs text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] bg-[var(--color-surface-1)] hover:bg-[var(--color-hover-surface)] border border-[var(--color-border-default)] rounded-[var(--radius-md)] transition-colors"
          >
            Mark all as read
          </button>
        )}
      </div>

      {/* Feed */}
      <div className="flex-1 overflow-y-auto">
        {activeGroups.length === 0 ? (
          <div className="flex flex-col items-center justify-center h-full gap-4 px-8">
            <svg width="120" height="90" viewBox="0 0 120 90" fill="none" className="opacity-20">
              <rect x="20" y="10" width="80" height="55" rx="6" stroke="var(--color-text-muted)" strokeWidth="1.5" strokeDasharray="4 3" />
              <path d="M20 25l40 22 40-22" stroke="var(--color-text-muted)" strokeWidth="1.5" />
              <path d="M40 75h40" stroke="var(--color-text-muted)" strokeWidth="1.5" strokeLinecap="round" strokeDasharray="3 4" />
            </svg>
            <div className="text-center">
              <p className="text-sm font-medium text-[var(--color-text-primary)]">
                {tab === "new" ? "All caught up" : "Nothing here yet"}
              </p>
              <p className="text-xs text-[var(--color-text-muted)] mt-1">
                {tab === "new" ? "No new notifications." : "Read notifications will appear here."}
              </p>
            </div>
          </div>
        ) : (
          <div className="max-w-4xl mx-auto py-4 px-4 space-y-3">
            {visibleGroups.map((group, gi) => {
              const issue = issues.find(it => it.id === group.issueId);
              const isFocused = keyboardNav && gi === focusedIndex;

              return (
                <div
                  key={group.issueId}
                  data-inbox-group={group.issueId}
                  className={`
                    rounded-[var(--radius-md)] border transition-colors
                    ${isFocused
                      ? "bg-[var(--color-surface-1)] border-[var(--color-accent-primary)] ring-1 ring-[var(--color-accent-primary)]"
                      : group.hasUnread
                        ? "bg-[var(--color-surface-1)] border-[var(--color-border-default)]"
                        : "bg-[var(--color-surface-1)] border-[var(--color-border-subtle)]"
                    }
                  `.trim().replace(/\s+/g, " ")}
                >
                  {/* Card header: status + ID + title + event count */}
                  <div className="flex items-center gap-2 px-4 py-3">
                    {issue && <StatusIcon status={issue.status} size={14} isInferred={issue.is_inferred} />}
                    <a
                      href={`#/issues/${group.issueId}`}
                      onClick={(e) => {
                        if (issue) {
                          e.preventDefault();
                          onIssueClick?.(issue);
                        }
                      }}
                      className="font-mono text-sm font-semibold text-[var(--color-accent-primary)] hover:text-[var(--color-accent-primary-hover)] transition-colors shrink-0"
                    >
                      {group.issueId}
                    </a>
                    <span className="text-sm truncate text-[var(--color-text-primary)]">
                      {group.issueTitle}
                    </span>
                    <span className="ml-auto text-xs tabular-nums text-[var(--color-text-muted)] shrink-0">
                      {group.events.length} {group.events.length === 1 ? "event" : "events"}
                    </span>
                  </div>

                  {/* Event cards — latest always shown, older behind toggle */}
                  <div className="px-3 pb-3 pt-1 space-y-2">
                    {(() => {
                      const latest = group.events[0];
                      const olderEvents = group.events.slice(1);
                      const isGroupOpen = expandedGroups.has(group.issueId);

                      const renderEvent = (evt: InboxItem, i: number) => {
                        const who = shortName(evt.created_by);
                        const desc = describeItem(evt, who);
                        if (!desc) return null;
                        const evtKey = `${evt.issue_id}-${evt.created_at}-${i}`;
                        const isExpanded = expandedComments.has(evtKey);
                        const hasDetail = !!desc.detail;
                        const previewText = desc.detail ? stripMarkdown(desc.detail) : "";
                        return (
                          <div
                            key={evtKey}
                            className={`rounded-[var(--radius-md)] bg-[var(--color-surface-2)] border border-[var(--color-border-subtle)] ${hasDetail ? "cursor-pointer hover:border-[var(--color-border-default)]" : ""} transition-colors`}
                            onClick={hasDetail ? () => toggleComment(evtKey) : undefined}
                          >
                            <div className="flex items-center gap-2.5 px-3 py-2.5">
                              <ActIcon k={desc.icon} />
                              <span className="text-sm text-[var(--color-text-secondary)] min-w-0 truncate">
                                {desc.sentence}
                              </span>
                              <span className="ml-auto text-xs tabular-nums text-[var(--color-text-muted)] shrink-0 whitespace-nowrap">
                                {formatRelativeTime(evt.created_at)}
                              </span>
                              {hasDetail && (
                                <svg className={`w-3.5 h-3.5 shrink-0 text-[var(--color-text-muted)] transition-transform ${isExpanded ? "rotate-180" : ""}`} fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                                  <path strokeLinecap="round" strokeLinejoin="round" d="M19.5 8.25l-7.5 7.5-7.5-7.5" />
                                </svg>
                              )}
                            </div>
                            {hasDetail && !isExpanded && (
                              <div className="px-3 pb-2.5 -mt-1">
                                <p className="text-xs text-[var(--color-text-muted)] italic truncate pl-6">
                                  &ldquo;{previewText.slice(0, 200)}{previewText.length > 200 ? "…" : ""}&rdquo;
                                </p>
                              </div>
                            )}
                            {hasDetail && isExpanded && (
                              <div className="px-3 pb-3 -mt-0.5">
                                <div className="ml-6 rounded-[var(--radius-md)] bg-[var(--color-surface)] border border-[var(--color-border-default)] px-3 py-2">
                                  <div className="prose-exponential text-sm">
                                    <Markdown remarkPlugins={[remarkGfm, remarkBreaks]}>
                                      {desc.detail!}
                                    </Markdown>
                                  </div>
                                </div>
                              </div>
                            )}
                          </div>
                        );
                      };

                      return (
                        <>
                          {renderEvent(latest, 0)}
                          {olderEvents.length > 0 && !isGroupOpen && (
                            <button
                              onClick={() => toggleGroup(group.issueId)}
                              className="w-full text-xs text-[var(--color-text-muted)] hover:text-[var(--color-text-secondary)] py-1 transition-colors"
                            >
                              {olderEvents.length} more {olderEvents.length === 1 ? "event" : "events"}
                            </button>
                          )}
                          {isGroupOpen && olderEvents.map((evt, i) => renderEvent(evt, i + 1))}
                          {isGroupOpen && olderEvents.length > 0 && (
                            <button
                              onClick={() => toggleGroup(group.issueId)}
                              className="w-full text-xs text-[var(--color-text-muted)] hover:text-[var(--color-text-secondary)] py-1 transition-colors"
                            >
                              Show less
                            </button>
                          )}
                        </>
                      );
                    })()}
                  </div>
                </div>
              );
            })}

            {hasMore && (
              <div className="flex justify-center pt-2 pb-4">
                <button
                  onClick={() => setVisibleCount(c => c + GROUPS_PER_PAGE)}
                  className="px-4 py-2 text-sm text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] bg-[var(--color-surface-1)] hover:bg-[var(--color-hover-surface)] border border-[var(--color-border-default)] rounded-[var(--radius-md)] transition-colors"
                >
                  Load more ({groups.length - visibleCount} remaining)
                </button>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
