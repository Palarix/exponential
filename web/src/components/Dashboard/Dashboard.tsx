import { useContext, useEffect, useMemo, useState } from "react";
import { fetchActivity, fetchMetrics, type ActivityEvent, type AttentionItem, type Issue, type PulseMetrics } from "../../api/client";
import { Avatar, Card, CopyableId, EmptyState, LabelColorsContext, PriorityIcon, StatusIcon, SubProgress } from "../ui";
// @ts-expect-error kept for future dashboard personalization
import { formatTriage } from "../../utils/format"; // eslint-disable-line
import { formatDuration } from "../../utils/format";
import { Section, SectionIcon, PulseCard, SECTION_ICONS } from "./Section";
import { ArrowUp, ArrowDown } from "lucide-react";
import { DailyVelocityChart, CumulativeChart } from "./charts";
import ActivityFeed from "./ActivityFeed";
import DistributionSection, { type DistFilter, type DistRow } from "./DistributionSection";

function attentionReason(item: AttentionItem): { label: string; icon: React.ReactNode; cls: string } {
  switch (item.kind) {
    case "blocker":
      return {
        label: `Blocked ${item.age_days ?? 0}d`,
        cls: "text-[var(--color-error)]",
        icon: <path strokeLinecap="round" strokeLinejoin="round" d="M18.364 18.364A9 9 0 005.636 5.636m12.728 12.728A9 9 0 015.636 5.636m12.728 12.728L5.636 5.636" />,
      };
    case "stale_wip":
      return {
        label: `Stale ${item.age_days ?? 0}d`,
        cls: "text-[var(--color-warning)]",
        icon: <path strokeLinecap="round" strokeLinejoin="round" d="M12 6v6h4.5m4.5 0a9 9 0 11-18 0 9 9 0 0118 0z" />,
      };
    case "high_priority":
      return {
        label: "Urgent",
        cls: "text-[rgb(244,124,66)]",
        icon: null,
      };
  }
}

const PRIORITY_LABELS: { value: number; label: string }[] = [
  { value: 1, label: "Urgent" },
  { value: 2, label: "High" },
  { value: 3, label: "Medium" },
  { value: 4, label: "Low" },
  { value: 0, label: "No priority" },
];

interface DashboardProps {
  issues: Issue[];
  onIssueClick?: (issue: Issue) => void;
  onNewIssue?: () => void;
}

export default function Dashboard({ issues, onIssueClick, onNewIssue }: DashboardProps) {
  const labelColors = useContext(LabelColorsContext);
  const [statusFilter, setStatusFilter] = useState<DistFilter>("all");
  const [labelFilter, setLabelFilter] = useState<DistFilter>("all");
  const [assigneeFilter, setAssigneeFilter] = useState<DistFilter>("all");
  const [priorityFilter, setPriorityFilter] = useState<DistFilter>("all");
  const [metrics, setMetrics] = useState<PulseMetrics | null>(null);
  const [activity, setActivity] = useState<ActivityEvent[]>([]);

  useEffect(() => {
    const load = () => {
      fetchMetrics().then(setMetrics).catch(() => {});
      fetchActivity().then(setActivity).catch(() => {});
    };
    load();
    const interval = setInterval(load, 30_000);
    const onFocus = () => load();
    window.addEventListener("focus", onFocus);
    return () => { clearInterval(interval); window.removeEventListener("focus", onFocus); };
  }, [issues]);

  const stats = {
    total: issues.length,
    backlog: issues.filter((i) => i.status === "BACKLOG").length,
    planned: issues.filter((i) => i.status === "PLANNED").length,
    doing: issues.filter((i) => i.status === "DOING").length,
    blocked: issues.filter((i) => i.status === "BLOCKED").length,
    done: issues.filter((i) => i.status === "DONE").length,
  };

  const STATUS_ORDER_MAP: Record<string, { label: string; idx: number }> = {
    BACKLOG: { label: "Backlog", idx: 0 },
    PLANNED: { label: "Planned", idx: 1 },
    DOING: { label: "In Progress", idx: 2 },
    BLOCKED: { label: "Blocked", idx: 3 },
    DONE: { label: "Done", idx: 4 },
  };

  const allStatuses = ["BACKLOG", "PLANNED", "DOING", "BLOCKED", "DONE"];

  const statusDistribution = useMemo(() => {
    const filtered = statusFilter === "active" ? issues.filter((i) => i.status !== "DONE") : issues;
    const counts = new Map<string, number>();
    for (const i of filtered) counts.set(i.status, (counts.get(i.status) || 0) + 1);
    const total = filtered.length;
    const visibleStatuses = statusFilter === "active" ? allStatuses.filter(s => s !== "DONE") : allStatuses;
    const rows: DistRow[] = visibleStatuses.map((status) => {
      const count = counts.get(status) || 0;
      return {
        key: status,
        label: (
          <span className="flex items-center gap-2">
            <StatusIcon status={status} size={14} />
            <span className="text-xs">{STATUS_ORDER_MAP[status]?.label || status}</span>
          </span>
        ),
        count,
        pct: total > 0 ? (count / total) * 100 : 0,
        barWidth: total > 0 ? (count / total) * 100 : 0,
      };
    });
    return { total, rows };
  }, [issues, statusFilter]);

  const labelDistribution = useMemo(() => {
    const filtered = labelFilter === "active" ? issues.filter((i) => i.status !== "DONE") : issues;
    const counts = new Map<string, number>();
    for (const i of filtered) for (const l of i.labels || []) counts.set(l, (counts.get(l) || 0) + 1);
    const total = filtered.length;

    const rows: DistRow[] = Array.from(counts.entries())
      .sort((a, b) => b[1] - a[1])
      .map(([label, count]) => ({
        key: label,
        label: (
          <span className="flex items-center gap-2">
            <span className="w-2 h-2 rounded-full shrink-0" style={{ background: labelColors[label] || labelColors[label.toLowerCase()] || "var(--color-text-muted)" }} />
            <span className="text-xs truncate">{label.charAt(0).toUpperCase() + label.slice(1)}</span>
          </span>
        ),
        count,
        pct: total > 0 ? (count / total) * 100 : 0,
        barWidth: total > 0 ? (count / total) * 100 : 0,
      }));
    return { total, rows };
  }, [issues, labelFilter]);

  const assigneeDistribution = useMemo(() => {
    const filtered = assigneeFilter === "active" ? issues.filter((i) => i.status !== "DONE") : issues;
    const counts = new Map<string, number>();
    let unassigned = 0;
    for (const i of filtered) {
      if (i.assignee) counts.set(i.assignee, (counts.get(i.assignee) || 0) + 1);
      else unassigned++;
    }
    const total = filtered.length;
    const namedRows: DistRow[] = Array.from(counts.entries())
      .sort((a, b) => b[1] - a[1])
      .map(([assignee, count]) => ({
        key: assignee,
        label: (
          <div className="flex items-center gap-2 min-w-0">
            <Avatar name={assignee} size="xs" />
            <span className="text-sm text-[var(--color-text-primary)] truncate">{assignee.split(" <")[0]}</span>
          </div>
        ),
        count,
        pct: total > 0 ? (count / total) * 100 : 0,
        barWidth: total > 0 ? (count / total) * 100 : 0,
      }));
    const unassignedRow: DistRow[] = unassigned > 0
      ? [{ key: "__unassigned__", label: <span className="text-sm text-[var(--color-text-muted)] italic">Unassigned</span>, count: unassigned, pct: total > 0 ? (unassigned / total) * 100 : 0, barWidth: total > 0 ? (unassigned / total) * 100 : 0 }]
      : [];
    return { total, rows: [...namedRows, ...unassignedRow] };
  }, [issues, assigneeFilter]);

  const priorityDistribution = useMemo(() => {
    const filtered = priorityFilter === "active" ? issues.filter((i) => i.status !== "DONE") : issues;
    const counts = new Map<number, number>();
    for (const i of filtered) { const p = i.priority || 0; counts.set(p, (counts.get(p) || 0) + 1); }
    const total = filtered.length;

    const rows: DistRow[] = PRIORITY_LABELS
      .filter((p) => (counts.get(p.value) || 0) > 0)
      .map((p) => {
        const count = counts.get(p.value) || 0;
        return {
          key: String(p.value),
          label: (
            <div className="flex items-center gap-2">
              <PriorityIcon priority={p.value} size={14} />
              <span className="text-sm text-[var(--color-text-primary)]">{p.label}</span>
            </div>
          ),
          count,
          pct: total > 0 ? (count / total) * 100 : 0,
          barWidth: total > 0 ? (count / total) * 100 : 0,
        };
      });
    return { total, rows };
  }, [issues, priorityFilter]);

  // @ts-expect-error kept for future dashboard personalization
  const statusCards = [
    { label: "Backlog", status: "BACKLOG", value: stats.backlog, color: "var(--color-status-backlog)" },
    { label: "Planned", status: "PLANNED", value: stats.planned, color: "var(--color-status-planned)" },
    { label: "In Progress", status: "DOING", value: stats.doing, color: "var(--color-status-doing)" },
    { label: "Blocked", status: "BLOCKED", value: stats.blocked, color: "var(--color-status-blocked)" },
    { label: "Done", status: "DONE", value: stats.done, color: "var(--color-status-done)" },
  ];

  if (issues.length === 0) {
    return (
      <EmptyState
        title="Welcome to Exponential"
        description="Track issues, plan sprints, and ship software — all from your terminal and this board. Create your first issue to get started."
        icon={
          <svg width="160" height="120" viewBox="0 0 160 120" fill="none">
            <rect x="20" y="15" width="120" height="80" rx="8" stroke="var(--color-text-muted)" strokeWidth="1.5" strokeDasharray="4 3" />
            <rect x="32" y="32" width="30" height="4" rx="2" fill="var(--color-text-muted)" />
            <rect x="32" y="42" width="50" height="3" rx="1.5" fill="var(--color-text-muted)" opacity="0.5" />
            <rect x="32" y="50" width="40" height="3" rx="1.5" fill="var(--color-text-muted)" opacity="0.5" />
            <rect x="32" y="64" width="30" height="4" rx="2" fill="var(--color-text-muted)" />
            <rect x="32" y="74" width="55" height="3" rx="1.5" fill="var(--color-text-muted)" opacity="0.5" />
            <circle cx="120" cy="50" r="14" stroke="var(--color-text-muted)" strokeWidth="1.5" />
            <path d="M116 50h8M120 46v8" stroke="var(--color-text-muted)" strokeWidth="1.5" strokeLinecap="round" />
          </svg>
        }
        actionLabel="Create your first issue"
        onAction={onNewIssue}
      />
    );
  }

  return (
    <div className="h-full flex flex-col">
      <div className="flex items-center gap-3 px-5 h-11 border-b border-[var(--color-border-subtle)] shrink-0">
        <span className="text-sm font-medium text-[var(--color-text-primary)]">Overview</span>
        <span className="ml-auto text-xs text-[var(--color-text-muted)] tabular-nums">
          {issues.length} issue{issues.length === 1 ? "" : "s"}
        </span>
      </div>

      <div className="flex-1 overflow-y-auto">
        <div className="max-w-7xl mx-auto space-y-3 py-3">
          {/* Pulse */}
          <Section title="Pulse" icon={<SectionIcon d={SECTION_ICONS.pulse} />} collapsible storageKey="exponential-dashboard-pulse-open">
            {/* Charts row: 50/50 */}
            <div className="px-5 py-3 grid grid-cols-2 gap-3">
              <Card variant="elevated" padding="sm" className="flex flex-col min-h-48">
                <div className="flex items-center justify-between mb-2">
                  <p className="text-xs uppercase tracking-wider text-[var(--color-text-muted)]">Daily Velocity</p>
                  <p className="text-xs text-[var(--color-text-muted)] tabular-nums">last 14 days</p>
                </div>
                <div className="flex-1 min-h-0">
                  {metrics && metrics.velocity.daily_buckets && metrics.velocity.daily_buckets.length > 0
                    ? <DailyVelocityChart buckets={metrics.velocity.daily_buckets} />
                    : <p className="text-2xl font-semibold text-[var(--color-text-primary)] leading-none">—</p>
                  }
                </div>
              </Card>
              {metrics && metrics.trends.weekly.length > 0 && (
                <Card variant="elevated" padding="sm" className="flex flex-col min-h-48">
                  <div className="flex items-center justify-between mb-2">
                    <p className="text-xs uppercase tracking-wider text-[var(--color-text-muted)]">Created vs Completed</p>
                    <div className="flex items-center gap-2 text-xs text-[var(--color-text-muted)]">
                      <span className="flex items-center gap-1"><span className="w-2 h-2 rounded-sm bg-[var(--color-text-muted)]" />created</span>
                      <span className="flex items-center gap-1"><span className="w-2 h-2 rounded-sm bg-[var(--color-accent-primary)]" />completed</span>
                    </div>
                  </div>
                  <div className="flex-1 min-h-0">
                    <CumulativeChart weekly={metrics.trends.weekly} />
                  </div>
                </Card>
              )}
            </div>

            {/* Stats row: Velocity + Cycle Time + Lead Time + Staleness */}
            <div className="px-5 pb-3 grid grid-cols-4 gap-3">
              <PulseCard title="Velocity">
                <table className="w-full text-center tabular-nums border-collapse">
                  <thead>
                    <tr className="text-xs uppercase tracking-wider text-[var(--color-text-muted)]">
                      <th className="font-normal py-1.5 border-r border-[var(--color-border-default)]">Last Week</th>
                      <th className="font-normal py-1.5">Current</th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr>
                      <td className="py-2 text-lg font-semibold text-[var(--color-text-primary)] border-r border-[var(--color-border-default)]">{metrics ? metrics.velocity.last_7d_points : "—"} <span className="text-xs font-normal text-[var(--color-text-muted)]">pts</span></td>
                      <td className="py-2 text-lg font-semibold text-[var(--color-text-primary)]">{metrics ? metrics.velocity.current_week_points : "—"} <span className="text-xs font-normal text-[var(--color-text-muted)]">pts</span></td>
                    </tr>
                  </tbody>
                </table>
                <p className="text-xs text-[var(--color-text-muted)] mt-auto pt-3 flex items-center gap-1">
                  {metrics && metrics.velocity.delta !== 0 ? (
                    <>
                      <span className={`inline-flex items-center ${metrics.velocity.delta > 0 ? "text-[var(--color-success)]" : "text-[var(--color-warning)]"}`}>
                        {metrics.velocity.delta > 0 ? <ArrowUp size={12} strokeWidth={2.5} /> : <ArrowDown size={12} strokeWidth={2.5} />}
                      </span>
                      <span>{metrics.velocity.delta > 0 ? "+" : ""}{metrics.velocity.delta}pts vs prior</span>
                    </>
                  ) : metrics && metrics.velocity.prior_7d_points > 0 ? (
                    <span>no change vs prior week</span>
                  ) : (
                    <span>no prior week data</span>
                  )}
                </p>
              </PulseCard>
              <PulseCard title="Cycle Time">
                {metrics && metrics.flow.cycle_count > 0 ? (
                  <>
                    <table className="w-full text-center tabular-nums border-collapse">
                      <thead>
                        <tr className="text-xs uppercase tracking-wider text-[var(--color-text-muted)]">
                          <th className="font-normal py-1.5 border-r border-[var(--color-border-default)]">Min</th>
                          <th className="font-normal py-1.5 border-r border-[var(--color-border-default)]">Median</th>
                          <th className="font-normal py-1.5 border-r border-[var(--color-border-default)]">P90</th>
                          <th className="font-normal py-1.5">Max</th>
                        </tr>
                      </thead>
                      <tbody>
                        <tr>
                          <td className="py-2 text-sm text-[var(--color-text-muted)] border-r border-[var(--color-border-default)]">{formatDuration(metrics.flow.cycle_time_min_hrs)}</td>
                          <td className="py-2 text-lg font-semibold text-[var(--color-text-primary)] border-r border-[var(--color-border-default)]">{formatDuration(metrics.flow.cycle_time_hrs)}</td>
                          <td className="py-2 text-sm text-[var(--color-text-muted)] border-r border-[var(--color-border-default)]">{formatDuration(metrics.flow.cycle_time_p90_hrs)}</td>
                          <td className="py-2 text-sm text-[var(--color-text-muted)]">{formatDuration(metrics.flow.cycle_time_max_hrs)}</td>
                        </tr>
                      </tbody>
                    </table>
                    <p className="text-xs text-[var(--color-text-muted)] mt-auto pt-3">Based on start → done · {metrics.flow.cycle_count} issues</p>
                  </>
                ) : (
                  <p className="text-2xl font-semibold text-[var(--color-text-primary)] leading-none">—</p>
                )}
              </PulseCard>
              <PulseCard title="Lead Time">
                {metrics && metrics.flow.lead_count > 0 ? (
                  <>
                    <table className="w-full text-center tabular-nums border-collapse">
                      <thead>
                        <tr className="text-xs uppercase tracking-wider text-[var(--color-text-muted)]">
                          <th className="font-normal py-1.5 border-r border-[var(--color-border-default)]">Min</th>
                          <th className="font-normal py-1.5 border-r border-[var(--color-border-default)]">Median</th>
                          <th className="font-normal py-1.5 border-r border-[var(--color-border-default)]">P90</th>
                          <th className="font-normal py-1.5">Max</th>
                        </tr>
                      </thead>
                      <tbody>
                        <tr>
                          <td className="py-2 text-sm text-[var(--color-text-muted)] border-r border-[var(--color-border-default)]">{formatDuration(metrics.flow.lead_time_min_hrs)}</td>
                          <td className="py-2 text-lg font-semibold text-[var(--color-text-primary)] border-r border-[var(--color-border-default)]">{formatDuration(metrics.flow.lead_time_hrs)}</td>
                          <td className="py-2 text-sm text-[var(--color-text-muted)] border-r border-[var(--color-border-default)]">{formatDuration(metrics.flow.lead_time_p90_hrs)}</td>
                          <td className="py-2 text-sm text-[var(--color-text-muted)]">{formatDuration(metrics.flow.lead_time_max_hrs)}</td>
                        </tr>
                      </tbody>
                    </table>
                    <p className="text-xs text-[var(--color-text-muted)] mt-auto pt-3">Based on created → done · {metrics.flow.lead_count} issues</p>
                  </>
                ) : (
                  <p className="text-2xl font-semibold text-[var(--color-text-primary)] leading-none">—</p>
                )}
              </PulseCard>
              <PulseCard title="Staleness">
                {(() => {
                  const s = metrics?.flow.staleness;
                  const total = metrics?.flow.staleness_total ?? 0;
                  if (!metrics || total === 0) return (
                    <>
                      <p className="text-2xl font-semibold text-[var(--color-text-primary)] leading-none">0</p>
                      <p className="text-xs text-[var(--color-text-muted)] mt-2">no open issues</p>
                    </>
                  );
                  const bars = [
                    { label: "<1d", count: s!.under_1d, cls: "bg-[var(--color-success)]" },
                    { label: "<3d", count: s!.under_3d, cls: "bg-[var(--color-success)]" },
                    { label: "<7d", count: s!.under_7d, cls: "bg-[var(--color-text-secondary)]" },
                    { label: "<14d", count: s!.under_14d, cls: "bg-[var(--color-warning)]" },
                    { label: "<30d", count: s!.under_30d, cls: "bg-[var(--color-warning)]" },
                    { label: ">30d", count: s!.over_30d, cls: "bg-[var(--color-error)]" },
                  ];
                  const max = Math.max(1, ...bars.map(b => b.count));
                  return (
                    <>
                      <div>
                        <div className="flex items-end gap-1 h-10">
                          {bars.map(b => (
                            <div key={b.label} className="flex-1 flex flex-col items-center justify-end h-full">
                              <span className="text-xs text-[var(--color-text-muted)] tabular-nums leading-none mb-1">{b.count > 0 ? b.count : ""}</span>
                              <div className={`w-3 rounded-t-sm ${b.cls} transition-all duration-500`} style={{ height: `${(b.count / max) * 100}%`, minHeight: b.count > 0 ? 2 : 0 }} />
                            </div>
                          ))}
                        </div>
                        <div className="flex gap-1 mt-1.5">
                          {bars.map(b => <span key={b.label} className="flex-1 text-center text-xs text-[var(--color-text-muted)] tabular-nums">{b.label}</span>)}
                        </div>
                      </div>
                      <p className="text-xs text-[var(--color-text-muted)] mt-auto pt-3 tabular-nums">Based on {total} open issues</p>
                    </>
                  );
                })()}
              </PulseCard>
            </div>

          </Section>

          {/* Composition */}
          <Section title="Composition" icon={<SectionIcon d={SECTION_ICONS.composition} />} collapsible defaultOpen={false} storageKey="exponential-dashboard-composition-open">
            <div className="px-5 py-4">
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
                <DistributionSection title="By status" filter={statusFilter} onFilterChange={setStatusFilter} rows={statusDistribution.rows} total={statusDistribution.total} emptyHasIssues="No issues." hideBars />
                <DistributionSection title="By label" filter={labelFilter} onFilterChange={setLabelFilter} rows={labelDistribution.rows} total={labelDistribution.total} emptyHasIssues="No labels on the filtered issues." />
                <DistributionSection title="By assignee" filter={assigneeFilter} onFilterChange={setAssigneeFilter} rows={assigneeDistribution.rows} total={assigneeDistribution.total} emptyHasIssues="No assignees on the filtered issues." />
                <DistributionSection title="By priority" filter={priorityFilter} onFilterChange={setPriorityFilter} rows={priorityDistribution.rows} total={priorityDistribution.total} emptyHasIssues="No priorities set on the filtered issues." />
              </div>
            </div>
          </Section>

          {/* Trends — hidden, superseded by Pulse charts */}

          {/* Needs Attention */}
          {metrics && metrics.attention.length > 0 && (
            <Section title="Needs Attention" icon={<SectionIcon d={SECTION_ICONS.attention} />} count={metrics.attention.length} collapsible storageKey="exponential-dashboard-attention-open">
              <div className="px-5 py-3 grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-2">
                {metrics.attention.map((item) => {
                  const issue = issues.find((i) => i.id === item.issue_id);
                  const reason = attentionReason(item);
                  return (
                    <button
                      key={item.issue_id}
                      onClick={() => issue && onIssueClick?.(issue)}
                      className="flex items-center gap-2 px-3 py-2.5 rounded-[var(--radius-md)] bg-[var(--color-surface-2)] border border-[var(--color-border-subtle)] hover:border-[var(--color-border-default)] transition-colors text-left"
                    >
                      <StatusIcon status={issue?.status || "BACKLOG"} size={14} isInferred={issue?.is_inferred} />
                      <span className="font-mono text-xs text-[var(--color-text-muted)] shrink-0">{item.issue_id}</span>
                      <span className="text-sm text-[var(--color-text-primary)] truncate flex-1">{item.title}</span>
                      {issue?.assignee && <Avatar name={issue.assignee} size="xs" />}
                      <span className={`shrink-0 ${reason.cls}`} title={reason.label}>
                        {reason.icon ? (
                          <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                            {reason.icon}
                          </svg>
                        ) : (
                          <PriorityIcon priority={item.priority || 1} size={16} />
                        )}
                      </span>
                    </button>
                  );
                })}
              </div>
            </Section>
          )}

          {/* Workload */}
          <Section title="Workload" icon={<SectionIcon d={SECTION_ICONS.workload} />} count={metrics?.workload.length} collapsible storageKey="exponential-dashboard-workload-open">
            {!metrics || metrics.workload.length === 0 ? (
              <div className="px-5 py-10 flex items-center justify-center"><p className="text-sm text-[var(--color-text-muted)] text-center">No assigned work.</p></div>
            ) : (
              <div className="py-2">
                <div className="grid grid-cols-[1fr_auto_auto_auto] gap-x-4 px-5 py-1 text-xs uppercase tracking-wider text-[var(--color-text-muted)]">
                  <span>Person</span><span className="text-right w-10">WIP</span><span className="text-right w-10">Pts</span><span className="text-right w-12">Blocked</span>
                </div>
                {metrics.workload.map((w) => (
                  <div key={w.assignee} className="grid grid-cols-[1fr_auto_auto_auto] gap-x-4 items-center px-5 py-2 hover:bg-[var(--color-hover-surface)] transition-colors" title={w.last_completed ? `Last completed ${w.last_completed}` : "No completed issues yet"}>
                    <div className="flex items-center gap-2 min-w-0">
                      <Avatar name={w.assignee} size="xs" />
                      <span className="text-sm text-[var(--color-text-primary)] truncate">{w.assignee.split(" <")[0]}</span>
                    </div>
                    <span className="w-10 text-right text-sm tabular-nums text-[var(--color-text-primary)]">{w.in_progress}</span>
                    <span className="w-10 text-right text-sm tabular-nums text-[var(--color-text-muted)]">{w.open_points}</span>
                    <span className={`w-12 text-right text-sm tabular-nums ${w.blocked > 0 ? "text-[var(--color-error)]" : "text-[var(--color-text-muted)]"}`}>{w.blocked}</span>
                  </div>
                ))}
              </div>
            )}
          </Section>

          {/* Active Epics */}
          <Section title="Active Epics" icon={<SectionIcon d={SECTION_ICONS.epics} />} count={metrics?.epics.length} collapsible storageKey="exponential-dashboard-epics-open">
            {!metrics || metrics.epics.length === 0 ? (
              <div className="px-5 py-10 flex items-center justify-center">
                <p className="text-sm text-[var(--color-text-muted)] text-center">No epics yet — create one with the <span className="font-mono">epic</span> label.</p>
              </div>
            ) : (
              <div className="py-2">
                {metrics.epics.map((ep) => {
                  const issue = issues.find((i) => i.id === ep.issue_id);
                  return (
                    <button key={ep.issue_id} onClick={() => issue && onIssueClick?.(issue)} className="flex items-center gap-3 w-full px-5 py-2 text-left transition-colors hover:bg-[var(--color-hover-surface)]">
                      <StatusIcon status={issue?.status || "PLANNED"} size={14} isInferred={issue?.is_inferred} />
                      <span className="text-sm text-[var(--color-text-primary)] truncate">{ep.title}</span>
                      <CopyableId id={ep.issue_id} className="text-xs shrink-0 tabular-nums" />
                      {ep.stale && <span className="text-xs uppercase tracking-wider font-medium px-2 py-1 rounded border shrink-0 text-[var(--color-warning)] border-[var(--color-warning)]/40 bg-[var(--color-warning)]/10">STALE</span>}
                      <span className="ml-auto flex items-center gap-2 text-xs text-[var(--color-text-muted)] shrink-0">
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
          <Section title="Recent Activity" icon={<SectionIcon d={SECTION_ICONS.activity} />} count={activity.length} collapsible storageKey="exponential-dashboard-activity-open">
            <ActivityFeed activity={activity} issues={issues} onIssueClick={onIssueClick} />
          </Section>

        </div>
      </div>
    </div>
  );
}
