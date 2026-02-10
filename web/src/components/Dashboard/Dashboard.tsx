import type { Issue } from '../../api/client';
import { Card, ProgressRing, Progress } from '../ui';

interface DashboardProps {
  issues: Issue[];
}

export default function Dashboard({ issues }: DashboardProps) {
  // Calculate stats
  const stats = {
    total: issues.length,
    backlog: issues.filter((i) => i.status === 'BACKLOG').length,
    planned: issues.filter((i) => i.status === 'PLANNED').length,
    doing: issues.filter((i) => i.status === 'DOING').length,
    blocked: issues.filter((i) => i.status === 'BLOCKED').length,
    done: issues.filter((i) => i.status === 'DONE').length,
    epics: issues.filter((i) => i.kind === 'EPIC').length,
    tasks: issues.filter((i) => i.kind === 'TASK').length,
    bugs: issues.filter((i) => i.kind === 'BUG').length,
  };

  const totalEstimate = issues.reduce((sum, i) => sum + (i.estimate || 0), 0);
  const totalLogged = issues.reduce((sum, i) => sum + (i.logged_effort || 0), 0);
  const completionRate = stats.total > 0 ? (stats.done / stats.total) * 100 : 0;

  const statusCards = [
    { label: 'Backlog', value: stats.backlog, color: 'var(--color-status-backlog)' },
    { label: 'Planned', value: stats.planned, color: 'var(--color-status-planned)' },
    { label: 'In Progress', value: stats.doing, color: 'var(--color-status-doing)' },
    { label: 'Blocked', value: stats.blocked, color: 'var(--color-status-blocked)' },
    { label: 'Done', value: stats.done, color: 'var(--color-status-done)' },
  ];

  const kindCards = [
    { label: 'Epics', value: stats.epics, color: 'var(--color-kind-epic)', icon: '🎯' },
    { label: 'Tasks', value: stats.tasks, color: 'var(--color-kind-task)', icon: '📋' },
    { label: 'Bugs', value: stats.bugs, color: 'var(--color-kind-bug)', icon: '🐛' },
  ];

  return (
    <div className="space-y-8 animate-fade-in">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-[var(--color-text-primary)]">Dashboard</h1>
          <p className="text-sm text-[var(--color-text-muted)] mt-1">
            Project overview and metrics
          </p>
        </div>
      </div>

      {/* Key Metrics Row */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        {/* Completion Ring */}
        <Card variant="elevated" className="flex items-center gap-6">
          <ProgressRing
            value={stats.done}
            max={stats.total || 1}
            size={100}
            strokeWidth={8}
            color="var(--color-success)"
            label={`${Math.round(completionRate)}%`}
            sublabel="complete"
          />
          <div>
            <p className="text-sm text-[var(--color-text-muted)]">Overall Progress</p>
            <p className="text-3xl font-bold text-[var(--color-text-primary)]">
              {stats.done}<span className="text-lg text-[var(--color-text-muted)]">/{stats.total}</span>
            </p>
            <p className="text-xs text-[var(--color-text-muted)] mt-1">issues completed</p>
          </div>
        </Card>

        {/* Effort Overview */}
        <Card variant="elevated">
          <p className="text-sm text-[var(--color-text-muted)] mb-2">Effort Tracking</p>
          <div className="flex items-baseline gap-2">
            <span className="text-3xl font-bold text-[var(--color-text-primary)]">{totalLogged}</span>
            <span className="text-lg text-[var(--color-text-muted)]">/ {totalEstimate} pts</span>
          </div>
          <Progress
            value={totalLogged}
            max={totalEstimate || 1}
            variant="accent"
            size="lg"
            showLabel
          />
        </Card>

        {/* Active Work */}
        <Card variant="elevated">
          <p className="text-sm text-[var(--color-text-muted)] mb-2">Active Work</p>
          <p className="text-3xl font-bold text-[var(--color-warning)]">{stats.doing}</p>
          <p className="text-xs text-[var(--color-text-muted)] mt-1">issues in progress</p>
          {stats.blocked > 0 && (
            <div className="mt-3 flex items-center gap-2 text-sm text-[var(--color-error)]">
              <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
              </svg>
              {stats.blocked} blocked
            </div>
          )}
        </Card>
      </div>



      {/* Issue Types */}
      <div>
        <h2 className="text-lg font-semibold text-[var(--color-text-primary)] mb-4">
          Issue Types
        </h2>
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
          {kindCards.map((card) => (
            <Card key={card.label} variant="glass" interactive>
              <div className="flex items-center gap-4">

                <div>
                  <p className="text-sm text-[var(--color-text-muted)]">{card.label}</p>
                  <p
                    className="text-2xl font-bold"
                    style={{ color: card.color }}
                  >
                    {card.value}
                  </p>
                </div>
              </div>
            </Card>
          ))}
        </div>
      </div>

      {/* Status Distribution */}
      <div>
        <h2 className="text-lg font-semibold text-[var(--color-text-primary)] mb-4">
          Status Distribution
        </h2>
        <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-5 gap-4">
          {statusCards.map((card) => (
            <Card
              key={card.label}
              variant="default"
              interactive
              className="text-center"
            >
              <p className="text-sm text-[var(--color-text-muted)] mb-1">{card.label}</p>
              <p
                className="text-3xl font-bold"
                style={{ color: card.color }}
              >
                {card.value}
              </p>
              <div
                className="h-1 mt-3 rounded-full"
                style={{
                  background: card.color,
                  opacity: 0.3,
                }}
              >
                <div
                  className="h-full rounded-full transition-all duration-500"
                  style={{
                    background: card.color,
                    width: `${stats.total ? (card.value / stats.total) * 100 : 0}%`
                  }}
                />
              </div>
            </Card>
          ))}
        </div>
      </div>

      {/* Recent Activity Placeholder */}
      <Card variant="default" className="border-dashed border-2">
        <div className="text-center py-8 text-[var(--color-text-muted)]">
          <svg className="w-12 h-12 mx-auto mb-3 opacity-50" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M12 6v6h4.5m4.5 0a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
          <p className="text-sm">Activity timeline coming soon</p>
        </div>
      </Card>
    </div>
  );
}
