import { useCallback, useEffect, useRef, useState, type HTMLAttributes, type ReactNode } from "react";
import { ChevronRight } from "lucide-react";
import type { Issue } from "../../api/types";
import { isTerminal } from "../../constants";
import { cn } from "../../utils/cn";
import { CopyableId, indicatorPill, StatusIcon, SubProgress } from "../ui";
import Tooltip from "../ui/Tooltip";
import { createHoverIntent } from "../IssueDetail/hover-intent";
import { IssuePreview } from "../IssueRef/IssuePreview";
import { childProgress, parentContext } from "./releases-utils";

/** How long the pointer rests on an issue row before its hover card opens (as on issue chips). */
const PREVIEW_DELAY = 250;

/**
 * A release's issues, flat. Read-only rows by default: a past release's list
 * is what it delivered, not to be moved. `renderRow` swaps in draggable rows.
 */
export function ReleaseIssueList({ issues, all, onIssueClick, empty = "No issues yet.", renderRow }: {
  issues: Issue[];
  all: Issue[];
  onIssueClick: (issue: Issue) => void;
  empty?: string;
  renderRow?: (issue: Issue) => ReactNode;
}) {
  if (issues.length === 0) {
    return <p className="px-2 py-2 text-xs text-[var(--color-text-muted)]">{empty}</p>;
  }
  return (
    <div className="border-t border-[var(--color-border-subtle)] pt-2 space-y-px">
      {issues.map((issue) => renderRow
        ? renderRow(issue)
        : <IssueRow key={issue.id} issue={issue} all={all} onIssueClick={onIssueClick} />)}
    </div>
  );
}

/**
 * One issue, compact for the half-width lists but in the backlog's order and
 * styles: ID, status, the parent as a "Parent ›" breadcrumb, the title, and
 * the sub-issue progress pill. Priority, labels and the work indicators stay
 * in the backlog; the hover card has the details. The breadcrumb gives way
 * before the title does. A click opens the issue. The row is a div, not a
 * button: it holds CopyableId, which is a button itself. Resting on the row
 * shows the issue hover card beside it, which closes when a drag starts.
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
  const rowRef = useRef<HTMLDivElement | null>(null);
  const [preview, setPreview] = useState(false);
  const [intent] = useState(() => createHoverIntent(PREVIEW_DELAY, setPreview));
  useEffect(() => () => intent.cancel(), [intent]);
  const setRefs = useCallback((el: HTMLDivElement | null) => {
    rowRef.current = el;
    setNodeRef?.(el);
  }, [setNodeRef]);
  const open = () => { intent.leave(); onIssueClick(issue); };
  return (
    <>
      <div
        ref={setRefs}
        role="button"
        tabIndex={0}
        onClick={open}
        onKeyDown={(e) => { if (e.key === "Enter") open(); }}
        onMouseEnter={() => intent.enter()}
        onMouseLeave={() => intent.leave()}
        onPointerDownCapture={() => intent.leave()}
        className={cn(
          "w-full flex items-center gap-3 px-3 h-10 rounded-[var(--radius-md)] cursor-pointer select-none transition-colors hover:bg-[var(--color-hover-surface-1)]",
          dragProps && "touch-none",
          isTerminal(issue.status) && "opacity-60",
          dragging && "opacity-30",
        )}
        {...dragProps}
      >
        <CopyableId id={issue.id} className="text-xs text-left shrink-0 tabular-nums" />
        <StatusIcon status={issue.status} size={14} isInferred={issue.is_inferred} />
        <span className="flex items-center gap-2 flex-1 min-w-0">
          {parent && <ParentCrumb parent={parent.parent} where={parent.where} />}
          <span className="text-sm text-[var(--color-text-primary)] truncate min-w-0 flex-1">{issue.title}</span>
        </span>
        {children && (
          <span className={indicatorPill}>
            <SubProgress done={children.done} total={children.total} size={12} />
            {children.done}/{children.total}
          </span>
        )}
      </div>
      {preview && !dragging && <IssuePreview anchorRef={rowRef} issue={issue} />}
    </>
  );
}

/**
 * A sub-issue's parent as the backlog shows it, "Parent ›" before the title;
 * the tooltip names the parent and the release it is in.
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
