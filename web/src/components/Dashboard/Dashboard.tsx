import { useEffect, useMemo, useState, type ReactNode } from 'react';
import Markdown from 'react-markdown';
import remarkBreaks from 'remark-breaks';
import remarkGfm from 'remark-gfm';
import { fetchActivity, fetchMetrics, type ActivityEvent, type AttentionItem, type Issue, type PulseMetrics } from '../../api/client';
import { Avatar, Card, CopyableId, LabelBadge, StatusIcon } from '../ui';

function stripMarkdown(text: string): string {
  return text
    .replace(/```[\s\S]*?```/g, ' ')           // code blocks
    .replace(/`([^`]+)`/g, '$1')                // inline code
    .replace(/^#{1,6}\s+/gm, '')                // headers
    .replace(/\*\*([^*]+)\*\*/g, '$1')          // bold
    .replace(/\*([^*]+)\*/g, '$1')              // italic
    .replace(/__([^_]+)__/g, '$1')              // bold alt
    .replace(/_([^_]+)_/g, '$1')                // italic alt
    .replace(/~~([^~]+)~~/g, '$1')              // strikethrough
    .replace(/!\[[^\]]*\]\([^)]+\)/g, '')       // images
    .replace(/\[([^\]]+)\]\([^)]+\)/g, '$1')    // links → text
    .replace(/^>\s?/gm, '')                     // blockquotes
    .replace(/^[-*+]\s+/gm, '')                 // unordered list markers
    .replace(/^\d+\.\s+/gm, '')                 // ordered list markers
    .replace(/\s+/g, ' ')                       // normalize whitespace
    .trim();
}

function SubProgress({ done, total }: { done: number; total: number }) {
  const pct = total > 0 ? (done / total) * 100 : 0;
  return (
    <svg width="14" height="14" viewBox="0 0 16 16" className="shrink-0">
      <circle cx="8" cy="8" r="6" fill="none" stroke="var(--color-bg-tertiary)" strokeWidth="2" />
      <circle
        cx="8"
        cy="8"
        r="6"
        fill="none"
        stroke={pct === 100 ? 'var(--color-success)' : 'var(--color-accent-primary)'}
        strokeWidth="2"
        strokeLinecap="round"
        strokeDasharray={`${pct * 0.377} 100`}
        transform="rotate(-90 8 8)"
      />
    </svg>
  );
}

function shortName(fullName: string): string {
  return fullName.split(' <')[0];
}

function formatRelativeTime(dateStr: string): string {
  const now = Date.now();
  const then = new Date(dateStr).getTime();
  const diff = now - then;
  const minutes = Math.floor(diff / 60000);
  if (minutes < 1) return 'just now';
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.floor(hours / 24);
  if (days < 30) return `${days}d ago`;
  const months = Math.floor(days / 30);
  return `${months}mo ago`;
}

type ActIconKey =
  | 'create' | 'comment'
  | 'status-done' | 'status-doing' | 'status-blocked' | 'status-planned' | 'status-backlog'
  | 'estimate' | 'rename' | 'description' | 'labels' | 'assign' | 'priority' | 'parent' | 'relations';

function ActIcon({ k }: { k: ActIconKey }) {
  const paths: Record<ActIconKey, ReactNode> = {
    'create': <path strokeLinecap="round" strokeLinejoin="round" d="M12 4.5v15m7.5-7.5h-15" />,
    'comment': <path strokeLinecap="round" strokeLinejoin="round" d="M2.25 12.76c0 1.6 1.123 2.994 2.707 3.227 1.068.157 2.148.279 3.238.364.466.037.893.281 1.153.671L12 21l2.652-3.978c.26-.39.687-.634 1.153-.671 1.09-.085 2.17-.207 3.238-.364 1.584-.233 2.707-1.626 2.707-3.228V6.741c0-1.602-1.123-2.995-2.707-3.228A48.394 48.394 0 0012 3c-2.392 0-4.744.175-7.043.513C3.373 3.746 2.25 5.14 2.25 6.741v6.018z" />,
    'status-done': <path strokeLinecap="round" strokeLinejoin="round" d="M9 12.75L11.25 15 15 9.75M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />,
    'status-doing': <path strokeLinecap="round" strokeLinejoin="round" d="M5.25 5.653c0-.856.917-1.398 1.667-.986l11.54 6.347a1.125 1.125 0 010 1.972l-11.54 6.347a1.125 1.125 0 01-1.667-.986V5.653z" />,
    'status-blocked': <path strokeLinecap="round" strokeLinejoin="round" d="M18.364 18.364A9 9 0 005.636 5.636m12.728 12.728A9 9 0 015.636 5.636m12.728 12.728L5.636 5.636" />,
    'status-planned': <path strokeLinecap="round" strokeLinejoin="round" d="M6.75 3v2.25M17.25 3v2.25M3 18.75V7.5a2.25 2.25 0 012.25-2.25h13.5A2.25 2.25 0 0121 7.5v11.25m-18 0A2.25 2.25 0 005.25 21h13.5A2.25 2.25 0 0021 18.75m-18 0v-7.5A2.25 2.25 0 015.25 9h13.5A2.25 2.25 0 0121 11.25v7.5" />,
    'status-backlog': <path strokeLinecap="round" strokeLinejoin="round" d="M20.25 7.5l-.625 10.632a2.25 2.25 0 01-2.247 2.118H6.622a2.25 2.25 0 01-2.247-2.118L3.75 7.5m6 4.125l2.25 2.25m0 0l2.25 2.25M12 13.875l2.25-2.25M12 13.875l-2.25 2.25M3.375 7.5h17.25c.621 0 1.125-.504 1.125-1.125v-1.5c0-.621-.504-1.125-1.125-1.125H3.375c-.621 0-1.125.504-1.125 1.125v1.5c0 .621.504 1.125 1.125 1.125z" />,
    'estimate': <path strokeLinecap="round" strokeLinejoin="round" d="M8 2L14 14H2L8 2Z" />,
    'rename': <path strokeLinecap="round" strokeLinejoin="round" d="M16.862 4.487l1.687-1.688a1.875 1.875 0 112.652 2.652L10.582 16.07a4.5 4.5 0 01-1.897 1.13L6 18l.8-2.685a4.5 4.5 0 011.13-1.897l8.932-8.931z" />,
    'description': <path strokeLinecap="round" strokeLinejoin="round" d="M19.5 14.25v-2.625a3.375 3.375 0 00-3.375-3.375h-1.5A1.125 1.125 0 0113.5 7.125v-1.5a3.375 3.375 0 00-3.375-3.375H8.25m2.25 0H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 00-9-9z" />,
    'labels': <path strokeLinecap="round" strokeLinejoin="round" d="M9.568 3H5.25A2.25 2.25 0 003 5.25v4.318c0 .597.237 1.17.659 1.591l9.581 9.581c.699.699 1.78.872 2.607.33a18.095 18.095 0 005.223-5.223c.542-.827.369-1.908-.33-2.607L11.16 3.66A2.25 2.25 0 009.568 3z" />,
    'assign': <path strokeLinecap="round" strokeLinejoin="round" d="M15.75 6a3.75 3.75 0 11-7.5 0 3.75 3.75 0 017.5 0zM4.501 20.118a7.5 7.5 0 0114.998 0A17.933 17.933 0 0112 21.75c-2.676 0-5.216-.584-7.499-1.632z" />,
    'priority': <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126zM12 15.75h.007v.008H12v-.008z" />,
    'parent': <path strokeLinecap="round" strokeLinejoin="round" d="M2.25 12.75V12A2.25 2.25 0 014.5 9.75h15A2.25 2.25 0 0121.75 12v.75m-8.69-6.44l-2.12-2.12a1.5 1.5 0 00-1.061-.44H4.5A2.25 2.25 0 002.25 6v12a2.25 2.25 0 002.25 2.25h15A2.25 2.25 0 0021.75 18V9a2.25 2.25 0 00-2.25-2.25h-5.379a1.5 1.5 0 01-1.06-.44z" />,
    'relations': <path strokeLinecap="round" strokeLinejoin="round" d="M13.19 8.688a4.5 4.5 0 011.242 7.244l-4.5 4.5a4.5 4.5 0 01-6.364-6.364l1.757-1.757m13.35-.622l1.757-1.757a4.5 4.5 0 00-6.364-6.364l-4.5 4.5a4.5 4.5 0 001.242 7.244" />,
  };
  return (
    <svg className="w-3.5 h-3.5 shrink-0 text-[var(--color-text-muted)]" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.75}>
      {paths[k]}
    </svg>
  );
}

const STATUS_VERB: Record<string, { verb: string; icon: ActIconKey }> = {
  BACKLOG: { verb: 'moved to Backlog', icon: 'status-backlog' },
  PLANNED: { verb: 'planned', icon: 'status-planned' },
  DOING: { verb: 'started', icon: 'status-doing' },
  BLOCKED: { verb: 'blocked', icon: 'status-blocked' },
  DONE: { verb: 'completed', icon: 'status-done' },
};

function describeActivity(evt: ActivityEvent): { icon: ActIconKey; content: ReactNode; preview?: string } | null {
  const p = evt.payload || {};
  switch (evt.type) {
    case 'CREATE':
      return { icon: 'create', content: <>created</> };
    case 'COMMENT':
      return { icon: 'comment', content: <>commented on</>, preview: String(p.text ?? '') };
    case 'UPDATE': {
      // Pick the dominant action for the icon (status > assignee > labels > estimate > priority > title > description > parent > relations).
      if (p.status) {
        const s = STATUS_VERB[String(p.status)] ?? { verb: `moved to ${String(p.status)}`, icon: 'status-backlog' as ActIconKey };
        return { icon: s.icon, content: <>{s.verb}</> };
      }
      if (p.assignee !== undefined) {
        const name = String(p.assignee);
        return {
          icon: 'assign',
          content: name
            ? <>assigned <span className="text-[var(--color-text-primary)]">{shortName(name)}</span> to</>
            : <>unassigned</>,
        };
      }
      if (Array.isArray(p.labels)) return { icon: 'labels', content: <>relabeled</> };
      if (p.estimate !== undefined) return { icon: 'estimate', content: <>set estimate to <span className="text-[var(--color-text-primary)]">{String(p.estimate)}</span> on</> };
      if (p.priority !== undefined) return { icon: 'priority', content: <>changed priority of</> };
      if (p.title) return { icon: 'rename', content: <>renamed</> };
      if (p.description !== undefined) return { icon: 'description', content: <>updated the description of</> };
      if (p.parent_id !== undefined) return { icon: 'parent', content: <>changed parent of</> };
      if (Array.isArray(p.dependencies)) return { icon: 'relations', content: <>updated relationships of</> };
      return null;
    }
    default:
      return null;
  }
}

function AttentionBadge({ item }: { item: AttentionItem }) {
  const styles: Record<AttentionItem['kind'], { label: string; cls: string }> = {
    blocker: { label: 'BLOCKED', cls: 'text-[var(--color-error)] border-[var(--color-error)]/40 bg-[var(--color-error)]/10' },
    stale_wip: { label: 'STALE', cls: 'text-[var(--color-warning)] border-[var(--color-warning)]/40 bg-[var(--color-warning)]/10' },
    high_priority: { label: item.priority === 1 ? 'URGENT' : 'HIGH', cls: 'text-[var(--color-text-secondary)] border-[var(--color-border-default)] bg-[var(--color-bg-tertiary)]' },
  };
  const s = styles[item.kind];
  return (
    <span className={`text-[10px] uppercase tracking-wider font-medium px-1.5 py-0.5 rounded border shrink-0 ${s.cls}`}>
      {s.label}
    </span>
  );
}

function attentionMeta(item: AttentionItem): string {
  if (item.kind === 'high_priority') return '';
  return `${item.age_days ?? 0}d`;
}

interface DashboardProps {
  issues: Issue[];
  onIssueClick?: (issue: Issue) => void;
}

type DistFilter = 'all' | 'active';

type DistRow = {
  key: string;
  label: ReactNode;
  count: number;
  pct: number;
  barWidth: number;
};

function DistributionSection({
  title,
  filter,
  onFilterChange,
  rows,
  total,
  emptyHasIssues = 'No data to display.',
  emptyNoIssues = 'No issues to display.',
}: {
  title: string;
  filter: DistFilter;
  onFilterChange: (f: DistFilter) => void;
  rows: DistRow[];
  total: number;
  emptyHasIssues?: string;
  emptyNoIssues?: string;
}) {
  return (
    <div>
      <div className="flex items-center justify-between mb-3">
        <h2 className="text-xs uppercase tracking-wider text-[var(--color-text-muted)]">{title}</h2>
        <div className="flex items-center gap-0.5 bg-[var(--color-bg-tertiary)] rounded-[var(--radius-md)] p-0.5">
          {(['active', 'all'] as const).map(opt => (
            <button
              key={opt}
              onClick={() => onFilterChange(opt)}
              className={`px-2.5 h-6 rounded-[var(--radius-sm)] text-xs transition-colors ${
                filter === opt
                  ? 'bg-[var(--color-bg-elevated)] text-[var(--color-text-primary)]'
                  : 'text-[var(--color-text-muted)] hover:text-[var(--color-text-secondary)]'
              }`}
            >
              {opt === 'active' ? 'Active' : 'All'}
            </button>
          ))}
        </div>
      </div>
      <Card variant="elevated" className="min-h-[160px] flex flex-col">
        {rows.length === 0 ? (
          <div className="flex-1 flex items-center justify-center">
            <p className="text-sm text-[var(--color-text-muted)] text-center">
              {total === 0 ? emptyNoIssues : emptyHasIssues}
            </p>
          </div>
        ) : (
          <div className="space-y-2">
            {rows.map(({ key, label, count, pct, barWidth }) => (
              <div key={key} className="flex items-center gap-2">
                <div className="flex-1 min-w-0">{label}</div>
                <div className="w-16 h-1.5 rounded-full bg-[var(--color-bg-tertiary)] overflow-hidden shrink-0">
                  <div
                    className="h-full rounded-full bg-[var(--color-text-secondary)] transition-all duration-500"
                    style={{ width: `${barWidth}%` }}
                  />
                </div>
                <div className="w-14 shrink-0 text-right text-xs tabular-nums">
                  <span className="text-[var(--color-text-primary)] font-medium">{count}</span>
                  <span className="text-[var(--color-text-muted)] ml-1">{pct.toFixed(0)}%</span>
                </div>
              </div>
            ))}
          </div>
        )}
      </Card>
    </div>
  );
}

function Sparkline({ buckets }: { buckets: PulseMetrics['velocity']['weekly_buckets'] }) {
  const max = Math.max(1, ...buckets.map(b => b.points));
  const barWidth = 6;
  const gap = 2;
  const height = 28;
  const width = buckets.length * (barWidth + gap) - gap;
  return (
    <svg
      width={width}
      height={height}
      className="text-[var(--color-text-secondary)] shrink-0"
      aria-label="velocity last 8 weeks"
    >
      {buckets.map((b, i) => {
        const h = (b.points / max) * height;
        return (
          <rect
            key={b.week_start}
            x={i * (barWidth + gap)}
            y={height - Math.max(1, h)}
            width={barWidth}
            height={Math.max(1, h)}
            fill="currentColor"
            rx={1}
          />
        );
      })}
    </svg>
  );
}

function PulseCard({ title, children }: { title: string; children: ReactNode }) {
  return (
    <Card variant="elevated" padding="sm" className="min-h-[112px]">
      <p className="text-xs uppercase tracking-wider text-[var(--color-text-muted)] mb-1.5">{title}</p>
      {children}
    </Card>
  );
}

function Section({
  title,
  icon,
  count,
  collapsible = false,
  defaultOpen = true,
  storageKey,
  children,
}: {
  title: string;
  icon: ReactNode;
  count?: number | string;
  collapsible?: boolean;
  defaultOpen?: boolean;
  storageKey?: string;
  children: ReactNode;
}) {
  const [open, setOpen] = useState<boolean>(() => {
    if (storageKey) {
      const stored = localStorage.getItem(storageKey);
      if (stored !== null) return stored === '1';
    }
    return defaultOpen;
  });

  const toggle = () => {
    if (!collapsible) return;
    setOpen(v => {
      const next = !v;
      if (storageKey) localStorage.setItem(storageKey, next ? '1' : '0');
      return next;
    });
  };

  return (
    <section>
      <div
        onClick={toggle}
        className={`flex items-center gap-2 px-5 py-2 bg-[var(--color-bg-secondary)] select-none ${collapsible ? 'cursor-pointer hover:bg-[var(--color-bg-hover)] transition-colors duration-[var(--duration-fast)]' : ''}`}
      >
        {collapsible && (
          <svg
            className={`w-3 h-3 text-[var(--color-text-muted)] transition-transform duration-100 ${open ? 'rotate-90' : ''}`}
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
            strokeWidth={2.5}
          >
            <path strokeLinecap="round" strokeLinejoin="round" d="M9 5l7 7-7 7" />
          </svg>
        )}
        <span className="text-[var(--color-text-muted)] flex shrink-0">{icon}</span>
        <span className="text-sm font-medium text-[var(--color-text-primary)]">{title}</span>
        {count !== undefined && (
          <span className="text-sm text-[var(--color-text-muted)] tabular-nums">{count}</span>
        )}
      </div>
      {(!collapsible || open) && <div>{children}</div>}
    </section>
  );
}

function SectionIcon({ d }: { d: string }) {
  return (
    <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.75}>
      <path strokeLinecap="round" strokeLinejoin="round" d={d} />
    </svg>
  );
}

const SECTION_ICONS = {
  pulse: 'M3 12h3l3-9 6 18 3-9h3',
  workload: 'M17 20h5v-2a4 4 0 00-3-3.87M9 20H2v-2a4 4 0 014-4h4a4 4 0 014 4v2M16 3.13a4 4 0 010 7.75M9 12a4 4 0 100-8 4 4 0 000 8z',
  attention: 'M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126zM12 15.75h.007v.008H12v-.008z',
  epics: 'M17.593 3.322c1.1.128 1.907 1.077 1.907 2.185V21L12 17.25 4.5 21V5.507c0-1.108.806-2.057 1.907-2.185a48.507 48.507 0 0111.186 0z',
  activity: 'M12 6v6h4.5m4.5 0a9 9 0 11-18 0 9 9 0 0118 0z',
  composition: 'M2.25 7.125C2.25 6.504 2.754 6 3.375 6h6c.621 0 1.125.504 1.125 1.125v3.75c0 .621-.504 1.125-1.125 1.125h-6a1.125 1.125 0 01-1.125-1.125v-3.75zM14.25 8.625c0-.621.504-1.125 1.125-1.125h5.25c.621 0 1.125.504 1.125 1.125v8.25c0 .621-.504 1.125-1.125 1.125h-5.25a1.125 1.125 0 01-1.125-1.125v-8.25zM3.75 16.125c0-.621.504-1.125 1.125-1.125h5.25c.621 0 1.125.504 1.125 1.125v2.25c0 .621-.504 1.125-1.125 1.125h-5.25a1.125 1.125 0 01-1.125-1.125v-2.25z',
  trends: 'M2.25 18L9 11.25l4.306 4.306a11.95 11.95 0 015.814-5.518l2.74-1.22m0 0l-5.94-2.281m5.94 2.281l-2.28 5.941',
};

function TrendChart({ weekly }: { weekly: { week_start: string; created: number; completed: number }[] }) {
  const max = Math.max(1, ...weekly.flatMap(w => [w.created, w.completed]));
  const barW = 6;
  const innerGap = 2;
  const groupGap = 6;
  const height = 56;
  const groupW = barW * 2 + innerGap;
  const width = weekly.length * (groupW + groupGap) - groupGap;
  return (
    <svg width={width} height={height} className="shrink-0" aria-label="created vs completed, last 8 weeks">
      {weekly.map((w, i) => {
        const x = i * (groupW + groupGap);
        const ch = (w.created / max) * height;
        const dh = (w.completed / max) * height;
        return (
          <g key={w.week_start}>
            <rect
              x={x}
              y={height - Math.max(1, ch)}
              width={barW}
              height={Math.max(1, ch)}
              className="fill-[var(--color-text-muted)]"
              rx={1}
            />
            <rect
              x={x + barW + innerGap}
              y={height - Math.max(1, dh)}
              width={barW}
              height={Math.max(1, dh)}
              className="fill-[var(--color-accent-primary)]"
              rx={1}
            />
          </g>
        );
      })}
    </svg>
  );
}

function formatTriage(mins: number): string {
  if (mins <= 0) return '—';
  if (mins < 60) return `${mins}m`;
  const hours = mins / 60;
  if (hours < 24) return `${Math.round(hours)}h`;
  const days = hours / 24;
  if (days < 10) return `${days.toFixed(1)}d`;
  return `${Math.round(days)}d`;
}

const PRIORITY_LABELS: { value: number; label: string; marker: string; markerClass: string }[] = [
  { value: 1, label: 'Urgent', marker: '!!!', markerClass: 'text-[var(--color-error)]' },
  { value: 2, label: 'High', marker: '!!', markerClass: 'text-[var(--color-warning)]' },
  { value: 3, label: 'Medium', marker: '!', markerClass: 'text-[var(--color-text-muted)]' },
  { value: 4, label: 'Low', marker: '', markerClass: '' },
  { value: 0, label: 'No priority', marker: '', markerClass: '' },
];

export default function Dashboard({ issues, onIssueClick }: DashboardProps) {
  const [labelFilter, setLabelFilter] = useState<DistFilter>('active');
  const [assigneeFilter, setAssigneeFilter] = useState<DistFilter>('active');
  const [priorityFilter, setPriorityFilter] = useState<DistFilter>('active');
  const [metrics, setMetrics] = useState<PulseMetrics | null>(null);
  const [activity, setActivity] = useState<ActivityEvent[]>([]);
  const [expandedComments, setExpandedComments] = useState<Set<string>>(() => new Set());

  const toggleComment = (key: string) => {
    setExpandedComments(prev => {
      const next = new Set(prev);
      if (next.has(key)) next.delete(key);
      else next.add(key);
      return next;
    });
  };

  useEffect(() => {
    const load = () => {
      fetchMetrics().then(setMetrics).catch(() => {});
      fetchActivity().then(setActivity).catch(() => {});
    };
    load();
    const interval = setInterval(load, 30_000);
    const onFocus = () => load();
    window.addEventListener('focus', onFocus);
    return () => {
      clearInterval(interval);
      window.removeEventListener('focus', onFocus);
    };
  }, [issues]);
  const stats = {
    total: issues.length,
    backlog: issues.filter((i) => i.status === 'BACKLOG').length,
    planned: issues.filter((i) => i.status === 'PLANNED').length,
    doing: issues.filter((i) => i.status === 'DOING').length,
    blocked: issues.filter((i) => i.status === 'BLOCKED').length,
    done: issues.filter((i) => i.status === 'DONE').length,
  };

  const labelDistribution = useMemo(() => {
    const filtered = labelFilter === 'active'
      ? issues.filter(i => i.status !== 'DONE')
      : issues;
    const counts = new Map<string, number>();
    for (const i of filtered) {
      for (const l of i.labels || []) {
        counts.set(l, (counts.get(l) || 0) + 1);
      }
    }
    const total = filtered.length;
    const max = Math.max(0, ...counts.values());
    const rows: DistRow[] = Array.from(counts.entries())
      .sort((a, b) => b[1] - a[1])
      .map(([label, count]) => ({
        key: label,
        label: <LabelBadge label={label} />,
        count,
        pct: total > 0 ? (count / total) * 100 : 0,
        barWidth: max > 0 ? (count / max) * 100 : 0,
      }));
    return { total, rows };
  }, [issues, labelFilter]);

  const assigneeDistribution = useMemo(() => {
    const filtered = assigneeFilter === 'active'
      ? issues.filter(i => i.status !== 'DONE')
      : issues;
    const counts = new Map<string, number>();
    let unassigned = 0;
    for (const i of filtered) {
      if (i.assignee) {
        counts.set(i.assignee, (counts.get(i.assignee) || 0) + 1);
      } else {
        unassigned++;
      }
    }
    const total = filtered.length;
    const allCounts = [...counts.values(), ...(unassigned > 0 ? [unassigned] : [])];
    const max = Math.max(0, ...allCounts);
    const namedRows: DistRow[] = Array.from(counts.entries())
      .sort((a, b) => b[1] - a[1])
      .map(([assignee, count]) => ({
        key: assignee,
        label: (
          <div className="flex items-center gap-2 min-w-0">
            <Avatar name={assignee} size="xs" />
            <span className="text-sm text-[var(--color-text-primary)] truncate">
              {assignee.split(' <')[0]}
            </span>
          </div>
        ),
        count,
        pct: total > 0 ? (count / total) * 100 : 0,
        barWidth: max > 0 ? (count / max) * 100 : 0,
      }));
    const unassignedRow: DistRow[] = unassigned > 0
      ? [{
          key: '__unassigned__',
          label: <span className="text-sm text-[var(--color-text-muted)] italic">Unassigned</span>,
          count: unassigned,
          pct: total > 0 ? (unassigned / total) * 100 : 0,
          barWidth: max > 0 ? (unassigned / max) * 100 : 0,
        }]
      : [];
    return { total, rows: [...namedRows, ...unassignedRow] };
  }, [issues, assigneeFilter]);

  const priorityDistribution = useMemo(() => {
    const filtered = priorityFilter === 'active'
      ? issues.filter(i => i.status !== 'DONE')
      : issues;
    const counts = new Map<number, number>();
    for (const i of filtered) {
      const p = i.priority || 0;
      counts.set(p, (counts.get(p) || 0) + 1);
    }
    const total = filtered.length;
    const max = Math.max(0, ...counts.values());
    const rows: DistRow[] = PRIORITY_LABELS
      .filter(p => (counts.get(p.value) || 0) > 0)
      .map(p => {
        const count = counts.get(p.value) || 0;
        return {
          key: String(p.value),
          label: (
            <div className="flex items-center gap-2">
              <span className={`text-xs font-medium tabular-nums w-6 ${p.markerClass}`}>
                {p.marker || '—'}
              </span>
              <span className="text-sm text-[var(--color-text-primary)]">{p.label}</span>
            </div>
          ),
          count,
          pct: total > 0 ? (count / total) * 100 : 0,
          barWidth: max > 0 ? (count / max) * 100 : 0,
        };
      });
    return { total, rows };
  }, [issues, priorityFilter]);

  const statusCards = [
    { label: 'Backlog', status: 'BACKLOG', value: stats.backlog, color: 'var(--color-status-backlog)' },
    { label: 'Planned', status: 'PLANNED', value: stats.planned, color: 'var(--color-status-planned)' },
    { label: 'In Progress', status: 'DOING', value: stats.doing, color: 'var(--color-status-doing)' },
    { label: 'Blocked', status: 'BLOCKED', value: stats.blocked, color: 'var(--color-status-blocked)' },
    { label: 'Done', status: 'DONE', value: stats.done, color: 'var(--color-status-done)' },
  ];

  return (
    <div className="h-full flex flex-col">
      {/* Header — matches the Issues view chrome */}
      <div className="flex items-center gap-3 px-5 h-11 border-b border-[var(--color-border-subtle)] shrink-0">
        <span className="text-sm font-medium text-[var(--color-text-primary)]">Overview</span>
        <span className="ml-auto text-xs text-[var(--color-text-muted)] tabular-nums">
          {issues.length} issue{issues.length === 1 ? '' : 's'}
        </span>
      </div>

      <div className="flex-1 overflow-y-auto">
        <div className="max-w-7xl mx-auto space-y-3 py-3">
        {/* Pulse */}
        <Section
          title="Pulse"
          icon={<SectionIcon d={SECTION_ICONS.pulse} />}
          collapsible
          storageKey="beats-dashboard-pulse-open"
        >
          <div className="px-5 py-3 grid grid-cols-2 lg:grid-cols-4 gap-3">
            <PulseCard title="Velocity">
              <div className="flex items-end justify-between gap-3">
                <div>
                  <p className="text-2xl font-semibold text-[var(--color-text-primary)] leading-none">
                    {metrics ? metrics.velocity.last_7d_points : '—'}
                  </p>
                  <p className="text-xs text-[var(--color-text-muted)] mt-2">pts last 7d</p>
                </div>
                {metrics && metrics.velocity.weekly_buckets.length > 0 && (
                  <Sparkline buckets={metrics.velocity.weekly_buckets} />
                )}
              </div>
            </PulseCard>

            <PulseCard title="Throughput">
              <p className="text-2xl font-semibold text-[var(--color-text-primary)] leading-none">
                {metrics ? metrics.throughput.last_7d : '—'}
              </p>
              <p className="text-xs text-[var(--color-text-muted)] mt-2">issues last 7d</p>
              {metrics && (
                <p className={`text-xs mt-1 ${
                  metrics.throughput.delta > 0
                    ? 'text-[var(--color-success)]'
                    : metrics.throughput.delta < 0
                      ? 'text-[var(--color-warning)]'
                      : 'text-[var(--color-text-muted)]'
                }`}>
                  {metrics.throughput.delta > 0 ? '+' : ''}{metrics.throughput.delta} vs prior 7d
                </p>
              )}
            </PulseCard>

            <PulseCard title="WIP">
              <p className="text-2xl font-semibold text-[var(--color-text-primary)] leading-none">
                {metrics ? metrics.wip.total : '—'}
              </p>
              <p className="text-xs text-[var(--color-text-muted)] mt-2">in progress</p>
              {metrics && metrics.wip.stale > 0 && (
                <div className="mt-1 flex items-center gap-1.5 text-xs text-[var(--color-warning)]">
                  <svg className="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                    <path strokeLinecap="round" strokeLinejoin="round" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                  </svg>
                  {metrics.wip.stale} stale &gt; {metrics.wip.stale_threshold_days}d
                </div>
              )}
            </PulseCard>

            <PulseCard title="Blockers">
              <p className="text-2xl font-semibold text-[var(--color-text-primary)] leading-none">
                {metrics ? metrics.blockers.total : '—'}
              </p>
              <p className="text-xs text-[var(--color-text-muted)] mt-2">blocked</p>
              {metrics && metrics.blockers.total > 0 && (
                <p className="text-xs text-[var(--color-error)] mt-1">
                  oldest {metrics.blockers.oldest_days}d
                </p>
              )}
            </PulseCard>
          </div>
        </Section>

        {/* Trends — three visual cards in one row */}
        <Section
          title="Trends"
          icon={<SectionIcon d={SECTION_ICONS.trends} />}
          collapsible
          storageKey="beats-dashboard-trends-open"
        >
          <div className="px-5 py-3 grid grid-cols-1 lg:grid-cols-3 gap-3">
            {/* Created vs Completed */}
            <Card variant="elevated" padding="sm" className="min-h-[112px] flex flex-col">
              <div className="flex items-center justify-between mb-2">
                <p className="text-xs uppercase tracking-wider text-[var(--color-text-muted)]">Created vs Completed</p>
                <div className="flex items-center gap-2 text-[10px] text-[var(--color-text-muted)]">
                  <span className="flex items-center gap-1">
                    <span className="w-2 h-2 rounded-sm bg-[var(--color-text-muted)]" />created
                  </span>
                  <span className="flex items-center gap-1">
                    <span className="w-2 h-2 rounded-sm bg-[var(--color-accent-primary)]" />completed
                  </span>
                </div>
              </div>
              {metrics ? (
                <div className="flex items-end justify-center">
                  <TrendChart weekly={metrics.trends.weekly} />
                </div>
              ) : (
                <p className="text-2xl font-semibold text-[var(--color-text-primary)] leading-none">—</p>
              )}
              <p className="mt-auto text-xs text-[var(--color-text-muted)]">last 8 weeks</p>
            </Card>

            {/* Median Triage Time */}
            <Card variant="elevated" padding="sm" className="min-h-[112px] flex flex-col">
              <p className="text-xs uppercase tracking-wider text-[var(--color-text-muted)] mb-1.5">Median Triage Time</p>
              <p className="text-2xl font-semibold text-[var(--color-text-primary)] leading-none">
                {metrics ? formatTriage(metrics.trends.median_triage_mins) : '—'}
              </p>
              <p className="mt-auto text-xs text-[var(--color-text-muted)]">
                {metrics && metrics.trends.triaged_count > 0
                  ? `across ${metrics.trends.triaged_count} triaged ${metrics.trends.triaged_count === 1 ? 'issue' : 'issues'}`
                  : 'no triaged issues yet'}
              </p>
            </Card>

            {/* Bug Age */}
            <Card variant="elevated" padding="sm" className="min-h-[112px] flex flex-col">
              <p className="text-xs uppercase tracking-wider text-[var(--color-text-muted)] mb-2">Bug Age</p>
              {(() => {
                const buckets = metrics?.trends.bug_age;
                const total = buckets
                  ? buckets.under_24h + buckets.under_48h + buckets.under_5d + buckets.under_14d + buckets.under_1mo + buckets.over_1mo
                  : 0;
                if (!metrics || total === 0) {
                  return (
                    <>
                      <p className="text-2xl font-semibold text-[var(--color-text-primary)] leading-none">0</p>
                      <p className="text-xs text-[var(--color-text-muted)] mt-2">no open bugs</p>
                    </>
                  );
                }
                const bars = [
                  { label: '<24h', count: buckets!.under_24h, cls: 'bg-[var(--color-text-secondary)]' },
                  { label: '<48h', count: buckets!.under_48h, cls: 'bg-[var(--color-text-secondary)]' },
                  { label: '<5d',  count: buckets!.under_5d,  cls: 'bg-[var(--color-warning)]' },
                  { label: '<14d', count: buckets!.under_14d, cls: 'bg-[var(--color-warning)]' },
                  { label: '<1mo', count: buckets!.under_1mo, cls: 'bg-[var(--color-error)]' },
                  { label: '>1mo', count: buckets!.over_1mo,  cls: 'bg-[var(--color-error)]' },
                ];
                const max = Math.max(1, ...bars.map(b => b.count));
                return (
                  <div className="mt-auto">
                    <div className="flex items-end gap-1 h-12">
                      {bars.map(b => (
                        <div key={b.label} className="flex-1 flex flex-col items-center justify-end h-full">
                          <span className="text-[10px] text-[var(--color-text-muted)] tabular-nums leading-none mb-0.5">
                            {b.count > 0 ? b.count : ''}
                          </span>
                          <div
                            className={`w-3 rounded-t-sm ${b.cls} transition-all duration-500`}
                            style={{ height: `${(b.count / max) * 100}%`, minHeight: b.count > 0 ? 2 : 0 }}
                          />
                        </div>
                      ))}
                    </div>
                    <div className="flex gap-1 mt-1.5">
                      {bars.map(b => (
                        <span key={b.label} className="flex-1 text-center text-[10px] text-[var(--color-text-muted)] tabular-nums">
                          {b.label}
                        </span>
                      ))}
                    </div>
                  </div>
                );
              })()}
            </Card>
          </div>
        </Section>

        {/* Needs Attention — only renders when there's something to surface */}
        {metrics && metrics.attention.length > 0 && (
          <Section
            title="Needs Attention"
            icon={<SectionIcon d={SECTION_ICONS.attention} />}
            count={metrics.attention.length}
            collapsible
            storageKey="beats-dashboard-attention-open"
          >
            <div className="py-2">
              {metrics.attention.map(item => {
                const issue = issues.find(i => i.id === item.issue_id);
                return (
                  <button
                    key={item.issue_id}
                    onClick={() => issue && onIssueClick?.(issue)}
                    className="flex items-center gap-3 w-full px-5 py-1.5 text-left transition-colors hover:bg-[var(--color-bg-hover)]"
                  >
                    <AttentionBadge item={item} />
                    <span className="text-sm text-[var(--color-text-primary)] truncate flex-1">
                      {item.title}
                    </span>
                    <span className="text-xs text-[var(--color-text-muted)] tabular-nums shrink-0">
                      {attentionMeta(item)}
                    </span>
                  </button>
                );
              })}
            </div>
          </Section>
        )}

        {/* Workload */}
        <Section
          title="Workload"
          icon={<SectionIcon d={SECTION_ICONS.workload} />}
          count={metrics?.workload.length}
          collapsible
          storageKey="beats-dashboard-workload-open"
        >
          {!metrics || metrics.workload.length === 0 ? (
            <div className="px-5 py-10 flex items-center justify-center">
              <p className="text-sm text-[var(--color-text-muted)] text-center">No assigned work.</p>
            </div>
          ) : (
            <div className="py-2">
              <div className="grid grid-cols-[1fr_auto_auto_auto] gap-x-4 px-5 py-1 text-[10px] uppercase tracking-wider text-[var(--color-text-muted)]">
                <span>Person</span>
                <span className="text-right w-10">WIP</span>
                <span className="text-right w-10">Pts</span>
                <span className="text-right w-12">Blocked</span>
              </div>
              {metrics.workload.map(w => (
                <div
                  key={w.assignee}
                  className="grid grid-cols-[1fr_auto_auto_auto] gap-x-4 items-center px-5 py-1.5 hover:bg-[var(--color-bg-hover)] transition-colors"
                  title={w.last_completed ? `Last completed ${w.last_completed}` : 'No completed issues yet'}
                >
                  <div className="flex items-center gap-2 min-w-0">
                    <Avatar name={w.assignee} size="xs" />
                    <span className="text-sm text-[var(--color-text-primary)] truncate">
                      {w.assignee.split(' <')[0]}
                    </span>
                  </div>
                  <span className="w-10 text-right text-sm tabular-nums text-[var(--color-text-primary)]">
                    {w.in_progress}
                  </span>
                  <span className="w-10 text-right text-sm tabular-nums text-[var(--color-text-muted)]">
                    {w.open_points}
                  </span>
                  <span className={`w-12 text-right text-sm tabular-nums ${w.blocked > 0 ? 'text-[var(--color-error)]' : 'text-[var(--color-text-muted)]'}`}>
                    {w.blocked}
                  </span>
                </div>
              ))}
            </div>
          )}
        </Section>

        {/* Active Epics */}
        <Section
          title="Active Epics"
          icon={<SectionIcon d={SECTION_ICONS.epics} />}
          count={metrics?.epics.length}
          collapsible
          storageKey="beats-dashboard-epics-open"
        >
          {!metrics || metrics.epics.length === 0 ? (
            <div className="px-5 py-10 flex items-center justify-center">
              <p className="text-sm text-[var(--color-text-muted)] text-center">
                No epics yet — create one with the <span className="font-mono">epic</span> label.
              </p>
            </div>
          ) : (
            <div className="py-2">
              {metrics.epics.map(ep => {
                const issue = issues.find(i => i.id === ep.issue_id);
                return (
                  <button
                    key={ep.issue_id}
                    onClick={() => issue && onIssueClick?.(issue)}
                    className="flex items-center gap-2.5 w-full px-5 py-1.5 text-left transition-colors hover:bg-[var(--color-bg-hover)]"
                  >
                    <StatusIcon status={issue?.status || 'PLANNED'} size={14} />
                    <span className="text-sm text-[var(--color-text-primary)] truncate">
                      {ep.title}
                    </span>
                    <CopyableId
                      id={ep.issue_id}
                      className="text-xs shrink-0 tabular-nums"
                    />
                    {ep.stale && (
                      <span className="text-[10px] uppercase tracking-wider font-medium px-1.5 py-0.5 rounded border shrink-0 text-[var(--color-warning)] border-[var(--color-warning)]/40 bg-[var(--color-warning)]/10">
                        STALE
                      </span>
                    )}
                    <span className="ml-auto flex items-center gap-1.5 text-xs text-[var(--color-text-muted)] shrink-0">
                      <SubProgress done={ep.children_done} total={ep.children_total} />
                      {ep.children_done}/{ep.children_total}
                    </span>
                  </button>
                );
              })}
            </div>
          )}
        </Section>

        {/* Recent Activity */}
        <Section
          title="Recent Activity"
          icon={<SectionIcon d={SECTION_ICONS.activity} />}
          count={activity.length}
          collapsible
          storageKey="beats-dashboard-activity-open"
        >
          {activity.length === 0 ? (
            <div className="px-5 py-10 flex items-center justify-center">
              <p className="text-sm text-[var(--color-text-muted)] text-center">No activity yet.</p>
            </div>
          ) : (
            <div className="py-2">
              {activity.map((evt, i) => {
                const desc = describeActivity(evt);
                if (!desc) return null;
                const issue = issues.find(it => it.id === evt.issue_id);
                const title = evt.issue_title || evt.issue_id;
                const key = `${evt.issue_id}-${evt.created_at}-${i}`;
                const isExpanded = expandedComments.has(key);
                const previewText = desc.preview ? stripMarkdown(desc.preview) : '';
                return (
                  <div key={key} className="px-5 py-2">
                    <div className="flex items-center gap-2 text-sm text-[var(--color-text-muted)]">
                      <Avatar name={evt.created_by} size="xs" />
                      <span className="text-[var(--color-text-primary)] font-medium shrink-0 ml-1">
                        {shortName(evt.created_by)}
                      </span>
                      <div className='shrink-0 flex items-center gap-1 text-warning'>
                        <ActIcon k={desc.icon} />
                        <span className="shrink-0">{desc.content}</span>
                      </div>
                      <button
                        onClick={() => issue && onIssueClick?.(issue)}
                        disabled={!issue}
                        className="text-[var(--color-text-primary)] hover:text-[var(--color-accent-primary)] truncate min-w-0 disabled:opacity-60 disabled:cursor-default"
                        title={title}
                      >
                        {title}
                      </button>
                      <span className="ml-auto text-xs tabular-nums shrink-0 whitespace-nowrap">
                        {formatRelativeTime(evt.created_at)}
                      </span>
                    </div>
                    {desc.preview && !isExpanded && (
                      <button
                        onClick={() => toggleComment(key)}
                        className="block w-full text-left ml-4 mt-0.5 text-sm text-[var(--color-text-muted)] italic truncate hover:text-[var(--color-text-secondary)] transition-colors cursor-pointer"
                        title="Expand comment"
                      >
                        “{previewText.slice(0, 140)}{previewText.length > 140 ? '…' : ''}”
                      </button>
                    )}
                    {desc.preview && isExpanded && (
                      <button
                        onClick={() => toggleComment(key)}
                        className="block w-full text-left ml-4 mt-1.5 rounded-[var(--radius-md)] bg-[var(--color-bg-secondary)] border border-[var(--color-border-default)] px-3 py-2 hover:border-[var(--color-border-focus)] transition-colors cursor-pointer"
                        title="Collapse comment"
                      >
                        <div className="prose-beats text-sm">
                          <Markdown remarkPlugins={[remarkGfm, remarkBreaks]}>
                            {desc.preview}
                          </Markdown>
                        </div>
                      </button>
                    )}
                  </div>
                );
              })}
            </div>
          )}
        </Section>

        {/* Composition — exploration cards, collapsed by default */}
        <Section
          title="Composition"
          icon={<SectionIcon d={SECTION_ICONS.composition} />}
          collapsible
          defaultOpen={false}
          storageKey="beats-dashboard-composition-open"
        >
          <div className="px-5 py-4 space-y-6">
            {/* By status */}
            <div>
              <h3 className="text-xs uppercase tracking-wider text-[var(--color-text-muted)] mb-3">By status</h3>
              <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-5 gap-3">
                {statusCards.map((card) => (
                  <Card key={card.label} variant="default" className="text-center">
                    <div className="flex items-center justify-center gap-1.5 mb-1.5">
                      <StatusIcon status={card.status} size={12} />
                      <p className="text-xs uppercase tracking-wider text-[var(--color-text-muted)]">{card.label}</p>
                    </div>
                    <p className="text-2xl font-semibold text-[var(--color-text-primary)] leading-none">
                      {card.value}
                    </p>
                    <div className="h-0.5 mt-3 rounded-full bg-[var(--color-bg-tertiary)]">
                      <div
                        className="h-full rounded-full transition-all duration-500"
                        style={{
                          background: card.color,
                          width: `${stats.total ? (card.value / stats.total) * 100 : 0}%`,
                        }}
                      />
                    </div>
                  </Card>
                ))}
              </div>
            </div>

            {/* By label / By assignee / By priority */}
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
              <DistributionSection
                title="By label"
                filter={labelFilter}
                onFilterChange={setLabelFilter}
                rows={labelDistribution.rows}
                total={labelDistribution.total}
                emptyHasIssues="No labels on the filtered issues."
              />
              <DistributionSection
                title="By assignee"
                filter={assigneeFilter}
                onFilterChange={setAssigneeFilter}
                rows={assigneeDistribution.rows}
                total={assigneeDistribution.total}
                emptyHasIssues="No assignees on the filtered issues."
              />
              <DistributionSection
                title="By priority"
                filter={priorityFilter}
                onFilterChange={setPriorityFilter}
                rows={priorityDistribution.rows}
                total={priorityDistribution.total}
                emptyHasIssues="No priorities set on the filtered issues."
              />
            </div>
          </div>
        </Section>
        </div>
      </div>
    </div>
  );
}
