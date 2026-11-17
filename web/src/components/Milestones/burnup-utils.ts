import type { MilestoneDay, MilestoneStep, MilestoneTally } from "../../api/types";
import type { BurnUpAxis, ProgressMode } from "./milestone-settings";

/** One drawn point of a burn-up, with the step or day it came from for the hover. */
export interface BurnUpPoint {
  scope: number;
  open: number;
  done: number;
  step?: MilestoneStep;
  day?: MilestoneDay;
}

const measure = (t: MilestoneTally, mode: ProgressMode) =>
  mode === "points"
    ? { scope: t.scope_points, open: t.open_points, done: t.completed_points }
    : { scope: t.scope, open: t.open, done: t.completed };

/**
 * The points a burn-up draws: the days (time axis) or the steps (order
 * axis), in issues or story points. In issues mode a step that changed only
 * points (an estimate) would draw a flat segment, so it is left out.
 */
export function burnUpPoints(
  progress: { days?: MilestoneDay[]; steps?: MilestoneStep[] } | undefined,
  axis: BurnUpAxis,
  mode: ProgressMode,
): BurnUpPoint[] {
  if (axis === "time") return (progress?.days ?? []).map((day) => ({ ...measure(day, mode), day }));
  const points: BurnUpPoint[] = [];
  for (const step of progress?.steps ?? []) {
    const p = { ...measure(step, mode), step };
    const prev = points[points.length - 1];
    if (mode === "issues" && prev && prev.scope === p.scope && prev.open === p.open && prev.done === p.done) continue;
    points.push(p);
  }
  return points;
}
