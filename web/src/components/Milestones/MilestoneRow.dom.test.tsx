// @vitest-environment jsdom
import { act } from "react";
import { afterEach, beforeAll, describe, expect, it, vi } from "vitest";
import * as TooltipPrimitive from "@radix-ui/react-tooltip";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { Milestone } from "../../api/types";
import { KeyboardNavProvider } from "../../keyboard";
import { mountDom, setupDom } from "../../test-dom";
import { MilestoneProgress } from "./MilestoneRow";

beforeAll(setupDom);

const milestone: Milestone = {
  id: "ms-2", number: 2, name: "1.1", label: "1.1", state: "OPEN", slot: "next", counts: { PLANNED: 2 }, total: 2, done: 0,
};

let cleanup: (() => void) | undefined;
afterEach(() => {
  cleanup?.();
  cleanup = undefined;
  vi.useRealTimers();
});

/** Rests the pointer on the progress pill past the hover delay; returns the open breakdown, if any. */
function hoverPill(m: Milestone) {
  vi.useFakeTimers();
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, enabled: false } } });
  const mounted = mountDom(
    <QueryClientProvider client={client}>
      <KeyboardNavProvider>
        <TooltipPrimitive.Provider>
          <MilestoneProgress milestone={m} today="2026-10-20" />
        </TooltipPrimitive.Provider>
      </KeyboardNavProvider>
    </QueryClientProvider>,
  );
  cleanup = mounted.cleanup;
  const pill = mounted.host.querySelector("button")!;
  act(() => { pill.dispatchEvent(new MouseEvent("mouseover", { bubbles: true })); });
  act(() => { vi.advanceTimersByTime(500); });
  return document.body.querySelector('[role="tooltip"]');
}

describe("MilestoneProgress", () => {
  it("opens the breakdown when the pointer rests on the pill", () => {
    expect(hoverPill(milestone)).not.toBeNull();
  });

  it("opens no breakdown for a milestone without issues", () => {
    expect(hoverPill({ ...milestone, counts: {}, total: 0 })).toBeNull();
  });
});
