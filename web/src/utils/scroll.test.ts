import { describe, expect, it } from "vitest";
import { hasMoreBelow, nextStick } from "./scroll";

describe("nextStick", () => {
  // A 500px viewport over 1000px of content: the end is at scrollTop 500.
  const at = (top: number) => ({ top, clientHeight: 500, scrollHeight: 1000 });

  it("sticks within 40px of the end", () => {
    expect(nextStick(false, 400, at(470))).toBe(true);
    expect(nextStick(false, 400, at(500))).toBe(true);
  });

  it("unsticks when the user scrolls up out of the grace zone", () => {
    expect(nextStick(true, 500, at(300))).toBe(false);
  });

  it("stays stuck when content grows below without an upward scroll", () => {
    // scrollTop unchanged, but the end moved 300px down.
    expect(nextStick(true, 500, { top: 500, clientHeight: 500, scrollHeight: 1300 })).toBe(true);
  });

  it("stays stuck on a downward scroll that has not reached the end yet", () => {
    expect(nextStick(true, 100, at(200))).toBe(true);
  });

  it("stays unstuck while scrolled up outside the grace zone", () => {
    expect(nextStick(false, 100, at(200))).toBe(false);
    expect(nextStick(false, 300, at(200))).toBe(false);
  });
});

describe("hasMoreBelow", () => {
  it("is false when content fits", () => {
    expect(hasMoreBelow(0, 500, 500)).toBe(false);
  });

  it("is true when content extends below the visible area", () => {
    expect(hasMoreBelow(0, 500, 800)).toBe(true);
    expect(hasMoreBelow(200, 500, 800)).toBe(true);
  });

  it("is false once scrolled to the end", () => {
    expect(hasMoreBelow(300, 500, 800)).toBe(false);
  });

  it("tolerates sub-pixel scroll positions at the end", () => {
    expect(hasMoreBelow(299.5, 500, 800)).toBe(false);
  });
});
