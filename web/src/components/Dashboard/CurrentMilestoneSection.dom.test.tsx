// @vitest-environment jsdom
import { act } from "react";
import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import * as TooltipPrimitive from "@radix-ui/react-tooltip";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { AppNavContext, type AppNav } from "../../app/contexts";
import { queryKeys } from "../../api/query-keys";
import type { Milestone, MilestoneDay, MilestonesResponse } from "../../api/types";
import { mountDom, setupDom } from "../../test-dom";
import CurrentMilestoneSection from "./CurrentMilestoneSection";

beforeAll(setupDom);

const current: Milestone = {
  id: "ms-2",
  number: 2,
  name: "1.1",
  label: "1.1",
  goal: "Live courier tracking",
  state: "OPEN",
  slot: "current",
  started_at: "2026-10-03T09:00:00Z",
  target_date: "2099-11-15",
  counts: { DOING: 2, DONE: 3 },
  total: 5,
  done: 3,
  points: { total: 13, done: 5 },
};

const TWO_DAYS: MilestoneDay[] = [
  { date: "2026-10-03", scope: 5, open: 5, completed: 0 },
  { date: "2026-10-04", scope: 5, open: 2, completed: 3 },
];

const boardWith = (milestone: Milestone): MilestonesResponse => ({
  current: milestone,
  next: { number: 3, name: "#3", state: "OPEN", slot: "next", placeholder: true, counts: {}, total: 0, done: 0 },
  upcoming: [],
  past: [],
});

let cleanup: (() => void) | undefined;
const navigate = vi.fn();

beforeEach(() => {
  navigate.mockReset();
  localStorage.clear();
});
afterEach(() => {
  cleanup?.();
  cleanup = undefined;
  vi.unstubAllGlobals();
});

function mount(milestone?: Milestone, days: MilestoneDay[] = TWO_DAYS) {
  const client = new QueryClient({ defaultOptions: { queries: { staleTime: Infinity, retry: false } } });
  if (milestone) {
    client.setQueryData(queryKeys.milestones, boardWith(milestone));
    client.setQueryData(queryKeys.milestoneProgress(`#${milestone.number}`), { days });
  }
  const mounted = mountDom(
    <QueryClientProvider client={client}>
      <AppNavContext.Provider value={{ navigate } as unknown as AppNav}>
        <TooltipPrimitive.Provider>
          <CurrentMilestoneSection />
        </TooltipPrimitive.Provider>
      </AppNavContext.Provider>
    </QueryClientProvider>,
  );
  cleanup = mounted.cleanup;
  return mounted.host;
}

const flush = () => act(async () => { await new Promise((resolve) => setTimeout(resolve)); });

describe("CurrentMilestoneSection", () => {
  it("shows the name, the goal and the progress in issues and points", () => {
    const text = mount(current).textContent;
    expect(text).toContain("Current Milestone");
    expect(text).toContain("1.1");
    expect(text).toContain("– Live courier tracking");
    expect(text).toContain("3/5 done · 5/13 pts");
  });

  it("leaves out the goal and the points when the milestone has neither", () => {
    const text = mount({ ...current, goal: undefined, points: undefined }).textContent;
    expect(text).toContain("3/5 done");
    expect(text).not.toContain("pts");
    expect(text).not.toContain("–");
  });

  it("shows the target date and the time bar", () => {
    const host = mount(current);
    expect(host.textContent).toContain("Target date");
    expect(host.textContent).toContain("Nov 15");
    expect(host.querySelector('[aria-label="Time to target date"]')).not.toBeNull();
  });

  it("shows no target date or time bar without a target date", () => {
    const host = mount({ ...current, target_date: undefined });
    expect(host.textContent).not.toContain("Target date");
    expect(host.querySelector('[aria-label="Time to target date"]')).toBeNull();
  });

  it("marks a target date that passed with open scope", () => {
    const host = mount({ ...current, target_date: "2026-10-10" });
    const date = [...host.querySelectorAll("span")].find((s) => s.textContent === "Oct 10");
    expect(date?.className).toContain("--color-warning");
  });

  it("does not mark a passed target date when all the scope is done", () => {
    const host = mount({ ...current, target_date: "2026-10-10", done: 5 });
    const date = [...host.querySelectorAll("span")].find((s) => s.textContent === "Oct 10");
    expect(date?.className).not.toContain("--color-warning");
  });

  it("shows the burndown with both dates and the legend", () => {
    const host = mount(current);
    expect(host.querySelector('[aria-label="Burndown"]')).not.toBeNull();
    expect(host.textContent).toContain("Oct 3");
    expect(host.textContent).toContain("Oct 4");
    expect(host.textContent).toContain("Open");
    expect(host.textContent).toContain("Scope");
  });

  it("shows the burndown on the first day, with the date once", () => {
    const host = mount(current, [TWO_DAYS[0]]);
    expect(host.querySelector('[aria-label="Burndown"]')).not.toBeNull();
    expect(host.textContent!.match(/Oct 3/g)).toHaveLength(1);
    expect(host.textContent).not.toContain("Not enough days");
  });

  it("shows only the progress for a milestone that was never started", () => {
    const host = mount({ ...current, started_at: undefined }, []);
    expect(host.textContent).toContain("3/5 done");
    expect(host.querySelector('[aria-label="Burndown"]')).toBeNull();
    expect(host.textContent).not.toContain("Scope");
  });

  it("opens Issues on All Issues with the @current filter from the name", () => {
    const host = mount(current);
    const name = [...host.querySelectorAll("button")].find((b) => b.textContent === "1.1")!;
    act(() => name.click());
    expect(navigate).toHaveBeenCalledWith("backlog", { tab: "all", filters: { milestones: ["@current"] } });
  });

  it("is hidden while Current is a placeholder", () => {
    const host = mount({ ...current, id: undefined, placeholder: true });
    expect(host.textContent).toBe("");
  });

  it("is hidden while Current has no scope", () => {
    const host = mount({ ...current, total: 0, done: 0, counts: {} });
    expect(host.textContent).toBe("");
  });

  it("is hidden while the milestones request is pending", () => {
    vi.stubGlobal("fetch", vi.fn(() => new Promise(() => {})));
    expect(mount().textContent).toBe("");
  });

  it("is hidden when the milestones request fails", async () => {
    vi.stubGlobal("fetch", vi.fn(() => Promise.reject(new Error("offline"))));
    const host = mount();
    await flush();
    expect(host.textContent).toBe("");
  });

  it("collapses and keeps its state in localStorage", () => {
    const host = mount(current);
    act(() => [...host.querySelectorAll("div")].find((d) => d.textContent === "Current Milestone")!.click());
    expect(host.textContent).not.toContain("3/5 done");
    expect(localStorage.getItem("exponential-dashboard-milestone-open")).toBe("0");
  });
});
