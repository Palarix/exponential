// @vitest-environment jsdom
import { act } from "react";
import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import * as TooltipPrimitive from "@radix-ui/react-tooltip";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { Milestone } from "../../api/types";
import { KeyboardNavProvider } from "../../keyboard";
import { mountDom, setupDom } from "../../test-dom";
import { queryKeys } from "../../api/query-keys";
import { MilestoneProgress, PastBurnUp } from "./MilestoneRow";

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

describe("PastBurnUp", () => {
  const t = (scope: number, open: number) => ({
    scope, open, completed: scope - open, scope_points: scope, open_points: open, completed_points: scope - open,
  });
  // Completed within one day: one day, three steps.
  const progress = {
    days: [{ date: "2026-08-20", ...t(2, 2) }],
    steps: [
      { at: "2026-08-20T09:00:00Z", ...t(2, 2), changes: [] },
      { at: "2026-08-20T10:00:00Z", ...t(2, 1), changes: [] },
      { at: "2026-08-20T11:00:00Z", ...t(2, 0), changes: [] },
    ],
  };
  const completed: Milestone = { ...milestone, state: "COMPLETED", slot: "past", started_at: "2026-08-20T09:00:00Z", completed_at: "2026-08-20T11:00:00Z" };

  beforeEach(() => localStorage.clear());

  function mount() {
    const client = new QueryClient({ defaultOptions: { queries: { staleTime: Infinity, retry: false } } });
    client.setQueryData(queryKeys.milestoneProgress("#2"), progress);
    const mounted = mountDom(<QueryClientProvider client={client}><PastBurnUp milestone={completed} /></QueryClientProvider>);
    cleanup = mounted.cleanup;
    return mounted.host;
  }

  it("keeps the Time rule: no chart under two days, with the switch back to Order", () => {
    localStorage.setItem("exponential-milestones-burnup-axis", "time");
    const host = mount();
    expect(host.querySelector('[aria-label="Burn-up"]')).toBeNull();
    expect(host.textContent).toContain("Completed within one day");
    act(() => host.querySelector<HTMLInputElement>('[aria-label="Burn-up axis"] input[value="order"]')!.click());
    expect(host.querySelector('[aria-label="Burn-up"]')).not.toBeNull();
    expect(host.textContent).toContain("3 steps");
  });
});
