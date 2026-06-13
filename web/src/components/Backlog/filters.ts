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
