import { forwardRef } from "react";
import { Milestone as MilestoneIcon } from "lucide-react";
import type { Issue } from "../../api/types";
import { useMilestones } from "../../api/queries";
import { cn } from "../../utils/cn";
import { popoverTriggerProps } from "../../utils/popover-trigger";
import { indicatorPill } from "../ui";
import Tooltip from "../ui/Tooltip";
import { milestoneChipTooltip } from "./milestone-assign-utils";

interface MilestoneChipProps {
  issue: Issue;
  /** The milestone picker is open on this chip. */
  expanded: boolean;
  onOpen: () => void;
}

/**
 * The milestone an issue is in, on backlog rows and board cards: the Milestones
 * icon and the name in an indicator pill. Like the row's other chips it is a
 * picker trigger; a click never opens the issue or starts a drag. Its host
 * renders the popover and anchors it to `[data-milestone-trigger]`.
 */
const MilestoneChip = forwardRef<HTMLButtonElement, MilestoneChipProps>(function MilestoneChip({ issue, expanded, onOpen }, ref) {
  const { data: board } = useMilestones();
  if (!issue.milestone_id || !issue.milestone) return null;

  const milestone = board
    ? [board.current, board.next, ...board.upcoming, ...board.past].find((r) => r.id === issue.milestone_id) ?? null
    : null;
  const tooltip = milestoneChipTooltip(issue.milestone, milestone);

  return (
    <Tooltip content={tooltip}>
      <button
        ref={ref}
        type="button"
        data-milestone-trigger
        aria-label={`${tooltip}, change milestone`}
        {...popoverTriggerProps("milestone", expanded ? "milestone" : null)}
        onClick={(e) => { e.stopPropagation(); onOpen(); }}
        onPointerDown={(e) => e.stopPropagation()}
        className={cn(indicatorPill, "max-w-32 hover:text-[var(--color-text-primary)] hover:border-[var(--color-border-default)] transition-colors")}
      >
        <MilestoneIcon size={12} className="shrink-0 text-[var(--color-accent-primary)]" />
        <span className="truncate">{issue.milestone}</span>
      </button>
    </Tooltip>
  );
});

export default MilestoneChip;
