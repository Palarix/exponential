export const STATUS_OPTIONS = [
  { value: "BACKLOG", label: "Backlog" },
  { value: "PLANNED", label: "Planned" },
  { value: "DOING", label: "In Progress" },
  { value: "BLOCKED", label: "Blocked" },
  { value: "DONE", label: "Done" },
];

export const ESTIMATE_OPTIONS = [0, 1, 2, 3, 5, 8];

export const PRIORITY_OPTIONS = [
  { value: 0, label: "No priority" },
  { value: 1, label: "Urgent" },
  { value: 2, label: "High" },
  { value: 3, label: "Medium" },
  { value: 4, label: "Low" },
];

export const BUILTIN_LABELS = [
  "bug",
  "feature",
  "epic",
  "improvement",
  "UI",
  "refactor",
];
