import { useState, useMemo } from "react";
import type { Cycle } from "../../api/client";
import { cycleLabel, filterCycles } from "../../utils/cycles";
import { Menu, MenuItem, MenuFilter, MenuLabel } from "./Menu";

interface CyclePickerProps {
  cycles: Cycle[];
  current?: string;
  /** Called with the chosen cycle ID, or "" for "No cycle". */
  onSelect: (cycleId: string) => void;
  onClose?: () => void;
}

export default function CyclePicker({ cycles, current, onSelect, onClose }: CyclePickerProps) {
  const [filterText, setFilterText] = useState("");
  const visible = useMemo(() => filterCycles(cycles, filterText), [cycles, filterText]);
  const hasQuery = filterText.trim().length > 0;

  const select = (cycleId: string) => { onSelect(cycleId); onClose?.(); };

  if (cycles.length === 0) {
    return (
      <Menu onClose={onClose} bare autoFocus={false}>
        <MenuLabel>Cycles not configured</MenuLabel>
      </Menu>
    );
  }
  return (
    <Menu onClose={onClose} bare autoFocus={false} maxHeight="18rem">
      <MenuFilter value={filterText} onChange={setFilterText} placeholder="Move to cycle..." />
      {current && !hasQuery && (
        <MenuItem label="No cycle" onClick={() => select("")} />
      )}
      {visible.map(c => (
        <MenuItem
          key={c.id}
          label={cycleLabel(c)}
          checked={c.id === current}
          onClick={() => select(c.id)}
        />
      ))}
      {visible.length === 0 && <MenuLabel>No matching cycles</MenuLabel>}
    </Menu>
  );
}
