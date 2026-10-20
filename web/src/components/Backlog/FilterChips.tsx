import { Fragment, type ReactNode } from "react";
import { EMPTY_FILTERS, chipLabel, outOfScope, releaseRefName, type BacklogFilters } from "./filters";
import { useReleases } from "../../api/queries";
import { shortName } from "../../utils/format";
import { STATUS_OPTIONS, PRIORITY_OPTIONS } from "../../constants";

interface FilterChipsProps {
  filters: BacklogFilters;
  onChange: (filters: BacklogFilters) => void;
  /** The tab's status scope; status values outside it show greyed. */
  scope?: string[];
}

const STATUS_LABELS = new Map(STATUS_OPTIONS.map((s) => [s.value, s.label]));
const PRIORITY_LABELS = new Map(PRIORITY_OPTIONS.map((p) => [String(p.value), p.label]));
const WORKED_BY_LABELS = new Map([["agent", "Agent"], ["human", "Human"]]);

export default function FilterChips({ filters, onChange, scope }: FilterChipsProps) {
  const { data: releases } = useReleases();
  const chips: { key: string; label: ReactNode; muted?: boolean; onClear: () => void }[] = [];

  if (filters.statuses.length > 0) {
    const allOut = filters.statuses.every((s) => outOfScope("status", s, scope));
    // When some values apply, grey the others; when none do, the whole chip is greyed.
    const out = (s: string) => !allOut && outOfScope("status", s, scope);
    chips.push({
      key: "status",
      label: filters.statuses.length <= 2
        ? <>Status: {filters.statuses.map((s, i) => (
            <Fragment key={s}>
              {i > 0 && ", "}
              <span className={out(s) ? "opacity-50" : undefined} title={out(s) ? "Not on this tab" : undefined}>{STATUS_LABELS.get(s) || s}</span>
            </Fragment>
          ))}</>
        : chipLabel("Status", filters.statuses, STATUS_LABELS),
      muted: allOut,
      onClear: () => onChange({ ...filters, statuses: [] }),
    });
  }
  if (filters.assignees.length > 0) {
    const display = filters.assignees.map((a) =>
      a === "__unassigned__" ? "Unassigned" : shortName(a)
    );
    chips.push({
      key: "assignee",
      label: display.length <= 2 ? `Assignee: ${display.join(", ")}` : `Assignee (${display.length})`,
      onClear: () => onChange({ ...filters, assignees: [] }),
    });
  }
  if (filters.workedBy.length > 0) {
    chips.push({
      key: "workedBy",
      label: chipLabel("Worked by", filters.workedBy, WORKED_BY_LABELS),
      onClear: () => onChange({ ...filters, workedBy: [] }),
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
  if (filters.releases.length > 0) {
    chips.push({
      key: "release",
      label: chipLabel("Release", filters.releases.map((r) => releaseRefName(r, releases))),
      onClear: () => onChange({ ...filters, releases: [] }),
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
      <button
        onClick={() => onChange(EMPTY_FILTERS)}
        className="text-xs text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] transition-colors shrink-0 whitespace-nowrap"
      >
        Clear filters
      </button>
      {chips.map((chip) => (
        <span
          key={chip.key}
          title={chip.muted ? "Not on this tab" : undefined}
          className={`flex items-center gap-1 px-2 py-0.5 rounded-full text-xs bg-[var(--color-hover-surface)] text-[var(--color-text-secondary)] border border-[var(--color-border-subtle)] whitespace-nowrap shrink-0 ${chip.muted ? "opacity-50" : ""}`}
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
    </div>
  );
}
