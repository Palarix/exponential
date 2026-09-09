export interface BacklogFilters {
  statuses: string[];
  labels: string[];
  assignees: string[];
  priorities: number[];
  epicId: string | null;
}

export const EMPTY_FILTERS: BacklogFilters = {
  statuses: [],
  labels: [],
  assignees: [],
  priorities: [],
  epicId: null,
};

export function hasActiveFilters(f: BacklogFilters): boolean {
  return (
    f.statuses.length > 0 ||
    f.labels.length > 0 ||
    f.assignees.length > 0 ||
    f.priorities.length > 0 ||
    f.epicId !== null
  );
}

export function matchesFilters(issue: { status: string; labels?: string[]; assignee?: string; priority: number; parent_id?: string }, filters: BacklogFilters): boolean {
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
  return true;
}

export type FilterDimension = "status" | "assignee" | "priority" | "labels" | "epic";

export function chipLabel(dimension: string, values: string[], lookup?: Map<string, string>): string {
  const names = lookup ? values.map((v) => lookup.get(v) || v) : values;
  if (names.length <= 2) return `${dimension}: ${names.join(", ")}`;
  return `${dimension} (${names.length})`;
}

export function getSelected(dim: FilterDimension, filters: BacklogFilters): string[] {
  switch (dim) {
    case "status": return filters.statuses;
    case "assignee": return filters.assignees;
    case "priority": return filters.priorities.map(String);
    case "labels": return filters.labels;
    case "epic": return filters.epicId ? [filters.epicId] : [];
  }
}

export function setSelected(dim: FilterDimension, filters: BacklogFilters, values: string[]): BacklogFilters {
  switch (dim) {
    case "status": return { ...filters, statuses: values };
    case "assignee": return { ...filters, assignees: values };
    case "priority": return { ...filters, priorities: values.map(Number) };
    case "labels": return { ...filters, labels: values };
    case "epic": return { ...filters, epicId: values[0] || null };
  }
}
