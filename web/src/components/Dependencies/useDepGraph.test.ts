import { describe, it, expect } from "vitest";
import {
  resolveIssue,
  collectEdges,
  isResolved,
  computeStats,
  type RawEdge,
} from "./useDepGraph";
import { makeIssue } from "../../test-utils";
import type { Issue } from "../../api/types";

describe("resolveIssue", () => {
  const issues = [
    makeIssue({ id: "xpo-aaa111" }),
    makeIssue({ id: "xpo-bbb222" }),
    makeIssue({ id: "xpo-ccc333" }),
  ];

  it("finds by exact id", () => {
    expect(resolveIssue(issues, "xpo-bbb222")?.id).toBe("xpo-bbb222");
  });

  it("finds by suffix match", () => {
    expect(resolveIssue(issues, "ccc333")?.id).toBe("xpo-ccc333");
  });

  it("returns undefined when no match", () => {
    expect(resolveIssue(issues, "xpo-zzz999")).toBeUndefined();
  });

  it("prefers exact match over suffix", () => {
    const ambiguous = [
      makeIssue({ id: "aaa111" }),
      makeIssue({ id: "xpo-aaa111" }),
    ];
    expect(resolveIssue(ambiguous, "aaa111")?.id).toBe("aaa111");
  });
});

describe("collectEdges", () => {
  it("returns empty for issues with no dependencies", () => {
    const issues = [makeIssue({ id: "a" }), makeIssue({ id: "b" })];
    expect(collectEdges(issues)).toEqual([]);
  });

  it("keeps blocked_by as canonical (no swap)", () => {
    const issues = [
      makeIssue({
        id: "a",
        dependencies: [{ source_id: "a", target_id: "b", kind: "blocked_by" }],
      }),
      makeIssue({ id: "b" }),
    ];
    const edges = collectEdges(issues);
    expect(edges).toHaveLength(1);
    expect(edges[0]).toEqual({ sourceId: "a", targetId: "b", kind: "blocked_by", originalKind: "blocked_by" });
  });

  it("canonicalizes blocks to blocked_by with swap", () => {
    const issues = [
      makeIssue({
        id: "a",
        dependencies: [{ source_id: "a", target_id: "b", kind: "blocks" }],
      }),
      makeIssue({ id: "b" }),
    ];
    const edges = collectEdges(issues);
    expect(edges).toHaveLength(1);
    expect(edges[0]).toEqual({ sourceId: "b", targetId: "a", kind: "blocked_by", originalKind: "blocks" });
  });

  it("canonicalizes dependency_of to depends_on", () => {
    const issues = [
      makeIssue({
        id: "a",
        dependencies: [{ source_id: "a", target_id: "b", kind: "dependency_of" }],
      }),
      makeIssue({ id: "b" }),
    ];
    const edges = collectEdges(issues);
    expect(edges[0]).toEqual({ sourceId: "b", targetId: "a", kind: "depends_on", originalKind: "dependency_of" });
  });

  it("deduplicates edges", () => {
    const issues = [
      makeIssue({
        id: "a",
        dependencies: [{ source_id: "a", target_id: "b", kind: "blocks" }],
      }),
      makeIssue({
        id: "b",
        dependencies: [{ source_id: "b", target_id: "a", kind: "blocked_by" }],
      }),
    ];
    const edges = collectEdges(issues);
    expect(edges).toHaveLength(1);
    expect(edges[0].kind).toBe("blocked_by");
  });

  it("skips edges referencing unknown issues", () => {
    const issues = [
      makeIssue({
        id: "a",
        dependencies: [{ source_id: "a", target_id: "unknown", kind: "blocks" }],
      }),
    ];
    expect(collectEdges(issues)).toEqual([]);
  });
});

describe("isResolved", () => {
  function issueMap(...issues: Issue[]): Map<string, Issue> {
    return new Map(issues.map((i) => [i.id, i]));
  }

  it("blocked_by edge is resolved when target (blocker) is DONE", () => {
    const edge: RawEdge = { sourceId: "a", targetId: "b", kind: "blocked_by", originalKind: "blocked_by" };
    const map = issueMap(
      makeIssue({ id: "a", status: "PLANNED" }),
      makeIssue({ id: "b", status: "DONE" }),
    );
    expect(isResolved(edge, map)).toBe(true);
  });

  it("blocked_by edge is unresolved when target (blocker) is not DONE", () => {
    const edge: RawEdge = { sourceId: "a", targetId: "b", kind: "blocked_by", originalKind: "blocked_by" };
    const map = issueMap(
      makeIssue({ id: "a", status: "DOING" }),
      makeIssue({ id: "b", status: "PLANNED" }),
    );
    expect(isResolved(edge, map)).toBe(false);
  });

  it("depends_on edge is resolved when target is DONE", () => {
    const edge: RawEdge = { sourceId: "a", targetId: "b", kind: "depends_on", originalKind: "depends_on" };
    const map = issueMap(
      makeIssue({ id: "a", status: "DOING" }),
      makeIssue({ id: "b", status: "DONE" }),
    );
    expect(isResolved(edge, map)).toBe(true);
  });

  it("depends_on edge is unresolved when target is not DONE", () => {
    const edge: RawEdge = { sourceId: "a", targetId: "b", kind: "depends_on", originalKind: "depends_on" };
    const map = issueMap(
      makeIssue({ id: "a", status: "DOING" }),
      makeIssue({ id: "b", status: "PLANNED" }),
    );
    expect(isResolved(edge, map)).toBe(false);
  });

  it("other kinds require both DONE", () => {
    const edge: RawEdge = { sourceId: "a", targetId: "b", kind: "relates_to", originalKind: "relates_to" };
    const bothDone = issueMap(
      makeIssue({ id: "a", status: "DONE" }),
      makeIssue({ id: "b", status: "DONE" }),
    );
    expect(isResolved(edge, bothDone)).toBe(true);

    const oneDone = issueMap(
      makeIssue({ id: "a", status: "DONE" }),
      makeIssue({ id: "b", status: "PLANNED" }),
    );
    expect(isResolved(edge, oneDone)).toBe(false);
  });

  it("returns false when issues are missing from map", () => {
    const edge: RawEdge = { sourceId: "a", targetId: "b", kind: "blocked_by", originalKind: "blocked_by" };
    expect(isResolved(edge, new Map())).toBe(false);
  });
});

describe("computeStats", () => {
  it("counts resolved and total blocker edges", () => {
    const issues = [
      makeIssue({
        id: "a",
        status: "DONE",
        dependencies: [{ source_id: "a", target_id: "b", kind: "blocks" }],
      }),
      makeIssue({
        id: "b",
        status: "PLANNED",
        dependencies: [{ source_id: "b", target_id: "c", kind: "depends_on" }],
      }),
      makeIssue({ id: "c", status: "DOING" }),
    ];
    const stats = computeStats(issues);
    expect(stats.total).toBe(2);
    expect(stats.resolved).toBe(1);
  });

  it("excludes non-blocker edges from count", () => {
    const issues = [
      makeIssue({
        id: "a",
        dependencies: [{ source_id: "a", target_id: "b", kind: "relates_to" }],
      }),
      makeIssue({ id: "b" }),
    ];
    const stats = computeStats(issues);
    expect(stats.total).toBe(0);
    expect(stats.resolved).toBe(0);
  });

  it("returns zeros for issues with no dependencies", () => {
    const stats = computeStats([makeIssue({ id: "a" })]);
    expect(stats).toEqual({ resolved: 0, total: 0 });
  });
});
