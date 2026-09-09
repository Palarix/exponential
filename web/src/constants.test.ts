import { describe, it, expect } from "vitest";
import { isTerminal, isCompleted } from "./constants";

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
