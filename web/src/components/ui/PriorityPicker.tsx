import { PRIORITY_OPTIONS } from "../../constants";
import PriorityIcon from "./PriorityIcon";
import { Menu, MenuItem } from "./Menu";

interface PriorityPickerProps {
  current: number;
  onSelect: (priority: number) => void;
  onClose?: () => void;
}

export default function PriorityPicker({ current, onSelect, onClose }: PriorityPickerProps) {
  return (
    <Menu onClose={onClose} bare>
      {PRIORITY_OPTIONS.map((opt, i) => (
        <MenuItem
          key={opt.value}
          label={opt.label}
          icon={<PriorityIcon priority={opt.value} size={14} />}
          checked={opt.value === current}
          shortcut={String(i + 1)}
          onClick={() => { onSelect(opt.value); onClose?.(); }}
        />
      ))}
    </Menu>
  );
}
