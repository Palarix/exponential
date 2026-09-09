import { describe, it, expect } from "vitest";
import { hasActiveFilters, matchesFilters, chipLabel, getSelected, setSelected, EMPTY_FILTERS, type BacklogFilters } from "./filters";

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
    const base: BacklogFilters = { statuses: ["DONE"], labels: ["bug"], assignees: [], priorities: [], epicId: null };
    const result = setSelected("assignee", base, ["alice"]);
    expect(result.statuses).toEqual(["DONE"]);
    expect(result.labels).toEqual(["bug"]);
  });
});
