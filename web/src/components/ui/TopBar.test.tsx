import { describe, expect, it } from "vitest";
import { renderMarkup } from "../../render-markup";
import { TopBar } from "./TopBar";

/** The class of each slot, in order: left, center (if any), right. */
const slots = (html: string) => [...html.matchAll(/<div class="([^"]*)" data-slot="(\w+)"/g)].map(([, cls, slot]) => ({ slot, cls }));

describe("TopBar", () => {
  it("lets the search shrink first: the sides keep their content width", () => {
    const html = renderMarkup(<TopBar left={<h1>Milestones</h1>} center={<input />} right={<button>New</button>} />);
    const [left, center, right] = slots(html);
    expect(center.slot).toBe("center");
    expect(center.cls).toContain("min-w-0");
    for (const side of [left, right]) {
      expect(side.cls).toContain("shrink-0");
      expect(side.cls).not.toContain("min-w-0");
    }
  });

  it("lets the sides truncate when there is no center slot", () => {
    const html = renderMarkup(<TopBar left={<h1>A long breadcrumb</h1>} right={<button>New</button>} />);
    const sides = slots(html);
    expect(sides.map((s) => s.slot)).toEqual(["left", "right"]);
    for (const side of sides) expect(side.cls).toContain("min-w-0");
  });
});
