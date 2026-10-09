import { describe, expect, it } from "vitest";
import { deriveAppConfig, effectiveLabelColors, withConfigLabels, patchIssueList, type RawConfig } from "./query-utils";
import { makeIssue } from "../test-utils";

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
