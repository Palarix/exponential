import { describe, expect, it } from "vitest";
import { DIALOG_LAYER, POPOVER_LAYER } from "./layers";

const z = (cls: string) => Number(cls.match(/^z-\[(\d+)\]$/)?.[1]);

describe("layers", () => {
  it("opens popovers above dialogs, so a dropdown in a dialog shows", () => {
    expect(z(POPOVER_LAYER)).toBeGreaterThan(z(DIALOG_LAYER));
  });

  it("keeps popovers below toasts and context menus (z-[100])", () => {
    expect(z(POPOVER_LAYER)).toBeLessThan(100);
  });
});
