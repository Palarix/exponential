import { describe, it, expect } from "vitest";
import { timelineConnectors } from "./activity-timeline-utils";

describe("timelineConnectors", () => {
  it("draws no segments for a single entry", () => {
    expect(timelineConnectors(0, 1)).toEqual({ above: false, below: false });
  });

  it("omits the segment above the first entry", () => {
    expect(timelineConnectors(0, 3)).toEqual({ above: false, below: true });
  });

  it("connects both sides of a middle entry", () => {
    expect(timelineConnectors(1, 3)).toEqual({ above: true, below: true });
  });

  it("omits the segment below the last entry", () => {
    expect(timelineConnectors(2, 3)).toEqual({ above: true, below: false });
  });
});
