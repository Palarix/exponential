import type { Issue } from '../../api/client';
import { Card, ProgressRing, StatusIcon } from '../ui';

interface DashboardProps {
  issues: Issue[];
}

export default function Dashboard({ issues }: DashboardProps) {
  const stats = {
    total: issues.length,
    backlog: issues.filter((i) => i.status === 'BACKLOG').length,
    planned: issues.filter((i) => i.status === 'PLANNED').length,
    doing: issues.filter((i) => i.status === 'DOING').length,
    blocked: issues.filter((i) => i.status === 'BLOCKED').length,
    done: issues.filter((i) => i.status === 'DONE').length,
  };

  const completionRate = stats.total > 0 ? (stats.done / stats.total) * 100 : 0;

  const statusCards = [
    { label: 'Backlog', status: 'BACKLOG', value: stats.backlog, color: 'var(--color-status-backlog)' },
    { label: 'Planned', status: 'PLANNED', value: stats.planned, color: 'var(--color-status-planned)' },
    { label: 'In Progress', status: 'DOING', value: stats.doing, color: 'var(--color-status-doing)' },
    { label: 'Blocked', status: 'BLOCKED', value: stats.blocked, color: 'var(--color-status-blocked)' },
    { label: 'Done', status: 'DONE', value: stats.done, color: 'var(--color-status-done)' },
  ];

  return (
    <div className="h-full flex flex-col">
      {/* Header */}
      <div className="flex items-center gap-3 px-5 h-11 border-b border-[var(--color-border-subtle)] shrink-0">
        <span className="text-sm font-medium text-[var(--color-text-primary)]">Dashboard</span>
      </div>

      <div className="flex-1 overflow-y-auto p-6 space-y-6 max-w-5xl">
        {/* Key Metrics */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          {/* Completion Ring */}
          <Card variant="elevated" className="flex items-center gap-5">
            <ProgressRing
              value={stats.done}
              max={stats.total || 1}
              size={80}
              strokeWidth={6}
              color="var(--color-success)"
              label={`${Math.round(completionRate)}%`}
              sublabel="complete"
            />
            <div>
              <p className="text-sm text-[var(--color-text-muted)]">Overall Progress</p>
              <p className="text-2xl font-bold text-[var(--color-text-primary)]">
                {stats.done}<span className="text-sm text-[var(--color-text-muted)]">/{stats.total}</span>
              </p>
              <p className="text-xs text-[var(--color-text-muted)]">issues completed</p>
            </div>
          </Card>

          {/* Active Work */}
          <Card variant="elevated">
            <p className="text-sm text-[var(--color-text-muted)] mb-1">Active Work</p>
            <p className="text-2xl font-bold text-[var(--color-warning)]">{stats.doing}</p>
            <p className="text-xs text-[var(--color-text-muted)]">issues in progress</p>
            {stats.blocked > 0 && (
              <div className="mt-2 flex items-center gap-1.5 text-sm text-[var(--color-error)]">
                <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                </svg>
                {stats.blocked} blocked
              </div>
            )}
          </Card>

          {/* Pipeline */}
          <Card variant="elevated">
            <p className="text-sm text-[var(--color-text-muted)] mb-1">Pipeline</p>
            <p className="text-2xl font-bold text-[var(--color-text-primary)]">{stats.backlog + stats.planned}</p>
            <p className="text-xs text-[var(--color-text-muted)]">in backlog & planned</p>
          </Card>
        </div>

        {/* Status Distribution */}
        <div>
          <h2 className="text-sm font-medium text-[var(--color-text-primary)] mb-3">Status Distribution</h2>
          <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-5 gap-3">
            {statusCards.map((card) => (
              <Card key={card.label} variant="default" className="text-center">
                <div className="flex items-center justify-center gap-1.5 mb-1">
                  <StatusIcon status={card.status} size={12} />
                  <p className="text-sm text-[var(--color-text-muted)]">{card.label}</p>
                </div>
                <p className="text-2xl font-bold" style={{ color: card.color }}>
                  {card.value}
                </p>
                <div className="h-0.5 mt-2 rounded-full bg-[var(--color-bg-tertiary)]">
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
      </div>
    </div>
  );
}
