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

export const LABEL_PRESET_COLORS = [
  '#26b5b0',
  '#e06091',
  '#d4a030',
  '#7ab030',
  '#8891a5',
  '#e07058',
];



