import { ESTIMATE_OPTIONS } from "../../constants";
import { Menu, MenuItem } from "./Menu";

interface EstimatePickerProps {
  current: number;
  onSelect: (estimate: number) => void;
  onClose?: () => void;
  /** Accessible name for the menu. */
  ariaLabel?: string;
}

function estimateLabel(est: number): string {
  return est === 0 ? "No estimate" : `${est} Point${est !== 1 ? "s" : ""}`;
}

export default function EstimatePicker({ current, onSelect, onClose, ariaLabel = "Change estimate" }: EstimatePickerProps) {
  return (
    <Menu onClose={onClose} aria-label={ariaLabel} bare>
      {ESTIMATE_OPTIONS.map((est, i) => (
        <MenuItem
          key={est}
          label={estimateLabel(est)}
          radio
          checked={est === current}
          shortcut={String(i + 1)}
          onClick={() => { onSelect(est); onClose?.(); }}
        />
      ))}
    </Menu>
  );
}
