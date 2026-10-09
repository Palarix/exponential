/** One turn of Tailwind's `animate-spin`. */
export const SPIN_PERIOD_MS = 1000;

/**
 * How long a spinner stays visible after its work settles: `null` if the
 * work finished within `delay` and never showed, otherwise enough to end on a
 * whole rotation (a full one if it hadn't started turning yet).
 */
export function spinHoldMs(t: { busyMs: number; visibleMs: number | null; delay: number; period: number }): number | null {
  if (t.visibleMs === null) return t.busyMs < t.delay ? null : t.period;
  if (t.visibleMs === 0) return t.period;
  const into = t.visibleMs % t.period;
  return into === 0 ? 0 : t.period - into;
}
