import { describe, it, expect } from "vitest";
import { truncate, kindColor, edgePath } from "./dep-graph-utils";

describe("truncate", () => {
  it("returns short strings unchanged", () => {
    expect(truncate("abc", 5)).toBe("abc");
    expect(truncate("abcde", 5)).toBe("abcde");
  });

  it("truncates and appends ellipsis", () => {
    expect(truncate("abcdef", 5)).toBe("abcde…");
  });

  it("handles empty string", () => {
    expect(truncate("", 5)).toBe("");
  });
});

describe("kindColor", () => {
  it("returns error color for blocks", () => {
    expect(kindColor("blocks")).toBe("var(--color-error)");
  });

  it("returns error color for depends_on", () => {
    expect(kindColor("depends_on")).toBe("var(--color-error)");
  });

  it("returns info color for relates_to", () => {
    expect(kindColor("relates_to")).toBe("var(--color-info)");
  });

  it("returns muted for duplicates", () => {
    expect(kindColor("duplicates")).toBe("var(--color-text-muted)");
  });

  it("returns error color for blocked_by", () => {
    expect(kindColor("blocked_by")).toBe("var(--color-error)");
  });

  it("returns error color for dependency_of", () => {
    expect(kindColor("dependency_of")).toBe("var(--color-error)");
  });

  it("returns muted for duplicated_by", () => {
    expect(kindColor("duplicated_by")).toBe("var(--color-text-muted)");
  });

  it("returns muted for unknown kinds", () => {
    expect(kindColor("unknown")).toBe("var(--color-text-muted)");
  });
});

describe("edgePath", () => {
  it("returns empty string for no points", () => {
    expect(edgePath([])).toBe("");
  });

  it("returns M command for single point", () => {
    expect(edgePath([{ x: 10, y: 20 }])).toBe("M10,20");
  });

  it("draws polyline through all points", () => {
    const path = edgePath([{ x: 0, y: 0 }, { x: 100, y: 50 }]);
    expect(path).toBe("M0,0L100,50");
  });

  it("starts at first point and ends at last point", () => {
    const points = [
      { x: 10, y: 20 },
      { x: 50, y: 50 },
      { x: 90, y: 20 },
    ];
    const path = edgePath(points);
    expect(path).toBe("M10,20L50,50L90,20");
  });

  it("handles orthogonal bend points", () => {
    const path = edgePath([
      { x: 0, y: 0 },
      { x: 50, y: 0 },
      { x: 50, y: 100 },
      { x: 100, y: 100 },
    ]);
    expect(path).toBe("M0,0L50,0L50,100L100,100");
  });
});
