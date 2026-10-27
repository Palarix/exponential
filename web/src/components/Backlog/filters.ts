import { jsonCodec } from "../../app/view-state-utils";
import type { MilestonesResponse } from "../../api/types";

export interface BacklogFilters {
  statuses: string[];
  labels: string[];
  assignees: string[];
  priorities: number[];
  epicId: string | null;
  workedBy: string[]; // "agent" | "human"
  /** Milestone refs: `@current`, `@next`, `none` or `#7`. Current and Next resolve each time the filter runs. */
  milestones: string[];
}

export const EMPTY_FILTERS: BacklogFilters = {
  statuses: [],
  labels: [],
  assignees: [],
  priorities: [],
  epicId: null,
  workedBy: [],
  milestones: [],
};

// Filters saved by older versions lack newer dimensions; fill them in.
export function parseStoredFilters(stored: string | null): BacklogFilters {
  if (!stored) return EMPTY_FILTERS;
  try {
    return { ...EMPTY_FILTERS, ...JSON.parse(stored) };
  } catch {
    return EMPTY_FILTERS;
  }
}

/** Persists filters to localStorage through `useViewState`. */
export const FILTERS_CODEC = jsonCodec(parseStoredFilters);

export function hasActiveFilters(f: BacklogFilters): boolean {
  return (
    f.statuses.length > 0 ||
    f.labels.length > 0 ||
    f.assignees.length > 0 ||
    f.priorities.length > 0 ||
    f.epicId !== null ||
    f.workedBy.length > 0 ||
    f.milestones.length > 0
  );
}

/**
 * The filters as a tab with the status scope `statuses` applies them: only
 * the selected statuses inside the scope count, and a Status filter with
 * none inside it is ignored, so a conflicting filter never empties a tab.
 */
export function filtersForScope(filters: BacklogFilters, statuses: string[]): BacklogFilters {
  if (filters.statuses.length === 0) return filters;
  return { ...filters, statuses: filters.statuses.filter((s) => statuses.includes(s)) };
}

/** Whether a filter value can't apply within the status `scope`: a status outside it. */
export function outOfScope(dim: FilterDimension, value: string, scope?: string[]): boolean {
  return dim === "status" && !!scope && !scope.includes(value);
}

/** Whether a dimension can't narrow the status `scope` at all: Status on a single-status tab. */
export function dimensionDisabled(dim: FilterDimension, scope?: string[]): boolean {
  return dim === "status" && !!scope && scope.length <= 1;
}

const milestoneByNumber = (ref: string, board: MilestonesResponse) =>
  [board.current, board.next, ...board.upcoming, ...board.past].find((r) => `#${r.number}` === ref);

/**
 * The milestone a ref names on `board`: its ID, "" for `none`, or null when
 * nothing matches (an unknown number, a placeholder slot, no board yet).
 */
export function resolveMilestoneRef(ref: string, board?: MilestonesResponse): string | null {
  if (ref === "none") return "";
  if (!board) return null;
  const milestone =
    ref === "@current" ? board.current
      : ref === "@next" ? board.next
        : milestoneByNumber(ref, board);
  return milestone?.id ?? null;
}

/** How a milestone ref reads in chips and menus: Current, Next, No milestone, or the milestone's name. */
export function milestoneRefName(ref: string, board?: MilestonesResponse): string {
  if (ref === "@current") return "Current";
  if (ref === "@next") return "Next";
  if (ref === "none") return "No milestone";
  return (board && milestoneByNumber(ref, board)?.name) ?? ref;
}

/**
 * The Milestone filter's menu: Current, Next, No milestone, then the Upcoming
 * and Past milestones by number, each with the count of `issues` it matches.
 */
export function milestoneFilterOptions(
  board: MilestonesResponse | undefined,
  issues: { milestone_id?: string }[],
): { value: string; label: string; count: number }[] {
  const option = (value: string, label: string) => {
    const id = resolveMilestoneRef(value, board);
    return { value, label, count: id === null ? 0 : issues.filter((i) => (i.milestone_id ?? "") === id).length };
  };
  if (!board) return [option("none", "No milestone")];
  return [
    option("@current", `Current · ${board.current.name}`),
    option("@next", `Next · ${board.next.name}`),
    option("none", "No milestone"),
    ...[...board.upcoming, ...board.past].map((r) => option(`#${r.number}`, r.name)),
  ];
}

export function matchesFilters(
  issue: { status: string; labels?: string[]; assignee?: string; assignee_via?: string; priority: number; parent_id?: string; milestone_id?: string },
  filters: BacklogFilters,
  milestones?: MilestonesResponse,
): boolean {
  if (filters.statuses.length > 0 && !filters.statuses.includes(issue.status))
    return false;
  if (
    filters.labels.length > 0 &&
    !filters.labels.some((l) => issue.labels?.includes(l))
  )
    return false;
  if (filters.assignees.length > 0) {
    const match = issue.assignee
      ? filters.assignees.includes(issue.assignee)
      : filters.assignees.includes("__unassigned__");
    if (!match) return false;
  }
  if (
    filters.priorities.length > 0 &&
    !filters.priorities.includes(issue.priority || 0)
  )
    return false;
  if (filters.epicId && issue.parent_id !== filters.epicId) return false;
  if (filters.workedBy.length > 0) {
    if (!issue.assignee) return false;
    if (!filters.workedBy.includes(issue.assignee_via ? "agent" : "human")) return false;
  }
  if (filters.milestones.length > 0) {
    const own = issue.milestone_id ?? "";
    if (!filters.milestones.some((ref) => resolveMilestoneRef(ref, milestones) === own)) return false;
  }
  return true;
}

export function matchesSearch(issue: { id: string; title: string; labels?: string[] }, query: string): boolean {
  const q = query.trim().toLowerCase();
  if (!q) return true;
  return (
    issue.title.toLowerCase().includes(q) ||
    issue.id.toLowerCase().includes(q) ||
    (issue.labels?.some((l) => l.toLowerCase().includes(q)) ?? false)
  );
}

export type FilterDimension = "status" | "assignee" | "workedBy" | "priority" | "labels" | "epic" | "milestone";

export function chipLabel(dimension: string, values: string[], lookup?: Map<string, string>): string {
  const names = lookup ? values.map((v) => lookup.get(v) || v) : values;
  if (names.length <= 2) return `${dimension}: ${names.join(", ")}`;
  return `${dimension} (${names.length})`;
}

export function getSelected(dim: FilterDimension, filters: BacklogFilters): string[] {
  switch (dim) {
    case "status": return filters.statuses;
    case "assignee": return filters.assignees;
    case "workedBy": return filters.workedBy;
    case "priority": return filters.priorities.map(String);
    case "labels": return filters.labels;
    case "epic": return filters.epicId ? [filters.epicId] : [];
    case "milestone": return filters.milestones;
  }
}

export function setSelected(dim: FilterDimension, filters: BacklogFilters, values: string[]): BacklogFilters {
  switch (dim) {
    case "status": return { ...filters, statuses: values };
    case "assignee": return { ...filters, assignees: values };
    case "workedBy": return { ...filters, workedBy: values };
    case "priority": return { ...filters, priorities: values.map(Number) };
    case "labels": return { ...filters, labels: values };
    case "epic": return { ...filters, epicId: values[0] || null };
    case "milestone": return { ...filters, milestones: values };
  }
}
