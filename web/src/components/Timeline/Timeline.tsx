import { useState, useEffect, useCallback, useMemo, useRef, type ReactNode } from "react";
import { createPortal } from "react-dom";
import { fetchTimeline, fetchCommitDetail } from "../../api/client";
import type { TimelineEntry, Issue, CommitDetail } from "../../api/client";
import { shortName, formatRelativeTime, formatShortDate, displayActor } from "../../utils/format";
import Tooltip from "../ui/Tooltip";
import {
  Plus,
  MessageSquareMore,
  GitMerge,
  Play,
  Ban,
  Archive,
  Triangle,
  User,
  Folder,
  Link,
  GitCommitVertical,
  Clock,
  Check,
  CircleDot,
  Pencil,
  SquareDashedBottomCode,
  Tag,
  TriangleAlert,
  Calendar,
} from "lucide-react";
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

const MUTED = "bg-[var(--color-hover-surface-2)] text-[var(--color-text-muted)]";

const CIRCLE_STYLE: Record<ActIconKey, string> = {
  create: "bg-[rgba(94,106,210,0.15)] text-[var(--color-accent-primary)]",
  commit: MUTED,
  merge: "bg-[rgba(94,106,210,0.15)] text-[var(--color-accent-primary)]",
  comment: MUTED,
  "status-done": "bg-[var(--color-success-bg)] text-[var(--color-success)]",
  "status-doing": "bg-[var(--color-warning-bg)] text-[var(--color-warning)]",
  "status-blocked": "bg-[var(--color-error-bg)] text-[var(--color-error)]",
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
  create: Plus,
  comment: MessageSquareMore,
  merge: GitMerge,
  commit: GitCommitVertical,
  "status-done": Check,
  "status-doing": Play,
  "status-blocked": Ban,
  "status-backlog": Archive,
  estimate: Triangle,
  assign: User,
  parent: Folder,
  relations: Link,
  "status-planned": CircleDot,
  rename: Pencil,
  description: SquareDashedBottomCode,
  labels: Tag,
  priority: TriangleAlert,
};

function ActIcon({ k }: { k: ActIconKey }) {
  const Icon = SHARED_ICONS[k];
  if (Icon) return <Icon className="w-2.5 h-2.5 shrink-0" />;
  return null;
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
  via?: string,
): EventDescription | null {
  const p = evt.payload || {};
  const name = (
    <Tooltip content={via || ""}><span className="font-medium text-[var(--color-text-primary)]">{who}</span></Tooltip>
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

function daySummary(entries: TimelineEntry[]): string {
  let commits = 0;
  let events = 0;
  for (const e of entries) {
    if (e.kind === "commit") commits++;
    else events++;
  }
  const parts: string[] = [];
  if (events > 0) parts.push(`${events} event${events === 1 ? "" : "s"}`);
  if (commits > 0) parts.push(`${commits} commit${commits === 1 ? "" : "s"}`);
  return parts.join(", ");
}

function entryActor(entry: TimelineEntry): string {
  if (entry.kind === "commit") return shortName(entry.author ?? "");
  const actor = displayActor(entry.created_by ?? "", entry.on_behalf_of);
  return actor.principal;
}

function extractContributors(entries: TimelineEntry[]): string[] {
  const seen = new Set<string>();
  for (const e of entries) {
    const name = entryActor(e);
    if (name) seen.add(name);
  }
  return Array.from(seen).sort((a, b) => a.localeCompare(b));
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
  const [rawEntries, setRawEntries] = useState<TimelineEntry[]>([]);
  const [filter, setFilter] = useState<KindFilter>("all");
  const [person, setPerson] = useState<string>("");
  const [loading, setLoading] = useState(true);
  const [limit, setLimit] = useState(100);

  const load = useCallback(async () => {
    try {
      const kind = filter === "all" ? undefined : filter;
      const data = await fetchTimeline(limit, kind);
      setRawEntries(data ?? []);
    } finally {
      setLoading(false);
    }
  }, [filter, limit]);

  useEffect(() => {
    load();
  }, [load]);

  const contributors = useMemo(() => extractContributors(rawEntries), [rawEntries]);
  const entries = useMemo(
    () => person ? rawEntries.filter((e) => entryActor(e) === person) : rawEntries,
    [rawEntries, person],
  );

  const handleLoadMore = () => setLimit((prev) => prev + 100);

  if (loading) {
    return (
      <div className="flex items-center justify-center h-full">
        <p className="text-sm text-[var(--color-text-muted)]">Loading timeline...</p>
      </div>
    );
  }

  if (rawEntries.length === 0) {
    return (
      <div className="h-full flex flex-col">
        <HeaderBar filter={filter} onFilterChange={setFilter} person={person} onPersonChange={setPerson} contributors={contributors} />
        <EmptyState
          icon={<Clock className="w-12 h-12" />}
          title="No activity yet"
          description="Events and commits will appear here as work progresses."
        />
      </div>
    );
  }

  const days = groupByDay(entries);

  return (
    <div className="h-full flex flex-col">
      <HeaderBar filter={filter} onFilterChange={setFilter} person={person} onPersonChange={setPerson} contributors={contributors} />
      <div className="flex-1 overflow-y-auto">
        <div className="max-w-7xl mx-auto py-2">
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

function HeaderBar({
  filter,
  onFilterChange,
  person,
  onPersonChange,
  contributors,
}: {
  filter: KindFilter;
  onFilterChange: (f: KindFilter) => void;
  person: string;
  onPersonChange: (p: string) => void;
  contributors: string[];
}) {
  return (
    <div className="flex items-center gap-3 px-5 h-11 border-b border-[var(--color-border-subtle)] shrink-0">
      <span className="text-sm font-medium text-[var(--color-text-primary)]">Timeline</span>

      <div className="ml-auto flex items-center gap-2">
        {contributors.length > 1 && (
          <PersonFilter person={person} onPersonChange={onPersonChange} contributors={contributors} />
        )}
        <div className="flex items-center gap-1 bg-[var(--color-bg-secondary)] rounded-[var(--radius-md)] p-0.5">
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
      </div>
    </div>
  );
}

function PersonFilter({
  person,
  onPersonChange,
  contributors,
}: {
  person: string;
  onPersonChange: (p: string) => void;
  contributors: string[];
}) {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    const handler = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, [open]);

  return (
    <div className="relative" ref={ref}>
      <button
        onClick={() => setOpen(!open)}
        className={`flex items-center gap-1.5 px-2.5 py-1 text-xs font-medium rounded-[var(--radius-sm)] transition-colors border ${
          person
            ? "border-[var(--color-accent-primary)] text-[var(--color-accent-primary)] bg-[var(--color-accent-primary)]/10"
            : "border-[var(--color-border-default)] text-[var(--color-text-muted)] hover:text-[var(--color-text-secondary)]"
        }`}
      >
        <User className="w-3 h-3" />
        {person || "Everyone"}
      </button>
      {open && (
        <div className="absolute right-0 top-full mt-1 z-20 w-48 bg-[var(--color-surface-3)] border border-[var(--color-border-default)] rounded-[var(--radius-md)] shadow-[var(--shadow-lg)] py-1">
          <button
            onClick={() => { onPersonChange(""); setOpen(false); }}
            className={`w-full text-left px-3 py-1.5 text-xs transition-colors ${
              !person ? "text-[var(--color-accent-primary)] font-medium" : "text-[var(--color-text-secondary)] hover:bg-[var(--color-hover-surface)]"
            }`}
          >
            Everyone
          </button>
          {contributors.map((c) => (
            <button
              key={c}
              onClick={() => { onPersonChange(c); setOpen(false); }}
              className={`w-full text-left px-3 py-1.5 text-xs transition-colors ${
                person === c ? "text-[var(--color-accent-primary)] font-medium" : "text-[var(--color-text-secondary)] hover:bg-[var(--color-hover-surface)]"
              }`}
            >
              {c}
            </button>
          ))}
        </div>
      )}
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
        <div className="w-24 shrink-0 flex flex-col justify-end pr-4">
          <div className="flex-1" />
        </div>
        <div className="w-7 flex flex-col items-center shrink-0">
          <div className={`w-px ${isFirst ? "h-3" : "min-h-6"} ${isFirst ? "bg-transparent" : "bg-[var(--color-border-default)]"}`} />
        </div>
        <div className="flex-1" />
      </div>
      <div className="flex items-center px-5">
        <div className="w-24 shrink-0 flex justify-end pr-4">
          <span className="text-xs font-semibold text-[var(--color-text-primary)] tracking-wider whitespace-nowrap">
            {dayLabel(day)}
          </span>
        </div>
        <div className="w-7 flex justify-center shrink-0">
          <Calendar size={16} className="shrink-0 text-[var(--color-text-muted)]" />
        </div>
        <div className="flex-1 pl-3 h-7">
          <span className="text-[13px] text-[var(--color-text-muted)]">
            {daySummary(entries)}
          </span>
        </div>
      </div>
      <div className="flex items-stretch px-5">
        <div className="w-24 shrink-0" />
        <div className="w-7 flex flex-col items-center shrink-0">
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

function CommitSHA({ sha }: { sha: string }) {
  const [detail, setDetail] = useState<CommitDetail | null>(null);
  const [open, setOpen] = useState(false);
  const [loading, setLoading] = useState(false);
  const btnRef = useRef<HTMLButtonElement>(null);
  const popRef = useRef<HTMLDivElement>(null);
  const [pos, setPos] = useState<{ top: number; left: number }>({ top: 0, left: 0 });

  const handleClick = async (e: React.MouseEvent) => {
    e.stopPropagation();
    if (open) { setOpen(false); return; }
    if (btnRef.current) {
      const rect = btnRef.current.getBoundingClientRect();
      const pad = 8;
      let top = rect.bottom + 4;
      let left = rect.left;
      if (left + 560 > window.innerWidth - pad) {
        left = window.innerWidth - 560 - pad;
      }
      if (top + 400 > window.innerHeight - pad) {
        top = rect.top - 400 - 4;
      }
      setPos({ top, left });
    }
    setOpen(true);
    if (!detail) {
      setLoading(true);
      try {
        const d = await fetchCommitDetail(sha);
        setDetail(d);
      } finally {
        setLoading(false);
      }
    }
  };

  useEffect(() => {
    if (!open) return;
    const handler = (e: MouseEvent) => {
      if (btnRef.current && !btnRef.current.contains(e.target as Node) &&
          popRef.current && !popRef.current.contains(e.target as Node)) {
        setOpen(false);
      }
    };
    const keyHandler = (e: KeyboardEvent) => { if (e.key === "Escape") setOpen(false); };
    const dismiss = (e: Event) => {
      if (popRef.current && popRef.current.contains(e.target as Node)) return;
      setOpen(false);
    };
    document.addEventListener("mousedown", handler);
    document.addEventListener("keydown", keyHandler);
    document.addEventListener("scroll", dismiss, true);
    return () => {
      document.removeEventListener("mousedown", handler);
      document.removeEventListener("keydown", keyHandler);
      document.removeEventListener("scroll", dismiss, true);
    };
  }, [open]);

  return (
    <>
      <button
        ref={btnRef}
        onClick={handleClick}
        className="font-mono text-xs text-[var(--color-text-secondary)] bg-[var(--color-surface-1)] border border-[var(--color-border-default)] px-1.5 py-0.5 rounded-[var(--radius-sm)] hover:border-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] transition-colors cursor-pointer align-middle"
      >
        {sha.slice(0, 7)}
      </button>
      {open && createPortal(
        <div
          ref={popRef}
          style={{ position: "fixed", top: pos.top, left: pos.left }}
          className="z-50 w-[560px] max-h-[400px] overflow-y-auto bg-[var(--color-surface-3)] border border-[var(--color-border-default)] rounded-[var(--radius-lg)] shadow-[var(--shadow-popover)]"
        >
          {loading ? (
            <div className="px-4 py-6 text-center text-xs text-[var(--color-text-muted)]">Loading...</div>
          ) : detail ? (
            <div>
              <div className="px-4 py-3 border-b border-[var(--color-border-subtle)]">
                <div className="flex items-center gap-2 mb-1">
                  <span className="font-mono text-xs text-[var(--color-text-muted)]">{detail.sha}</span>
                </div>
                <p className="text-sm font-medium text-[var(--color-text-primary)] leading-snug">{detail.subject}</p>
                {detail.body && (
                  <p className="text-xs text-[var(--color-text-secondary)] mt-1.5 leading-relaxed whitespace-pre-wrap">{detail.body}</p>
                )}
                <div className="flex items-center gap-2 mt-2 text-[11px] text-[var(--color-text-muted)]">
                  <span>{detail.author}</span>
                  <span>·</span>
                  <span>{formatRelativeTime(detail.date)}</span>
                </div>
              </div>
              {detail.files.length > 0 && (
                <div className="px-4 py-2 max-h-48 overflow-y-auto">
                  <div className="text-[11px] text-[var(--color-text-muted)] mb-1.5">{detail.files.length} file{detail.files.length === 1 ? "" : "s"} changed</div>
                  {detail.files.map((f) => (
                    <div key={f.path} className="flex items-center gap-2 py-0.5 text-xs">
                      <span className="truncate min-w-0 text-[var(--color-text-secondary)]">{f.path}</span>
                      <span className="ml-auto shrink-0 tabular-nums">
                        {f.additions > 0 && <span className="text-[var(--color-success)]">+{f.additions}</span>}
                        {f.additions > 0 && f.deletions > 0 && " "}
                        {f.deletions > 0 && <span className="text-[var(--color-error)]">-{f.deletions}</span>}
                      </span>
                    </div>
                  ))}
                </div>
              )}
            </div>
          ) : null}
        </div>,
        document.body,
      )}
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
        <CommitSHA sha={entry.sha ?? ""} />
        {entry.issue_id
          ? <>
              {" on "}
              <span className="font-mono text-xs text-[var(--color-text-secondary)]">{entry.branch}</span>
            </>
          : <> <span className="text-[var(--color-text-primary)]">{entry.message}</span></>
        }
      </>
    : (() => {
        const actor = displayActor(entry.created_by ?? "", entry.on_behalf_of);
        const desc = describeIssueEvent(entry, actor.principal, actor.via);
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
        <div className={`w-5 h-5 rounded-full shrink-0 flex items-center justify-center ${circleStyle}`}>
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
