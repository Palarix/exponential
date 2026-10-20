import { describe, expect, it } from "vitest";
import { motionHint, notifyCopy } from "./preferences-utils";

describe("motionHint", () => {
  it("says what the system asks for under System default", () => {
    expect(motionHint("system", true)).toBe("Your system asks for reduced motion, so movement becomes a fade.");
    expect(motionHint("system", false)).toBe("Your system allows motion, so animations play.");
  });

  it("explains On and Off, whatever the system says", () => {
    expect(motionHint("on", true)).toBe("Animations always play, even if your system asks for reduced motion.");
    expect(motionHint("off", false)).toBe("No movement or fades. Loading and progress indicators still animate.");
  });
});

describe("notifyCopy", () => {
  it("offers the switch when the browser can notify", () => {
    expect(notifyCopy("on")).toEqual({ switchable: true });
    expect(notifyCopy("off")).toEqual({ switchable: true });
  });

  it("explains instead of a switch when it cannot", () => {
    expect(notifyCopy("blocked")).toEqual({ switchable: false, status: "Blocked by the browser", hint: "Allow notifications for this site in your browser's settings." });
    expect(notifyCopy("unsupported")).toEqual({ switchable: false, status: "Not supported" });
  });
});
