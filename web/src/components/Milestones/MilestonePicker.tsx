import { useState, useMemo } from "react";
import type { Milestone } from "../../api/client";
import { Menu, MenuItem, MenuFilter, MenuLabel } from "../ui/Menu";
import { filterPickerOptions, type PickerOption } from "./milestone-assign-utils";
import { MilestoneTitle } from "./MilestoneRow";

interface MilestonePickerProps {
  /** The open milestones, from `pickerOptions`. */
  options: PickerOption[];
  /** The issue's `milestone_id`. */
  current?: string;
  /** Called with the chosen milestone, or null for "No milestone". */
  onSelect: (milestone: Milestone | null) => void;
  onClose?: () => void;
  /** Accessible name for the menu. */
  ariaLabel?: string;
}

export default function MilestonePicker({ options, current, onSelect, onClose, ariaLabel = "Change milestone" }: MilestonePickerProps) {
  const [filterText, setFilterText] = useState("");
  const visible = useMemo(() => filterPickerOptions(options, filterText), [options, filterText]);
  const hasQuery = filterText.trim().length > 0;

  const select = (milestone: Milestone | null) => { onSelect(milestone); onClose?.(); };

  return (
    <Menu onClose={onClose} aria-label={ariaLabel} bare autoFocus={false} maxHeight="18rem" className="w-80 max-w-full">
      <MenuFilter value={filterText} onChange={setFilterText} placeholder="Move to milestone..." />
      {current && !hasQuery && (
        <MenuItem label="No milestone" onClick={() => select(null)} />
      )}
      {visible.map(({ milestone, slotLabel, disabled, completed }) => (
        <MenuItem
          key={milestone.number}
          label={completed ? `${milestone.name}, Completed` : slotLabel ? `${milestone.name}, ${slotLabel}` : milestone.name}
          content={
            <span className="flex items-center gap-2 min-w-0">
              <MilestoneTitle milestone={milestone} />
              {completed && (
                <span className="shrink-0 px-2 h-5 inline-flex items-center rounded-full text-xs font-medium bg-[var(--color-hover-surface)] text-[var(--color-text-muted)]">
                  Completed
                </span>
              )}
            </span>
          }
          radio
          checked={!!milestone.id && milestone.id === current}
          disabled={disabled}
          suffix={disabled && !completed && <span className="text-xs text-[var(--color-text-muted)]">Save Current first</span>}
          onClick={() => select(milestone)}
        />
      ))}
      {visible.length === 0 && <MenuLabel>No matching milestones</MenuLabel>}
    </Menu>
  );
}
