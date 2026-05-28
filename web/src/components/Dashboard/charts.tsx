import type { PulseMetrics } from "../../api/client";

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

export function TrendChart({ weekly }: { weekly: { week_start: string; created: number; completed: number }[] }) {
  const max = Math.max(1, ...weekly.flatMap((w) => [w.created, w.completed]));
  const barW = 6;
  const innerGap = 2;
  const groupGap = 6;
  const height = 56;
  const groupW = barW * 2 + innerGap;
  const width = weekly.length * (groupW + groupGap) - groupGap;
  return (
    <svg width={width} height={height} className="shrink-0" aria-label="created vs completed, last 8 weeks">
      {weekly.map((w, i) => {
        const x = i * (groupW + groupGap);
        const ch = (w.created / max) * height;
        const dh = (w.completed / max) * height;
        return (
          <g key={w.week_start}>
            <rect x={x} y={height - Math.max(1, ch)} width={barW} height={Math.max(1, ch)} className="fill-[var(--color-text-muted)]" rx={1} />
            <rect x={x + barW + innerGap} y={height - Math.max(1, dh)} width={barW} height={Math.max(1, dh)} className="fill-[var(--color-accent-primary)]" rx={1} />
          </g>
        );
      })}
    </svg>
  );
}
