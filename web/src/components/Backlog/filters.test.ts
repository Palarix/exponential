import { describe, it, expect } from "vitest";
import type { Release, ReleasesResponse } from "../../api/types";
import { hasActiveFilters, matchesFilters, matchesSearch, chipLabel, getSelected, setSelected, parseStoredFilters, filtersForScope, outOfScope, dimensionDisabled, releaseFilterOptions, releaseRefName, EMPTY_FILTERS, type BacklogFilters } from "./filters";

describe("hasActiveFilters", () => {
  it("returns false for empty filters", () => {
    expect(hasActiveFilters(EMPTY_FILTERS)).toBe(false);
  });

  it("returns true when statuses has entries", () => {
    expect(hasActiveFilters({ ...EMPTY_FILTERS, statuses: ["DONE"] })).toBe(true);
  });

  it("returns true when labels has entries", () => {
    expect(hasActiveFilters({ ...EMPTY_FILTERS, labels: ["bug"] })).toBe(true);
  });

  it("returns true when assignees has entries", () => {
    expect(hasActiveFilters({ ...EMPTY_FILTERS, assignees: ["alice"] })).toBe(true);
  });

  it("returns true when priorities has entries", () => {
    expect(hasActiveFilters({ ...EMPTY_FILTERS, priorities: [1] })).toBe(true);
  });

  it("returns true when epicId is set", () => {
    expect(hasActiveFilters({ ...EMPTY_FILTERS, epicId: "xpo-abc" })).toBe(true);
  });

  it("returns true when multiple filters are active", () => {
    const f: BacklogFilters = {
      statuses: ["DOING"],
      labels: ["bug"],
      assignees: [],
      priorities: [],
      epicId: null,
      workedBy: [],
      releases: [],
    };
    expect(hasActiveFilters(f)).toBe(true);
  });
});

describe("matchesFilters", () => {
  const base = { status: "PLANNED", labels: ["bug"], assignee: "alice", priority: 2, parent_id: "epic-1" };

  it("matches when no filters are active", () => {
    expect(matchesFilters(base, EMPTY_FILTERS)).toBe(true);
  });

  it("filters by status", () => {
    expect(matchesFilters(base, { ...EMPTY_FILTERS, statuses: ["PLANNED"] })).toBe(true);
    expect(matchesFilters(base, { ...EMPTY_FILTERS, statuses: ["DONE"] })).toBe(false);
  });

  it("filters by labels (any match)", () => {
    expect(matchesFilters(base, { ...EMPTY_FILTERS, labels: ["bug"] })).toBe(true);
    expect(matchesFilters(base, { ...EMPTY_FILTERS, labels: ["feature"] })).toBe(false);
    expect(matchesFilters(base, { ...EMPTY_FILTERS, labels: ["feature", "bug"] })).toBe(true);
  });

  it("filters by assignee", () => {
    expect(matchesFilters(base, { ...EMPTY_FILTERS, assignees: ["alice"] })).toBe(true);
    expect(matchesFilters(base, { ...EMPTY_FILTERS, assignees: ["bob"] })).toBe(false);
  });

  it("matches unassigned issues with __unassigned__ sentinel", () => {
    const unassigned = { ...base, assignee: undefined };
    expect(matchesFilters(unassigned, { ...EMPTY_FILTERS, assignees: ["__unassigned__"] })).toBe(true);
    expect(matchesFilters(unassigned, { ...EMPTY_FILTERS, assignees: ["alice"] })).toBe(false);
  });

  it("filters by priority", () => {
    expect(matchesFilters(base, { ...EMPTY_FILTERS, priorities: [2] })).toBe(true);
    expect(matchesFilters(base, { ...EMPTY_FILTERS, priorities: [1, 3] })).toBe(false);
  });

  it("treats missing priority as 0", () => {
    const noPriority = { ...base, priority: 0 };
    expect(matchesFilters(noPriority, { ...EMPTY_FILTERS, priorities: [0] })).toBe(true);
  });

  it("filters by epicId (parent_id)", () => {
    expect(matchesFilters(base, { ...EMPTY_FILTERS, epicId: "epic-1" })).toBe(true);
    expect(matchesFilters(base, { ...EMPTY_FILTERS, epicId: "epic-2" })).toBe(false);
  });

  it("handles issues with no labels", () => {
    const noLabels = { ...base, labels: undefined };
    expect(matchesFilters(noLabels, { ...EMPTY_FILTERS, labels: ["bug"] })).toBe(false);
    expect(matchesFilters(noLabels, EMPTY_FILTERS)).toBe(true);
  });

  it("applies all filter dimensions together", () => {
    const filters: BacklogFilters = {
      statuses: ["PLANNED"],
      labels: ["bug"],
      assignees: ["alice"],
      priorities: [2],
      epicId: "epic-1",
      workedBy: [],
      releases: [],
    };
    expect(matchesFilters(base, filters)).toBe(true);

    const wrongStatus = { ...base, status: "DONE" };
    expect(matchesFilters(wrongStatus, filters)).toBe(false);
  });
});

describe("chipLabel", () => {
  it("lists up to 2 values inline", () => {
    expect(chipLabel("Status", ["DONE"])).toBe("Status: DONE");
    expect(chipLabel("Status", ["DONE", "DOING"])).toBe("Status: DONE, DOING");
  });

  it("shows count for 3+ values", () => {
    expect(chipLabel("Status", ["A", "B", "C"])).toBe("Status (3)");
  });

  it("uses lookup map for display names", () => {
    const lookup = new Map([["DONE", "Done"], ["DOING", "In Progress"]]);
    expect(chipLabel("Status", ["DONE", "DOING"], lookup)).toBe("Status: Done, In Progress");
  });

  it("falls back to raw value if not in lookup", () => {
    const lookup = new Map([["DONE", "Done"]]);
    expect(chipLabel("Status", ["DONE", "UNKNOWN"], lookup)).toBe("Status: Done, UNKNOWN");
  });
});

describe("getSelected", () => {
  const filters: BacklogFilters = {
    statuses: ["DONE"],
    labels: ["bug"],
    assignees: ["alice"],
    priorities: [1, 2],
    epicId: "epic-1",
    workedBy: [],
    releases: [],
  };

  it("gets statuses", () => {
    expect(getSelected("status", filters)).toEqual(["DONE"]);
  });

  it("gets assignees", () => {
    expect(getSelected("assignee", filters)).toEqual(["alice"]);
  });

  it("gets priorities as strings", () => {
    expect(getSelected("priority", filters)).toEqual(["1", "2"]);
  });

  it("gets labels", () => {
    expect(getSelected("labels", filters)).toEqual(["bug"]);
  });

  it("gets epic as single-element array", () => {
    expect(getSelected("epic", filters)).toEqual(["epic-1"]);
  });

  it("returns empty array when epicId is null", () => {
    expect(getSelected("epic", EMPTY_FILTERS)).toEqual([]);
  });
});

describe("setSelected", () => {
  it("sets statuses", () => {
    const result = setSelected("status", EMPTY_FILTERS, ["DONE", "DOING"]);
    expect(result.statuses).toEqual(["DONE", "DOING"]);
  });

  it("sets assignees", () => {
    const result = setSelected("assignee", EMPTY_FILTERS, ["alice"]);
    expect(result.assignees).toEqual(["alice"]);
  });

  it("converts priority strings to numbers", () => {
    const result = setSelected("priority", EMPTY_FILTERS, ["1", "3"]);
    expect(result.priorities).toEqual([1, 3]);
  });

  it("sets labels", () => {
    const result = setSelected("labels", EMPTY_FILTERS, ["bug"]);
    expect(result.labels).toEqual(["bug"]);
  });

  it("sets epicId from first value", () => {
    const result = setSelected("epic", EMPTY_FILTERS, ["epic-1"]);
    expect(result.epicId).toBe("epic-1");
  });

  it("clears epicId when values empty", () => {
    const withEpic = { ...EMPTY_FILTERS, epicId: "epic-1" };
    const result = setSelected("epic", withEpic, []);
    expect(result.epicId).toBeNull();
  });

  it("preserves other fields", () => {
    const base: BacklogFilters = { statuses: ["DONE"], labels: ["bug"], assignees: [], priorities: [], epicId: null, workedBy: [], releases: [] };
    const result = setSelected("assignee", base, ["alice"]);
    expect(result.statuses).toEqual(["DONE"]);
    expect(result.labels).toEqual(["bug"]);
  });
});

describe("workedBy filter", () => {
  const byAgent = { status: "DOING", assignee: "Alice <a@x.com>", assignee_via: "claude-code/2.1.263 <agent@mcp>", priority: 0 };
  const byHand = { status: "DOING", assignee: "Alice <a@x.com>", priority: 0 };
  const unassigned = { status: "DOING", priority: 0 };

  it("counts as an active filter", () => {
    expect(hasActiveFilters({ ...EMPTY_FILTERS, workedBy: ["agent"] })).toBe(true);
  });

  it("matches agent-worked issues", () => {
    const f = { ...EMPTY_FILTERS, workedBy: ["agent"] };
    expect(matchesFilters(byAgent, f)).toBe(true);
    expect(matchesFilters(byHand, f)).toBe(false);
    expect(matchesFilters(unassigned, f)).toBe(false);
  });

  it("matches hand-worked issues", () => {
    const f = { ...EMPTY_FILTERS, workedBy: ["human"] };
    expect(matchesFilters(byAgent, f)).toBe(false);
    expect(matchesFilters(byHand, f)).toBe(true);
    expect(matchesFilters(unassigned, f)).toBe(false);
  });

  it("round-trips through getSelected/setSelected", () => {
    const f = setSelected("workedBy", EMPTY_FILTERS, ["agent"]);
    expect(getSelected("workedBy", f)).toEqual(["agent"]);
  });
});

describe("releases filter", () => {
  const rel = (number: number, overrides: Partial<Release> = {}): Release => ({
    id: `rel-${number}`, number, name: `#${number}`, state: "OPEN", counts: {}, total: 0, done: 0, ...overrides,
  });
  const board: ReleasesResponse = {
    current: rel(1, { slot: "current", name: "1.5" }),
    next: rel(2, { slot: "next", name: "1.6" }),
    upcoming: [rel(3, { slot: "upcoming" })],
    past: [rel(4, { slot: "past", name: "1.4", state: "RELEASED" })],
  };
  const inRel = (id?: string) => ({ status: "PLANNED", priority: 0, release_id: id });
  const f = (...releases: string[]) => ({ ...EMPTY_FILTERS, releases });

  it("counts as an active filter", () => {
    expect(hasActiveFilters(f("@current"))).toBe(true);
  });

  it("resolves @current and @next against the board", () => {
    expect(matchesFilters(inRel("rel-1"), f("@current"), board)).toBe(true);
    expect(matchesFilters(inRel("rel-2"), f("@current"), board)).toBe(false);
    expect(matchesFilters(inRel("rel-2"), f("@next"), board)).toBe(true);
  });

  it("follows Current when it changes", () => {
    const after: ReleasesResponse = { ...board, current: board.next, next: board.upcoming[0], upcoming: [] };
    expect(matchesFilters(inRel("rel-1"), f("@current"), after)).toBe(false);
    expect(matchesFilters(inRel("rel-2"), f("@current"), after)).toBe(true);
  });

  it("matches issues without a release with none", () => {
    expect(matchesFilters(inRel(undefined), f("none"), board)).toBe(true);
    expect(matchesFilters(inRel(""), f("none"))).toBe(true);
    expect(matchesFilters(inRel("rel-1"), f("none"), board)).toBe(false);
  });

  it("matches a release by number, past releases included", () => {
    expect(matchesFilters(inRel("rel-3"), f("#3"), board)).toBe(true);
    expect(matchesFilters(inRel("rel-4"), f("#4"), board)).toBe(true);
    expect(matchesFilters(inRel("rel-1"), f("#3"), board)).toBe(false);
  });

  it("matches any of several refs", () => {
    expect(matchesFilters(inRel("rel-2"), f("@current", "@next"), board)).toBe(true);
    expect(matchesFilters(inRel(undefined), f("@current", "none"), board)).toBe(true);
  });

  it("matches nothing for a deleted number, a placeholder slot or a missing board", () => {
    expect(matchesFilters(inRel("rel-9"), f("#9"), board)).toBe(false);
    const empty: ReleasesResponse = { ...board, next: rel(5, { id: undefined, placeholder: true, slot: "next" }) };
    expect(matchesFilters(inRel(undefined), f("@next"), empty)).toBe(false);
    expect(matchesFilters(inRel("rel-1"), f("@current"))).toBe(false);
  });

  it("does not put an issue reopened in a past release in @current", () => {
    expect(matchesFilters({ ...inRel("rel-4"), status: "DOING" }, f("@current"), board)).toBe(false);
  });

  it("round-trips through getSelected/setSelected", () => {
    expect(getSelected("release", setSelected("release", EMPTY_FILTERS, ["@next", "#3"]))).toEqual(["@next", "#3"]);
  });

  it("offers Current, Next, No release, then Upcoming and Past releases, with counts", () => {
    const issues = [inRel("rel-1"), inRel("rel-1"), inRel("rel-3"), inRel(undefined)];
    expect(releaseFilterOptions(board, issues)).toEqual([
      { value: "@current", label: "Current · 1.5", count: 2 },
      { value: "@next", label: "Next · 1.6", count: 0 },
      { value: "none", label: "No release", count: 1 },
      { value: "#3", label: "#3", count: 1 },
      { value: "#4", label: "1.4", count: 0 },
    ]);
  });

  it("offers only No release before the board loads", () => {
    expect(releaseFilterOptions(undefined, [inRel(undefined)])).toEqual([{ value: "none", label: "No release", count: 1 }]);
  });

  it("names refs for chips and menus", () => {
    expect(releaseRefName("@current", board)).toBe("Current");
    expect(releaseRefName("@next", board)).toBe("Next");
    expect(releaseRefName("none", board)).toBe("No release");
    expect(releaseRefName("#4", board)).toBe("1.4");
    expect(releaseRefName("#9", board)).toBe("#9");
    expect(releaseRefName("#4")).toBe("#4");
  });
});

describe("filtersForScope", () => {
  const ACTIVE = ["PLANNED", "DOING", "BLOCKED"];

  it("keeps the selected statuses inside the tab's scope", () => {
    expect(filtersForScope({ ...EMPTY_FILTERS, statuses: ["DONE", "DOING"] }, ACTIVE).statuses).toEqual(["DOING"]);
  });

  it("ignores the Status filter when none of its values are in scope", () => {
    expect(filtersForScope({ ...EMPTY_FILTERS, statuses: ["DONE"] }, ACTIVE).statuses).toEqual([]);
  });

  it("leaves the other dimensions alone", () => {
    const f = { ...EMPTY_FILTERS, statuses: ["DONE"], labels: ["bug"], releases: ["@current"] };
    expect(filtersForScope(f, ACTIVE)).toEqual({ ...f, statuses: [] });
  });

  it("marks only status values outside the scope as not applying", () => {
    expect(outOfScope("status", "DONE", ACTIVE)).toBe(true);
    expect(outOfScope("status", "DOING", ACTIVE)).toBe(false);
    expect(outOfScope("labels", "DONE", ACTIVE)).toBe(false);
    expect(outOfScope("status", "DONE")).toBe(false);
  });

  it("disables the Status entry only for a single-status scope", () => {
    expect(dimensionDisabled("status", ["BACKLOG"])).toBe(true);
    expect(dimensionDisabled("status", ACTIVE)).toBe(false);
    expect(dimensionDisabled("status")).toBe(false);
    expect(dimensionDisabled("labels", ["BACKLOG"])).toBe(false);
  });

  it("returns the same filters without a status filter", () => {
    const f = { ...EMPTY_FILTERS, labels: ["bug"] };
    expect(filtersForScope(f, ACTIVE)).toBe(f);
  });
});

describe("parseStoredFilters", () => {
  it("fills dimensions missing from filters saved by older versions", () => {
    const f = parseStoredFilters(JSON.stringify({ statuses: ["DOING"], labels: [], assignees: [], priorities: [], epicId: null }));
    expect(f.statuses).toEqual(["DOING"]);
    expect(f.workedBy).toEqual([]);
    expect(f.releases).toEqual([]);
  });

  it("falls back to empty filters on missing or corrupt input", () => {
    expect(parseStoredFilters(null)).toEqual(EMPTY_FILTERS);
    expect(parseStoredFilters("{nope")).toEqual(EMPTY_FILTERS);
  });
});

describe("matchesSearch", () => {
  const issue = { id: "xpo-a1b2c3", title: "Fix Login Redirect", labels: ["bug", "Frontend"] };

  it("matches everything for an empty or whitespace-only query", () => {
    expect(matchesSearch(issue, "")).toBe(true);
    expect(matchesSearch(issue, "   ")).toBe(true);
  });

  it("matches a title substring case-insensitively", () => {
    expect(matchesSearch(issue, "login redir")).toBe(true);
  });

  it("matches an ID substring", () => {
    expect(matchesSearch(issue, "A1B2")).toBe(true);
  });

  it("matches a label substring", () => {
    expect(matchesSearch(issue, "front")).toBe(true);
  });

  it("trims surrounding whitespace from the query", () => {
    expect(matchesSearch(issue, "  bug  ")).toBe(true);
  });

  it("handles issues without labels", () => {
    expect(matchesSearch({ id: "xpo-1", title: "Thing" }, "bug")).toBe(false);
  });

  it("returns false when nothing matches", () => {
    expect(matchesSearch(issue, "zzz")).toBe(false);
  });
});
