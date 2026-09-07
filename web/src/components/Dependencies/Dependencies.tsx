import { useState, useMemo, useCallback } from 'react';
import type { Issue } from '../../api/client';
import { LabelBadge, Toggle } from '../ui';
import StatusIcon from '../ui/StatusIcon';
import DepGraphView from './DepGraph';
import { useDepGraph, collectEdges, computeStats, resolveIssue, isResolved } from './useDepGraph';

const KIND_LABELS: Record<string, string> = {
  blocks: 'Blocks',
  depends_on: 'Depends on',
  relates_to: 'Relates to',
  related: 'Relates to',
  duplicates: 'Duplicates',
};

const KIND_FILTER_OPTIONS = [
  { value: '', label: 'All types' },
  { value: 'blocks', label: 'Blocks' },
  { value: 'depends_on', label: 'Depends on' },
  { value: 'relates_to', label: 'Relates to' },
  { value: 'duplicates', label: 'Duplicates' },
];

interface DependenciesProps {
  issues: Issue[];
  onIssueClick?: (issue: Issue) => void;
  focusIssueId?: string | null;
  onFocusChange?: (issueId: string | null) => void;
}

export default function Dependencies({ issues, onIssueClick, focusIssueId, onFocusChange }: DependenciesProps) {
  const [showCompleted, setShowCompleted] = useState(() => localStorage.getItem('exponential-deps-show-completed') === 'true');
  const [kindFilter, setKindFilter] = useState('');
  const [search, setSearch] = useState('');

  const handleShowCompleted = useCallback((next: boolean) => {
    setShowCompleted(next);
    localStorage.setItem('exponential-deps-show-completed', String(next));
  }, []);

  const stats = useMemo(() => computeStats(issues), [issues]);

  const focusIssue = focusIssueId ? resolveIssue(issues, focusIssueId) : undefined;

  if (focusIssue && focusIssueId) {
    return (
      <FocusedGraph
        issues={issues}
        focusIssue={focusIssue}
        focusIssueId={focusIssue.id}
        showCompleted={showCompleted}
        onShowCompletedChange={handleShowCompleted}
        onIssueClick={onIssueClick}
        onBack={() => onFocusChange?.(null)}
      />
    );
  }

  return (
    <TableView
      issues={issues}
      stats={stats}
      showCompleted={showCompleted}
      onShowCompletedChange={handleShowCompleted}
      kindFilter={kindFilter}
      onKindFilterChange={setKindFilter}
      search={search}
      onSearchChange={setSearch}
      onRowClick={(issueId) => onFocusChange?.(issueId)}
      onIssueClick={onIssueClick}
    />
  );
}

function FocusedGraph({
  issues,
  focusIssue,
  focusIssueId,
  showCompleted,
  onShowCompletedChange,
  onIssueClick,
  onBack,
}: {
  issues: Issue[];
  focusIssue: Issue;
  focusIssueId: string;
  showCompleted: boolean;
  onShowCompletedChange: (v: boolean) => void;
  onIssueClick?: (issue: Issue) => void;
  onBack: () => void;
}) {
  const graph = useDepGraph(issues, focusIssueId, showCompleted);

  return (
    <DepGraphView
      graph={graph}
      focusIssue={focusIssue}
      showCompleted={showCompleted}
      onShowCompletedChange={onShowCompletedChange}
      onIssueClick={onIssueClick}
      onBack={onBack}
    />
  );
}

interface TableViewProps {
  issues: Issue[];
  stats: { resolved: number; total: number };
  showCompleted: boolean;
  onShowCompletedChange: (v: boolean) => void;
  kindFilter: string;
  onKindFilterChange: (v: string) => void;
  search: string;
  onSearchChange: (v: string) => void;
  onRowClick: (issueId: string) => void;
  onIssueClick?: (issue: Issue) => void;
}

function TableView({
  issues,
  stats,
  showCompleted,
  onShowCompletedChange,
  kindFilter,
  onKindFilterChange,
  search,
  onSearchChange,
  onRowClick,
}: TableViewProps) {
  const issueMap = useMemo(() => new Map(issues.map(i => [i.id, i])), [issues]);

  const rows = useMemo(() => {
    const edges = collectEdges(issues);
    let filtered = edges;

    if (!showCompleted) {
      filtered = filtered.filter(e => !isResolved(e, issueMap));
    }

    if (kindFilter) {
      const normalizedFilter = kindFilter === 'relates_to' ? new Set(['relates_to', 'related']) : new Set([kindFilter]);
      filtered = filtered.filter(e => normalizedFilter.has(e.kind));
    }

    if (search) {
      const q = search.toLowerCase();
      filtered = filtered.filter(e => {
        const s = issueMap.get(e.sourceId);
        const t = issueMap.get(e.targetId);
        return (
          e.sourceId.toLowerCase().includes(q) ||
          e.targetId.toLowerCase().includes(q) ||
          s?.title.toLowerCase().includes(q) ||
          t?.title.toLowerCase().includes(q)
        );
      });
    }

    return filtered;
  }, [issues, issueMap, showCompleted, kindFilter, search]);

  const searchRef = useCallback((el: HTMLInputElement | null) => {
    if (!el) return;
    const handler = (e: KeyboardEvent) => {
      if (e.key === '/' && !e.metaKey && !e.ctrlKey && document.activeElement !== el) {
        e.preventDefault();
        el.focus();
      }
    };
    document.addEventListener('keydown', handler);
    return () => document.removeEventListener('keydown', handler);
  }, []);

  return (
    <div className="h-full flex flex-col">
      {/* Header */}
      <div className="flex items-center gap-3 px-5 h-11 border-b border-[var(--color-border-subtle)] shrink-0">
        <span className="text-sm font-medium text-[var(--color-text-primary)] shrink-0">Dependencies</span>

        {stats.total > 0 && (
          <span className="text-xs tabular-nums shrink-0" style={{ color: stats.resolved === stats.total ? 'var(--color-success)' : 'var(--color-text-muted)' }}>
            {stats.resolved} of {stats.total} blocker{stats.total !== 1 ? 's' : ''} resolved
          </span>
        )}

        {/* Search — centered, flexible width */}
        <div className="flex-1 flex justify-center px-4">
          <div className="relative w-full max-w-md">
            <input
              ref={searchRef}
              type="text"
              placeholder="Search dependencies..."
              value={search}
              onChange={e => onSearchChange(e.target.value)}
              className="text-sm h-8 pl-8 pr-3 w-full rounded-[var(--radius-md)] border border-[var(--color-border-subtle)] bg-[var(--color-surface-0)] text-[var(--color-text-primary)] outline-none focus:border-[var(--color-border-focus)] placeholder:text-[var(--color-text-muted)]"
            />
            <svg className="absolute left-2.5 top-1/2 -translate-y-1/2 w-4 h-4 text-[var(--color-text-muted)]" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M21 21l-5.197-5.197m0 0A7.5 7.5 0 105.196 5.196a7.5 7.5 0 0010.607 10.607z" />
            </svg>
          </div>
        </div>

        {/* Kind filter */}
        <select
          className="text-xs h-7 px-2 rounded-[var(--radius-sm)] border border-[var(--color-border-subtle)] bg-[var(--color-surface-0)] text-[var(--color-text-secondary)] outline-none shrink-0"
          value={kindFilter}
          onChange={e => onKindFilterChange(e.target.value)}
        >
          {KIND_FILTER_OPTIONS.map(o => (
            <option key={o.value} value={o.value}>{o.label}</option>
          ))}
        </select>

        <Toggle
          checked={showCompleted}
          onChange={onShowCompletedChange}
          label="Completed"
        />
      </div>

      {/* Table */}
      {rows.length === 0 ? (
        <EmptyState hasAnyDeps={issues.some(i => i.dependencies && i.dependencies.length > 0)} />
      ) : (
        <div className="flex-1 overflow-y-auto px-5 py-3">
          <div className="space-y-1.5">
            {rows.map((edge, i) => {
              const source = issueMap.get(edge.sourceId);
              const target = issueMap.get(edge.targetId);
              if (!source || !target) return null;
              return (
                <DepRow
                  key={i}
                  source={source}
                  target={target}
                  kind={edge.kind}
                  onClick={() => onRowClick(source.id)}
                />
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
}

function IssueCell({ issue }: { issue: Issue }) {
  const isDone = issue.status === 'DONE';
  return (
    <div className="flex items-center gap-2.5 min-w-0 flex-1 px-3 py-2">
      <StatusIcon status={issue.status} size={14} isInferred={issue.is_inferred} />
      <span className="text-[11px] font-mono text-[var(--color-text-muted)] shrink-0">{issue.id.replace(/^xpo-/, '')}</span>
      {issue.labels?.[0] && <LabelBadge label={issue.labels[0]} borderless />}
      <span
        className="text-sm text-[var(--color-text-primary)] truncate"
        style={{
          textDecoration: isDone ? 'line-through' : undefined,
          opacity: isDone ? 0.6 : 1,
        }}
      >
        {issue.title}
      </span>
    </div>
  );
}

function DepRow({ source, target, kind, onClick }: { source: Issue; target: Issue; kind: string; onClick: () => void }) {
  return (
    <div
      className="flex items-center rounded-[var(--radius-md)] border border-[var(--color-border-subtle)] hover:border-[var(--color-border-default)] hover:bg-[var(--color-hover-surface)] cursor-pointer transition-all duration-[var(--duration-fast)]"
      onClick={onClick}
    >
      <IssueCell issue={source} />

      {/* Kind column — fixed width center divider */}
      <div className="flex items-center justify-center shrink-0 w-28 px-1">
        <div className="flex items-center gap-1.5">
          <div className="w-4 h-px" style={{ background: kindColor(kind) }} />
          <span className="text-[11px] font-medium whitespace-nowrap" style={{ color: kindColor(kind) }}>
            {KIND_LABELS[kind] ?? kind.replace(/_/g, ' ')}
          </span>
          <svg className="w-2.5 h-2.5 shrink-0" style={{ color: kindColor(kind) }} fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M13.5 4.5L21 12m0 0l-7.5 7.5M21 12H3" />
          </svg>
        </div>
      </div>

      <IssueCell issue={target} />
    </div>
  );
}

function kindColor(kind: string): string {
  const colors: Record<string, string> = {
    blocks: 'var(--color-error)',
    depends_on: 'var(--color-error)',
    relates_to: 'var(--color-info)',
    related: 'var(--color-info)',
    duplicates: 'var(--color-text-muted)',
  };
  return colors[kind] ?? 'var(--color-text-muted)';
}

function EmptyState({ hasAnyDeps }: { hasAnyDeps: boolean }) {
  return (
    <div className="flex-1 flex items-center justify-center">
      <div className="text-center py-16 text-[var(--color-text-muted)]">
        <svg className="w-10 h-10 mx-auto mb-3 opacity-30" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
          <path strokeLinecap="round" strokeLinejoin="round" d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1" />
        </svg>
        <p className="text-sm">{hasAnyDeps ? 'No dependencies match the current filters' : 'No dependencies'}</p>
        {!hasAnyDeps && <p className="text-sm mt-1">Issues aren't linked to each other yet</p>}
      </div>
    </div>
  );
}
