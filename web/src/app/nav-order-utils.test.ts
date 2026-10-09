import { describe, expect, it } from "vitest";
import { resolveNavOrder } from "./nav-order-utils";

const fallback = ["a", "b", "c", "d"];

describe("resolveNavOrder", () => {
  it("uses the published order when it contains the issue", () => {
    expect(resolveNavOrder(["c", "a"], fallback, "a")).toEqual(["c", "a"]);
  });

  it("falls back when nothing is published", () => {
    expect(resolveNavOrder(null, fallback, "b")).toBe(fallback);
  });

  it("falls back when the published order does not contain the issue", () => {
    expect(resolveNavOrder(["c", "d"], fallback, "a")).toBe(fallback);
  });

  it("falls back when the published order is empty", () => {
    expect(resolveNavOrder([], fallback, "a")).toBe(fallback);
  });

  it("returns the published order when no issue is selected", () => {
    expect(resolveNavOrder(["c"], fallback, null)).toEqual(["c"]);
  });
});
