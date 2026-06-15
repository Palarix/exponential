import { useState } from 'react';
import type { Issue } from '../../api/client';
import { LabelBadge, StatusBadge, Toggle } from '../ui';

interface DependenciesProps {
  issues: Issue[];
  onIssueClick?: (issue: Issue) => void;
}

export default function Dependencies({ issues, onIssueClick }: DependenciesProps) {
  const [showCompleted, setShowCompleted] = useState(() => localStorage.getItem('beats-deps-show-completed') === 'true');

  const allDependencies: { source: Issue; target: Issue; kind: string }[] = [];

  for (const issue of issues) {
    if (issue.dependencies) {
      for (const dep of issue.dependencies) {
        const target = issues.find((i) => i.id === dep.target_id);
        if (target) {
          allDependencies.push({ source: issue, target, kind: dep.kind });
        }
      }
    }
  }

  const dependencies = showCompleted
    ? allDependencies
    : allDependencies.filter(dep => !(dep.source.status === 'DONE' && dep.target.status === 'DONE'));

  const completedCount = allDependencies.length - allDependencies.filter(dep => !(dep.source.status === 'DONE' && dep.target.status === 'DONE')).length;

  const byKind = dependencies.reduce((acc, dep) => {
    if (!acc[dep.kind]) acc[dep.kind] = [];
    acc[dep.kind].push(dep);
    return acc;
  }, {} as Record<string, typeof dependencies>);

  const kindColors: Record<string, string> = {
    blocked_by: 'var(--color-error)',
    blocks: 'var(--color-error)',
    child: 'var(--color-accent-primary)',
    parent: 'var(--color-accent-primary)',
    related: 'var(--color-info)',
  };

  return (
    <div className="h-full flex flex-col">
      {/* Header */}
      <div className="flex items-center gap-3 px-5 h-11 border-b border-[var(--color-border-subtle)] shrink-0">
        <span className="text-sm font-medium text-[var(--color-text-primary)]">Dependencies</span>
        <span className="text-xs text-[var(--color-text-muted)] tabular-nums">
          {dependencies.length} relationship{dependencies.length !== 1 ? 's' : ''}
        </span>
        <div className="flex-1" />
        {completedCount > 0 && (
          <Toggle
            checked={showCompleted}
            onChange={(next) => {
              setShowCompleted(next);
              localStorage.setItem('beats-deps-show-completed', String(next));
            }}
            label={`Show completed (${completedCount})`}
          />
        )}
      </div>

      <div className="flex-1 overflow-y-auto p-5 space-y-6 max-w-7xl mx-auto">
        {dependencies.length === 0 ? (
          <div className="text-center py-16 text-[var(--color-text-muted)]">
            <svg className="w-10 h-10 mx-auto mb-3 opacity-30" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1" />
            </svg>
            <p className="text-sm">No dependencies</p>
            <p className="text-sm mt-1">Issues aren't linked to each other yet</p>
          </div>
        ) : (
          Object.entries(byKind).map(([kind, deps]) => (
            <div key={kind}>
              <div className="flex items-center gap-2 mb-3">
                <h2 className="text-sm font-medium text-[var(--color-text-primary)] capitalize">
                  {kind.replace('_', ' ')}
                </h2>
                <span
                  className="text-xs tabular-nums px-2 py-1 rounded-[var(--radius-sm)]"
                  style={{
                    background: `${kindColors[kind] || 'var(--color-text-muted)'}15`,
                    color: kindColors[kind] || 'var(--color-text-muted)',
                  }}
                >
                  {deps.length}
                </span>
              </div>

              <div className="space-y-1">
                {deps.map((dep, i) => (
                  <DependencyRow
                    key={i}
                    source={dep.source}
                    target={dep.target}
                    kind={dep.kind}
                    color={kindColors[kind] || 'var(--color-text-muted)'}
                    onIssueClick={onIssueClick}
                  />
                ))}
              </div>
            </div>
          ))
        )}
      </div>
    </div>
  );
}

function DependencyRow({
  source,
  target,
  kind,
  color,
  onIssueClick,
}: {
  source: Issue;
  target: Issue;
  kind: string;
  color: string;
  onIssueClick?: (issue: Issue) => void;
}) {
  return (
    <div className="flex items-center border border-[var(--color-border-subtle)] rounded-[var(--radius-md)] overflow-hidden">
      {/* Source */}
      <IssueLink issue={source} onClick={() => onIssueClick?.(source)} />

      {/* Arrow */}
      <div className="flex items-center gap-2 px-3 shrink-0" style={{ color }}>
        <div className="w-5 h-px" style={{ background: color }} />
        <span className="text-xs font-medium whitespace-nowrap">{kind.replace('_', ' ')}</span>
        <svg className="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}>
          <path strokeLinecap="round" strokeLinejoin="round" d="M14 5l7 7m0 0l-7 7m7-7H3" />
        </svg>
      </div>

      {/* Target */}
      <IssueLink issue={target} onClick={() => onIssueClick?.(target)} />
    </div>
  );
}

function IssueLink({ issue, onClick }: { issue: Issue; onClick?: () => void }) {
  return (
    <div
      className="flex-1 flex items-center gap-2 px-3 py-2 hover:bg-[var(--color-bg-hover)] cursor-pointer transition-colors duration-[var(--duration-fast)] min-w-0"
      onClick={onClick}
    >
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2 mb-1">
          <span className="text-xs font-mono text-[var(--color-text-muted)]">{issue.id}</span>
          {issue.labels?.map((label: string) => (
            <LabelBadge key={label} label={label} />
          ))}
        </div>
        <p className="text-sm text-[var(--color-text-primary)] truncate">{issue.title}</p>
      </div>
      <StatusBadge status={issue.status} />
    </div>
  );
}
