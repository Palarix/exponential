import type { Issue } from '../../api/client';
import { Card, LabelBadge, StatusBadge } from '../ui';

interface DependenciesProps {
  issues: Issue[];
  onIssueClick?: (issue: Issue) => void;
}

export default function Dependencies({ issues, onIssueClick }: DependenciesProps) {
  // Find all dependency relationships
  const dependencies: { source: Issue; target: Issue; kind: string }[] = [];

  for (const issue of issues) {
    if (issue.dependencies) {
      for (const dep of issue.dependencies) {
        const target = issues.find((i) => i.id === dep.target_id);
        if (target) {
          dependencies.push({
            source: issue,
            target,
            kind: dep.kind,
          });
        }
      }
    }
  }

  // Group by kind
  const byKind = dependencies.reduce((acc, dep) => {
    if (!acc[dep.kind]) acc[dep.kind] = [];
    acc[dep.kind].push(dep);
    return acc;
  }, {} as Record<string, typeof dependencies>);

  const kindIcons: Record<string, string> = {
    blocked_by: '🚫',
    blocks: '⛔',
    child: '📎',
    parent: '📂',
    related: '🔗',
  };

  const kindColors: Record<string, string> = {
    blocked_by: 'var(--color-error)',
    blocks: 'var(--color-error)',
    child: 'var(--color-kind-epic)',
    parent: 'var(--color-kind-epic)',
    related: 'var(--color-info)',
  };

  return (
    <div className="space-y-6 animate-fade-in">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-[var(--color-text-primary)]">Dependencies</h1>
          <p className="text-sm text-[var(--color-text-muted)] mt-1">
            {dependencies.length} relationship{dependencies.length !== 1 ? 's' : ''} found
          </p>
        </div>
      </div>

      {dependencies.length === 0 ? (
        <Card variant="default" className="border-dashed border-2">
          <div className="text-center py-12 text-[var(--color-text-muted)]">
            <svg className="w-16 h-16 mx-auto mb-4 opacity-40" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1" />
            </svg>
            <p className="text-lg font-medium mb-1">No dependencies</p>
            <p className="text-sm">Issues aren't linked to each other yet</p>
          </div>
        </Card>
      ) : (
        <div className="space-y-8">
          {Object.entries(byKind).map(([kind, deps]) => (
            <div key={kind}>
              <div className="flex items-center gap-2 mb-4">
                <span className="text-xl">{kindIcons[kind] || '🔗'}</span>
                <h2 className="text-lg font-semibold text-[var(--color-text-primary)] capitalize">
                  {kind.replace('_', ' ')}
                </h2>
                <span
                  className="px-2 py-0.5 text-xs font-medium rounded-[var(--radius-full)]"
                  style={{
                    background: `${kindColors[kind] || 'var(--color-text-muted)'}20`,
                    color: kindColors[kind] || 'var(--color-text-muted)'
                  }}
                >
                  {deps.length}
                </span>
              </div>

              <div className="space-y-3">
                {deps.map((dep, i) => (
                  <DependencyCard
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
          ))}
        </div>
      )}
    </div>
  );
}

function DependencyCard({
  source,
  target,
  kind,
  color,
  onIssueClick
}: {
  source: Issue;
  target: Issue;
  kind: string;
  color: string;
  onIssueClick?: (issue: Issue) => void;
}) {
  return (
    <Card variant="default" padding="none" className="overflow-hidden">
      <div className="flex items-stretch">
        {/* Source Issue */}
        <IssueLink issue={source} onClick={() => onIssueClick?.(source)} />

        {/* Relationship Arrow */}
        <div
          className="flex items-center justify-center px-4 min-w-[140px]"
          style={{ background: `${color}10` }}
        >
          <div className="flex items-center gap-2">
            <div className="w-8 h-0.5 rounded-full" style={{ background: color }} />
            <span
              className="text-xs font-medium whitespace-nowrap"
              style={{ color }}
            >
              {kind.replace('_', ' ')}
            </span>
            <svg
              className="w-4 h-4"
              style={{ color }}
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
              strokeWidth={2}
            >
              <path strokeLinecap="round" strokeLinejoin="round" d="M14 5l7 7m0 0l-7 7m7-7H3" />
            </svg>
          </div>
        </div>

        {/* Target Issue */}
        <IssueLink issue={target} onClick={() => onIssueClick?.(target)} />
      </div>
    </Card>
  );
}

function IssueLink({ issue, onClick }: { issue: Issue; onClick?: () => void }) {
  return (
    <div
      className="
        flex-1 p-4 
        hover:bg-[var(--color-bg-hover)] 
        cursor-pointer 
        transition-colors duration-[var(--duration-fast)]
        group
      "
      onClick={onClick}
    >
      <div className="flex items-center gap-3">
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 mb-1">
            <div className="flex gap-1 overflow-hidden">
              {issue.labels?.map((label: string) => (
                <LabelBadge key={label} label={label} />
              ))}
            </div>
            <span className="text-[10px] font-mono text-[var(--color-text-muted)]">
              {issue.id}
            </span>
          </div>
          <p className="text-sm font-medium text-[var(--color-text-primary)] truncate group-hover:text-[var(--color-text-accent)] transition-colors">
            {issue.title}
          </p>
        </div>
        <StatusBadge status={issue.status} />
      </div>
    </div>
  );
}
