import { describe, expect, it } from "vitest";
import { navigableColumnIds, resolveFocusColumn, stepColumn } from "./board-nav";

const columns = [{ id: "BACKLOG" }, { id: "PLANNED" }, { id: "DOING" }, { id: "BLOCKED" }, { id: "DONE" }];

describe("navigableColumnIds", () => {
  it("drops hidden and collapsed columns, keeping board order", () => {
    expect(navigableColumnIds(columns, new Set(["BACKLOG"]), new Set(["BLOCKED"]))).toEqual(["PLANNED", "DOING", "DONE"]);
  });
});

describe("stepColumn", () => {
  const ids = ["PLANNED", "DOING", "BLOCKED", "DONE"];
  const containers = { PLANNED: ["a"], DOING: ["b"], BLOCKED: [], DONE: ["c"] };

  it("moves to the next non-empty column, skipping empty ones", () => {
    expect(stepColumn(ids, 1, 1, containers)).toBe(3);
    expect(stepColumn(ids, 3, -1, containers)).toBe(1);
  });

  it("stays put at the edges", () => {
    expect(stepColumn(ids, 3, 1, containers)).toBe(3);
    expect(stepColumn(ids, 0, -1, containers)).toBe(0);
  });
});

describe("resolveFocusColumn", () => {
  const ids = ["PLANNED", "DOING", "DONE"];

  it("keeps a focused column that has cards", () => {
    expect(resolveFocusColumn(ids, 1, { PLANNED: ["a"], DOING: ["b"], DONE: [] })).toBe(1);
  });

  it("falls back to the first non-empty column when the focused one is empty or gone", () => {
    expect(resolveFocusColumn(ids, 0, { PLANNED: [], DOING: [], DONE: ["c"] })).toBe(2);
    expect(resolveFocusColumn(ids, 7, { PLANNED: ["a"], DOING: [], DONE: [] })).toBe(0);
  });

  it("returns -1 when no column has cards", () => {
    expect(resolveFocusColumn(ids, 0, {})).toBe(-1);
  });
});
