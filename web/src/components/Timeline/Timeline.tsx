import { useState, useEffect, useCallback, type ReactNode } from "react";
import { fetchTimeline } from "../../api/client";
import type { TimelineEntry, Issue } from "../../api/client";
import { shortName, formatRelativeTime, formatShortDate } from "../../utils/format";
import {
  PlusIcon,
  CommentIcon,
  MergeIcon,
  CheckCircleIcon,
  PlayIcon,
  BlockedIcon,
  ArchiveIcon,
  TriangleIcon,
  UserIcon,
  FolderIcon,
  LinkIcon,
  GitCommitIcon,
  TimelineIcon,
} from "../ui/icons";
import EmptyState from "../ui/EmptyState";
import StatusIcon from "../ui/StatusIcon";

type KindFilter = "all" | "issue_event" | "commit";

type ActIconKey =
  | "create"
  | "comment"
  | "merge"
  | "commit"
  | "status-done"
  | "status-doing"
  | "status-blocked"
  | "status-planned"
  | "status-backlog"
  | "estimate"
  | "rename"
  | "description"
  | "labels"
  | "assign"
  | "priority"
  | "parent"
  | "relations";

const MUTED = "ring-[var(--color-text-muted)] text-[var(--color-text-secondary)]";

const CIRCLE_STYLE: Record<ActIconKey, string> = {
  create: "ring-[var(--color-accent-primary)] text-[var(--color-accent-primary)]",
  commit: MUTED,
  merge: "ring-[var(--color-accent-primary)] text-[var(--color-accent-primary)]",
  comment: MUTED,
  "status-done": "ring-[var(--color-success)] text-[var(--color-success)]",
  "status-doing": "ring-[var(--color-warning)] text-[var(--color-warning)]",
  "status-blocked": "ring-[var(--color-error)] text-[var(--color-error)]",
  "status-planned": MUTED,
  "status-backlog": MUTED,
  estimate: MUTED,
  rename: MUTED,
  description: MUTED,
  labels: MUTED,
  assign: MUTED,
  priority: MUTED,
  parent: MUTED,
  relations: MUTED,
};

const SHARED_ICONS: Partial<
  Record<ActIconKey, (props: { className?: string }) => ReactNode>
> = {
  create: PlusIcon,
  comment: CommentIcon,
  merge: MergeIcon,
  commit: GitCommitIcon,
  "status-done": CheckCircleIcon,
  "status-doing": PlayIcon,
  "status-blocked": BlockedIcon,
  "status-backlog": ArchiveIcon,
  estimate: TriangleIcon,
  assign: UserIcon,
  parent: FolderIcon,
  relations: LinkIcon,
};

function ActIcon({ k }: { k: ActIconKey }) {
  const cls = "w-3 h-3 shrink-0";
  const Shared = SHARED_ICONS[k];
  if (Shared) return <Shared className={cls} />;
  const paths: Partial<Record<ActIconKey, ReactNode>> = {
    "status-planned": (
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        d="M6.75 3v2.25M17.25 3v2.25M3 18.75V7.5a2.25 2.25 0 012.25-2.25h13.5A2.25 2.25 0 0121 7.5v11.25m-18 0A2.25 2.25 0 005.25 21h13.5A2.25 2.25 0 0021 18.75m-18 0v-7.5A2.25 2.25 0 015.25 9h13.5A2.25 2.25 0 0121 11.25v7.5"
      />
    ),
    rename: (
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        d="M16.862 4.487l1.687-1.688a1.875 1.875 0 112.652 2.652L10.582 16.07a4.5 4.5 0 01-1.897 1.13L6 18l.8-2.685a4.5 4.5 0 011.13-1.897l8.932-8.931z"
      />
    ),
    description: (
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        d="M19.5 14.25v-2.625a3.375 3.375 0 00-3.375-3.375h-1.5A1.125 1.125 0 0113.5 7.125v-1.5a3.375 3.375 0 00-3.375-3.375H8.25m2.25 0H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 00-9-9z"
      />
    ),
    labels: (
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        d="M9.568 3H5.25A2.25 2.25 0 003 5.25v4.318c0 .597.237 1.17.659 1.591l9.581 9.581c.699.699 1.78.872 2.607.33a18.095 18.095 0 005.223-5.223c.542-.827.369-1.908-.33-2.607L11.16 3.66A2.25 2.25 0 009.568 3z"
      />
    ),
    priority: (
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126zM12 15.75h.007v.008H12v-.008z"
      />
    ),
  };
  return (
    <svg
      className={cls}
      fill="none"
      viewBox="0 0 24 24"
      stroke="currentColor"
      strokeWidth={1.75}
    >
      {paths[k]}
    </svg>
  );
}

function resolveIconKey(entry: TimelineEntry): ActIconKey | null {
  if (entry.kind === "commit") return "commit";
  const p = entry.payload || {};
  switch (entry.event_type) {
    case "CREATE": return "create";
    case "COMMENT": return "comment";
    case "MERGE": return "merge";
    case "UPDATE": {
      if (p.status) {
        const icons: Record<string, ActIconKey> = {
          BACKLOG: "status-backlog", PLANNED: "status-planned",
          DOING: "status-doing", BLOCKED: "status-blocked", DONE: "status-done",
        };
        return icons[String(p.status)] ?? "status-backlog";
      }
      if (p.assignee !== undefined) return "assign";
      if (Array.isArray(p.labels)) return "labels";
      if (p.estimate !== undefined) return "estimate";
      if (p.priority !== undefined) return "priority";
      if (p.title) return "rename";
      if (p.description !== undefined) return "description";
      if (p.parent_id !== undefined) return "parent";
      if (Array.isArray(p.dependencies)) return "relations";
      return null;
    }
    default: return null;
  }
}

interface EventDescription {
  before: ReactNode;
  after?: ReactNode;
}

function describeIssueEvent(
  evt: TimelineEntry,
  who: string,
): EventDescription | null {
  const p = evt.payload || {};
  const name = (
    <span className="font-medium text-[var(--color-text-primary)]">{who}</span>
  );
  switch (evt.event_type) {
    case "CREATE":
      return { before: <>{name} created</> };
    case "COMMENT":
      return { before: <>{name} commented on</> };
    case "MERGE": {
      const strategy = p.strategy ? ` via ${String(p.strategy)}` : "";
      return { before: <>{name} merged</>, after: strategy || undefined };
    }
    case "UPDATE": {
      if (p.status) {
        const verbs: Record<string, [string, string]> = {
          BACKLOG: ["moved", "to Backlog"],
          PLANNED: ["marked", "as Planned"],
          DOING: ["started working on", ""],
          BLOCKED: ["marked", "as Blocked"],
          DONE: ["completed", ""],
        };
        const status = String(p.status);
        const [verb, suffix] = verbs[status] ?? ["moved", `to ${status}`];
        return { before: <>{name} {verb}</>, after: suffix || undefined };
      }
      if (p.assignee !== undefined) {
        const assignee = String(p.assignee);
        if (!assignee) return { before: <>{name} unassigned</> };
        return { before: <>{name} assigned</>, after: <>to <span className="font-medium text-[var(--color-text-primary)]">{shortName(assignee)}</span></> };
      }
      if (Array.isArray(p.labels)) return { before: <>{name} relabeled</> };
      if (p.estimate !== undefined) return { before: <>{name} estimated</> };
      if (p.priority !== undefined) return { before: <>{name} changed priority on</> };
      if (p.title) return { before: <>{name} renamed</> };
      if (p.description !== undefined) return { before: <>{name} updated the description of</> };
      if (p.parent_id !== undefined) return { before: <>{name} changed parent of</> };
      if (Array.isArray(p.dependencies)) return { before: <>{name} updated relationships on</> };
      return null;
    }
    default: return null;
  }
}

function groupByDay(entries: TimelineEntry[]): Map<string, TimelineEntry[]> {
  const groups = new Map<string, TimelineEntry[]>();
  for (const entry of entries) {
    const day = entry.timestamp.slice(0, 10);
    const existing = groups.get(day);
    if (existing) {
      existing.push(entry);
    } else {
      groups.set(day, [entry]);
    }
  }
  return groups;
}

function dayLabel(dateStr: string): string {
  const today = new Date().toISOString().slice(0, 10);
  const yesterday = new Date(Date.now() - 86400000).toISOString().slice(0, 10);
  if (dateStr === today) return "Today";
  if (dateStr === yesterday) return "Yesterday";
  return formatShortDate(dateStr);
}

const FILTER_OPTIONS: { value: KindFilter; label: string }[] = [
  { value: "all", label: "All" },
  { value: "issue_event", label: "Issues" },
  { value: "commit", label: "Commits" },
];

export default function Timeline({
  issues,
  onIssueClick,
}: {
  issues: Issue[];
  onIssueClick?: (issue: Issue) => void;
}) {
  const [entries, setEntries] = useState<TimelineEntry[]>([]);
  const [filter, setFilter] = useState<KindFilter>("all");
  const [loading, setLoading] = useState(true);
  const [limit, setLimit] = useState(100);

  const load = useCallback(async () => {
    try {
      const kind = filter === "all" ? undefined : filter;
      const data = await fetchTimeline(limit, kind);
      setEntries(data ?? []);
    } finally {
      setLoading(false);
    }
  }, [filter, limit]);

  useEffect(() => {
    load();
  }, [load]);

  const handleLoadMore = () => setLimit((prev) => prev + 100);

  if (loading) {
    return (
      <div className="flex items-center justify-center h-full">
        <p className="text-sm text-[var(--color-text-muted)]">Loading timeline...</p>
      </div>
    );
  }

  if (entries.length === 0) {
    return (
      <div className="h-full flex flex-col">
        <div className="flex items-center gap-3 px-5 h-11 border-b border-[var(--color-border-subtle)] shrink-0">
          <span className="text-sm font-medium text-[var(--color-text-primary)]">Timeline</span>
          <FilterToggle filter={filter} onFilterChange={setFilter} />
        </div>
        <EmptyState
          icon={<TimelineIcon className="w-12 h-12" />}
          title="No activity yet"
          description="Events and commits will appear here as work progresses."
        />
      </div>
    );
  }

  const days = groupByDay(entries);

  return (
    <div className="h-full flex flex-col">
      <div className="flex items-center gap-3 px-5 h-11 border-b border-[var(--color-border-subtle)] shrink-0">
        <span className="text-sm font-medium text-[var(--color-text-primary)]">Timeline</span>
        <FilterToggle filter={filter} onFilterChange={setFilter} />
      </div>
      <div className="flex-1 overflow-y-auto">
        <div className="py-2">
          {Array.from(days.entries()).map(([day, dayEntries], dayIdx) => (
            <DayGroup
              key={day}
              day={day}
              entries={dayEntries}
              issues={issues}
              onIssueClick={onIssueClick}
              isFirst={dayIdx === 0}
              isLast={dayIdx === days.size - 1 && entries.length < limit}
            />
          ))}
          {entries.length >= limit && (
            <div className="flex items-stretch px-5">
              <div className="w-24 shrink-0" />
              <div className="w-7 flex flex-col items-center shrink-0">
                <div className="w-px flex-1 bg-[var(--color-border-default)]" />
                <div className="w-1.5 h-1.5 rounded-full shrink-0 bg-[var(--color-text-muted)]" />
                <div className="w-px flex-1 bg-transparent" />
              </div>
              <div className="flex-1 flex items-center py-2 pl-3">
                <button
                  onClick={handleLoadMore}
                  className="text-sm text-[var(--color-accent-primary)] hover:text-[var(--color-accent-hover)] transition-colors"
                >
                  Load more
                </button>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

function FilterToggle({
  filter,
  onFilterChange,
}: {
  filter: KindFilter;
  onFilterChange: (f: KindFilter) => void;
}) {
  return (
    <div className="ml-auto flex items-center gap-1 bg-[var(--color-bg-secondary)] rounded-[var(--radius-md)] p-0.5">
      {FILTER_OPTIONS.map((opt) => (
        <button
          key={opt.value}
          onClick={() => onFilterChange(opt.value)}
          className={`px-2.5 py-1 text-xs font-medium rounded-[var(--radius-sm)] transition-colors ${
            filter === opt.value
              ? "bg-[var(--color-bg-primary)] text-[var(--color-text-primary)] shadow-sm"
              : "text-[var(--color-text-muted)] hover:text-[var(--color-text-secondary)]"
          }`}
        >
          {opt.label}
        </button>
      ))}
    </div>
  );
}

function DayGroup({
  day,
  entries,
  issues,
  onIssueClick,
  isFirst,
  isLast,
}: {
  day: string;
  entries: TimelineEntry[];
  issues: Issue[];
  onIssueClick?: (issue: Issue) => void;
  isFirst: boolean;
  isLast: boolean;
}) {
  return (
    <>
      {/* Day header row — bullet on the continuous line */}
      <div className="flex items-stretch px-5">
        <div className="w-24 shrink-0 flex items-end justify-end pr-4 pb-1">
          <span className="text-xs font-semibold text-[var(--color-text-primary)] tracking-wider whitespace-nowrap">
            {dayLabel(day)}
          </span>
        </div>
        <div className="w-7 flex flex-col items-center shrink-0">
          <div className={`w-px ${isFirst ? "h-3" : "min-h-6"} ${isFirst ? "bg-transparent" : "bg-[var(--color-border-default)]"}`} />
          <div className="text-[var(--color-text-muted)] text-[8px] leading-none shrink-0">●</div>
          <div className="w-px h-2 bg-[var(--color-border-default)]" />
        </div>
        <div className="flex-1" />
      </div>

      {/* Activity entries */}
      {entries.map((entry, i) => (
        <TimelineRow
          key={`${entry.timestamp}-${entry.issue_id}-${entry.sha ?? ""}-${i}`}
          entry={entry}
          issues={issues}
          onIssueClick={onIssueClick}
          isLastEntry={isLast && i === entries.length - 1}
        />
      ))}
    </>
  );
}

function IssueLink({
  issue,
  title,
  onClick,
}: {
  issue: Issue | undefined;
  title: string;
  onClick?: (issue: Issue) => void;
}) {
  return (
    <button
      onClick={() => issue && onClick?.(issue)}
      disabled={!issue}
      className="font-medium text-[var(--color-text-primary)] hover:text-[var(--color-accent-primary)] disabled:opacity-60 disabled:cursor-default transition-colors"
      title={title}
    >
      {issue && <StatusIcon status={issue.status} size={12} isInferred={issue.is_inferred} className="inline-block align-[-1px] mr-1 ml-0.5" />}
      {title}
    </button>
  );
}

function TimelineRow({
  entry,
  issues,
  onIssueClick,
  isLastEntry,
}: {
  entry: TimelineEntry;
  issues: Issue[];
  onIssueClick?: (issue: Issue) => void;
  isLastEntry: boolean;
}) {
  const issue = issues.find((it) => it.id === entry.issue_id);
  const title = entry.issue_title || entry.issue_id;
  const iconKey = resolveIconKey(entry);
  if (!iconKey) return null;

  const circleStyle = CIRCLE_STYLE[iconKey];
  const isCommit = entry.kind === "commit";

  const description = isCommit
    ? <>
        <span className="font-medium text-[var(--color-text-primary)]">{shortName(entry.author ?? "")}</span>
        {" committed "}
        <span className="font-mono text-xs text-[var(--color-text-secondary)] bg-[var(--color-surface-1)] border border-[var(--color-border-default)] px-1.5 py-0.5 rounded-[var(--radius-sm)]">
          {entry.sha?.slice(0, 7)}
        </span>
        {entry.issue_id
          ? <>
              {" on "}
              <IssueLink issue={issue} title={title} onClick={onIssueClick} />
            </>
          : <> <span className="text-[var(--color-text-primary)]">{entry.message}</span></>
        }
      </>
    : (() => {
        const desc = describeIssueEvent(entry, shortName(entry.created_by ?? ""));
        if (!desc) return null;
        return <>
          {desc.before}
          {" "}
          <IssueLink issue={issue} title={title} onClick={onIssueClick} />
          {desc.after && <> {desc.after}</>}
        </>;
      })();

  if (!description) return null;

  return (
    <div className="flex items-stretch px-5 min-h-12 group">
      {/* Continuous vertical line with icon circle */}
      <div className="w-24 shrink-0 flex flex-col items-end pr-4">
        <div className="flex-1" />
      </div>
      <div className="w-7 flex flex-col items-center shrink-0">
        <div className="w-px flex-1 bg-[var(--color-border-default)]" />
        <div className={`w-5 h-5 rounded-full shrink-0 flex items-center justify-center bg-[var(--color-bg-primary)] ring-1 ${circleStyle}`}>
          <ActIcon k={iconKey} />
        </div>
        <div className={`w-px flex-1 ${isLastEntry ? "bg-transparent" : "bg-[var(--color-border-default)]"}`} />
      </div>

      {/* Content */}
      <div className="flex-1 flex items-center gap-2 py-2.5 pl-3 pr-4 min-w-0 rounded-[var(--radius-md)] hover:bg-[var(--color-hover-surface)] transition-colors">
        <span className="text-sm text-[var(--color-text-secondary)] truncate min-w-0">
          {description}
        </span>
        <span className="ml-auto text-[11px] tabular-nums shrink-0 whitespace-nowrap text-[var(--color-text-muted)] opacity-0 group-hover:opacity-100 transition-opacity">
          {formatRelativeTime(entry.timestamp)}
        </span>
      </div>
    </div>
  );
}
