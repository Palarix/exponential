import { useEffect, useMemo, useState } from "react";
import { fetchActivity, fetchMetrics, type ActivityEvent, type AttentionItem, type Issue, type PulseMetrics } from "../../api/client";
import { Avatar, Card, CopyableId, LabelBadge, StatusIcon, SubProgress } from "../ui";
import { formatTriage } from "../../utils/format";
import { Section, SectionIcon, PulseCard, SECTION_ICONS } from "./Section";
import { Sparkline, TrendChart } from "./charts";
import ActivityFeed from "./ActivityFeed";
import DistributionSection, { type DistFilter, type DistRow } from "./DistributionSection";

function AttentionBadge({ item }: { item: AttentionItem }) {
  const styles: Record<AttentionItem["kind"], { label: string; cls: string }> = {
    blocker: { label: "BLOCKED", cls: "text-[var(--color-error)] border-[var(--color-error)]/40 bg-[var(--color-error)]/10" },
    stale_wip: { label: "STALE", cls: "text-[var(--color-warning)] border-[var(--color-warning)]/40 bg-[var(--color-warning)]/10" },
    high_priority: { label: item.priority === 1 ? "URGENT" : "HIGH", cls: "text-[var(--color-text-secondary)] border-[var(--color-border-default)] bg-[var(--color-bg-tertiary)]" },
  };
  const s = styles[item.kind];
  return (
    <span className={`text-[10px] uppercase tracking-wider font-medium px-2 py-1 rounded border shrink-0 ${s.cls}`}>
      {s.label}
    </span>
  );
}

function attentionMeta(item: AttentionItem): string {
  if (item.kind === "high_priority") return "";
  return `${item.age_days ?? 0}d`;
}

const PRIORITY_LABELS: { value: number; label: string; marker: string; markerClass: string }[] = [
  { value: 1, label: "Urgent", marker: "!!!", markerClass: "text-[var(--color-error)]" },
  { value: 2, label: "High", marker: "!!", markerClass: "text-[var(--color-warning)]" },
  { value: 3, label: "Medium", marker: "!", markerClass: "text-[var(--color-text-muted)]" },
  { value: 4, label: "Low", marker: "", markerClass: "" },
  { value: 0, label: "No priority", marker: "", markerClass: "" },
];

interface DashboardProps {
  issues: Issue[];
  onIssueClick?: (issue: Issue) => void;
}

export default function Dashboard({ issues, onIssueClick }: DashboardProps) {
  const [labelFilter, setLabelFilter] = useState<DistFilter>("active");
  const [assigneeFilter, setAssigneeFilter] = useState<DistFilter>("active");
  const [priorityFilter, setPriorityFilter] = useState<DistFilter>("active");
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

  const labelDistribution = useMemo(() => {
    const filtered = labelFilter === "active" ? issues.filter((i) => i.status !== "DONE") : issues;
    const counts = new Map<string, number>();
    for (const i of filtered) for (const l of i.labels || []) counts.set(l, (counts.get(l) || 0) + 1);
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
    const filtered = assigneeFilter === "active" ? issues.filter((i) => i.status !== "DONE") : issues;
    const counts = new Map<string, number>();
    let unassigned = 0;
    for (const i of filtered) {
      if (i.assignee) counts.set(i.assignee, (counts.get(i.assignee) || 0) + 1);
      else unassigned++;
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
            <span className="text-sm text-[var(--color-text-primary)] truncate">{assignee.split(" <")[0]}</span>
          </div>
        ),
        count,
        pct: total > 0 ? (count / total) * 100 : 0,
        barWidth: max > 0 ? (count / max) * 100 : 0,
      }));
    const unassignedRow: DistRow[] = unassigned > 0
      ? [{ key: "__unassigned__", label: <span className="text-sm text-[var(--color-text-muted)] italic">Unassigned</span>, count: unassigned, pct: total > 0 ? (unassigned / total) * 100 : 0, barWidth: max > 0 ? (unassigned / max) * 100 : 0 }]
      : [];
    return { total, rows: [...namedRows, ...unassignedRow] };
  }, [issues, assigneeFilter]);

  const priorityDistribution = useMemo(() => {
    const filtered = priorityFilter === "active" ? issues.filter((i) => i.status !== "DONE") : issues;
    const counts = new Map<number, number>();
    for (const i of filtered) { const p = i.priority || 0; counts.set(p, (counts.get(p) || 0) + 1); }
    const total = filtered.length;
    const max = Math.max(0, ...counts.values());
    const rows: DistRow[] = PRIORITY_LABELS
      .filter((p) => (counts.get(p.value) || 0) > 0)
      .map((p) => {
        const count = counts.get(p.value) || 0;
        return {
          key: String(p.value),
          label: (
            <div className="flex items-center gap-2">
              <span className={`text-xs font-medium tabular-nums w-6 ${p.markerClass}`}>{p.marker || "—"}</span>
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
    { label: "Backlog", status: "BACKLOG", value: stats.backlog, color: "var(--color-status-backlog)" },
    { label: "Planned", status: "PLANNED", value: stats.planned, color: "var(--color-status-planned)" },
    { label: "In Progress", status: "DOING", value: stats.doing, color: "var(--color-status-doing)" },
    { label: "Blocked", status: "BLOCKED", value: stats.blocked, color: "var(--color-status-blocked)" },
    { label: "Done", status: "DONE", value: stats.done, color: "var(--color-status-done)" },
  ];

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
          <Section title="Pulse" icon={<SectionIcon d={SECTION_ICONS.pulse} />} collapsible storageKey="beats-dashboard-pulse-open">
            <div className="px-5 py-3 grid grid-cols-2 lg:grid-cols-4 gap-3">
              <PulseCard title="Velocity">
                <div className="flex items-end justify-between gap-3">
                  <div>
                    <p className="text-2xl font-semibold text-[var(--color-text-primary)] leading-none">{metrics ? metrics.velocity.last_7d_points : "—"}</p>
                    <p className="text-xs text-[var(--color-text-muted)] mt-2">pts last 7d</p>
                  </div>
                  {metrics && metrics.velocity.weekly_buckets.length > 0 && <Sparkline buckets={metrics.velocity.weekly_buckets} />}
                </div>
              </PulseCard>
              <PulseCard title="Throughput">
                <p className="text-2xl font-semibold text-[var(--color-text-primary)] leading-none">{metrics ? metrics.throughput.last_7d : "—"}</p>
                <p className="text-xs text-[var(--color-text-muted)] mt-2">issues last 7d</p>
                {metrics && (
                  <p className={`text-xs mt-1 ${metrics.throughput.delta > 0 ? "text-[var(--color-success)]" : metrics.throughput.delta < 0 ? "text-[var(--color-warning)]" : "text-[var(--color-text-muted)]"}`}>
                    {metrics.throughput.delta > 0 ? "+" : ""}{metrics.throughput.delta} vs prior 7d
                  </p>
                )}
              </PulseCard>
              <PulseCard title="WIP">
                <p className="text-2xl font-semibold text-[var(--color-text-primary)] leading-none">{metrics ? metrics.wip.total : "—"}</p>
                <p className="text-xs text-[var(--color-text-muted)] mt-2">in progress</p>
                {metrics && metrics.wip.stale > 0 && (
                  <div className="mt-1 flex items-center gap-2 text-xs text-[var(--color-warning)]">
                    <svg className="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                      <path strokeLinecap="round" strokeLinejoin="round" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                    </svg>
                    {metrics.wip.stale} stale &gt; {metrics.wip.stale_threshold_days}d
                  </div>
                )}
              </PulseCard>
              <PulseCard title="Blockers">
                <p className="text-2xl font-semibold text-[var(--color-text-primary)] leading-none">{metrics ? metrics.blockers.total : "—"}</p>
                <p className="text-xs text-[var(--color-text-muted)] mt-2">blocked</p>
                {metrics && metrics.blockers.total > 0 && <p className="text-xs text-[var(--color-error)] mt-1">oldest {metrics.blockers.oldest_days}d</p>}
              </PulseCard>
            </div>
          </Section>

          {/* Trends */}
          <Section title="Trends" icon={<SectionIcon d={SECTION_ICONS.trends} />} collapsible storageKey="beats-dashboard-trends-open">
            <div className="px-5 py-3 grid grid-cols-1 lg:grid-cols-3 gap-3">
              <Card variant="elevated" padding="sm" className="min-h-[112px] flex flex-col">
                <div className="flex items-center justify-between mb-2">
                  <p className="text-xs uppercase tracking-wider text-[var(--color-text-muted)]">Created vs Completed</p>
                  <div className="flex items-center gap-2 text-[10px] text-[var(--color-text-muted)]">
                    <span className="flex items-center gap-1"><span className="w-2 h-2 rounded-sm bg-[var(--color-text-muted)]" />created</span>
                    <span className="flex items-center gap-1"><span className="w-2 h-2 rounded-sm bg-[var(--color-accent-primary)]" />completed</span>
                  </div>
                </div>
                {metrics ? <div className="flex items-end justify-center"><TrendChart weekly={metrics.trends.weekly} /></div> : <p className="text-2xl font-semibold text-[var(--color-text-primary)] leading-none">—</p>}
                <p className="mt-auto text-xs text-[var(--color-text-muted)]">last 8 weeks</p>
              </Card>
              <Card variant="elevated" padding="sm" className="min-h-[112px] flex flex-col">
                <p className="text-xs uppercase tracking-wider text-[var(--color-text-muted)] mb-2">Median Triage Time</p>
                <p className="text-2xl font-semibold text-[var(--color-text-primary)] leading-none">{metrics ? formatTriage(metrics.trends.median_triage_mins) : "—"}</p>
                <p className="mt-auto text-xs text-[var(--color-text-muted)]">
                  {metrics && metrics.trends.triaged_count > 0
                    ? `across ${metrics.trends.triaged_count} triaged ${metrics.trends.triaged_count === 1 ? "issue" : "issues"}`
                    : "no triaged issues yet"}
                </p>
              </Card>
              <Card variant="elevated" padding="sm" className="min-h-[112px] flex flex-col">
                <p className="text-xs uppercase tracking-wider text-[var(--color-text-muted)] mb-2">Bug Age</p>
                {(() => {
                  const buckets = metrics?.trends.bug_age;
                  const total = buckets ? buckets.under_24h + buckets.under_48h + buckets.under_5d + buckets.under_14d + buckets.under_1mo + buckets.over_1mo : 0;
                  if (!metrics || total === 0) return (<><p className="text-2xl font-semibold text-[var(--color-text-primary)] leading-none">0</p><p className="text-xs text-[var(--color-text-muted)] mt-2">no open bugs</p></>);
                  const bars = [
                    { label: "<24h", count: buckets!.under_24h, cls: "bg-[var(--color-text-secondary)]" },
                    { label: "<48h", count: buckets!.under_48h, cls: "bg-[var(--color-text-secondary)]" },
                    { label: "<5d", count: buckets!.under_5d, cls: "bg-[var(--color-warning)]" },
                    { label: "<14d", count: buckets!.under_14d, cls: "bg-[var(--color-warning)]" },
                    { label: "<1mo", count: buckets!.under_1mo, cls: "bg-[var(--color-error)]" },
                    { label: ">1mo", count: buckets!.over_1mo, cls: "bg-[var(--color-error)]" },
                  ];
                  const max = Math.max(1, ...bars.map((b) => b.count));
                  return (
                    <div className="mt-auto">
                      <div className="flex items-end gap-1 h-12">
                        {bars.map((b) => (
                          <div key={b.label} className="flex-1 flex flex-col items-center justify-end h-full">
                            <span className="text-[10px] text-[var(--color-text-muted)] tabular-nums leading-none mb-1">{b.count > 0 ? b.count : ""}</span>
                            <div className={`w-3 rounded-t-sm ${b.cls} transition-all duration-500`} style={{ height: `${(b.count / max) * 100}%`, minHeight: b.count > 0 ? 2 : 0 }} />
                          </div>
                        ))}
                      </div>
                      <div className="flex gap-1 mt-2">
                        {bars.map((b) => <span key={b.label} className="flex-1 text-center text-[10px] text-[var(--color-text-muted)] tabular-nums">{b.label}</span>)}
                      </div>
                    </div>
                  );
                })()}
              </Card>
            </div>
          </Section>

          {/* Needs Attention */}
          {metrics && metrics.attention.length > 0 && (
            <Section title="Needs Attention" icon={<SectionIcon d={SECTION_ICONS.attention} />} count={metrics.attention.length} collapsible storageKey="beats-dashboard-attention-open">
              <div className="py-2">
                {metrics.attention.map((item) => {
                  const issue = issues.find((i) => i.id === item.issue_id);
                  return (
                    <button key={item.issue_id} onClick={() => issue && onIssueClick?.(issue)} className="flex items-center gap-3 w-full px-5 py-2 text-left transition-colors hover:bg-[var(--color-bg-hover)]">
                      <AttentionBadge item={item} />
                      <span className="text-sm text-[var(--color-text-primary)] truncate flex-1">{item.title}</span>
                      <span className="text-xs text-[var(--color-text-muted)] tabular-nums shrink-0">{attentionMeta(item)}</span>
                    </button>
                  );
                })}
              </div>
            </Section>
          )}

          {/* Workload */}
          <Section title="Workload" icon={<SectionIcon d={SECTION_ICONS.workload} />} count={metrics?.workload.length} collapsible storageKey="beats-dashboard-workload-open">
            {!metrics || metrics.workload.length === 0 ? (
              <div className="px-5 py-10 flex items-center justify-center"><p className="text-sm text-[var(--color-text-muted)] text-center">No assigned work.</p></div>
            ) : (
              <div className="py-2">
                <div className="grid grid-cols-[1fr_auto_auto_auto] gap-x-4 px-5 py-1 text-[10px] uppercase tracking-wider text-[var(--color-text-muted)]">
                  <span>Person</span><span className="text-right w-10">WIP</span><span className="text-right w-10">Pts</span><span className="text-right w-12">Blocked</span>
                </div>
                {metrics.workload.map((w) => (
                  <div key={w.assignee} className="grid grid-cols-[1fr_auto_auto_auto] gap-x-4 items-center px-5 py-2 hover:bg-[var(--color-bg-hover)] transition-colors" title={w.last_completed ? `Last completed ${w.last_completed}` : "No completed issues yet"}>
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
          <Section title="Active Epics" icon={<SectionIcon d={SECTION_ICONS.epics} />} count={metrics?.epics.length} collapsible storageKey="beats-dashboard-epics-open">
            {!metrics || metrics.epics.length === 0 ? (
              <div className="px-5 py-10 flex items-center justify-center">
                <p className="text-sm text-[var(--color-text-muted)] text-center">No epics yet — create one with the <span className="font-mono">epic</span> label.</p>
              </div>
            ) : (
              <div className="py-2">
                {metrics.epics.map((ep) => {
                  const issue = issues.find((i) => i.id === ep.issue_id);
                  return (
                    <button key={ep.issue_id} onClick={() => issue && onIssueClick?.(issue)} className="flex items-center gap-3 w-full px-5 py-2 text-left transition-colors hover:bg-[var(--color-bg-hover)]">
                      <StatusIcon status={issue?.status || "PLANNED"} size={14} />
                      <span className="text-sm text-[var(--color-text-primary)] truncate">{ep.title}</span>
                      <CopyableId id={ep.issue_id} className="text-xs shrink-0 tabular-nums" />
                      {ep.stale && <span className="text-[10px] uppercase tracking-wider font-medium px-2 py-1 rounded border shrink-0 text-[var(--color-warning)] border-[var(--color-warning)]/40 bg-[var(--color-warning)]/10">STALE</span>}
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
          <Section title="Recent Activity" icon={<SectionIcon d={SECTION_ICONS.activity} />} count={activity.length} collapsible storageKey="beats-dashboard-activity-open">
            <ActivityFeed activity={activity} issues={issues} onIssueClick={onIssueClick} />
          </Section>

          {/* Composition */}
          <Section title="Composition" icon={<SectionIcon d={SECTION_ICONS.composition} />} collapsible defaultOpen={false} storageKey="beats-dashboard-composition-open">
            <div className="px-5 py-4 space-y-6">
              <div>
                <h3 className="text-xs uppercase tracking-wider text-[var(--color-text-muted)] mb-3">By status</h3>
                <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-5 gap-3">
                  {statusCards.map((card) => (
                    <Card key={card.label} variant="default" className="text-center">
                      <div className="flex items-center justify-center gap-2 mb-2">
                        <StatusIcon status={card.status} size={12} />
                        <p className="text-xs uppercase tracking-wider text-[var(--color-text-muted)]">{card.label}</p>
                      </div>
                      <p className="text-2xl font-semibold text-[var(--color-text-primary)] leading-none">{card.value}</p>
                      <div className="h-1 mt-3 rounded-full bg-[var(--color-bg-tertiary)]">
                        <div className="h-full rounded-full transition-all duration-500" style={{ background: card.color, width: `${stats.total ? (card.value / stats.total) * 100 : 0}%` }} />
                      </div>
                    </Card>
                  ))}
                </div>
              </div>
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                <DistributionSection title="By label" filter={labelFilter} onFilterChange={setLabelFilter} rows={labelDistribution.rows} total={labelDistribution.total} emptyHasIssues="No labels on the filtered issues." />
                <DistributionSection title="By assignee" filter={assigneeFilter} onFilterChange={setAssigneeFilter} rows={assigneeDistribution.rows} total={assigneeDistribution.total} emptyHasIssues="No assignees on the filtered issues." />
                <DistributionSection title="By priority" filter={priorityFilter} onFilterChange={setPriorityFilter} rows={priorityDistribution.rows} total={priorityDistribution.total} emptyHasIssues="No priorities set on the filtered issues." />
              </div>
            </div>
          </Section>
        </div>
      </div>
    </div>
  );
}
