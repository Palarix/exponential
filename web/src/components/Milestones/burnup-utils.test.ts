import { describe, expect, it } from "vitest";
import type { MilestoneDay, MilestoneStep, MilestoneTally } from "../../api/types";
import { burnUpPoints } from "./burnup-utils";

const tally = (scope: number, open: number, scopePts = scope, openPts = open): MilestoneTally => ({
  scope, open, completed: scope - open, scope_points: scopePts, open_points: openPts, completed_points: scopePts - openPts,
});
const step = (at: string, t: MilestoneTally): MilestoneStep => ({ at, changes: [], ...t });
const day = (date: string, t: MilestoneTally): MilestoneDay => ({ date, ...t });

const steps = [
  step("2026-10-01T09:00:00Z", tally(3, 3, 8, 8)),
  step("2026-10-01T10:00:00Z", tally(3, 2, 8, 5)),
  // An estimate change: points only.
  step("2026-10-01T11:00:00Z", tally(3, 2, 10, 7)),
  step("2026-10-01T12:00:00Z", tally(3, 0, 10, 0)),
];
const days = [day("2026-10-01", tally(3, 3, 8, 8)), day("2026-10-02", tally(3, 0, 10, 0))];

describe("burnUpPoints", () => {
  it("draws every step in points mode", () => {
    const points = burnUpPoints({ days, steps }, "order", "points");
    expect(points.map((p) => [p.scope, p.open])).toEqual([[8, 8], [8, 5], [10, 7], [10, 0]]);
    expect(points.map((p) => p.done)).toEqual([0, 3, 3, 10]);
    expect(points[1].step).toBe(steps[1]);
  });

  it("skips the steps whose issue counts did not change in issues mode", () => {
    const points = burnUpPoints({ days, steps }, "order", "issues");
    expect(points.map((p) => [p.scope, p.open])).toEqual([[3, 3], [3, 2], [3, 0]]);
    expect(points.map((p) => p.step)).toEqual([steps[0], steps[1], steps[3]]);
    expect(points.map((p) => p.done)).toEqual([0, 1, 3]);
  });

  it("draws the days on the time axis, in issues or points", () => {
    expect(burnUpPoints({ days, steps }, "time", "issues").map((p) => [p.scope, p.open])).toEqual([[3, 3], [3, 0]]);
    const points = burnUpPoints({ days, steps }, "time", "points");
    expect(points.map((p) => [p.scope, p.open])).toEqual([[8, 8], [10, 0]]);
    expect(points[0].day).toBe(days[0]);
  });

  it("is empty without progress", () => {
    expect(burnUpPoints(undefined, "order", "issues")).toEqual([]);
    expect(burnUpPoints({ days: [] }, "order", "issues")).toEqual([]);
  });
});
