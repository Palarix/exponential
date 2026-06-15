import { useState, useMemo, type ReactNode } from "react";
import Markdown from "react-markdown";
import remarkBreaks from "remark-breaks";
import remarkGfm from "remark-gfm";
import type { InboxItem, Issue } from "../../api/client";
import { Avatar } from "../ui";
import { shortName, formatRelativeTime, stripMarkdown } from "../../utils/format";

type ActIconKey =
  | "create" | "comment" | "merge"
  | "status-done" | "status-doing" | "status-blocked" | "status-planned" | "status-backlog"
  | "assign" | "update";

const ICON_COLORS: Partial<Record<ActIconKey, string>> = {
  "status-done": "text-[var(--color-success)]",
  "status-doing": "text-[var(--color-accent-primary)]",
  "status-blocked": "text-[var(--color-error)]",
  "status-planned": "text-[var(--color-warning)]",
  merge: "text-[var(--color-success)]",
  assign: "text-[var(--color-accent-primary)]",
  comment: "text-[var(--color-accent-primary)]",
  create: "text-[var(--color-success)]",
};

function ActIcon({ k }: { k: ActIconKey }) {
  const color = ICON_COLORS[k] ?? "text-[var(--color-text-muted)]";
  const paths: Record<ActIconKey, ReactNode> = {
    create: <path strokeLinecap="round" strokeLinejoin="round" d="M12 4.5v15m7.5-7.5h-15" />,
    comment: <path strokeLinecap="round" strokeLinejoin="round" d="M2.25 12.76c0 1.6 1.123 2.994 2.707 3.227 1.068.157 2.148.279 3.238.364.466.037.893.281 1.153.671L12 21l2.652-3.978c.26-.39.687-.634 1.153-.671 1.09-.085 2.17-.207 3.238-.364 1.584-.233 2.707-1.626 2.707-3.228V6.741c0-1.602-1.123-2.995-2.707-3.228A48.394 48.394 0 0012 3c-2.392 0-4.744.175-7.043.513C3.373 3.746 2.25 5.14 2.25 6.741v6.018z" />,
    merge: <path strokeLinecap="round" strokeLinejoin="round" d="M3 7.5L7.5 3m0 0L12 7.5M7.5 3v13.5m13.5-3L16.5 18m0 0L12 13.5M16.5 18V4.5" />,
    "status-done": <path strokeLinecap="round" strokeLinejoin="round" d="M9 12.75L11.25 15 15 9.75M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />,
    "status-doing": <path strokeLinecap="round" strokeLinejoin="round" d="M5.25 5.653c0-.856.917-1.398 1.667-.986l11.54 6.347a1.125 1.125 0 010 1.972l-11.54 6.347a1.125 1.125 0 01-1.667-.986V5.653z" />,
    "status-blocked": <path strokeLinecap="round" strokeLinejoin="round" d="M18.364 18.364A9 9 0 005.636 5.636m12.728 12.728A9 9 0 015.636 5.636m12.728 12.728L5.636 5.636" />,
    "status-planned": <path strokeLinecap="round" strokeLinejoin="round" d="M6.75 3v2.25M17.25 3v2.25M3 18.75V7.5a2.25 2.25 0 012.25-2.25h13.5A2.25 2.25 0 0121 7.5v11.25m-18 0A2.25 2.25 0 005.25 21h13.5A2.25 2.25 0 0021 18.75m-18 0v-7.5A2.25 2.25 0 015.25 9h13.5A2.25 2.25 0 0121 11.25v7.5" />,
    "status-backlog": <path strokeLinecap="round" strokeLinejoin="round" d="M20.25 7.5l-.625 10.632a2.25 2.25 0 01-2.247 2.118H6.622a2.25 2.25 0 01-2.247-2.118L3.75 7.5m6 4.125l2.25 2.25m0 0l2.25 2.25M12 13.875l2.25-2.25M12 13.875l-2.25 2.25M3.375 7.5h17.25c.621 0 1.125-.504 1.125-1.125v-1.5c0-.621-.504-1.125-1.125-1.125H3.375c-.621 0-1.125.504-1.125 1.125v1.5c0 .621.504 1.125 1.125 1.125z" />,
    assign: <path strokeLinecap="round" strokeLinejoin="round" d="M15.75 6a3.75 3.75 0 11-7.5 0 3.75 3.75 0 017.5 0zM4.501 20.118a7.5 7.5 0 0114.998 0A17.933 17.933 0 0112 21.75c-2.676 0-5.216-.584-7.499-1.632z" />,
    update: <path strokeLinecap="round" strokeLinejoin="round" d="M16.862 4.487l1.687-1.688a1.875 1.875 0 112.652 2.652L10.582 16.07a4.5 4.5 0 01-1.897 1.13L6 18l.8-2.685a4.5 4.5 0 011.13-1.897l8.932-8.931z" />,
  };
  return (
    <svg className={`w-4 h-4 shrink-0 ${color}`} fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.75}>
      {paths[k]}
    </svg>
  );
}

const STATUS_VERB: Record<string, { verb: string; icon: ActIconKey }> = {
  BACKLOG: { verb: "moved to Backlog", icon: "status-backlog" },
  PLANNED: { verb: "planned", icon: "status-planned" },
  DOING: { verb: "started work on", icon: "status-doing" },
  BLOCKED: { verb: "blocked", icon: "status-blocked" },
  DONE: { verb: "completed", icon: "status-done" },
};

function describeItem(item: InboxItem): { icon: ActIconKey; action: ReactNode; detail?: string } | null {
  const p = item.payload || {};
  switch (item.type) {
    case "CREATE":
      return { icon: "create", action: <>created this issue</> };
    case "COMMENT":
      return { icon: "comment", action: <>commented</>, detail: String(p.text ?? "") };
    case "MERGE": {
      const strategy = p.strategy ? ` via ${String(p.strategy)}` : "";
      return { icon: "merge", action: <>merged{strategy}</> };
    }
    case "UPDATE": {
      if (p.status) {
        const s = STATUS_VERB[String(p.status)] ?? { verb: `moved to ${String(p.status)}`, icon: "status-backlog" as ActIconKey };
        return { icon: s.icon, action: <>{s.verb}</> };
      }
      if (p.assignee !== undefined) {
        const name = String(p.assignee);
        if (!name) return { icon: "assign", action: <>unassigned</> };
        return { icon: "assign", action: <>assigned to <span className="font-medium text-[var(--color-text-primary)]">{shortName(name)}</span></> };
      }
      return { icon: "update", action: <>updated</> };
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
            className="px-3 py-1.5 text-xs text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] bg-[var(--color-bg-secondary)] hover:bg-[var(--color-bg-hover)] border border-[var(--color-border-default)] rounded-[var(--radius-md)] transition-colors"
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
          <div className="max-w-2xl mx-auto py-4 px-4 space-y-3">
            {visibleGroups.map(group => {
              const issue = issues.find(it => it.id === group.issueId);
              const isOpen = expandedGroups.has(group.issueId);
              const latest = group.events[0];
              const latestDesc = describeItem(latest);
              const extraCount = group.events.length - 1;

              return (
                <div
                  key={group.issueId}
                  className={`
                    rounded-[var(--radius-md)] border transition-colors
                    ${group.hasUnread
                      ? "bg-[var(--color-bg-secondary)] border-[var(--color-border-default)]"
                      : "bg-[var(--color-bg-secondary)] border-[var(--color-border-subtle)]"
                    }
                  `.trim().replace(/\s+/g, " ")}
                >
                  <div className="px-4 py-3">
                    {/* Card header: issue link + timestamp */}
                    <div className="flex items-center gap-2">
                      {group.hasUnread && (
                        <span className="w-2 h-2 rounded-full bg-[var(--color-accent-primary)] shrink-0" />
                      )}
                      <a
                        href={`#/issues/${group.issueId}`}
                        onClick={(e) => {
                          if (issue) {
                            e.preventDefault();
                            onIssueClick?.(issue);
                          }
                        }}
                        className={`text-sm font-medium hover:text-[var(--color-accent-primary)] transition-colors ${group.hasUnread ? "text-[var(--color-text-primary)]" : "text-[var(--color-text-secondary)]"}`}
                      >
                        {group.issueId}
                      </a>
                      <span className={`text-sm truncate ${group.hasUnread ? "text-[var(--color-text-secondary)]" : "text-[var(--color-text-muted)]"}`}>
                        {group.issueTitle}
                      </span>
                      <span className="ml-auto text-xs tabular-nums text-[var(--color-text-muted)] shrink-0 whitespace-nowrap">
                        {formatRelativeTime(latest.created_at)}
                      </span>
                    </div>

                    {/* Latest event summary */}
                    {latestDesc && (
                      <div className="flex items-center gap-2 mt-2 text-sm text-[var(--color-text-muted)]">
                        <ActIcon k={latestDesc.icon} />
                        <Avatar name={latest.created_by} size="xs" />
                        <span className="text-[var(--color-text-secondary)] shrink-0">{shortName(latest.created_by)}</span>
                        <span className="shrink-0">{latestDesc.action}</span>
                      </div>
                    )}

                    {/* Expand toggle for additional events */}
                    {extraCount > 0 && (
                      <button
                        onClick={() => toggleGroup(group.issueId)}
                        className="mt-2 text-xs text-[var(--color-text-muted)] hover:text-[var(--color-text-secondary)] transition-colors"
                      >
                        {isOpen ? "Hide" : `+${extraCount} more update${extraCount > 1 ? "s" : ""}`}
                      </button>
                    )}
                  </div>

                  {/* Expanded event list */}
                  {isOpen && extraCount > 0 && (
                    <div className="border-t border-[var(--color-border-subtle)] px-4 py-2 space-y-2">
                      {group.events.slice(1).map((evt, i) => {
                        const desc = describeItem(evt);
                        if (!desc) return null;
                        const evtKey = `${evt.issue_id}-${evt.created_at}-${i}`;
                        const commentExpanded = expandedComments.has(evtKey);
                        const hasDetail = !!desc.detail;
                        const previewText = desc.detail ? stripMarkdown(desc.detail) : "";
                        return (
                          <div key={evtKey} className="text-sm">
                            <div className="flex items-center gap-2 text-[var(--color-text-muted)]">
                              <ActIcon k={desc.icon} />
                              <Avatar name={evt.created_by} size="xs" />
                              <span className="text-[var(--color-text-secondary)] shrink-0">{shortName(evt.created_by)}</span>
                              <span className="shrink-0">{desc.action}</span>
                              <span className="ml-auto text-xs tabular-nums shrink-0 whitespace-nowrap">
                                {formatRelativeTime(evt.created_at)}
                              </span>
                            </div>
                            {hasDetail && !commentExpanded && (
                              <button
                                onClick={() => toggleComment(evtKey)}
                                className="mt-1 pl-6 w-full text-left text-xs text-[var(--color-text-muted)] italic truncate hover:text-[var(--color-text-secondary)] transition-colors"
                              >
                                &ldquo;{previewText.slice(0, 200)}{previewText.length > 200 ? "…" : ""}&rdquo;
                              </button>
                            )}
                            {hasDetail && commentExpanded && (
                              <button
                                onClick={() => toggleComment(evtKey)}
                                className="mt-1 ml-6 block w-[calc(100%-1.5rem)] text-left rounded-[var(--radius-md)] bg-[var(--color-bg-primary)] border border-[var(--color-border-default)] px-3 py-2 hover:border-[var(--color-border-focus)] transition-colors cursor-pointer"
                              >
                                <div className="prose-beats text-sm">
                                  <Markdown remarkPlugins={[remarkGfm, remarkBreaks]}>
                                    {desc.detail!}
                                  </Markdown>
                                </div>
                              </button>
                            )}
                          </div>
                        );
                      })}
                    </div>
                  )}

                  {/* Comment detail for latest event (when group not expanded) */}
                  {!isOpen && latestDesc?.detail && (() => {
                    const evtKey = `latest-${group.issueId}`;
                    const commentExpanded = expandedComments.has(evtKey);
                    const previewText = stripMarkdown(latestDesc.detail!);
                    return (
                      <div className="px-4 pb-3">
                        {!commentExpanded ? (
                          <button
                            onClick={() => toggleComment(evtKey)}
                            className="w-full text-left text-xs text-[var(--color-text-muted)] italic truncate hover:text-[var(--color-text-secondary)] transition-colors"
                          >
                            &ldquo;{previewText.slice(0, 200)}{previewText.length > 200 ? "…" : ""}&rdquo;
                          </button>
                        ) : (
                          <button
                            onClick={() => toggleComment(evtKey)}
                            className="block w-full text-left rounded-[var(--radius-md)] bg-[var(--color-bg-primary)] border border-[var(--color-border-default)] px-3 py-2 hover:border-[var(--color-border-focus)] transition-colors cursor-pointer"
                          >
                            <div className="prose-beats text-sm">
                              <Markdown remarkPlugins={[remarkGfm, remarkBreaks]}>
                                {latestDesc.detail!}
                              </Markdown>
                            </div>
                          </button>
                        )}
                      </div>
                    );
                  })()}
                </div>
              );
            })}

            {hasMore && (
              <div className="flex justify-center pt-2 pb-4">
                <button
                  onClick={() => setVisibleCount(c => c + GROUPS_PER_PAGE)}
                  className="px-4 py-2 text-sm text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] bg-[var(--color-bg-secondary)] hover:bg-[var(--color-bg-hover)] border border-[var(--color-border-default)] rounded-[var(--radius-md)] transition-colors"
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
