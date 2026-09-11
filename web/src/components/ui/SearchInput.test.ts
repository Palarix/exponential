import { describe, expect, it } from "vitest";
import { searchInputKbdHint } from "./search-input-utils";

describe("searchInputKbdHint", () => {
  it("shows / when value is empty", () => {
    expect(searchInputKbdHint("")).toBe("/");
  });

  it("shows Esc when value is non-empty", () => {
    expect(searchInputKbdHint("hello")).toBe("Esc");
  });

  it("shows / for whitespace-only value", () => {
    expect(searchInputKbdHint("   ")).toBe("/");
  });
});
