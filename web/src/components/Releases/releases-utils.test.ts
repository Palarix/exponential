import { describe, expect, it } from "vitest";
import type { Release, ReleasesResponse } from "../../api/types";
import { makeIssue } from "../../test-utils";
import {
  acceptsIssueDrop, childProgress, matchesIssueSearch, parentContext, issuesInRelease, openReleases, orderWarnings, releaseDropRef, reorderMove, savedOrder, timeBar,
  noReleaseIssues, releaseFilter,
} from "./releases-utils";

const rel = (number: number, overrides: Partial<Release> = {}): Release => ({
  id: `rel-${number}`,
  number,
  name: `#${number}`,
  state: "OPEN",
  counts: {},
  total: 0,
  done: 0,
  ...overrides,
});

const placeholder = (number: number, slot: "current" | "next", locked = false): Release =>
  rel(number, { id: undefined, placeholder: true, slot, locked });

const board = (overrides: Partial<ReleasesResponse> = {}): ReleasesResponse => ({
  current: rel(1, { slot: "current" }),
  next: rel(2, { slot: "next" }),
  upcoming: [],
  past: [],
  ...overrides,
});

describe("openReleases", () => {
  it("lists Current and Next, placeholders included, then Upcoming", () => {
    const b = board({ current: rel(1), next: placeholder(2, "next") });
    expect(openReleases(b).map((r) => r.number)).toEqual([1, 2]);
  });
});

describe("savedOrder", () => {
  it("lists Current, Next and Upcoming in order", () => {
    const b = board({ upcoming: [rel(3), rel(4)] });
    expect(savedOrder(b).map((r) => r.number)).toEqual([1, 2, 3, 4]);
  });

  it("leaves placeholders out", () => {
    expect(savedOrder(board({ next: placeholder(2, "next") })).map((r) => r.number)).toEqual([1]);
    expect(savedOrder(board({ current: placeholder(1, "current"), next: placeholder(2, "next", true) }))).toEqual([]);
  });
});

describe("orderWarnings", () => {
  it("flags both releases when a later one has an earlier target date", () => {
    const w = orderWarnings([
      rel(1, { name: "1.5", target_date: "2026-12-01" }),
      rel(2, { name: "1.6", target_date: "2026-11-01" }),
    ]);
    expect(w.get(1)).toEqual(["1.6"]);
    expect(w.get(2)).toEqual(["1.5"]);
  });

  it("does not flag releases in target date order, or with equal dates", () => {
    const w = orderWarnings([
      rel(1, { target_date: "2026-11-01" }),
      rel(2, { target_date: "2026-11-01" }),
      rel(3, { target_date: "2026-12-01" }),
    ]);
    expect(w.size).toBe(0);
  });

  it("skips releases without a target date and placeholders", () => {
    const w = orderWarnings([
      rel(1, { target_date: "2026-12-01" }),
      rel(2),
      rel(3, { target_date: "2026-12-15" }),
      { ...placeholder(4, "next"), target_date: "2026-01-01" },
    ]);
    expect(w.size).toBe(0);
  });

  it("lists every release a release conflicts with", () => {
    const w = orderWarnings([
      rel(1, { name: "A", target_date: "2026-12-01" }),
      rel(2, { name: "B", target_date: "2026-11-15" }),
      rel(3, { name: "C", target_date: "2026-11-01" }),
    ]);
    expect(w.get(1)).toEqual(["B", "C"]);
    expect(w.get(2)).toEqual(["A", "C"]);
    expect(w.get(3)).toEqual(["A", "B"]);
  });
});

describe("reorderMove", () => {
  const ids = ["a", "b", "c", "d"];

  it("places a release moved down after the one it was dropped on", () => {
    expect(reorderMove(ids, "a", "c")).toEqual({ id: "a", after: "c" });
  });

  it("places a release moved up before the one it was dropped on", () => {
    expect(reorderMove(ids, "d", "a")).toEqual({ id: "d", before: "a" });
  });

  it("does nothing for a drop on itself or an unknown target", () => {
    expect(reorderMove(ids, "b", "b")).toBeNull();
    expect(reorderMove(ids, "b", "zzz")).toBeNull();
  });
});

describe("timeBar", () => {
  it("fills from the start to today, relative to the target date", () => {
    expect(timeBar("2026-10-01T09:00:00Z", "2026-10-11", "2026-10-06", true)).toEqual({ fraction: 0.5, overdue: false });
  });

  it("is full and overdue once the target date passed with open scope", () => {
    expect(timeBar("2026-10-01", "2026-10-11", "2026-10-20", true)).toEqual({ fraction: 1, overdue: true });
    expect(timeBar("2026-10-01", "2026-10-11", "2026-10-20", false)).toEqual({ fraction: 1, overdue: false });
  });

  it("is empty before the start", () => {
    expect(timeBar("2026-10-10", "2026-10-20", "2026-10-01", true)).toEqual({ fraction: 0, overdue: false });
  });

  it("has no bar without a target date or a start", () => {
    expect(timeBar("2026-10-01", undefined, "2026-10-06", true)).toBeNull();
    expect(timeBar(undefined, "2026-10-11", "2026-10-06", true)).toBeNull();
  });
});

describe("issuesInRelease", () => {
  it("lists the release's issues, open ones first, each group by sort order", () => {
    const issues = [
      makeIssue({ id: "d1", status: "DONE", release_id: "rel-1", sort_order: "a0" }),
      makeIssue({ id: "o2", status: "PLANNED", release_id: "rel-1", sort_order: "a2" }),
      makeIssue({ id: "o1", status: "DOING", release_id: "rel-1", sort_order: "a1" }),
      makeIssue({ id: "x", status: "PLANNED", release_id: "rel-2" }),
    ];
    expect(issuesInRelease(issues, "rel-1").map((i) => i.id)).toEqual(["o1", "o2", "d1"]);
  });

  it("keeps only the issues that match a search", () => {
    const issues = [
      makeIssue({ id: "xpo-aaa111", title: "Courier tracking", release_id: "rel-1" }),
      makeIssue({ id: "xpo-bbb222", title: "Menu sync", release_id: "rel-1" }),
    ];
    expect(issuesInRelease(issues, "rel-1", "courier").map((i) => i.id)).toEqual(["xpo-aaa111"]);
  });

  it("is empty for a placeholder", () => {
    expect(issuesInRelease([makeIssue({ release_id: "" })], undefined)).toEqual([]);
  });
});

describe("matchesIssueSearch", () => {
  const issue = makeIssue({ id: "xpo-abc123", title: "Fix Login" });

  it("matches the title or the ID, ignoring case and surrounding spaces", () => {
    expect(matchesIssueSearch(issue, "login")).toBe(true);
    expect(matchesIssueSearch(issue, " ABC1 ")).toBe(true);
    expect(matchesIssueSearch(issue, "search")).toBe(false);
  });

  it("matches everything when the search is empty", () => {
    expect(matchesIssueSearch(issue, "  ")).toBe(true);
  });
});

describe("noReleaseIssues", () => {
  const issues = [
    makeIssue({ id: "xpo-aaa111", title: "Fix login", status: "PLANNED" }),
    makeIssue({ id: "xpo-bbb222", title: "Add search", status: "BACKLOG", sort_order: "a1" }),
    makeIssue({ id: "xpo-ccc333", title: "Done thing", status: "DONE" }),
    makeIssue({ id: "xpo-ddd444", title: "Canceled", status: "CANCELED" }),
    makeIssue({ id: "xpo-eee555", title: "In a release", status: "PLANNED", release_id: "rel-1" }),
  ];

  it("lists open issues without a release", () => {
    expect(noReleaseIssues(issues, "").map((i) => i.id)).toEqual(["xpo-aaa111", "xpo-bbb222"]);
  });

  it("filters by title or ID, ignoring case", () => {
    expect(noReleaseIssues(issues, "LOGIN").map((i) => i.id)).toEqual(["xpo-aaa111"]);
    expect(noReleaseIssues(issues, "bbb2").map((i) => i.id)).toEqual(["xpo-bbb222"]);
  });
});

describe("issue drops", () => {
  const inOne = makeIssue({ release_id: "rel-1" });
  const loose = makeIssue({});

  it("sends the release ID, the slot ref of a placeholder, or empty for No release", () => {
    expect(releaseDropRef(rel(3))).toBe("rel-3");
    expect(releaseDropRef(placeholder(1, "current"))).toBe("@current");
    expect(releaseDropRef(placeholder(2, "next"))).toBe("@next");
    expect(releaseDropRef(null)).toBe("");
  });

  it("accepts drops on another open release and on placeholders", () => {
    expect(acceptsIssueDrop(rel(2), inOne)).toBe(true);
    expect(acceptsIssueDrop(placeholder(2, "next"), inOne)).toBe(true);
    expect(acceptsIssueDrop(rel(1), loose)).toBe(true);
  });

  it("refuses the issue's own release, a locked Next and released releases", () => {
    expect(acceptsIssueDrop(rel(1), inOne)).toBe(false);
    expect(acceptsIssueDrop(placeholder(2, "next", true), inOne)).toBe(false);
    expect(acceptsIssueDrop(rel(9, { state: "RELEASED" }), inOne)).toBe(false);
  });

  it("accepts No release only for issues in a release", () => {
    expect(acceptsIssueDrop(null, inOne)).toBe(true);
    expect(acceptsIssueDrop(null, loose)).toBe(false);
  });
});

describe("parentContext", () => {
  const epic = makeIssue({ id: "epic", title: "Courier app", release_id: "rel-3", release: "1.2" });
  const kid = makeIssue({ id: "kid", parent_id: "epic" });

  it("names the parent and the release it is in", () => {
    expect(parentContext(kid, [epic, kid])).toEqual({ parent: epic, where: "in 1.2" });
  });

  it("says when the parent is in no release", () => {
    const loose = { ...epic, release_id: undefined, release: undefined };
    expect(parentContext(kid, [loose, kid])?.where).toBe("no release");
  });

  it("has nothing for an issue without a parent, or whose parent is not in the list", () => {
    expect(parentContext(epic, [epic, kid])).toBeNull();
    expect(parentContext(kid, [kid])).toBeNull();
  });
});

describe("childProgress", () => {
  it("counts all direct children and the done ones", () => {
    const issues = [
      makeIssue({ id: "epic" }),
      makeIssue({ id: "a", parent_id: "epic", status: "DONE" }),
      makeIssue({ id: "b", parent_id: "epic", status: "DOING" }),
      makeIssue({ id: "c", parent_id: "a" }),
    ];
    expect(childProgress(issues[0], issues)).toEqual({ done: 1, total: 2 });
  });

  it("has nothing for an issue without children", () => {
    expect(childProgress(makeIssue({ id: "solo" }), [makeIssue({ id: "solo" })])).toBeNull();
  });
});

describe("releaseFilter", () => {
  it("opens Current and Next in Issues by slot, so the filter follows them", () => {
    expect(releaseFilter(rel(1, { slot: "current" }))).toBe("@current");
    expect(releaseFilter(placeholder(2, "next"))).toBe("@next");
  });

  it("opens an Upcoming release in Issues by number", () => {
    expect(releaseFilter(rel(3, { slot: "upcoming" }))).toBe("#3");
  });

  it("has no filter for a Past release: it expands in place", () => {
    expect(releaseFilter(rel(4, { slot: "past", state: "RELEASED" }))).toBeNull();
  });
});
