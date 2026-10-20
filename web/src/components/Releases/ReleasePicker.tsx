import { useState, useMemo } from "react";
import type { Release } from "../../api/client";
import { Menu, MenuItem, MenuFilter, MenuLabel } from "../ui/Menu";
import { filterPickerOptions, type PickerOption } from "./release-assign-utils";
import { ReleaseTitle } from "./ReleaseRow";

interface ReleasePickerProps {
  /** The open releases, from `pickerOptions`. */
  options: PickerOption[];
  /** The issue's `release_id`. */
  current?: string;
  /** Called with the chosen release, or null for "No release". */
  onSelect: (release: Release | null) => void;
  onClose?: () => void;
  /** Accessible name for the menu. */
  ariaLabel?: string;
}

export default function ReleasePicker({ options, current, onSelect, onClose, ariaLabel = "Change release" }: ReleasePickerProps) {
  const [filterText, setFilterText] = useState("");
  const visible = useMemo(() => filterPickerOptions(options, filterText), [options, filterText]);
  const hasQuery = filterText.trim().length > 0;

  const select = (release: Release | null) => { onSelect(release); onClose?.(); };

  return (
    <Menu onClose={onClose} aria-label={ariaLabel} bare autoFocus={false} maxHeight="18rem" className="w-80 max-w-full">
      <MenuFilter value={filterText} onChange={setFilterText} placeholder="Move to release..." />
      {current && !hasQuery && (
        <MenuItem label="No release" onClick={() => select(null)} />
      )}
      {visible.map(({ release, slotLabel, disabled, released }) => (
        <MenuItem
          key={release.number}
          label={released ? `${release.name}, Released` : slotLabel ? `${release.name}, ${slotLabel}` : release.name}
          content={
            <span className="flex items-center gap-2 min-w-0">
              <ReleaseTitle release={release} />
              {released && (
                <span className="shrink-0 px-2 h-5 inline-flex items-center rounded-full text-xs font-medium bg-[var(--color-hover-surface)] text-[var(--color-text-muted)]">
                  Released
                </span>
              )}
            </span>
          }
          radio
          checked={!!release.id && release.id === current}
          disabled={disabled}
          suffix={disabled && !released && <span className="text-xs text-[var(--color-text-muted)]">Save Current first</span>}
          onClick={() => select(release)}
        />
      ))}
      {visible.length === 0 && <MenuLabel>No matching releases</MenuLabel>}
    </Menu>
  );
}
