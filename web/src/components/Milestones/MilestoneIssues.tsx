import { useEffect, useRef, useState, type HTMLAttributes, type ReactNode } from "react";
import { ChevronRight } from "lucide-react";
import type { Issue } from "../../api/types";
import { isTerminal } from "../../constants";
import { cn } from "../../utils/cn";
import { Avatar, CopyableId, EstimateBadge, indicatorPill, OverflowLabels, StatusIcon, SubProgress } from "../ui";
import Tooltip from "../ui/Tooltip";
import { createHoverIntent } from "../IssueDetail/hover-intent";
import { IssuePreview } from "../IssueRef/IssuePreview";
import { childProgress, parentContext, rowEstimate } from "./milestones-utils";

/** How long the pointer rests on an issue title before its hover card opens (as on issue chips). */
const PREVIEW_DELAY = 250;

/** Rows in an expanded milestone start their content in the milestone title's column. */
export const ROW_INSET = "pl-7 pr-2";

/** A muted note in the list of an expanded row, inset like the issue rows. */
export function ListNote({ children }: { children: ReactNode }) {
  return <div className={cn(ROW_INSET, "py-2 text-xs text-[var(--color-text-muted)]")}>{children}</div>;
}

/**
 * A milestone's issues, flat. Read-only rows by default: a past milestone's list
 * is what it delivered, not to be moved. `renderRow` swaps in draggable rows.
 */
export function MilestoneIssueList({ issues, all, onIssueClick, empty = "No issues.", renderRow }: {
  issues: Issue[];
  all: Issue[];
  onIssueClick: (issue: Issue) => void;
  empty?: ReactNode;
  renderRow?: (issue: Issue) => ReactNode;
}) {
  if (issues.length === 0) return <ListNote>{empty}</ListNote>;
  return (
    <div className="space-y-px">
      {issues.map((issue) => renderRow
        ? renderRow(issue)
        : <IssueRow key={issue.id} issue={issue} all={all} onIssueClick={onIssueClick} />)}
    </div>
  );
}

/**
 * One issue in the backlog's order and styles: ID, status, the parent as a
 * "Parent ›" breadcrumb, the title and the sub-issue progress pill, then on
 * the far right labels, estimate and assignee. Those are read-only here: the
 * backlog's pickers belong to its keyboard navigation and popovers. The hover
 * card names the issue and shows what the row leaves out: a description
 * excerpt and counts of comments,
 * attachments, sub-issues, commits and open blockers. The breadcrumb gives way
 * before the title does. A click opens the issue. The row is a div, not a
 * button: it holds CopyableId, which is a button itself. Resting on the title
 * shows the issue hover card above it, which closes when a drag starts.
 */
export function IssueRow({ issue, all, onIssueClick, setNodeRef, dragProps, dragging = false }: {
  issue: Issue;
  /** Every issue, for the parent breadcrumb and the child progress. */
  all: Issue[];
  onIssueClick: (issue: Issue) => void;
  setNodeRef?: (el: HTMLElement | null) => void;
  dragProps?: HTMLAttributes<HTMLDivElement>;
  dragging?: boolean;
}) {
  const parent = parentContext(issue, all);
  const children = childProgress(issue, all);
  const titleRef = useRef<HTMLSpanElement | null>(null);
  const [preview, setPreview] = useState(false);
  const [intent] = useState(() => createHoverIntent(PREVIEW_DELAY, setPreview));
  useEffect(() => () => intent.cancel(), [intent]);
  const open = () => { intent.leave(); onIssueClick(issue); };
  return (
    <>
      <div
        ref={setNodeRef}
        // OverflowLabels fits the labels to a share of this row's width.
        data-backlog-row
        role="button"
        tabIndex={0}
        onClick={open}
        onKeyDown={(e) => { if (e.key === "Enter") open(); }}
        onPointerDownCapture={() => intent.leave()}
        className={cn(
          "w-full flex items-center gap-3 h-10 rounded-[var(--radius-md)] cursor-pointer select-none transition-colors hover:bg-[var(--color-hover-surface)]",
          ROW_INSET,
          dragProps && "touch-none",
          isTerminal(issue.status) && "opacity-60",
          dragging && "opacity-30",
        )}
        {...dragProps}
      >
        <CopyableId id={issue.id} className="text-xs text-left shrink-0 tabular-nums" />
        <StatusIcon status={issue.status} size={14} isInferred={issue.is_inferred} />
        <span className="flex items-center gap-2 min-w-0">
          {parent && <ParentCrumb parent={parent.parent} where={parent.where} />}
          <span
            ref={titleRef}
            onMouseEnter={() => intent.enter()}
            onMouseLeave={() => intent.leave()}
            className="text-sm text-[var(--color-text-primary)] truncate min-w-0"
          >
            {issue.title}
          </span>
        </span>
        {children && (
          <span className={cn(indicatorPill, "shrink-0")}>
            <SubProgress done={children.done} total={children.total} size={12} />
            {children.done}/{children.total}
          </span>
        )}
        <div className="flex-1" />
        {issue.labels && issue.labels.length > 0 && (
          // Below a stack width of 40rem the title keeps the room.
          <span className="shrink-0 hidden @[40rem]:inline-flex">
            <OverflowLabels labels={issue.labels} />
          </span>
        )}
        <span className="flex items-center w-10 justify-end shrink-0">
          <EstimateBadge value={rowEstimate(issue, all)} />
        </span>
        <span className="flex shrink-0">
          {issue.assignee ? (
            <Avatar name={issue.assignee} size="sm" />
          ) : (
            <span className="flex opacity-50" title="No assignee">
              <Avatar size="sm" />
            </span>
          )}
        </span>
      </div>
      {preview && !dragging && <IssuePreview anchorRef={titleRef} issue={issue} placement="top-start" />}
    </>
  );
}

/**
 * A sub-issue's parent as the backlog shows it, "Parent ›" before the title;
 * the tooltip names the parent and the milestone it is in.
 */
function ParentCrumb({ parent, where }: { parent: Issue; where: string }) {
  const text = `Parent: ${parent.id} ${parent.title} · ${where}`;
  return (
    <Tooltip content={text}>
      <span aria-label={text} className="flex items-center gap-2 min-w-0 max-w-[40%] shrink-[3]">
        <span className="text-sm text-[var(--color-text-muted)] truncate">{parent.title}</span>
        <ChevronRight className="w-3 h-3 text-[var(--color-text-muted)] shrink-0" />
      </span>
    </Tooltip>
  );
}
