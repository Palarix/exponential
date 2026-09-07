import { useState, useEffect, useMemo } from 'react';
import { fetchCycles, fetchCycleProgress } from '../../api/client';
import type { Issue, Cycle, CycleProgressDay } from '../../api/client';
import { formatShortDate } from '../../utils/format';
import { StatusIcon } from '../ui';
import { UserRound } from "lucide-react";

interface CyclesProps {
  issues: Issue[];
  onIssueClick: (issue: Issue) => void;
  onRefresh: () => Promise<void>;
  selectedCycleId: string | null;
  onCycleSelect: (id: string | null) => void;
}

// --- Timeline Index View ---

function CycleStatusBadge({ status }: { status: string }) {
  const styles: Record<string, string> = {
    current: 'text-[var(--color-accent-primary)]',
    upcoming: 'text-[var(--color-text-secondary)]',
    planned: 'text-[var(--color-text-muted)]',
    completed: 'text-[var(--color-text-muted)]',
  };
  const labels: Record<string, string> = {
    current: 'Current',
    upcoming: 'Upcoming',
    planned: 'Planned',
    completed: 'Completed',
  };
  return (
    <span className={`text-xs font-medium ${styles[status] || styles.completed}`}>
      {labels[status] || status}
    </span>
  );
}

function CycleIcon({ status }: { status: string }) {
  if (status === 'current') {
    return (
      <div className="w-4 h-4 rounded-full border-2 border-[var(--color-accent-primary)] flex items-center justify-center shrink-0">
        <svg className="w-2 h-2 text-[var(--color-accent-primary)]" viewBox="0 0 10 10" fill="currentColor">
          <path d="M2.5 1.5v7l6-3.5z" />
        </svg>
      </div>
    );
  }
  if (status === 'completed') {
    return (
      <div className="w-4 h-4 rounded-full border-2 border-[var(--color-border-default)] flex items-center justify-center shrink-0">
        <svg className="w-2 h-2 text-[var(--color-text-muted)]" viewBox="0 0 10 10" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
          <path d="M2 5.5l2 2 4-4.5" />
        </svg>
      </div>
    );
  }
  const playIcon = (
    <svg className="w-2 h-2 text-[var(--color-text-muted)]" viewBox="0 0 10 10" fill="currentColor">
      <path d="M2.5 1.5v7l6-3.5z" />
    </svg>
  );
  if (status === 'planned') {
    return (
      <div className="w-4 h-4 rounded-full border-2 border-dashed border-[var(--color-border-default)] flex items-center justify-center shrink-0">
        {playIcon}
      </div>
    );
  }
  return (
    <div className="w-4 h-4 rounded-full border-2 border-[var(--color-text-secondary)] flex items-center justify-center shrink-0">
      {playIcon}
    </div>
  );
}

function BreakdownBar({ cycle, issues }: { cycle: Cycle; issues: Issue[] }) {
  const counts = useMemo(() => {
    const cycleIssues = issues.filter(i => {
      if (i.status === 'DONE') return i.cycle_id === cycle.id;
      return i.effective_cycle_id === cycle.id;
    });
    let planned = 0, started = 0, completed = 0;
    for (const i of cycleIssues) {
      if (i.status === 'DONE') completed++;
      else if (i.status === 'DOING' || i.status === 'BLOCKED') started++;
      else planned++;
    }
    return { planned, started, completed, total: cycleIssues.length };
  }, [cycle.id, issues]);

  if (counts.total === 0) return null;

  return (
    <div className="flex items-center gap-2">
      <div className="w-32 h-1.5 rounded-full bg-[var(--color-hover-surface)] overflow-hidden flex">
        {counts.completed > 0 && (
          <div className="h-full bg-[var(--color-success)]" style={{ width: `${(counts.completed / counts.total) * 100}%` }} />
        )}
        {counts.started > 0 && (
          <div className="h-full bg-[var(--color-status-doing)]" style={{ width: `${(counts.started / counts.total) * 100}%` }} />
        )}
        {counts.planned > 0 && (
          <div className="h-full bg-[var(--color-text-muted)] opacity-30" style={{ width: `${(counts.planned / counts.total) * 100}%` }} />
        )}
      </div>
      <span className="text-xs text-[var(--color-text-muted)] tabular-nums whitespace-nowrap">
        {counts.completed}/{counts.total}
      </span>
    </div>
  );
}

function TimelineDot({ color, hollow }: { color: string; hollow: boolean }) {
  if (hollow) {
    return <div className={`w-2 h-2 rounded-full border-[1.5px] ${color}`} />;
  }
  return <div className={`w-2 h-2 rounded-full ${color}`} />;
}

function TimelineSeparator({ date, dotColor, hollow }: { date: string; dotColor: string; hollow: boolean }) {
  return (
    <div className="flex items-center px-4">
      <span className="text-sm text-[var(--color-text-muted)] tabular-nums w-14 shrink-0">{date}</span>
      <div className="w-5 flex justify-center shrink-0">
        <TimelineDot color={dotColor} hollow={hollow} />
      </div>
      <div className="flex-1 border-t border-[var(--color-border-subtle)] ml-2" />
    </div>
  );
}

function TimelineRow({ cycle, issues, lineColor, onClick }: { cycle: Cycle; issues: Issue[]; lineColor: string; onClick: () => void }) {
  return (
    <div className="flex items-stretch">
      <div className="w-14 shrink-0 ml-4" />
      <div className="w-5 flex justify-center shrink-0">
        <div className={`w-px ${lineColor}`} />
      </div>

      <button
        onClick={onClick}
        className="group flex-1 flex items-center gap-4 pl-4 pr-6 py-3 min-w-0 text-left transition-colors hover:bg-[var(--color-surface-1)] rounded-[var(--radius-md)] mr-4 ml-2"
      >
        <CycleIcon status={cycle.status} />

        <span className={`w-20 shrink-0 text-sm font-medium ${cycle.status === 'current' ? 'text-[var(--color-text-primary)]' : 'text-[var(--color-text-secondary)]'}`}>
          Cycle {cycle.number}
        </span>

        <div className="flex-1 min-w-0">
          <BreakdownBar cycle={cycle} issues={issues} />
        </div>

        <div className="flex items-center gap-3 shrink-0">
          <CycleStatusBadge status={cycle.status} />
          <svg className="w-4 h-4 text-[var(--color-text-muted)] opacity-0 group-hover:opacity-100 transition-opacity" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M8.25 4.5l7.5 7.5-7.5 7.5" />
          </svg>
        </div>
      </button>
    </div>
  );
}

function CyclesTimeline({ cycles, issues, onSelect }: { cycles: Cycle[]; issues: Issue[]; onSelect: (id: string) => void }) {
  const sorted = useMemo(() => [...cycles].reverse(), [cycles]);

  const currentIdx = sorted.findIndex(c => c.status === 'current');

  function dotProps(idx: number) {
    const c = sorted[idx];
    const nextInList = idx + 1 < sorted.length ? sorted[idx + 1] : null;
    if (c.status === 'current') return { dotColor: 'bg-[var(--color-accent-primary)]', hollow: false };
    if (nextInList?.status === 'current') return { dotColor: 'border-[var(--color-accent-primary)]', hollow: true };
    if (c.status === 'upcoming' || c.status === 'planned') return { dotColor: 'border-[var(--color-text-muted)]', hollow: true };
    return { dotColor: 'bg-[var(--color-text-muted)]', hollow: false };
  }

  const topBoundary = useMemo(() => {
    if (sorted.length === 0) return '';
    const d = new Date(sorted[0].end + 'T00:00:00');
    d.setDate(d.getDate() + 1);
    return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
  }, [sorted]);

  function lineColor(rowIdx: number) {
    if (currentIdx < 0) return 'bg-[var(--color-border-default)]';
    return rowIdx === currentIdx ? 'bg-[var(--color-accent-primary)]' : 'bg-[var(--color-border-default)]';
  }

  return (
    <div className="h-full overflow-y-auto">
      <div className="max-w-7xl mx-auto">
      <div className="px-6 pt-5 pb-3 flex items-center justify-between">
        <h1 className="text-lg font-semibold text-[var(--color-text-primary)]">Cycles</h1>
      </div>
      <div className="pb-4">
        {sorted.length > 0 && (
          <TimelineSeparator date={formatShortDate(topBoundary)} dotColor={sorted[0].status === 'current' ? 'border-[var(--color-accent-primary)]' : 'border-[var(--color-text-muted)]'} hollow={true} />
        )}
        {sorted.map((c, i) => {
          const dp = dotProps(i);
          return (
            <div key={c.id}>
              <TimelineRow cycle={c} issues={issues} lineColor={lineColor(i)} onClick={() => onSelect(c.id)} />
              <TimelineSeparator date={formatShortDate(c.start)} dotColor={dp.dotColor} hollow={dp.hollow} />
            </div>
          );
        })}
      </div>
      </div>
    </div>
  );
}

// --- Progress Chart ---

function ProgressChart({ cycleId }: { cycleId: string }) {
  const [days, setDays] = useState<CycleProgressDay[]>([]);

  useEffect(() => {
    fetchCycleProgress(cycleId)
      .then(data => setDays(data.days || []))
      .catch(() => {});
  }, [cycleId]);

  if (days.length < 2) return null;

  const initialScope = days[0].scope;
  const remaining = days.map(d => d.scope - d.completed);
  const maxVal = Math.max(initialScope, ...days.map(d => d.scope), 1);
  const w = 220;
  const h = 100;
  const px = 4;
  const py = 6;
  const plotW = w - px * 2;
  const plotH = h - py * 2;

  const toX = (i: number) => px + (i / (days.length - 1)) * plotW;
  const toY = (v: number) => py + plotH - (v / maxVal) * plotH;

  const idealLine = `M${toX(0)},${toY(initialScope)} L${toX(days.length - 1)},${toY(0)}`;

  const scopeLine = days.map((d, i) => `${i === 0 ? 'M' : 'L'}${toX(i)},${toY(d.scope)}`).join(' ');

  const remainingLine = remaining.map((v, i) => `${i === 0 ? 'M' : 'L'}${toX(i)},${toY(v)}`).join(' ');
  const remainingFill = remainingLine + ` L${toX(remaining.length - 1)},${toY(0)} L${toX(0)},${toY(0)} Z`;

  const firstDate = formatShortDate(days[0].date);
  const lastDate = formatShortDate(days[days.length - 1].date);

  return (
    <div>
      <svg viewBox={`0 0 ${w} ${h + 14}`} className="w-full" preserveAspectRatio="none">
        <defs>
          <linearGradient id="remaining-grad" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stopColor="var(--color-accent-primary)" stopOpacity="0.2" />
            <stop offset="100%" stopColor="var(--color-accent-primary)" stopOpacity="0.02" />
          </linearGradient>
        </defs>

        {[0, 0.5, 1].map(frac => (
          <line key={frac} x1={px} x2={w - px} y1={toY(frac * maxVal)} y2={toY(frac * maxVal)} stroke="var(--color-border-subtle)" strokeWidth="0.5" />
        ))}

        <path d={idealLine} fill="none" stroke="var(--color-text-muted)" strokeWidth="1" strokeDasharray="4 3" opacity="0.5" />

        <path d={scopeLine} fill="none" stroke="var(--color-text-muted)" strokeWidth="1" strokeDasharray="2 2" opacity="0.4" />

        <path d={remainingFill} fill="url(#remaining-grad)" />
        <path d={remainingLine} fill="none" stroke="var(--color-accent-primary)" strokeWidth="2" />

        {remaining.length > 0 && (
          <circle cx={toX(remaining.length - 1)} cy={toY(remaining[remaining.length - 1])} r="2.5" fill="var(--color-accent-primary)" />
        )}

        <text x={px} y={h + 12} fontSize="8" fill="var(--color-text-muted)" fontFamily="var(--font-sans)">{firstDate}</text>
        <text x={w - px} y={h + 12} fontSize="8" fill="var(--color-text-muted)" fontFamily="var(--font-sans)" textAnchor="end">{lastDate}</text>
      </svg>

      <div className="flex items-center gap-3 mt-1">
        <div className="flex items-center gap-1">
          <div className="w-3 h-0.5 bg-[var(--color-accent-primary)]" />
          <span className="text-xs text-[var(--color-text-muted)]">Remaining</span>
        </div>
        <div className="flex items-center gap-1">
          <div className="w-3 h-0.5 border-t border-dashed border-[var(--color-text-muted)] opacity-50" />
          <span className="text-xs text-[var(--color-text-muted)]">Ideal</span>
        </div>
        <div className="flex items-center gap-1">
          <div className="w-3 h-0.5 border-t border-dotted border-[var(--color-text-muted)] opacity-40" />
          <span className="text-xs text-[var(--color-text-muted)]">Scope</span>
        </div>
      </div>
    </div>
  );
}

// --- Detail View ---

function CycleDetail({ cycle, issues, onIssueClick, onBack }: {
  cycle: Cycle;
  issues: Issue[];
  onIssueClick: (issue: Issue) => void;
  onBack: () => void;
}) {
  const cycleIssues = useMemo(() => {
    return issues.filter(i => {
      if (i.status === 'DONE') return i.cycle_id === cycle.id;
      return i.effective_cycle_id === cycle.id;
    });
  }, [issues, cycle.id]);

  const byStatus = useMemo(() => {
    const groups: Record<string, Issue[]> = {};
    for (const issue of cycleIssues) {
      const s = issue.status;
      if (!groups[s]) groups[s] = [];
      groups[s].push(issue);
    }
    return groups;
  }, [cycleIssues]);

  const statusOrder = ['DOING', 'PLANNED', 'BACKLOG', 'BLOCKED', 'DONE'];
  const scopeCount = cycleIssues.length;
  const startedCount = (byStatus['DOING']?.length || 0) + (byStatus['BLOCKED']?.length || 0);
  const completedCount = byStatus['DONE']?.length || 0;
  const pct = scopeCount > 0 ? Math.round((completedCount / scopeCount) * 100) : 0;

  const assigneeBreakdown = useMemo(() => {
    const map = new Map<string, { total: number; done: number }>();
    for (const issue of cycleIssues) {
      const name = issue.assignee ? issue.assignee.split(' <')[0] : 'Unassigned';
      const entry = map.get(name) || { total: 0, done: 0 };
      entry.total++;
      if (issue.status === 'DONE') entry.done++;
      map.set(name, entry);
    }
    return Array.from(map.entries())
      .map(([name, { total, done }]) => ({ name, total, done }))
      .sort((a, b) => b.total - a.total);
  }, [cycleIssues]);

  return (
    <div className="flex h-full">
      {/* Issue list */}
      <div className="flex-1 overflow-y-auto">
        <div className="px-6 pt-5 pb-3">
          <button
            onClick={onBack}
            className="flex items-center gap-1 text-xs text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] transition-colors mb-2"
          >
            <svg className="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M15.75 19.5L8.25 12l7.5-7.5" />
            </svg>
            Cycles
          </button>
          <h2 className="text-lg font-semibold text-[var(--color-text-primary)]">
            Cycle {cycle.number}
          </h2>
          <p className="text-xs text-[var(--color-text-muted)] mt-0.5">
            {formatShortDate(cycle.start)} &rarr; {formatShortDate(cycle.end)}
          </p>
        </div>

        {cycleIssues.length === 0 ? (
          <div className="px-6 py-12 text-center text-sm text-[var(--color-text-muted)]">
            No issues in this cycle.
          </div>
        ) : (
          <div className="px-4">
            {statusOrder.map(status => {
              const group = byStatus[status];
              if (!group?.length) return null;
              return (
                <div key={status} className="mb-4">
                  <div className="flex items-center gap-2 px-2 py-1.5 text-xs text-[var(--color-text-muted)] uppercase tracking-wider">
                    <StatusIcon status={status} size={14} />
                    <span>{status.replace('_', ' ')}</span>
                    <span className="text-[var(--color-text-muted)]">{group.length}</span>
                  </div>
                  {group.map(issue => (
                    <button
                      key={issue.id}
                      onClick={() => onIssueClick(issue)}
                      className="w-full flex items-center gap-3 px-2 py-2 rounded-[var(--radius-md)] hover:bg-[var(--color-hover-surface)] text-left transition-colors"
                    >
                      <StatusIcon status={issue.status} size={16} isInferred={issue.is_inferred} />
                      <span className="text-sm text-[var(--color-text-primary)] truncate flex-1">
                        {issue.title}
                      </span>
                      <span className="text-xs text-[var(--color-text-muted)] shrink-0">
                        {issue.id.slice(-6)}
                      </span>
                    </button>
                  ))}
                </div>
              );
            })}
          </div>
        )}
      </div>

      {/* Stats sidebar */}
      <div className="w-72 shrink-0 border-l border-[var(--color-border-subtle)] overflow-y-auto">
        {/* Header badges */}
        <div className="flex items-center gap-2 px-4 pt-4 pb-2">
          <span className={`px-2 py-0.5 rounded-full text-xs font-medium ${
            cycle.status === 'current' ? 'bg-[var(--color-accent-primary)]/15 text-[var(--color-accent-primary)]' :
            cycle.status === 'upcoming' ? 'bg-[var(--color-hover-surface)] text-[var(--color-text-secondary)]' :
            cycle.status === 'completed' ? 'bg-[var(--color-success)]/15 text-[var(--color-success)]' :
            'bg-[var(--color-hover-surface)] text-[var(--color-text-muted)]'
          } capitalize`}>
            {cycle.status}
          </span>
          <span className="px-2 py-0.5 rounded-full text-xs text-[var(--color-text-muted)] bg-[var(--color-hover-surface)]">
            {formatShortDate(cycle.start)} &rarr; {formatShortDate(cycle.end)}
          </span>
        </div>

        {/* Title */}
        <div className="flex items-center gap-2 px-4 pb-4">
          <CycleIcon status={cycle.status} />
          <h3 className="text-base font-semibold text-[var(--color-text-primary)]">
            Cycle {cycle.number}
          </h3>
        </div>

        <div className="border-t border-[var(--color-border-subtle)]" />

        {/* Progress stats */}
        <div className="px-4 py-3">
          <div className="text-xs font-medium text-[var(--color-text-muted)] mb-2">Progress</div>
          <div className="flex items-start justify-between gap-2">
            <div>
              <div className="flex items-center gap-1">
                <div className="w-2 h-2 rounded-sm bg-[var(--color-text-muted)] opacity-40" />
                <span className="text-xs text-[var(--color-text-muted)]">Scope</span>
              </div>
              <span className="text-sm font-semibold text-[var(--color-text-primary)] tabular-nums">{scopeCount}</span>
            </div>
            <div>
              <div className="flex items-center gap-1">
                <div className="w-2 h-2 rounded-sm bg-[var(--color-status-doing)]" />
                <span className="text-xs text-[var(--color-text-muted)]">Started</span>
              </div>
              <div className="flex items-baseline gap-1">
                <span className="text-sm font-semibold text-[var(--color-text-primary)] tabular-nums">{startedCount}</span>
                {scopeCount > 0 && <span className="text-xs text-[var(--color-text-muted)]">{Math.round((startedCount / scopeCount) * 100)}%</span>}
              </div>
            </div>
            <div>
              <div className="flex items-center gap-1">
                <div className="w-2 h-2 rounded-sm bg-[var(--color-success)]" />
                <span className="text-xs text-[var(--color-text-muted)]">Done</span>
              </div>
              <div className="flex items-baseline gap-1">
                <span className="text-sm font-semibold text-[var(--color-text-primary)] tabular-nums">{completedCount}</span>
                {scopeCount > 0 && <span className="text-xs text-[var(--color-text-muted)]">{pct}%</span>}
              </div>
            </div>
          </div>
        </div>

        {/* Chart */}
        <div className="px-4 pb-3">
          <ProgressChart cycleId={cycle.id} />
        </div>

        <div className="border-t border-[var(--color-border-subtle)]" />

        {/* Assignee breakdown */}
        <div className="px-4 py-3">
          <div className="text-xs font-medium text-[var(--color-text-muted)] mb-2">Assignees</div>
          <div className="space-y-2">
            {assigneeBreakdown.map(a => (
              <div key={a.name} className="flex items-center gap-2">
                {a.name === 'Unassigned' ? (
                  <div className="w-5 h-5 rounded-full bg-[var(--color-hover-surface)] flex items-center justify-center shrink-0">
                    <UserRound className="w-3 h-3 text-[var(--color-text-muted)]" />
                  </div>
                ) : (
                  <div className="w-5 h-5 rounded-full bg-[var(--color-accent-primary)] flex items-center justify-center shrink-0 text-xs font-medium text-white">
                    {a.name.charAt(0).toUpperCase()}
                  </div>
                )}
                <span className="text-xs text-[var(--color-text-secondary)] flex-1 truncate">{a.name}</span>
                <span className="text-xs text-[var(--color-text-muted)] tabular-nums">{a.done}/{a.total}</span>
              </div>
            ))}
          </div>
        </div>

        <div className="border-t border-[var(--color-border-subtle)]" />

        {/* Status breakdown */}
        <div className="px-4 py-3">
          <div className="text-xs font-medium text-[var(--color-text-muted)] mb-2">Status</div>
          <div className="space-y-1.5">
            {statusOrder.map(status => {
              const count = byStatus[status]?.length || 0;
              if (count === 0) return null;
              return (
                <div key={status} className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <StatusIcon status={status} size={14} />
                    <span className="text-xs text-[var(--color-text-secondary)] capitalize">
                      {status.toLowerCase().replace('_', ' ')}
                    </span>
                  </div>
                  <span className="text-xs text-[var(--color-text-muted)] tabular-nums">{count}</span>
                </div>
              );
            })}
          </div>
        </div>
      </div>
    </div>
  );
}

// --- Root ---

export default function Cycles({ issues, onIssueClick, selectedCycleId, onCycleSelect }: CyclesProps) {
  const [cycles, setCycles] = useState<Cycle[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchCycles()
      .then(data => {
        if (data.enabled && data.cycles) setCycles(data.cycles);
      })
      .catch(() => {})
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    fetchCycles()
      .then(data => {
        if (data.enabled && data.cycles) setCycles(data.cycles);
      })
      .catch(() => {});
  }, [issues]);

  const selectedCycle = selectedCycleId ? cycles.find(c => c.id === selectedCycleId) ?? null : null;

  if (loading) {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="w-5 h-5 border-2 border-[var(--color-text-muted)] border-t-transparent rounded-full animate-spin" />
      </div>
    );
  }

  if (cycles.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center h-full gap-4 text-center">
        <svg className="w-12 h-12 text-[var(--color-text-muted)] opacity-40" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1}>
          <path strokeLinecap="round" strokeLinejoin="round" d="M16.023 9.348h4.992v-.001M2.985 19.644v-4.992m0 0h4.992m-4.993 0l3.181 3.183a8.25 8.25 0 0013.803-3.7M4.031 9.865a8.25 8.25 0 0113.803-3.7l3.181 3.182M21.016 5.176v4.993" />
        </svg>
        <div>
          <h3 className="text-sm font-medium text-[var(--color-text-primary)]">No cycles configured</h3>
          <p className="text-xs text-[var(--color-text-muted)] mt-1">
            Run <code className="px-1.5 py-0.5 bg-[var(--color-surface-1)] rounded text-xs font-mono">xpo cycle init</code> to set up iterations.
          </p>
        </div>
      </div>
    );
  }

  if (selectedCycle) {
    return (
      <CycleDetail
        cycle={selectedCycle}
        issues={issues}
        onIssueClick={onIssueClick}
        onBack={() => onCycleSelect(null)}
      />
    );
  }

  return <CyclesTimeline cycles={cycles} issues={issues} onSelect={onCycleSelect} />;
}
