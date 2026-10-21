import { describe, expect, it } from "vitest";
import { fieldTriggerClass } from "./field-trigger";

describe("fieldTriggerClass", () => {
  it("is a light wash of the text colour with a tint border, so it shows on any surface", () => {
    const cls = fieldTriggerClass(false);
    expect(cls).toContain("bg-[var(--color-field-rest)]");
    expect(cls).toContain("hover:bg-[var(--color-field-hover)]");
    expect(cls).toContain("border-[var(--color-tint-border)]");
  });

  it("swaps the border for red when a required field is empty", () => {
    const cls = fieldTriggerClass(true);
    expect(cls).toContain("border-[var(--color-error)]/40");
    expect(cls).not.toContain("border-[var(--color-tint-border)]");
  });
});
