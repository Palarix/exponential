export const STATUS_OPTIONS = [
  { value: "BACKLOG", label: "Backlog" },
  { value: "PLANNED", label: "Planned" },
  { value: "DOING", label: "In Progress" },
  { value: "BLOCKED", label: "Blocked" },
  { value: "DONE", label: "Done" },
  { value: "CANCELED", label: "Canceled" },
  { value: "DUPLICATE", label: "Duplicate" },
];

export const TERMINAL_STATUSES = new Set(["DONE", "CANCELED", "DUPLICATE"]);

export function isTerminal(status: string): boolean {
  return TERMINAL_STATUSES.has(status);
}

export function isCompleted(status: string): boolean {
  return status === "DONE";
}

export const ESTIMATE_OPTIONS = [0, 1, 2, 3, 5, 8];

export const PRIORITY_OPTIONS = [
  { value: 0, label: "No priority" },
  { value: 1, label: "Urgent" },
  { value: 2, label: "High" },
  { value: 3, label: "Medium" },
  { value: 4, label: "Low" },
];

/** Colors offered for custom labels: distinct from each other and from the built-in label colors. */
export const LABEL_PRESET_COLORS = [
  '#e85d9c', // pink
  '#a33a6e', // plum
  '#f28c38', // orange
  '#b4501f', // rust
  '#e6c935', // yellow
  '#86881a', // olive
  '#a3c93a', // lime
  '#3fb26b', // green
  '#2e7a2c', // forest
  '#1fb5b0', // teal
  '#0e6a7c', // petrol
  '#8a8f98', // gray
];



