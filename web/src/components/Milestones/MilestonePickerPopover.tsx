import type { RefObject } from "react";
import type { Issue } from "../../api/types";
import { useMilestones } from "../../api/queries";
import Popover, { PopoverPanel } from "../ui/Popover";
import { useMilestoneAssign } from "./milestone-assign-context";
import { pickerOptions } from "./milestone-assign-utils";
import MilestonePicker from "./MilestonePicker";

/** The milestone picker for `issue` as a popover on `anchorRef`; a pick goes through the assign path. */
export default function MilestonePickerPopover({ issue, anchorRef, onClose }: {
  issue: Issue;
  anchorRef: RefObject<HTMLElement | null>;
  onClose: () => void;
}) {
  const { data: milestones } = useMilestones();
  const { assign } = useMilestoneAssign();
  if (!milestones) return null;
  // The popover is portalled, but React still bubbles its events through the
  // host: without this a pick would also click the row (opening the issue)
  // or start a drag.
  const stop = (e: React.SyntheticEvent) => e.stopPropagation();
  return (
    <span className="contents" onClick={stop} onPointerDown={stop}>
      <Popover anchorRef={anchorRef} onClose={onClose}>
        <PopoverPanel>
          <MilestonePicker
            options={pickerOptions(milestones, issue.milestone_id)}
            current={issue.milestone_id}
            onSelect={(milestone) => assign([issue.id], milestone)}
            onClose={onClose}
          />
        </PopoverPanel>
      </Popover>
    </span>
  );
}
