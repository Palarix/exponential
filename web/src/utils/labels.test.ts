import { describe, it, expect } from "vitest";
import {
  canonicalLabel,
  labelColor,
  toggleLabel,
  deduplicateLabels,
  mergeAndSort,
  splitLabels,
  contrastTextColor,
  filterLabelsByName,
  isBuiltinLabel,
  partitionBuiltinLabels,
  recentIssuesWithLabel,
} from "./labels";

const CONFIG: Record<string, string> = {
  bug: "#e06091",
  feature: "#26b5b0",
  Epic: "#d4a030",
};

describe("canonicalLabel", () => {
  it("returns the canonical key for a case-insensitive match", () => {
    expect(canonicalLabel("BUG", CONFIG)).toBe("bug");
    expect(canonicalLabel("Feature", CONFIG)).toBe("feature");
    expect(canonicalLabel("epic", CONFIG)).toBe("Epic");
  });

  it("returns the input unchanged when no match", () => {
    expect(canonicalLabel("unknown", CONFIG)).toBe("unknown");
  });
});

describe("labelColor", () => {
  it("returns the color for a matching label", () => {
    expect(labelColor("bug", CONFIG)).toBe("#e06091");
    expect(labelColor("BUG", CONFIG)).toBe("#e06091");
  });

  it("returns the CSS var fallback when no match", () => {
    expect(labelColor("nope", CONFIG)).toBe("var(--color-text-muted)");
  });
});

describe("toggleLabel", () => {
  it("adds a label when not present", () => {
    expect(toggleLabel([], "bug")).toEqual(["bug"]);
    expect(toggleLabel(["feature"], "bug")).toEqual(["feature", "bug"]);
  });

  it("removes a label case-insensitively", () => {
    expect(toggleLabel(["bug", "feature"], "BUG")).toEqual(["feature"]);
  });

  it("removes case-insensitively when label is present with different case", () => {
    expect(toggleLabel(["feature"], "Feature")).toEqual([]);
  });
});

describe("deduplicateLabels", () => {
  it("removes case-insensitive duplicates and canonicalizes", () => {
    expect(deduplicateLabels(["bug", "BUG", "feature"], CONFIG)).toEqual([
      "bug",
      "feature",
    ]);
  });

  it("preserves order of first occurrence", () => {
    expect(
      deduplicateLabels(["feature", "bug", "FEATURE"], CONFIG),
    ).toEqual(["feature", "bug"]);
  });

  it("handles labels not in config", () => {
    expect(deduplicateLabels(["custom", "CUSTOM"], CONFIG)).toEqual([
      "custom",
    ]);
  });
});

describe("mergeAndSort", () => {
  it("deduplicates and sorts locale-insensitively", () => {
    expect(mergeAndSort(CONFIG, ["feature", "bug", "FEATURE"])).toEqual([
      "bug",
      "feature",
    ]);
  });

  it("returns empty for empty input", () => {
    expect(mergeAndSort(CONFIG, [])).toEqual([]);
  });
});

describe("splitLabels", () => {
  const defaults = [
    { name: "bug", color: "#e06091" },
    { name: "feature", color: "#26b5b0" },
  ];

  it("splits into primary and metadata", () => {
    const result = splitLabels(["bug", "custom", "feature"], defaults);
    expect(result.primary).toEqual(["bug", "feature"]);
    expect(result.metadata).toEqual(["custom"]);
  });

  it("sorts primary by default order, metadata alphabetically", () => {
    const result = splitLabels(
      ["feature", "zebra", "bug", "alpha"],
      defaults,
    );
    expect(result.primary).toEqual(["bug", "feature"]);
    expect(result.metadata).toEqual(["alpha", "zebra"]);
  });

  it("handles empty input", () => {
    const result = splitLabels([], defaults);
    expect(result.primary).toEqual([]);
    expect(result.metadata).toEqual([]);
  });

  it("is case-insensitive on default matching", () => {
    const result = splitLabels(["BUG"], defaults);
    expect(result.primary).toEqual(["BUG"]);
    expect(result.metadata).toEqual([]);
  });
});

describe("contrastTextColor", () => {
  it("returns black for light backgrounds", () => {
    expect(contrastTextColor("#ffffff")).toBe("#000000");
    expect(contrastTextColor("#d4a030")).toBe("#000000");
  });

  it("returns white for dark backgrounds", () => {
    expect(contrastTextColor("#000000")).toBe("#ffffff");
    expect(contrastTextColor("#1a1a2e")).toBe("#ffffff");
  });

  it("handles hex without #", () => {
    expect(contrastTextColor("ffffff")).toBe("#000000");
  });

  it("returns correct contrast for mid-range colors", () => {
    expect(contrastTextColor("#808080")).toBe("#000000");
    expect(contrastTextColor("#333333")).toBe("#ffffff");
  });
});

describe("filterLabelsByName", () => {
  const LABELS = [{ name: "bug" }, { name: "Feature" }, { name: "frontend" }, { name: "Epic" }];

  it("returns all labels for an empty query", () => {
    expect(filterLabelsByName(LABELS, "")).toBe(LABELS);
  });

  it("returns all labels for a whitespace-only query", () => {
    expect(filterLabelsByName(LABELS, "   ")).toBe(LABELS);
  });

  it("matches case-insensitively", () => {
    expect(filterLabelsByName(LABELS, "EPIC")).toEqual([{ name: "Epic" }]);
  });

  it("matches substrings anywhere in the name", () => {
    expect(filterLabelsByName(LABELS, "e")).toEqual([
      { name: "Feature" },
      { name: "frontend" },
      { name: "Epic" },
    ]);
    expect(filterLabelsByName(LABELS, "ont")).toEqual([{ name: "frontend" }]);
  });

  it("trims surrounding whitespace from the query", () => {
    expect(filterLabelsByName(LABELS, "  bug ")).toEqual([{ name: "bug" }]);
  });

  it("returns an empty list when nothing matches", () => {
    expect(filterLabelsByName(LABELS, "zzz")).toEqual([]);
  });
});

describe("isBuiltinLabel", () => {
  const defaults = [{ name: "Bug", color: "#f00" }, { name: "feature", color: "" }];

  it("matches default labels case-insensitively", () => {
    expect(isBuiltinLabel("bug", defaults)).toBe(true);
    expect(isBuiltinLabel("FEATURE", defaults)).toBe(true);
  });

  it("rejects other labels", () => {
    expect(isBuiltinLabel("ux", defaults)).toBe(false);
    expect(isBuiltinLabel("bug", [])).toBe(false);
  });
});

describe("partitionBuiltinLabels", () => {
  const defaults = [{ name: "feature", color: "" }, { name: "Bug", color: "#f00" }];

  it("orders built-ins by default_labels and keeps the custom order", () => {
    const labels = [{ name: "api" }, { name: "bug" }, { name: "Feature" }, { name: "ux" }];
    const { builtin, custom } = partitionBuiltinLabels(labels, defaults);
    expect(builtin.map((l) => l.name)).toEqual(["Feature", "bug"]);
    expect(custom.map((l) => l.name)).toEqual(["api", "ux"]);
  });

  it("puts everything in custom when there are no defaults", () => {
    const { builtin, custom } = partitionBuiltinLabels([{ name: "bug" }], []);
    expect(builtin).toEqual([]);
    expect(custom).toEqual([{ name: "bug" }]);
  });
});

describe("recentIssuesWithLabel", () => {
  const issues = [
    { id: "a", labels: ["UX"], updated_at: "2026-01-01T00:00:00Z" },
    { id: "b", labels: ["bug"], updated_at: "2026-03-01T00:00:00Z" },
    { id: "c", labels: ["ux", "bug"], updated_at: "2026-02-01T00:00:00Z" },
    { id: "d", labels: ["ux"], updated_at: "2025-01-01T00:00:00Z" },
    { id: "e", updated_at: "2026-04-01T00:00:00Z" },
  ];

  it("returns the most recently updated issues with the label, case-insensitively", () => {
    expect(recentIssuesWithLabel(issues, "ux", 2).map((i) => i.id)).toEqual(["c", "a"]);
  });

  it("returns none when no issue has the label", () => {
    expect(recentIssuesWithLabel(issues, "infra", 2)).toEqual([]);
  });
});
