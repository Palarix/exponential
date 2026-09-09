import { EMPTY_FILTERS, chipLabel, type BacklogFilters } from "./filters";
import { STATUS_OPTIONS, PRIORITY_OPTIONS } from "../../constants";

interface FilterChipsProps {
  filters: BacklogFilters;
  onChange: (filters: BacklogFilters) => void;
}

const STATUS_LABELS = new Map(STATUS_OPTIONS.map((s) => [s.value, s.label]));
const PRIORITY_LABELS = new Map(PRIORITY_OPTIONS.map((p) => [String(p.value), p.label]));

export default function FilterChips({ filters, onChange }: FilterChipsProps) {
  const chips: { key: string; label: string; onClear: () => void }[] = [];

  if (filters.statuses.length > 0) {
    chips.push({
      key: "status",
      label: chipLabel("Status", filters.statuses, STATUS_LABELS),
      onClear: () => onChange({ ...filters, statuses: [] }),
    });
  }
  if (filters.assignees.length > 0) {
    const display = filters.assignees.map((a) =>
      a === "__unassigned__" ? "Unassigned" : a.split(" <")[0]
    );
    chips.push({
      key: "assignee",
      label: display.length <= 2 ? `Assignee: ${display.join(", ")}` : `Assignee (${display.length})`,
      onClear: () => onChange({ ...filters, assignees: [] }),
    });
  }
  if (filters.priorities.length > 0) {
    chips.push({
      key: "priority",
      label: chipLabel("Priority", filters.priorities.map(String), PRIORITY_LABELS),
      onClear: () => onChange({ ...filters, priorities: [] }),
    });
  }
  if (filters.labels.length > 0) {
    chips.push({
      key: "labels",
      label: chipLabel("Label", filters.labels),
      onClear: () => onChange({ ...filters, labels: [] }),
    });
  }
  if (filters.epicId) {
    chips.push({
      key: "epic",
      label: `Epic: ${filters.epicId}`,
      onClear: () => onChange({ ...filters, epicId: null }),
    });
  }

  if (chips.length === 0) return null;

  return (
    <div className="flex items-center gap-1.5 min-w-0 overflow-x-auto">
      {chips.map((chip) => (
        <span
          key={chip.key}
          className="flex items-center gap-1 px-2 py-0.5 rounded-full text-xs bg-[var(--color-hover-surface)] text-[var(--color-text-secondary)] border border-[var(--color-border-subtle)] whitespace-nowrap shrink-0"
        >
          {chip.label}
          <button
            onClick={chip.onClear}
            className="ml-0.5 text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] transition-colors"
          >
            <svg className="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </span>
      ))}
      <button
        onClick={() => onChange(EMPTY_FILTERS)}
        className="text-xs text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] transition-colors shrink-0 px-1"
        title="Clear all filters"
      >
        <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
          <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
        </svg>
      </button>
    </div>
  );
}
