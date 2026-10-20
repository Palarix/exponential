import { jsonCodec } from "../../app/view-state-utils";
import type { ReleasesResponse } from "../../api/types";

export interface BacklogFilters {
  statuses: string[];
  labels: string[];
  assignees: string[];
  priorities: number[];
  epicId: string | null;
  workedBy: string[]; // "agent" | "human"
  /** Release refs: `@current`, `@next`, `none` or `#7`. Current and Next resolve each time the filter runs. */
  releases: string[];
}

export const EMPTY_FILTERS: BacklogFilters = {
  statuses: [],
  labels: [],
  assignees: [],
  priorities: [],
  epicId: null,
  workedBy: [],
  releases: [],
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
    f.releases.length > 0
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

const releaseByNumber = (ref: string, board: ReleasesResponse) =>
  [board.current, board.next, ...board.upcoming, ...board.past].find((r) => `#${r.number}` === ref);

/**
 * The release a ref names on `board`: its ID, "" for `none`, or null when
 * nothing matches (an unknown number, a placeholder slot, no board yet).
 */
export function resolveReleaseRef(ref: string, board?: ReleasesResponse): string | null {
  if (ref === "none") return "";
  if (!board) return null;
  const release =
    ref === "@current" ? board.current
      : ref === "@next" ? board.next
        : releaseByNumber(ref, board);
  return release?.id ?? null;
}

/** How a release ref reads in chips and menus: Current, Next, No release, or the release's name. */
export function releaseRefName(ref: string, board?: ReleasesResponse): string {
  if (ref === "@current") return "Current";
  if (ref === "@next") return "Next";
  if (ref === "none") return "No release";
  return (board && releaseByNumber(ref, board)?.name) ?? ref;
}

/**
 * The Release filter's menu: Current, Next, No release, then the Upcoming
 * and Past releases by number, each with the count of `issues` it matches.
 */
export function releaseFilterOptions(
  board: ReleasesResponse | undefined,
  issues: { release_id?: string }[],
): { value: string; label: string; count: number }[] {
  const option = (value: string, label: string) => {
    const id = resolveReleaseRef(value, board);
    return { value, label, count: id === null ? 0 : issues.filter((i) => (i.release_id ?? "") === id).length };
  };
  if (!board) return [option("none", "No release")];
  return [
    option("@current", `Current · ${board.current.name}`),
    option("@next", `Next · ${board.next.name}`),
    option("none", "No release"),
    ...[...board.upcoming, ...board.past].map((r) => option(`#${r.number}`, r.name)),
  ];
}

export function matchesFilters(
  issue: { status: string; labels?: string[]; assignee?: string; assignee_via?: string; priority: number; parent_id?: string; release_id?: string },
  filters: BacklogFilters,
  releases?: ReleasesResponse,
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
  if (filters.releases.length > 0) {
    const own = issue.release_id ?? "";
    if (!filters.releases.some((ref) => resolveReleaseRef(ref, releases) === own)) return false;
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

export type FilterDimension = "status" | "assignee" | "workedBy" | "priority" | "labels" | "epic" | "release";

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
    case "release": return filters.releases;
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
    case "release": return { ...filters, releases: values };
  }
}
