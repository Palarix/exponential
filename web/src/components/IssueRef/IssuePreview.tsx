import type { ReactNode, RefObject } from "react";
import type { Issue } from "../../api/client";
import { PRIORITY_OPTIONS, STATUS_OPTIONS } from "../../constants";
import { ActorName, Avatar, LabelBadge, Popover, StatusIcon } from "../ui";
import type { Placement } from "../ui/popover-utils";
import { PriorityIcon } from "../IssueDetail/icons";

/**
 * What the card shows. `full` names the issue (ID, assignee, title, status,
 * priority) for links that show only an ID. `details` is for board rows that
 * already show the ID, title and status: it adds what they leave out.
 */
export type IssuePreviewVariant = "full" | "details";

/** Hover card for an issue, beside its anchor. */
export function IssuePreview({
  anchorRef,
  issue,
  placement = "right-start",
  variant = "full",
}: {
  anchorRef: RefObject<HTMLElement | null>;
  issue: Issue;
  placement?: Placement;
  variant?: IssuePreviewVariant;
}) {
  return (
    <Popover anchorRef={anchorRef} onClose={() => {}} placement={placement} offset={8} className="pointer-events-none">
      <IssuePreviewCard issue={issue} variant={variant} />
    </Popover>
  );
}

export function IssuePreviewCard({ issue, variant = "full" }: { issue: Issue; variant?: IssuePreviewVariant }) {
  const priority = PRIORITY_OPTIONS.find(p => p.value === (issue.priority || 0))?.label;
  const card = "w-80 bg-[var(--color-surface-3)] border border-[var(--color-border-default)] rounded-[var(--radius-lg)] shadow-[var(--shadow-popover)] p-3";

  if (variant === "details") {
    const labels = issue.labels ?? [];
    return (
      <div role="tooltip" className={card}>
        <dl className="grid grid-cols-[72px_minmax(0,1fr)] items-center gap-x-3 gap-y-2 text-xs">
          <DetailRow name="Assignee">
            {issue.assignee ? (
              <span className="flex items-center gap-1.5 min-w-0">
                <Avatar name={issue.assignee} size="sm" />
                <ActorName name={issue.assignee} via={issue.assignee_via} className="truncate" />
              </span>
            ) : <span className="text-[var(--color-text-muted)]">Unassigned</span>}
          </DetailRow>
          <DetailRow name="Priority">
            <span className="flex items-center gap-1.5">
              <PriorityIcon priority={issue.priority || 0} size={12} />
              {priority}
            </span>
          </DetailRow>
          {issue.estimate > 0 && <DetailRow name="Estimate"><span className="tabular-nums">{issue.estimate}</span></DetailRow>}
          {labels.length > 0 && (
            <DetailRow name="Labels">
              <span className="flex flex-wrap gap-1">
                {labels.map(l => <LabelBadge key={l} label={l} />)}
              </span>
            </DetailRow>
          )}
        </dl>
      </div>
    );
  }

  const status = STATUS_OPTIONS.find(s => s.value === issue.status)?.label ?? issue.status;
  return (
    <div role="tooltip" className={card}>
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
  );
}

function DetailRow({ name, children }: { name: string; children: ReactNode }) {
  return (
    <>
      <dt className="text-[var(--color-text-muted)]">{name}</dt>
      <dd className="min-w-0 text-[var(--color-text-secondary)]">{children}</dd>
    </>
  );
}
