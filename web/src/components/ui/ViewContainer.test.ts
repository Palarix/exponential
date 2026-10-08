import { describe, expect, it } from "vitest";
import { viewContentClass, viewInnerClass } from "./view-container-utils";

describe("viewContentClass", () => {
  it("uses the scroll-stable utility when scrolling", () => {
    expect(viewContentClass(true)).toBe("flex-1 min-h-0 scroll-stable");
  });

  it("clips overflow when not scrolling", () => {
    expect(viewContentClass(false)).toBe("flex-1 min-h-0 overflow-hidden");
  });

  it("never emits Tailwind overflow-y utilities", () => {
    for (const scroll of [true, false]) {
      expect(viewContentClass(scroll)).not.toMatch(/overflow-y-/);
    }
  });
});

describe("viewInnerClass", () => {
  it("returns null for full-width views", () => {
    expect(viewInnerClass(undefined)).toBeNull();
  });

  it("centers a max-w-7xl column", () => {
    expect(viewInnerClass("7xl")).toBe("mx-auto w-full max-w-7xl");
  });

  it("centers the issue-detail column", () => {
    expect(viewInnerClass("detail")).toBe("mx-auto w-full max-w-[76rem]");
  });
});
