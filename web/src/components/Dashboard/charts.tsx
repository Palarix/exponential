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

export function DailyVelocityChart({ buckets }: { buckets: { date: string; points: number }[] }) {
  const data = buckets.map((b) => ({
    date: b.date.slice(5),
    points: b.points,
  }));

  return (
    <ResponsiveContainer width="100%" height={200}>
      <AreaChart data={data} margin={{ top: 4, right: 4, bottom: 0, left: -20 }}>
        <defs>
          <linearGradient id="dv-gradient" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stopColor="var(--color-accent-primary)" stopOpacity={0.25} />
            <stop offset="100%" stopColor="var(--color-accent-primary)" stopOpacity={0} />
          </linearGradient>
        </defs>
        <CartesianGrid vertical strokeDasharray="3 3" stroke="var(--color-border-subtle)" horizontal={false} />
        <XAxis dataKey="date" tick={AXIS_STYLE} axisLine={false} tickLine={false} interval={0} />
        <YAxis tick={AXIS_STYLE} axisLine={false} tickLine={false} allowDecimals={false} />
        <Tooltip {...TOOLTIP_STYLE} />
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
        <XAxis dataKey="week" tick={AXIS_STYLE} axisLine={false} tickLine={false} />
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
