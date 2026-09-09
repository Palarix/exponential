import { describe, it, expect } from "vitest";
import { computeAppendKey, sortIssuesWithinGroups, sortGroup } from "./sort";
import { makeIssue } from "../test-utils";

describe("computeAppendKey", () => {
  it("returns a key for an empty array", () => {
    const key = computeAppendKey([]);
    expect(typeof key).toBe("string");
    expect(key.length).toBeGreaterThan(0);
  });

  it("returns a key greater than all existing sort_order values", () => {
    const issues = [
      makeIssue({ sort_order: "a0" }),
      makeIssue({ sort_order: "a1" }),
      makeIssue({ sort_order: "a2" }),
    ];
    const key = computeAppendKey(issues);
    expect(key > "a2").toBe(true);
  });

  it("handles issues with empty sort_order", () => {
    const issues = [
      makeIssue({ sort_order: "" }),
      makeIssue({ sort_order: "a1" }),
    ];
    const key = computeAppendKey(issues);
    expect(key > "a1").toBe(true);
  });
});

describe("sortIssuesWithinGroups", () => {
  const backlog = makeIssue({ id: "b", status: "BACKLOG", sort_order: "a1", title: "Bravo" });
  const planned = makeIssue({ id: "p", status: "PLANNED", sort_order: "a0", title: "Alpha" });
  const doing = makeIssue({ id: "d", status: "DOING", sort_order: "a2", title: "Charlie" });
  const done1 = makeIssue({
    id: "d1", status: "DONE", title: "Done One",
    updated_at: "2026-01-02T00:00:00Z",
  });
  const done2 = makeIssue({
    id: "d2", status: "DONE", title: "Done Two",
    updated_at: "2026-01-03T00:00:00Z",
  });

  it("sorts by status group order in manual mode", () => {
    const sorted = sortIssuesWithinGroups([doing, backlog, planned], "manual");
    expect(sorted.map((i) => i.id)).toEqual(["b", "p", "d"]);
  });

  it("sorts within same status by sort_order in manual mode", () => {
    const p1 = makeIssue({ id: "p1", status: "PLANNED", sort_order: "a2" });
    const p2 = makeIssue({ id: "p2", status: "PLANNED", sort_order: "a0" });
    const sorted = sortIssuesWithinGroups([p1, p2], "manual");
    expect(sorted.map((i) => i.id)).toEqual(["p2", "p1"]);
  });

  it("sorts DONE group by updated_at regardless of sort key", () => {
    const sorted = sortIssuesWithinGroups([done1, done2], "title");
    expect(sorted.map((i) => i.id)).toEqual(["d2", "d1"]);
  });

  it("sorts by title within same status group", () => {
    const a = makeIssue({ id: "a", status: "PLANNED", title: "Alpha" });
    const b = makeIssue({ id: "b", status: "PLANNED", title: "Bravo" });
    const sorted = sortIssuesWithinGroups([b, a], "title");
    expect(sorted.map((i) => i.id)).toEqual(["a", "b"]);
  });

  it("sorts by priority (lower number = higher priority)", () => {
    const urgent = makeIssue({ id: "u", status: "PLANNED", priority: 1 });
    const low = makeIssue({ id: "l", status: "PLANNED", priority: 4 });
    const none = makeIssue({ id: "n", status: "PLANNED", priority: 0 });
    const sorted = sortIssuesWithinGroups([none, low, urgent], "priority");
    expect(sorted.map((i) => i.id)).toEqual(["u", "l", "n"]);
  });

  it("sorts by estimate (higher first)", () => {
    const big = makeIssue({ id: "big", status: "PLANNED", estimate: 8 });
    const small = makeIssue({ id: "small", status: "PLANNED", estimate: 1 });
    const sorted = sortIssuesWithinGroups([small, big], "estimate");
    expect(sorted.map((i) => i.id)).toEqual(["big", "small"]);
  });

  it("sorts by created_at (newest first)", () => {
    const old = makeIssue({ id: "old", status: "PLANNED", created_at: "2026-01-01T00:00:00Z" });
    const recent = makeIssue({ id: "new", status: "PLANNED", created_at: "2026-06-01T00:00:00Z" });
    const sorted = sortIssuesWithinGroups([old, recent], "created");
    expect(sorted.map((i) => i.id)).toEqual(["new", "old"]);
  });
});

describe("sortGroup", () => {
  it("sorts by sort_order in manual mode", () => {
    const a = makeIssue({ id: "a", sort_order: "a2" });
    const b = makeIssue({ id: "b", sort_order: "a0" });
    const sorted = sortGroup([a, b], "manual");
    expect(sorted.map((i) => i.id)).toEqual(["b", "a"]);
  });

  it("sorts by the given sort key in non-manual mode", () => {
    const a = makeIssue({ id: "a", title: "Zebra" });
    const b = makeIssue({ id: "b", title: "Alpha" });
    const sorted = sortGroup([a, b], "title");
    expect(sorted.map((i) => i.id)).toEqual(["b", "a"]);
  });

  it("sorts DONE items by updated_at even in manual mode", () => {
    const d1 = makeIssue({ id: "d1", status: "DONE", sort_order: "a0", updated_at: "2026-01-01T00:00:00Z" });
    const d2 = makeIssue({ id: "d2", status: "DONE", sort_order: "a2", updated_at: "2026-06-01T00:00:00Z" });
    const sorted = sortGroup([d1, d2], "manual");
    expect(sorted.map((i) => i.id)).toEqual(["d2", "d1"]);
  });
});
