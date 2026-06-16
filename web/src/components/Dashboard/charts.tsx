import {
  ResponsiveContainer,
  AreaChart,
  Area,
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
} from "recharts";
import type { PulseMetrics } from "../../api/client";

const AXIS_STYLE = {
  fontSize: 10,
  fill: "var(--color-text-muted)",
};

const GRID_STYLE = {
  stroke: "var(--color-border-subtle)",
  strokeDasharray: "3 3",
};

const TOOLTIP_STYLE = {
  contentStyle: {
    background: "var(--color-bg-elevated)",
    border: "1px solid var(--color-border-default)",
    borderRadius: "var(--radius-md)",
    fontSize: 12,
    padding: "6px 10px",
    boxShadow: "var(--shadow-popover)",
  },
  labelStyle: { color: "var(--color-text-muted)", marginBottom: 2 },
  itemStyle: { color: "var(--color-text-primary)", padding: 0 },
};

export function Sparkline({ buckets }: { buckets: PulseMetrics["velocity"]["weekly_buckets"] }) {
  const max = Math.max(1, ...buckets.map((b) => b.points));
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

export function DistributionBar({ median, p75, p90, max, unit = "d" }: {
  min: number; median: number; p75: number; p90: number; max: number; unit?: string;
}) {
  // Cap axis at p90, show max as trailing label
  const axisMax = p90 || 1;
  const pct = (v: number) => Math.min((v / axisMax) * 100, 100);

  return (
    <div className="mt-2">
      <div className="relative h-3 rounded-full overflow-hidden bg-[var(--color-bg-tertiary)]">
        {/* Green: 0 → p50 */}
        <div className="absolute inset-y-0 left-0 bg-[var(--color-success)]" style={{ width: `${pct(median)}%` }} />
        {/* Amber: p50 → p75 */}
        <div className="absolute inset-y-0 bg-[var(--color-warning)]" style={{ left: `${pct(median)}%`, width: `${pct(p75) - pct(median)}%` }} />
        {/* Red: p75 → p90 */}
        <div className="absolute inset-y-0 bg-[var(--color-error)]" style={{ left: `${pct(p75)}%`, width: `${pct(p90) - pct(p75)}%` }} />
      </div>
      <div className="flex items-center mt-1.5 text-xs tabular-nums text-[var(--color-text-muted)]">
        <span className="text-[var(--color-success)]">p50 {median}{unit}</span>
        <span className="mx-1">·</span>
        <span className="text-[var(--color-warning)]">p75 {p75}{unit}</span>
        <span className="mx-1">·</span>
        <span className="text-[var(--color-error)]">p90 {p90}{unit}</span>
        {max > p90 && (
          <>
            <span className="mx-1">·</span>
            <span>max {max}{unit}</span>
          </>
        )}
      </div>
    </div>
  );
}

const DAY_NAMES = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];
function dateToDayName(iso: unknown) {
  return DAY_NAMES[new Date(String(iso) + "T00:00:00").getDay()];
}

export function DailyVelocityChart({ buckets }: { buckets: { date: string; points: number }[] }) {
  return (
    <ResponsiveContainer width="100%" height={200}>
      <AreaChart data={buckets} margin={{ top: 4, right: 4, bottom: 0, left: -20 }}>
        <defs>
          <linearGradient id="dv-gradient" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stopColor="var(--color-accent-primary)" stopOpacity={0.25} />
            <stop offset="100%" stopColor="var(--color-accent-primary)" stopOpacity={0} />
          </linearGradient>
        </defs>
        <CartesianGrid vertical strokeDasharray="3 3" stroke="var(--color-border-subtle)" horizontal={false} />
        <XAxis dataKey="date" tick={AXIS_STYLE} axisLine={false} tickLine={false} interval={0} padding={{ left: 12, right: 12 }} tickFormatter={dateToDayName} />
        <YAxis tick={AXIS_STYLE} axisLine={false} tickLine={false} allowDecimals={false} />
        <Tooltip {...TOOLTIP_STYLE} labelFormatter={dateToDayName} />
        <Area
          type="step"
          dataKey="points"
          stroke="var(--color-accent-primary)"
          strokeWidth={2}
          fill="url(#dv-gradient)"
          dot={{ r: 2.5, fill: "var(--color-accent-primary)", strokeWidth: 0 }}
          activeDot={{ r: 4, fill: "var(--color-accent-primary)", strokeWidth: 0 }}
          name="Points"
        />
      </AreaChart>
    </ResponsiveContainer>
  );
}

export function CumulativeChart({ weekly }: { weekly: { week_start: string; created: number; completed: number }[] }) {
  let cumCreated = 0;
  let cumCompleted = 0;
  const data = weekly.map((w) => {
    cumCreated += w.created;
    cumCompleted += w.completed;
    return { week: w.week_start.slice(5), created: cumCreated, completed: cumCompleted };
  });

  return (
    <ResponsiveContainer width="100%" height="100%">
      <AreaChart data={data} margin={{ top: 4, right: 4, bottom: 0, left: -20 }}>
        <defs>
          <linearGradient id="cum-created" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stopColor="var(--color-text-muted)" stopOpacity={0.2} />
            <stop offset="100%" stopColor="var(--color-text-muted)" stopOpacity={0} />
          </linearGradient>
          <linearGradient id="cum-completed" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stopColor="var(--color-accent-primary)" stopOpacity={0.25} />
            <stop offset="100%" stopColor="var(--color-accent-primary)" stopOpacity={0} />
          </linearGradient>
        </defs>
        <CartesianGrid {...GRID_STYLE} horizontal vertical={false} />
        <XAxis dataKey="week" tick={AXIS_STYLE} axisLine={false} tickLine={false} padding={{ left: 12, right: 12 }} />
        <YAxis tick={AXIS_STYLE} axisLine={false} tickLine={false} allowDecimals={false} />
        <Tooltip {...TOOLTIP_STYLE} />
        <Area type="monotone" dataKey="created" stroke="var(--color-text-muted)" strokeWidth={1.5} fill="url(#cum-created)" name="Created" dot={false} />
        <Area type="monotone" dataKey="completed" stroke="var(--color-accent-primary)" strokeWidth={1.5} fill="url(#cum-completed)" name="Completed" dot={false} />
      </AreaChart>
    </ResponsiveContainer>
  );
}

export function TrendChart({ weekly }: { weekly: { week_start: string; created: number; completed: number }[] }) {
  const data = weekly.map((w) => ({
    week: w.week_start.slice(5),
    created: w.created,
    completed: w.completed,
  }));

  return (
    <ResponsiveContainer width="100%" height={120}>
      <BarChart data={data} margin={{ top: 4, right: 4, bottom: 0, left: -20 }}>
        <CartesianGrid {...GRID_STYLE} horizontal vertical={false} />
        <XAxis dataKey="week" tick={AXIS_STYLE} axisLine={false} tickLine={false} />
        <YAxis tick={AXIS_STYLE} axisLine={false} tickLine={false} allowDecimals={false} />
        <Tooltip {...TOOLTIP_STYLE} />
        <Bar dataKey="created" fill="var(--color-text-muted)" radius={[2, 2, 0, 0]} name="Created" />
        <Bar dataKey="completed" fill="var(--color-accent-primary)" radius={[2, 2, 0, 0]} name="Completed" />
      </BarChart>
    </ResponsiveContainer>
  );
}
