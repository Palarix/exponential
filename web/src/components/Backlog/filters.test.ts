import { describe, it, expect } from "vitest";
import { hasActiveFilters, matchesFilters, matchesSearch, chipLabel, getSelected, setSelected, parseStoredFilters, EMPTY_FILTERS, type BacklogFilters } from "./filters";

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
    const base: BacklogFilters = { statuses: ["DONE"], labels: ["bug"], assignees: [], priorities: [], epicId: null, workedBy: [] };
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

describe("parseStoredFilters", () => {
  it("fills dimensions missing from filters saved by older versions", () => {
    const f = parseStoredFilters(JSON.stringify({ statuses: ["DOING"], labels: [], assignees: [], priorities: [], epicId: null }));
    expect(f.statuses).toEqual(["DOING"]);
    expect(f.workedBy).toEqual([]);
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
