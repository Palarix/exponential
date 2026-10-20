import { describe, expect, it } from "vitest";
import { easeToward, hasMoreBelow, nextStick, parseDuration } from "./scroll";

describe("easeToward", () => {
  it("moves part of the way, more for a longer frame", () => {
    const short = easeToward(0, 100, 16, 150);
    const long = easeToward(0, 100, 32, 150);
    expect(short).toBeGreaterThan(0);
    expect(short).toBeLessThan(100);
    expect(long).toBeGreaterThan(short);
  });

  it("does not depend on frame rate: two half frames land where one full frame does", () => {
    const once = easeToward(0, 100, 32, 150);
    const twice = easeToward(easeToward(0, 100, 16, 150), 100, 16, 150);
    expect(twice).toBeCloseTo(once, 6);
  });

  it("snaps to the target within 1px", () => {
    expect(easeToward(99.5, 100, 1, 150)).toBe(100);
  });

  it("moves at least 1px, so a scroller that rounds scrollTop never stalls", () => {
    expect(easeToward(197, 200, 1, 150)).toBe(198);
    expect(easeToward(200, 197, 1, 150)).toBe(199);
  });

  it("never overshoots", () => {
    expect(easeToward(0, 100, 10_000, 150)).toBe(100);
  });

  it("jumps when there is no time constant", () => {
    expect(easeToward(0, 100, 16, 0)).toBe(100);
  });
});

describe("parseDuration", () => {
  it("reads milliseconds and seconds", () => {
    expect(parseDuration(" 150ms", 1)).toBe(150);
    expect(parseDuration("0.2s", 1)).toBe(200);
  });

  it("falls back on anything else", () => {
    expect(parseDuration("", 150)).toBe(150);
    expect(parseDuration("fast", 150)).toBe(150);
  });
});

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
