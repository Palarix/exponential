// @vitest-environment jsdom
import { act } from "react";
import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import * as TooltipPrimitive from "@radix-ui/react-tooltip";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { AppNavContext, type AppNav } from "../../app/contexts";
import { queryKeys } from "../../api/query-keys";
import type { Milestone, MilestoneDay, MilestoneProgressResponse, MilestoneStep, MilestonesResponse } from "../../api/types";
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
  { date: "2026-10-03", scope: 5, open: 5, completed: 0, scope_points: 13, open_points: 13, completed_points: 0 },
  { date: "2026-10-04", scope: 5, open: 2, completed: 3, scope_points: 13, open_points: 8, completed_points: 5 },
];

const STEPS: MilestoneStep[] = [
  { at: "2026-10-03T09:00:00Z", scope: 5, open: 5, completed: 0, scope_points: 13, open_points: 13, completed_points: 0, changes: [] },
  {
    at: "2026-10-03T10:00:00Z", scope: 5, open: 3, completed: 2, scope_points: 13, open_points: 10, completed_points: 3,
    changes: [{ issue_id: "xpo-1", title: "Map view", kind: "done" }, { issue_id: "xpo-2", title: "Courier pins", kind: "done" }],
  },
  // An estimate change: the issue counts stay.
  {
    at: "2026-10-03T11:00:00Z", scope: 5, open: 3, completed: 2, scope_points: 15, open_points: 12, completed_points: 3,
    changes: [{ issue_id: "xpo-3", title: "ETA", kind: "estimated" }],
  },
  {
    at: "2026-10-04T10:00:00Z", scope: 5, open: 2, completed: 3, scope_points: 15, open_points: 10, completed_points: 5,
    changes: [{ issue_id: "xpo-4", title: "Push alerts", kind: "done" }],
  },
];

type Progress = Pick<MilestoneProgressResponse, "days" | "steps">;
const PROGRESS: Progress = { days: TWO_DAYS, steps: STEPS };

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

function mount(milestone?: Milestone, progress: Progress = PROGRESS) {
  const client = new QueryClient({ defaultOptions: { queries: { staleTime: Infinity, retry: false } } });
  if (milestone) {
    client.setQueryData(queryKeys.milestones, boardWith(milestone));
    client.setQueryData(queryKeys.milestoneProgress(`#${milestone.number}`), progress);
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

  it("shows the burn-up with both dates and the legend", () => {
    localStorage.setItem("exponential-milestones-burnup-axis", "time");
    const host = mount(current);
    expect(host.querySelector('[aria-label="Burn-up"]')).not.toBeNull();
    expect(host.textContent).toContain("Oct 3");
    expect(host.textContent).toContain("Oct 4");
    expect(host.textContent).toContain("Done");
    expect(host.textContent).toContain("Scope");
  });

  it("shows the burn-up on the first day, with the date once", () => {
    localStorage.setItem("exponential-milestones-burnup-axis", "time");
    const host = mount(current, { days: [TWO_DAYS[0]], steps: [STEPS[0]] });
    expect(host.querySelector('[aria-label="Burn-up"]')).not.toBeNull();
    expect(host.textContent!.match(/Oct 3/g)).toHaveLength(1);
    expect(host.textContent).not.toContain("Not enough days");
  });

  it("shows only the progress for a milestone that was never started", () => {
    const host = mount({ ...current, started_at: undefined }, { days: [], steps: [] });
    expect(host.textContent).toContain("3/5 done");
    expect(host.querySelector('[aria-label="Burn-up"]')).toBeNull();
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
  describe("burn-up axis and mode", () => {
    const axis = (host: HTMLElement) => host.querySelector<HTMLInputElement>('[aria-label="Burn-up axis"] input:checked')?.value;
    const radio = (host: HTMLElement, value: string) => host.querySelector<HTMLInputElement>(`[aria-label="Burn-up axis"] input[value="${value}"]`)!;
    const hover = (host: HTMLElement, i: number) => {
      const rect = host.querySelector(`[aria-label="Burn-up"] rect[data-point="${i}"]`)!;
      act(() => { rect.dispatchEvent(new MouseEvent("mouseover", { bubbles: true })); });
      return host.querySelector('[role="tooltip"]');
    };

    it("defaults to Order and counts the drawn steps, skipping points-only steps in issues mode", () => {
      const host = mount(current);
      expect(axis(host)).toBe("order");
      expect(host.textContent).toContain("3 steps");
      expect(host.querySelectorAll('[aria-label="Burn-up"] rect[data-point]')).toHaveLength(3);
    });

    it("switches to Time and keeps the choice", () => {
      const host = mount(current);
      act(() => radio(host, "time").click());
      expect(axis(host)).toBe("time");
      expect(localStorage.getItem("exponential-milestones-burnup-axis")).toBe("time");
      expect(host.textContent).not.toContain("steps");
      expect(host.querySelectorAll('[aria-label="Burn-up"] rect[data-point]')).toHaveLength(2);
    });

    it("uses the stored axis", () => {
      localStorage.setItem("exponential-milestones-burnup-axis", "time");
      expect(axis(mount(current))).toBe("time");
    });

    it("draws every step in story points when the pill shows points", () => {
      localStorage.setItem("exponential-milestones-show-points", "true");
      const host = mount(current);
      expect(host.textContent).toContain("4 steps");
      expect(hover(host, 2)?.textContent).toContain("Done 3 of 15 pts");
    });

    it("shows a step's changes on hover", () => {
      const host = mount(current);
      const tip = hover(host, 1)!;
      expect(tip.textContent).toContain("Done 2 of 5");
      expect(tip.textContent).toContain("Done");
      expect(tip.textContent).toContain("Map view");
      expect(tip.textContent).toContain("Courier pins");
      expect(hover(host, 0)?.textContent).toContain("Start");
    });

    it("shows the date and the counts on hover in Time mode", () => {
      localStorage.setItem("exponential-milestones-burnup-axis", "time");
      const tip = hover(mount(current), 1)!;
      expect(tip.textContent).toContain("Oct 4");
      expect(tip.textContent).toContain("Done 3 of 5");
    });
  });
});
