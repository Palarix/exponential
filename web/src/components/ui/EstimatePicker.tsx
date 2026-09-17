import { ESTIMATE_OPTIONS } from "../../constants";
import { Menu, MenuItem } from "./Menu";

interface EstimatePickerProps {
  current: number;
  onSelect: (estimate: number) => void;
  onClose?: () => void;
}

function estimateLabel(est: number): string {
  return est === 0 ? "No estimate" : `${est} Point${est !== 1 ? "s" : ""}`;
}

export default function EstimatePicker({ current, onSelect, onClose }: EstimatePickerProps) {
  return (
    <Menu onClose={onClose} bare>
      {ESTIMATE_OPTIONS.map((est, i) => (
        <MenuItem
          key={est}
          label={estimateLabel(est)}
          checked={est === current}
          shortcut={String(i + 1)}
          onClick={() => { onSelect(est); onClose?.(); }}
        />
      ))}
    </Menu>
  );
}
