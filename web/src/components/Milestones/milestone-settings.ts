import { useCallback, useSyncExternalStore } from "react";

/** What the progress pill and the burn-ups count: issues, or story points. */
export type ProgressMode = "issues" | "points";

/** The burn-up's x-axis: one step per write (order), or one point per day (time). */
export type BurnUpAxis = "order" | "time";

/** Where the progress pills' issues/points choice is saved. */
export const PROGRESS_MODE_KEY = "exponential-milestones-show-points";

/** Where the burn-up's Order/Time choice is saved. */
export const BURNUP_AXIS_KEY = "exponential-milestones-burnup-axis";

// Server rendering (and the markup tests) has no localStorage: it renders the
// defaults.

// One listener set for every stored milestone setting: a change in one view
// (the pill in Milestones) shows at once in the others (the Dashboard chart).
const listeners = new Set<() => void>();

function subscribe(listener: () => void) {
  listeners.add(listener);
  window.addEventListener("storage", listener);
  return () => {
    listeners.delete(listener);
    window.removeEventListener("storage", listener);
  };
}

function write(key: string, value: string) {
  localStorage.setItem(key, value);
  listeners.forEach((l) => l());
}

const readProgressMode = (): ProgressMode => localStorage.getItem(PROGRESS_MODE_KEY) === "true" ? "points" : "issues";
const readBurnUpAxis = (): BurnUpAxis => localStorage.getItem(BURNUP_AXIS_KEY) === "time" ? "time" : "order";

/** The stored pill mode, and a toggle between issues and points. */
export function useProgressMode(): [ProgressMode, () => void] {
  const mode = useSyncExternalStore(subscribe, readProgressMode, () => "issues" as const);
  const toggle = useCallback(() => write(PROGRESS_MODE_KEY, String(readProgressMode() !== "points")), []);
  return [mode, toggle];
}

/** The stored burn-up axis (Order by default), and its setter. */
export function useBurnUpAxis(): [BurnUpAxis, (axis: BurnUpAxis) => void] {
  const axis = useSyncExternalStore(subscribe, readBurnUpAxis, () => "order" as const);
  const set = useCallback((next: BurnUpAxis) => write(BURNUP_AXIS_KEY, next), []);
  return [axis, set];
}
