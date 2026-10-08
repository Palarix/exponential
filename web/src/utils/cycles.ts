import type { Cycle } from "../api/types";

export function cycleLabel(cycle: Cycle): string {
  return `Cycle ${cycle.number} (${cycle.status})`;
}

export function filterCycles(cycles: Cycle[], query: string): Cycle[] {
  const q = query.trim().toLowerCase();
  const open = cycles.filter(c => c.status !== "completed");
  if (!q) return open;
  return open.filter(c => cycleLabel(c).toLowerCase().includes(q));
}
