import { describe, expect, it } from "vitest";
import { renderMarkup } from "../../render-markup";
import NotifyBell from "./NotifyBell";

describe("NotifyBell", () => {
  const noop = () => {};

  it("renders nothing when the browser has no notifications", () => {
    expect(renderMarkup(<NotifyBell state="unsupported" onToggle={noop} />)).toBe("");
  });

  it("is a pressed bell when on, and an unpressed bell-off when off", () => {
    const on = renderMarkup(<NotifyBell state="on" onToggle={noop} />);
    expect(on).toMatch(/aria-pressed="true"/);
    expect(on).toContain("lucide-bell");
    expect(on).not.toContain("lucide-bell-off");
    const off = renderMarkup(<NotifyBell state="off" onToggle={noop} />);
    expect(off).toMatch(/aria-pressed="false"/);
    expect(off).toContain("lucide-bell-off");
  });

  it("explains a blocked permission and is disabled", () => {
    const blocked = renderMarkup(<NotifyBell state="blocked" onToggle={noop} />);
    expect(blocked).toContain("disabled");
    expect(blocked).toMatch(/title="[^"]*blocked[^"]*browser/i);
  });
});
