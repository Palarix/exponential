import { describe, expect, it } from "vitest";
import { hasMoreBelow } from "./scroll";

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
