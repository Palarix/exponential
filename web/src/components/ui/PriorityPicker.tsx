import { PRIORITY_OPTIONS } from "../../constants";
import PriorityIcon from "./PriorityIcon";
import { Menu, MenuItem } from "./Menu";

interface PriorityPickerProps {
  current: number;
  onSelect: (priority: number) => void;
  onClose?: () => void;
  /** Accessible name for the menu. */
  ariaLabel?: string;
}

export default function PriorityPicker({ current, onSelect, onClose, ariaLabel = "Set priority" }: PriorityPickerProps) {
  return (
    <Menu onClose={onClose} aria-label={ariaLabel} bare>
      {PRIORITY_OPTIONS.map((opt, i) => (
        <MenuItem
          key={opt.value}
          label={opt.label}
          icon={<PriorityIcon priority={opt.value} size={14} />}
          radio
          checked={opt.value === current}
          shortcut={String(i + 1)}
          onClick={() => { onSelect(opt.value); onClose?.(); }}
        />
      ))}
    </Menu>
  );
}
