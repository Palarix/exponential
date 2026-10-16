import { describe, it, expect } from "vitest";
import { isTerminal, isCompleted, LABEL_PRESET_COLORS } from "./constants";

describe("isTerminal", () => {
  it("returns true for terminal statuses", () => {
    expect(isTerminal("DONE")).toBe(true);
    expect(isTerminal("CANCELED")).toBe(true);
    expect(isTerminal("DUPLICATE")).toBe(true);
  });

  it("returns false for non-terminal statuses", () => {
    expect(isTerminal("BACKLOG")).toBe(false);
    expect(isTerminal("PLANNED")).toBe(false);
    expect(isTerminal("DOING")).toBe(false);
    expect(isTerminal("BLOCKED")).toBe(false);
  });

  it("returns false for unknown statuses", () => {
    expect(isTerminal("")).toBe(false);
    expect(isTerminal("done")).toBe(false);
  });
});

describe("isCompleted", () => {
  it("returns true only for DONE", () => {
    expect(isCompleted("DONE")).toBe(true);
  });

  it("returns false for other terminal statuses", () => {
    expect(isCompleted("CANCELED")).toBe(false);
    expect(isCompleted("DUPLICATE")).toBe(false);
  });

  it("returns false for non-terminal statuses", () => {
    expect(isCompleted("BACKLOG")).toBe(false);
    expect(isCompleted("DOING")).toBe(false);
  });
});

describe("LABEL_PRESET_COLORS", () => {
  const builtin = ["#eb5757", "#b36cd9", "#5e6ad2", "#4da6e8"];
  const rgb = (hex: string) => [1, 3, 5].map((i) => parseInt(hex.slice(i, i + 2), 16));
  const distance = (a: string, b: string) => Math.hypot(...rgb(a).map((v, i) => v - rgb(b)[i]));

  it("keeps clear of the built-in label colors", () => {
    for (const c of LABEL_PRESET_COLORS) {
      for (const b of builtin) expect(distance(c, b), `${c} vs ${b}`).toBeGreaterThan(60);
    }
  });

  it("makes every swatch clearly different from the others", () => {
    LABEL_PRESET_COLORS.forEach((a, i) => {
      for (const b of LABEL_PRESET_COLORS.slice(i + 1)) expect(distance(a, b), `${a} vs ${b}`).toBeGreaterThan(60);
    });
  });
});
