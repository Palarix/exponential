import { describe, it, expect } from "vitest";
import type { Cycle } from "../api/types";
import { cycleLabel, filterCycles } from "./cycles";

function makeCycle(number: number, status: Cycle["status"]): Cycle {
  return { id: `c${number}`, number, start: "2026-01-01", end: "2026-01-14", status, done: 0, total: 0 };
}

describe("cycleLabel", () => {
  it("formats number and status", () => {
    expect(cycleLabel(makeCycle(12, "current"))).toBe("Cycle 12 (current)");
  });
});

describe("filterCycles", () => {
  const cycles = [
    makeCycle(11, "completed"),
    makeCycle(12, "current"),
    makeCycle(13, "upcoming"),
    makeCycle(21, "planned"),
  ];
  const ids = (cs: Cycle[]) => cs.map(c => c.id);

  it("excludes completed cycles for an empty query", () => {
    expect(ids(filterCycles(cycles, ""))).toEqual(["c12", "c13", "c21"]);
  });

  it("treats a whitespace-only query as empty", () => {
    expect(ids(filterCycles(cycles, "  "))).toEqual(["c12", "c13", "c21"]);
  });

  it("matches on cycle number", () => {
    expect(ids(filterCycles(cycles, "13"))).toEqual(["c13"]);
  });

  it("matches the full label case-insensitively", () => {
    expect(ids(filterCycles(cycles, "CYCLE 12"))).toEqual(["c12"]);
  });

  it("matches on status", () => {
    expect(ids(filterCycles(cycles, "upcoming"))).toEqual(["c13"]);
  });

  it("never returns completed cycles even when they match", () => {
    expect(ids(filterCycles(cycles, "11"))).toEqual([]);
    expect(ids(filterCycles(cycles, "completed"))).toEqual([]);
  });

  it("trims the query", () => {
    expect(ids(filterCycles(cycles, " planned "))).toEqual(["c21"]);
  });
});
