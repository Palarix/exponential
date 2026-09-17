import { STATUS_OPTIONS } from "../../constants";
import StatusIcon from "./StatusIcon";
import { Menu, MenuItem } from "./Menu";

interface StatusPickerProps {
  current: string;
  onSelect: (status: string) => void;
  onClose?: () => void;
}

export default function StatusPicker({ current, onSelect, onClose }: StatusPickerProps) {
  return (
    <Menu onClose={onClose} bare>
      {STATUS_OPTIONS.map((opt, i) => (
        <MenuItem
          key={opt.value}
          label={opt.label}
          icon={<StatusIcon status={opt.value} size={14} />}
          checked={opt.value === current}
          shortcut={String(i + 1)}
          onClick={() => { onSelect(opt.value); onClose?.(); }}
        />
      ))}
    </Menu>
  );
}
