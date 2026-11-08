import { describe, expect, it } from "vitest";
import type { Milestone, MilestonesResponse } from "../../api/types";
import { makeIssue } from "../../test-utils";
import {
  acceptsIssueDrop, childProgress, matchesIssueSearch, parentContext, issuesInMilestone, openMilestones, orderWarnings, milestoneDropRef, reorderMove, savedOrder, timeBar,
  noMilestoneIssues, milestoneFilter, moveOptions, moveRequest, milestoneMemberCounts, headerClickToggles, rowEstimate, showsOnDashboard,
  anyMatch, defaultExpanded, parseExpanded,
} from "./milestones-utils";

const rel = (number: number, overrides: Partial<Milestone> = {}): Milestone => ({
  id: `ms-${number}`,
  number,
  name: `#${number}`,
  state: "OPEN",
  counts: {},
  total: 0,
  done: 0,
  ...overrides,
});

const placeholder = (number: number, slot: "current" | "next", locked = false): Milestone =>
  rel(number, { id: undefined, placeholder: true, slot, locked });

const board = (overrides: Partial<MilestonesResponse> = {}): MilestonesResponse => ({
  current: rel(1, { slot: "current" }),
  next: rel(2, { slot: "next" }),
  upcoming: [],
  past: [],
  ...overrides,
});

describe("openMilestones", () => {
  it("lists Current and Next, placeholders included, then Upcoming", () => {
    const b = board({ current: rel(1), next: placeholder(2, "next") });
    expect(openMilestones(b).map((r) => r.number)).toEqual([1, 2]);
  });
});

describe("showsOnDashboard", () => {
  it("shows a saved milestone with scope", () => {
    expect(showsOnDashboard(rel(1, { total: 3 }))).toBe(true);
  });

  it("shows a saved milestone whose scope is all done", () => {
    expect(showsOnDashboard(rel(1, { total: 3, done: 3 }))).toBe(true);
  });

  it("hides a milestone without scope", () => {
    expect(showsOnDashboard(rel(1))).toBe(false);
  });

  it("hides a placeholder", () => {
    expect(showsOnDashboard(placeholder(1, "current"))).toBe(false);
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
  it("flags both milestones when a later one has an earlier target date", () => {
    const w = orderWarnings([
      rel(1, { name: "1.5", target_date: "2026-12-01" }),
      rel(2, { name: "1.6", target_date: "2026-11-01" }),
    ]);
    expect(w.get(1)).toEqual(["1.6"]);
    expect(w.get(2)).toEqual(["1.5"]);
  });

  it("does not flag milestones in target date order, or with equal dates", () => {
    const w = orderWarnings([
      rel(1, { target_date: "2026-11-01" }),
      rel(2, { target_date: "2026-11-01" }),
      rel(3, { target_date: "2026-12-01" }),
    ]);
    expect(w.size).toBe(0);
  });

  it("skips milestones without a target date and placeholders", () => {
    const w = orderWarnings([
      rel(1, { target_date: "2026-12-01" }),
      rel(2),
      rel(3, { target_date: "2026-12-15" }),
      { ...placeholder(4, "next"), target_date: "2026-01-01" },
    ]);
    expect(w.size).toBe(0);
  });

  it("lists every milestone a milestone conflicts with", () => {
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

  it("places a milestone moved down after the one it was dropped on", () => {
    expect(reorderMove(ids, "a", "c")).toEqual({ id: "a", after: "c" });
  });

  it("places a milestone moved up before the one it was dropped on", () => {
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

describe("issuesInMilestone", () => {
  it("lists the milestone's issues, open ones first, each group by sort order", () => {
    const issues = [
      makeIssue({ id: "d1", status: "DONE", milestone_id: "ms-1", sort_order: "a0" }),
      makeIssue({ id: "o2", status: "PLANNED", milestone_id: "ms-1", sort_order: "a2" }),
      makeIssue({ id: "o1", status: "DOING", milestone_id: "ms-1", sort_order: "a1" }),
      makeIssue({ id: "x", status: "PLANNED", milestone_id: "ms-2" }),
    ];
    expect(issuesInMilestone(issues, "ms-1").map((i) => i.id)).toEqual(["o1", "o2", "d1"]);
  });

  it("keeps only the issues that match a search", () => {
    const issues = [
      makeIssue({ id: "xpo-aaa111", title: "Courier tracking", milestone_id: "ms-1" }),
      makeIssue({ id: "xpo-bbb222", title: "Menu sync", milestone_id: "ms-1" }),
    ];
    expect(issuesInMilestone(issues, "ms-1", "courier").map((i) => i.id)).toEqual(["xpo-aaa111"]);
  });

  it("is empty for a placeholder", () => {
    expect(issuesInMilestone([makeIssue({ milestone_id: "" })], undefined)).toEqual([]);
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

describe("noMilestoneIssues", () => {
  const issues = [
    makeIssue({ id: "xpo-aaa111", title: "Fix login", status: "PLANNED" }),
    makeIssue({ id: "xpo-bbb222", title: "Add search", status: "BACKLOG", sort_order: "a1" }),
    makeIssue({ id: "xpo-ccc333", title: "Done thing", status: "DONE" }),
    makeIssue({ id: "xpo-ddd444", title: "Canceled", status: "CANCELED" }),
    makeIssue({ id: "xpo-eee555", title: "In a milestone", status: "PLANNED", milestone_id: "ms-1" }),
  ];

  it("lists open issues without a milestone", () => {
    expect(noMilestoneIssues(issues, "").map((i) => i.id)).toEqual(["xpo-aaa111", "xpo-bbb222"]);
  });

  it("filters by title or ID, ignoring case", () => {
    expect(noMilestoneIssues(issues, "LOGIN").map((i) => i.id)).toEqual(["xpo-aaa111"]);
    expect(noMilestoneIssues(issues, "bbb2").map((i) => i.id)).toEqual(["xpo-bbb222"]);
  });
});

describe("issue drops", () => {
  const inOne = makeIssue({ milestone_id: "ms-1" });
  const loose = makeIssue({});

  it("sends the milestone ID, the slot ref of a placeholder, or empty for No milestone", () => {
    expect(milestoneDropRef(rel(3))).toBe("ms-3");
    expect(milestoneDropRef(placeholder(1, "current"))).toBe("@current");
    expect(milestoneDropRef(placeholder(2, "next"))).toBe("@next");
    expect(milestoneDropRef(null)).toBe("");
  });

  it("accepts drops on another open milestone and on placeholders", () => {
    expect(acceptsIssueDrop(rel(2), inOne)).toBe(true);
    expect(acceptsIssueDrop(placeholder(2, "next"), inOne)).toBe(true);
    expect(acceptsIssueDrop(rel(1), loose)).toBe(true);
  });

  it("refuses the issue's own milestone, a locked Next and completed milestones", () => {
    expect(acceptsIssueDrop(rel(1), inOne)).toBe(false);
    expect(acceptsIssueDrop(placeholder(2, "next", true), inOne)).toBe(false);
    expect(acceptsIssueDrop(rel(9, { state: "COMPLETED" }), inOne)).toBe(false);
  });

  it("accepts No milestone only for issues in a milestone", () => {
    expect(acceptsIssueDrop(null, inOne)).toBe(true);
    expect(acceptsIssueDrop(null, loose)).toBe(false);
  });
});

describe("parentContext", () => {
  const epic = makeIssue({ id: "epic", title: "Courier app", milestone_id: "ms-3", milestone: "1.2" });
  const kid = makeIssue({ id: "kid", parent_id: "epic" });

  it("names the parent and the milestone it is in", () => {
    expect(parentContext(kid, [epic, kid])).toEqual({ parent: epic, where: "in 1.2" });
  });

  it("says when the parent is in no milestone", () => {
    const loose = { ...epic, milestone_id: undefined, milestone: undefined };
    expect(parentContext(kid, [loose, kid])?.where).toBe("no milestone");
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

describe("rowEstimate", () => {
  const kids = [
    makeIssue({ id: "a", parent_id: "epic", status: "DONE", estimate: 3 }),
    makeIssue({ id: "b", parent_id: "epic", estimate: 5 }),
    makeIssue({ id: "c", parent_id: "epic" }),
  ];

  it("is the issue's own estimate without children", () => {
    expect(rowEstimate(makeIssue({ id: "solo", estimate: 2 }), [])).toBe(2);
  });

  it("is an open parent's remaining child points, an unestimated child counting 1", () => {
    const epic = makeIssue({ id: "epic", estimate: 8 });
    expect(rowEstimate(epic, [epic, ...kids])).toBe(6);
  });

  it("is a closed parent's points done", () => {
    const epic = makeIssue({ id: "epic", status: "DONE" });
    expect(rowEstimate(epic, [epic, ...kids])).toBe(3);
  });
});

describe("milestoneFilter", () => {
  it("opens Current and Next in Issues by slot, so the filter follows them", () => {
    expect(milestoneFilter(rel(1, { slot: "current" }))).toBe("@current");
    expect(milestoneFilter(placeholder(2, "next"))).toBe("@next");
  });

  it("opens an Upcoming milestone in Issues by number", () => {
    expect(milestoneFilter(rel(3, { slot: "upcoming" }))).toBe("#3");
  });

  it("has no filter for a Past milestone: it expands in place", () => {
    expect(milestoneFilter(rel(4, { slot: "past", state: "COMPLETED" }))).toBeNull();
  });
});

describe("moveOptions", () => {
  const b = board({
    current: rel(1, { slot: "current", name: "1.5" }),
    next: rel(2, { slot: "next", name: "1.6" }),
    upcoming: [rel(3, { slot: "upcoming", name: "1.7" })],
  });
  const labels = (r: Milestone, x = b) => moveOptions(x, r).options.map((o) => o.label);

  it("lists the other open milestones by slot, then No milestone, never the milestone itself", () => {
    expect(labels(b.next)).toEqual(["1.5 (Current)", "1.7", "No milestone"]);
    expect(labels(b.current)).toEqual(["1.6 (Next)", "1.7", "No milestone"]);
  });

  it("defaults to the milestone after it", () => {
    expect(moveOptions(b, b.current).value).toBe("ms-2");
    expect(moveOptions(b, b.next).value).toBe("ms-3");
  });

  it("defaults to the milestone before it when it is the last one", () => {
    expect(moveOptions(b, b.upcoming[0]).value).toBe("ms-2");
  });

  it("offers a new milestone when there is no other open milestone", () => {
    const alone = board({ current: rel(1, { slot: "current", name: "1.5" }), next: placeholder(2, "next"), upcoming: [] });
    const { options, value } = moveOptions(alone, alone.current);
    expect(options.map((o) => o.label)).toEqual(["A new milestone", "No milestone"]);
    expect(value).toBe("new");
  });
});

describe("moveRequest", () => {
  it("removes for No milestone, moves by default for a new milestone, and moves to a picked milestone", () => {
    expect(moveRequest("none")).toEqual({ issues: "remove" });
    expect(moveRequest("new")).toEqual({ issues: "move_to_next" });
    expect(moveRequest("ms-3")).toEqual({ issues: "move_to_next", to: "ms-3" });
  });
});

describe("milestoneMemberCounts", () => {
  it("counts the open members and all members, parents included", () => {
    const issues = [
      makeIssue({ id: "xpo-a", milestone_id: "ms-1", status: "DOING" }),
      makeIssue({ id: "xpo-b", milestone_id: "ms-1", status: "DONE" }),
      makeIssue({ id: "xpo-c", milestone_id: "ms-1", status: "CANCELED" }),
      makeIssue({ id: "xpo-d", milestone_id: "ms-1", status: "BACKLOG", parent_id: "xpo-a" }),
      makeIssue({ id: "xpo-e", milestone_id: "ms-2", status: "DOING" }),
    ];
    expect(milestoneMemberCounts(issues, "ms-1")).toEqual({ open: 2, all: 4 });
  });

  it("counts nothing for a milestone without an ID", () => {
    expect(milestoneMemberCounts([makeIssue({ id: "xpo-a" })], undefined)).toEqual({ open: 0, all: 0 });
  });
});

describe("headerClickToggles", () => {
  const header = { contains: (n: unknown) => n === inside || n === inButton };
  const inside = { closest: () => null };
  const inButton = { closest: (sel: string) => (sel.includes("button") ? {} : null) };
  const portal = { closest: () => null };

  it("toggles on a click on the header itself", () => {
    expect(headerClickToggles(inside as never, header as never)).toBe(true);
  });

  it("leaves clicks on the header's buttons to the buttons", () => {
    expect(headerClickToggles(inButton as never, header as never)).toBe(false);
  });

  it("ignores clicks that bubble from portals: menus, popovers and dialogs", () => {
    expect(headerClickToggles(portal as never, header as never)).toBe(false);
  });
});

describe("defaultExpanded", () => {
  it("opens a saved Current and No milestone", () => {
    expect(defaultExpanded(board())).toEqual([1, "none"]);
  });

  it("opens only No milestone while Current is a placeholder", () => {
    expect(defaultExpanded(board({ current: placeholder(1, "current") }))).toEqual(["none"]);
  });
});

describe("parseExpanded", () => {
  it("reads the saved milestone numbers and No milestone", () => {
    expect(parseExpanded('[3,"none",7]')).toEqual([3, "none", 7]);
    expect(parseExpanded("[]")).toEqual([]);
  });

  it("returns null when nothing is saved or the value is not a list of keys", () => {
    expect(parseExpanded(null)).toBeNull();
    expect(parseExpanded("{")).toBeNull();
    expect(parseExpanded('{"a":1}')).toBeNull();
    expect(parseExpanded('[1,"other",true]')).toEqual([1]);
  });
});

describe("anyMatch", () => {
  const ids = new Set(["ms-1", "ms-9"]);
  const issues = [
    makeIssue({ id: "xpo-a", title: "Courier map", milestone_id: "ms-1" }),
    makeIssue({ id: "xpo-b", title: "Hidden in a deleted milestone", milestone_id: "ms-5" }),
    makeIssue({ id: "xpo-c", title: "Loose task" }),
    makeIssue({ id: "xpo-d", title: "Closed loose task", status: "DONE" }),
  ];

  it("finds a match in a listed milestone or in No milestone", () => {
    expect(anyMatch(issues, "courier", ids)).toBe(true);
    expect(anyMatch(issues, "loose", ids)).toBe(true);
  });

  it("ignores issues the stack does not list", () => {
    expect(anyMatch(issues, "hidden", ids)).toBe(false);
    expect(anyMatch(issues, "closed loose", ids)).toBe(false);
    expect(anyMatch(issues, "zzz", ids)).toBe(false);
  });
});
