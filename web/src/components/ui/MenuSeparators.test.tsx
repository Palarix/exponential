import { describe, expect, it } from "vitest";
import { renderMarkup } from "../../render-markup";
import { MenuDivider, MenuFilter } from "./Menu";

describe("menu separators", () => {
  // A fixed border color can match or undercut the menu's surface; the tint lifts off any surface.
  it("draws MenuDivider with the surface-mixed tint", () => {
    const html = renderMarkup(<MenuDivider />);
    expect(html).toContain('role="separator"');
    expect(html).toContain("border-[var(--color-tint-border)]");
    expect(html).not.toContain("--color-border-subtle");
  });

  it("draws MenuFilter's line as the same divider", () => {
    const html = renderMarkup(<MenuFilter value="" onChange={() => {}} />);
    expect(html).toContain('role="separator"');
    expect(html).toContain("border-[var(--color-tint-border)]");
    expect(html).not.toContain("--color-border-subtle");
  });
});
