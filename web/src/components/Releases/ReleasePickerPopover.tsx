import type { RefObject } from "react";
import type { Issue } from "../../api/types";
import { useReleases } from "../../api/queries";
import Popover, { PopoverPanel } from "../ui/Popover";
import { useReleaseAssign } from "./release-assign-context";
import { pickerOptions } from "./release-assign-utils";
import ReleasePicker from "./ReleasePicker";

/** The release picker for `issue` as a popover on `anchorRef`; a pick goes through the assign path. */
export default function ReleasePickerPopover({ issue, anchorRef, onClose }: {
  issue: Issue;
  anchorRef: RefObject<HTMLElement | null>;
  onClose: () => void;
}) {
  const { data: releases } = useReleases();
  const { assign } = useReleaseAssign();
  if (!releases) return null;
  // The popover is portalled, but React still bubbles its events through the
  // host: without this a pick would also click the row (opening the issue)
  // or start a drag.
  const stop = (e: React.SyntheticEvent) => e.stopPropagation();
  return (
    <span className="contents" onClick={stop} onPointerDown={stop}>
      <Popover anchorRef={anchorRef} onClose={onClose}>
        <PopoverPanel>
          <ReleasePicker
            options={pickerOptions(releases, issue.release_id)}
            current={issue.release_id}
            onSelect={(release) => assign([issue.id], release)}
            onClose={onClose}
          />
        </PopoverPanel>
      </Popover>
    </span>
  );
}
