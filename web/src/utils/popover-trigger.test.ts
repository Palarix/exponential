import { describe, expect, it } from "vitest";
import { popoverTriggerProps } from "./popover-trigger";

describe("popoverTriggerProps", () => {
  it("reports expanded only for the open popover", () => {
    expect(popoverTriggerProps("status", "status")).toEqual({ "aria-haspopup": "menu", "aria-expanded": true });
    expect(popoverTriggerProps("status", "priority")).toEqual({ "aria-haspopup": "menu", "aria-expanded": false });
    expect(popoverTriggerProps("status", null)).toEqual({ "aria-haspopup": "menu", "aria-expanded": false });
  });
});
