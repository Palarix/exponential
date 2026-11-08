import { describe, expect, it, vi } from "vitest";
import { renderMarkup } from "../../render-markup";
import { navCommandItems } from "./nav-items";

describe("navCommandItems", () => {
  it("lists the sidebar's views in sidebar order, with the sidebar's names", () => {
    expect(navCommandItems("", false, vi.fn()).map((i) => i.label)).toEqual([
      "Overview", "Issues", "Board", "Milestones", "Timeline", "Labels", "Dependencies", "My Issues", "Notifications",
    ]);
  });

  it("lists Cycles only when cycles are enabled", () => {
    const labels = navCommandItems("", true, vi.fn()).map((i) => i.label);
    expect(labels.slice(0, 5)).toEqual(["Overview", "Issues", "Board", "Cycles", "Milestones"]);
  });

  it("leaves out Agent, Drafts and Preferences", () => {
    const labels = navCommandItems("", true, vi.fn()).map((i) => i.label);
    for (const hidden of ["Agent", "Drafts", "Preferences"]) expect(labels).not.toContain(hidden);
  });

  it("groups under Navigation, with the sidebar's icon", () => {
    const [milestones] = navCommandItems("milestones", false, vi.fn());
    expect(milestones.group).toBe("Navigation");
    expect(milestones.label).toBe("Milestones");
    expect(renderMarkup(<>{milestones.icon}</>)).toContain("lucide-milestone");
  });

  it("narrows by the query and opens the chosen view", () => {
    const go = vi.fn();
    const items = navCommandItems("mile", false, go);
    expect(items.map((i) => i.label)).toEqual(["Milestones"]);
    items[0].action();
    expect(go).toHaveBeenCalledWith("milestones");
  });
});
