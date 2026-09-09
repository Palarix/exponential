import { describe, it, expect } from "vitest";
import {
  canonicalLabel,
  labelColor,
  toggleLabel,
  deduplicateLabels,
  mergeAndSort,
  splitLabels,
  contrastTextColor,
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
