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

export const LABEL_PRESET_COLORS = [
  '#26b5b0',
  '#e06091',
  '#d4a030',
  '#7ab030',
  '#8891a5',
  '#e07058',
];

// Canonical "default" labels — match the seeded set in internal/beats/setup.go.
// Shown at the top of label pickers in this fixed order. Suppressed when the
// project config sets `hide_default_labels: true`.
export const DEFAULT_LABELS: { name: string; color: string }[] = [
  { name: 'epic',        color: '#5e6ad2' },
  { name: 'feature',     color: '#b36cd9' },
  { name: 'bug',         color: '#eb5757' },
  { name: 'improvement', color: '#4da6e8' },
];

