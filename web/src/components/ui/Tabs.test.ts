import { describe, expect, it } from "vitest";
import { cycleTabs } from "./tabs-utils";

const FOUR_TABS = ["all", "active", "backlog", "done"] as const;
const TWO_TABS = ["assigned", "created"] as const;

describe("cycleTabs", () => {
  describe("] (next)", () => {
    it("advances to the next tab", () => {
      expect(cycleTabs(FOUR_TABS, "all", 1)).toBe("active");
      expect(cycleTabs(FOUR_TABS, "active", 1)).toBe("backlog");
      expect(cycleTabs(FOUR_TABS, "backlog", 1)).toBe("done");
    });

    it("wraps from last to first", () => {
      expect(cycleTabs(FOUR_TABS, "done", 1)).toBe("all");
    });

    it("works with two tabs", () => {
      expect(cycleTabs(TWO_TABS, "assigned", 1)).toBe("created");
      expect(cycleTabs(TWO_TABS, "created", 1)).toBe("assigned");
    });
  });

  describe("[ (previous)", () => {
    it("goes to the previous tab", () => {
      expect(cycleTabs(FOUR_TABS, "done", -1)).toBe("backlog");
      expect(cycleTabs(FOUR_TABS, "backlog", -1)).toBe("active");
      expect(cycleTabs(FOUR_TABS, "active", -1)).toBe("all");
    });

    it("wraps from first to last", () => {
      expect(cycleTabs(FOUR_TABS, "all", -1)).toBe("done");
    });
  });

  describe("edge cases", () => {
    it("returns null for a single item", () => {
      expect(cycleTabs(["only"] as const, "only", 1)).toBeNull();
      expect(cycleTabs(["only"] as const, "only", -1)).toBeNull();
    });

    it("returns null for an empty list", () => {
      expect(cycleTabs([] as unknown as readonly string[], "x", 1)).toBeNull();
    });

    it("returns null when activeId is not in the list", () => {
      expect(cycleTabs(FOUR_TABS, "bogus", 1)).toBeNull();
      expect(cycleTabs(FOUR_TABS, "bogus", -1)).toBeNull();
    });
  });
});
