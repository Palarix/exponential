import { describe, expect, it } from "vitest";
import { cardPopoverAnchorSelector } from "./board-popover";

describe("cardPopoverAnchorSelector", () => {
  it("anchors status, estimate and labels popovers to the card", () => {
    for (const type of ["status", "estimate", "labels"] as const) {
      expect(cardPopoverAnchorSelector("xpo-aaa111", type)).toBe('[data-board-card="xpo-aaa111"]');
    }
  });

  it("anchors the assignee popover to the card's avatar trigger", () => {
    expect(cardPopoverAnchorSelector("xpo-aaa111", "assignee")).toBe(
      '[data-board-card="xpo-aaa111"] [data-assignee-trigger]',
    );
  });
});
