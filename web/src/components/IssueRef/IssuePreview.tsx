import type { RefObject } from "react";
import type { Issue } from "../../api/client";
import { PRIORITY_OPTIONS, STATUS_OPTIONS } from "../../constants";
import { ActorName, Avatar, Popover, StatusIcon } from "../ui";
import type { Placement } from "../ui/popover-utils";
import { PriorityIcon } from "../IssueDetail/icons";

/** Hover card with an issue's ID, assignee, title, status and priority. */
export function IssuePreview({
  anchorRef,
  issue,
  placement = "right-start",
}: {
  anchorRef: RefObject<HTMLElement | null>;
  issue: Issue;
  placement?: Placement;
}) {
  const status = STATUS_OPTIONS.find(s => s.value === issue.status)?.label ?? issue.status;
  const priority = PRIORITY_OPTIONS.find(p => p.value === (issue.priority || 0))?.label;
  return (
    <Popover anchorRef={anchorRef} onClose={() => {}} placement={placement} offset={8} className="pointer-events-none">
      <div role="tooltip" className="w-80 bg-[var(--color-surface-3)] border border-[var(--color-border-default)] rounded-[var(--radius-lg)] shadow-[var(--shadow-popover)] p-3">
        <div className="flex items-center gap-2 text-xs text-[var(--color-text-muted)]">
          <span className="font-mono">{issue.id}</span>
          {issue.assignee && (
            <span className="ml-auto flex items-center gap-1.5 min-w-0">
              <Avatar name={issue.assignee} size="sm" />
              <ActorName name={issue.assignee} via={issue.assignee_via} className="truncate text-[var(--color-text-secondary)]" />
            </span>
          )}
        </div>
        <div className="mt-1.5 text-sm text-[var(--color-text-primary)] truncate">{issue.title}</div>
        <div className="my-2.5 border-t border-[var(--color-border-subtle)]" />
        <div className="flex items-center gap-4 text-xs text-[var(--color-text-secondary)]">
          <span className="flex items-center gap-1.5">
            <StatusIcon status={issue.status} size={12} isInferred={issue.is_inferred} />
            {status}
          </span>
          <span className="flex items-center gap-1.5">
            <PriorityIcon priority={issue.priority || 0} size={12} />
            {priority}
          </span>
        </div>
      </div>
    </Popover>
  );
}
