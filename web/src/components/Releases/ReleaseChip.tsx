import { forwardRef } from "react";
import { Package } from "lucide-react";
import type { Issue } from "../../api/types";
import { useReleases } from "../../api/queries";
import { cn } from "../../utils/cn";
import { popoverTriggerProps } from "../../utils/popover-trigger";
import { indicatorPill } from "../ui";
import Tooltip from "../ui/Tooltip";
import { releaseChipTooltip } from "./release-assign-utils";

interface ReleaseChipProps {
  issue: Issue;
  /** The release picker is open on this chip. */
  expanded: boolean;
  onOpen: () => void;
}

/**
 * The release an issue is in, on backlog rows and board cards: the Releases
 * icon and the name in an indicator pill. Like the row's other chips it is a
 * picker trigger; a click never opens the issue or starts a drag. Its host
 * renders the popover and anchors it to `[data-release-trigger]`.
 */
const ReleaseChip = forwardRef<HTMLButtonElement, ReleaseChipProps>(function ReleaseChip({ issue, expanded, onOpen }, ref) {
  const { data: board } = useReleases();
  if (!issue.release_id || !issue.release) return null;

  const release = board
    ? [board.current, board.next, ...board.upcoming, ...board.past].find((r) => r.id === issue.release_id) ?? null
    : null;
  const tooltip = releaseChipTooltip(issue.release, release);

  return (
    <Tooltip content={tooltip}>
      <button
        ref={ref}
        type="button"
        data-release-trigger
        aria-label={`${tooltip}, change release`}
        {...popoverTriggerProps("release", expanded ? "release" : null)}
        onClick={(e) => { e.stopPropagation(); onOpen(); }}
        onPointerDown={(e) => e.stopPropagation()}
        className={cn(indicatorPill, "max-w-32 hover:text-[var(--color-text-primary)] hover:border-[var(--color-border-default)] transition-colors")}
      >
        <Package size={12} className="shrink-0 text-[var(--color-accent-primary)]" />
        <span className="truncate">{issue.release}</span>
      </button>
    </Tooltip>
  );
});

export default ReleaseChip;
