import { describe, expect, it } from "vitest";
import { deriveAppConfig, effectiveLabelColors, withConfigLabels, applyIssuePatches, renameLabel, removeLabel, cyclesFromResponse, firstQueryError, type RawConfig } from "./query-utils";
import { makeIssue } from "../test-utils";
import { ApiError } from "./client";

const raw: RawConfig = {
  auto_commit: false,
  prefix: "xpo-",
  version: "1.2.3",
  labels: { ui: "#f00", Bug: "#0f0" },
  name: "exponential",
  hide_default_labels: false,
  default_labels: ["bug", "feature"],
  contributors: ["A <a@x>"],
  cycles: { enabled: true, duration: "1w", start_day: "monday", anchor_date: "2026-01-05" },
};

describe("deriveAppConfig", () => {
  it("uses App's previous defaults before config has loaded", () => {
    expect(deriveAppConfig(undefined)).toEqual({
      prefix: "issue-",
      version: "",
      projectName: "",
      configLabels: {},
      contributors: [],
      hideDefaultLabels: false,
      cyclesEnabled: false,
      defaultLabels: [],
    });
  });

  it("derives fields from loaded config", () => {
    const c = deriveAppConfig(raw);
    expect(c.prefix).toBe("xpo-");
    expect(c.version).toBe("1.2.3");
    expect(c.projectName).toBe("exponential");
    expect(c.contributors).toEqual(["A <a@x>"]);
    expect(c.cyclesEnabled).toBe(true);
    expect(c.configLabels).toEqual({ ui: "#f00", Bug: "#0f0" });
  });

  it("colors default labels by exact, then lowercase, name", () => {
    const c = deriveAppConfig({ ...raw, labels: { bug: "#0f0" }, default_labels: ["Bug", "feature"] });
    expect(c.defaultLabels).toEqual([{ name: "Bug", color: "#0f0" }, { name: "feature", color: "" }]);
  });

  it("tolerates missing optional fields", () => {
    const c = deriveAppConfig({ ...raw, labels: undefined as never, contributors: undefined, cycles: undefined, default_labels: undefined as never, version: "" });
    expect(c.configLabels).toEqual({});
    expect(c.contributors).toEqual([]);
    expect(c.cyclesEnabled).toBe(false);
    expect(c.defaultLabels).toEqual([]);
  });
});

describe("effectiveLabelColors", () => {
  it("layers config labels over default label colors", () => {
    const c = deriveAppConfig({ ...raw, labels: { bug: "#111", ui: "#222" }, default_labels: ["bug", "feature"] });
    expect(effectiveLabelColors({ ...c, defaultLabels: [{ name: "bug", color: "#aaa" }, { name: "feature", color: "#bbb" }] }))
      .toEqual({ bug: "#111", feature: "#bbb", ui: "#222" });
  });

  it("ignores default labels when hidden", () => {
    const c = { ...deriveAppConfig(raw), hideDefaultLabels: true };
    expect(effectiveLabelColors(c)).toEqual(c.configLabels);
  });
});

describe("withConfigLabels", () => {
  it("replaces labels on loaded config", () => {
    expect(withConfigLabels(raw, { x: "#000" })).toEqual({ ...raw, labels: { x: "#000" } });
  });

  it("leaves unloaded config alone", () => {
    expect(withConfigLabels(undefined, { x: "#000" })).toBeUndefined();
  });
});

describe("applyIssuePatches", () => {
  it("merges each patch into its issue and leaves the others untouched", () => {
    const a = makeIssue({ id: "a", title: "A" });
    const b = makeIssue({ id: "b", title: "B" });
    const c = makeIssue({ id: "c", status: "PLANNED" });
    const { next } = applyIssuePatches([a, b, c], [
      { issueId: "b", patch: { title: "B2" } },
      { issueId: "c", patch: { status: "DOING" } },
    ]);
    expect(next?.[0]).toBe(a);
    expect(next?.[1]).toEqual({ ...b, title: "B2" });
    expect(next?.[2]).toEqual({ ...c, status: "DOING" });
  });

  it("returns an undo holding the previous value of each patched field", () => {
    const a = makeIssue({ id: "a", status: "PLANNED", sort_order: "a0", title: "A" });
    const { undo } = applyIssuePatches([a], [{ issueId: "a", patch: { status: "DOING", sort_order: "a5" } }]);
    expect(undo).toEqual([{ issueId: "a", patch: { status: "PLANNED", sort_order: "a0" } }]);
  });

  it("undoes in reverse order so repeated patches to one issue restore the original", () => {
    const a = makeIssue({ id: "a", status: "BACKLOG" });
    const first = applyIssuePatches([a], [
      { issueId: "a", patch: { status: "PLANNED" } },
      { issueId: "a", patch: { status: "DOING" } },
    ]);
    expect(first.next?.[0].status).toBe("DOING");
    expect(applyIssuePatches(first.next, first.undo).next?.[0].status).toBe("BACKLOG");
  });

  it("rolls back only its own fields, keeping a later concurrent edit", () => {
    const a = makeIssue({ id: "a", status: "PLANNED", estimate: 0 });
    const statusEdit = applyIssuePatches([a], [{ issueId: "a", patch: { status: "DOING" } }]);
    const estimateEdit = applyIssuePatches(statusEdit.next, [{ issueId: "a", patch: { estimate: 3 } }]);
    const rolledBack = applyIssuePatches(estimateEdit.next, statusEdit.undo).next?.[0];
    expect(rolledBack?.status).toBe("PLANNED");
    expect(rolledBack?.estimate).toBe(3);
  });

  it("skips patches for issues that aren't in the list", () => {
    const a = makeIssue({ id: "a" });
    const { next, undo } = applyIssuePatches([a], [{ issueId: "missing", patch: { title: "x" } }]);
    expect(next?.[0]).toBe(a);
    expect(undo).toEqual([]);
  });

  it("leaves an unloaded list alone", () => {
    expect(applyIssuePatches(undefined, [{ issueId: "a", patch: { title: "x" } }])).toEqual({ next: undefined, undo: [] });
  });
});

describe("renameLabel", () => {
  it("replaces the old name with the new name and color, keeping the others", () => {
    expect(renameLabel({ ui: "#f00", bug: "#0f0" }, "ui", "frontend", "#00f")).toEqual({ bug: "#0f0", frontend: "#00f" });
  });

  it("recolors in place when the name is unchanged", () => {
    expect(renameLabel({ ui: "#f00" }, "ui", "ui", "#00f")).toEqual({ ui: "#00f" });
  });
});

describe("removeLabel", () => {
  it("drops the named label without mutating the input", () => {
    const labels = { ui: "#f00", bug: "#0f0" };
    expect(removeLabel(labels, "ui")).toEqual({ bug: "#0f0" });
    expect(labels).toEqual({ ui: "#f00", bug: "#0f0" });
  });
});

describe("cyclesFromResponse", () => {
  const cycle = { id: "c1", number: 1, start: "2026-01-05", end: "2026-01-12", status: "current" as const, done: 0, total: 0 };

  it("returns the cycles when enabled", () => {
    expect(cyclesFromResponse({ enabled: true, cycles: [cycle] })).toEqual([cycle]);
  });

  it("returns a stable empty list when disabled, missing or not loaded", () => {
    const empty = cyclesFromResponse(undefined);
    expect(empty).toEqual([]);
    expect(cyclesFromResponse({ enabled: false, cycles: [cycle] })).toBe(empty);
    expect(cyclesFromResponse({ enabled: true, cycles: null as unknown as [] })).toBe(empty);
  });
});

describe("firstQueryError", () => {
  const ok = { status: "success" as const, error: null };
  const apiError = (status: number, body: string) =>
    new ApiError(new Response(body, { status, statusText: "Err" }), body);

  it("is null when every query succeeded", () => {
    expect(firstQueryError([ok, ok])).toBeNull();
  });

  it("skips entries that weren't refetched", () => {
    expect(firstQueryError([false, ok])).toBeNull();
  });

  it("returns the first failure's API message", () => {
    const err = apiError(409, "worktree is locked");
    expect(firstQueryError([ok, { status: "error", error: err }, { status: "error", error: new Error("later") }])).toBe(err.message);
  });

  it("falls back to a generic message for non-API errors", () => {
    expect(firstQueryError([{ status: "error", error: new TypeError("Failed to fetch") }])).toBe("Failed to load");
  });
});
