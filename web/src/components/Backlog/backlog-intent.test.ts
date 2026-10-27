import { describe, expect, it } from "vitest";
import { intentTarget } from "./backlog-intent";
import { EMPTY_FILTERS } from "./filters";

const current = { ...EMPTY_FILTERS, assignees: ["alice"] };

describe("intentTarget", () => {
  it("stays on the open tab without a tab in the intent", () => {
    expect(intentTarget({ filters: { labels: ["ux"] } }, "active", current)).toEqual({
      tab: "active",
      filters: { ...current, labels: ["ux"] },
    });
  });

  it("switches to the intent's tab and merges into the shared filters", () => {
    expect(intentTarget({ tab: "all", filters: { milestones: ["@current"] } }, "active", current)).toEqual({
      tab: "all",
      filters: { ...current, milestones: ["@current"] },
    });
  });

  it("leaves the filters alone without filters in the intent", () => {
    expect(intentTarget({ tab: "done" }, "all", current)).toEqual({ tab: "done", filters: null });
  });
});
