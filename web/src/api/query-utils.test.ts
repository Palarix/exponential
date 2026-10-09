import { describe, expect, it } from "vitest";
import { deriveAppConfig, effectiveLabelColors, withConfigLabels, patchIssueList, cyclesFromResponse, firstQueryError, type RawConfig } from "./query-utils";
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

describe("patchIssueList", () => {
  it("merges the patch into the matching issue only", () => {
    const a = makeIssue({ id: "a", title: "A" });
    const b = makeIssue({ id: "b", title: "B" });
    const next = patchIssueList([a, b], "b", { title: "B2" });
    expect(next?.[0]).toBe(a);
    expect(next?.[1]).toEqual({ ...b, title: "B2" });
  });

  it("leaves an unloaded list alone", () => {
    expect(patchIssueList(undefined, "a", { title: "x" })).toBeUndefined();
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
