import { describe, expect, it } from "vitest";
import { spinHoldMs } from "./spinner";

const period = 1000;

describe("spinHoldMs", () => {
  it("never shows work that settled before the delay", () => {
    expect(spinHoldMs({ busyMs: 40, visibleMs: null, delay: 150, period })).toBeNull();
  });

  it("gives instant work with no delay one full rotation", () => {
    expect(spinHoldMs({ busyMs: 5, visibleMs: null, delay: 0, period })).toBe(1000);
  });

  it("gives work that outlasted the delay but hadn't shown yet one full rotation", () => {
    expect(spinHoldMs({ busyMs: 150, visibleMs: null, delay: 150, period })).toBe(1000);
  });

  it("finishes the current rotation once visible", () => {
    expect(spinHoldMs({ busyMs: 450, visibleMs: 300, delay: 150, period })).toBe(700);
    expect(spinHoldMs({ busyMs: 2450, visibleMs: 2300, delay: 150, period })).toBe(700);
  });

  it("stops at once on an exact rotation boundary", () => {
    expect(spinHoldMs({ busyMs: 1000, visibleMs: 1000, delay: 0, period })).toBe(0);
  });

  it("completes a first rotation that just started", () => {
    expect(spinHoldMs({ busyMs: 0, visibleMs: 0, delay: 0, period })).toBe(1000);
  });
});
