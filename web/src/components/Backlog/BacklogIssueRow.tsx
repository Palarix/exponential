import { memo } from "react";
import type { Issue } from "../../api/client";
import {
  Avatar,
  BranchBadge,
  CopyableId,
  EstimateBadge,
  OverflowLabels,
  Popover,
  StatusIcon,
  PriorityIcon,
  StatusPicker,
  PriorityPicker,
  LabelPicker,
  EstimatePicker,
  SubProgress,
} from "../ui";
import { ChevronRight, Paperclip } from "lucide-react";
import { formatShortDate } from "../../utils/format";
import { isTerminal } from "../../constants";
import { IssueRowDnd } from "./DndComponents";
import type { TreeGuide, HierarchyMode } from "./useBacklogRows";

type PopoverType = "status" | "labels" | "estimate" | "priority";

interface Props {
  issue: Issue;
  depth: number;
  hasChildren: boolean;
  childDone: number;
  childTotal: number;
  childPointsDone: number;
  childPointsTotal: number;
  parentBreadcrumb?: string;
  isGhostParent?: boolean;
  isGhostChild?: boolean;
  treeGuides: TreeGuide[];
  rowIndex: number;
  hierarchyMode: HierarchyMode;
  isFocused: boolean;
  keyboardNav: boolean;
  isNodeExpanded: boolean;
  canDrag: boolean;
  isDropTarget: boolean;
  dndId: string;
  isDraggedOrBatch: boolean;
  dragBatchCount: number;
  popoverType: PopoverType | null;
  allKnownLabels: string[];
  cycleNumber?: number;
  onIssueClick?: (issue: Issue) => void;
  onToggleNode: (id: string) => void;
  onMouseEnter: (index: number) => void;
  onOpenPopover: (rowIndex: number, type: PopoverType) => void;
  onClosePopover: () => void;
  onQuickStatus: (issueId: string, status: string) => void;
  onQuickPriority: (issueId: string, priority: number) => void;
  onQuickEstimate: (issueId: string, estimate: number) => void;
  onQuickLabelToggle: (issue: Issue, label: string) => void;
  onConfigLabelsChange?: (labels: Record<string, string>) => void;
}

export const BacklogIssueRow = memo(function BacklogIssueRow({
  issue,
  depth,
  hasChildren,
  childDone,
  childTotal,
  childPointsDone,
  childPointsTotal,
  parentBreadcrumb,
  isGhostParent,
  isGhostChild,
  treeGuides,
  rowIndex,
  hierarchyMode,
  isFocused,
  keyboardNav,
  isNodeExpanded,
  canDrag,
  isDropTarget,
  dndId,
  isDraggedOrBatch,
  dragBatchCount,
  popoverType,
  allKnownLabels,
  cycleNumber,
  onIssueClick,
  onToggleNode,
  onMouseEnter,
  onOpenPopover,
  onClosePopover,
  onQuickStatus,
  onQuickPriority,
  onQuickEstimate,
  onQuickLabelToggle,
  onConfigLabelsChange,
}: Props) {
  const indent = depth * 24;
  const isGhostRow = !!isGhostParent || !!isGhostChild;

  return (
    <div className="relative" data-context-issue={issue.id} data-depth={depth}>
      <IssueRowDnd id={dndId} canDrag={canDrag} enabled={isDropTarget}>
        {(setRowRef, dragProps) => (
          <div
            ref={setRowRef}
            data-row={rowIndex}
            data-backlog-row
            {...dragProps.attributes}
            {...dragProps.listeners}
            onClick={() => onIssueClick?.(issue)}
            onMouseEnter={() => onMouseEnter(rowIndex)}
            className={`relative flex items-center gap-3 px-5 h-10 border-b border-[var(--color-border-subtle)] cursor-pointer transition-colors duration-[var(--duration-fast)] select-none group ${isGhostRow ? "opacity-50" : ""} ${isFocused && keyboardNav ? "bg-[var(--color-hover-surface)] ring-1 ring-inset ring-[var(--color-accent-primary)]/40" : isFocused ? "bg-[var(--color-hover-surface)]" : keyboardNav ? "" : "hover:bg-[var(--color-hover-surface)]"} ${isDraggedOrBatch ? "opacity-40" : ""}`}
            style={{ paddingLeft: `${20 + indent}px` }}
          >
            {treeGuides.map((guide, k) =>
              guide !== "blank" ? (
                <svg
                  key={k}
                  className="absolute top-0 h-10 pointer-events-none text-[var(--color-border-default)]"
                  style={{ left: `${20 + k * 24}px`, width: "24px" }}
                  viewBox="0 0 24 40"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="1.5"
                >
                  {(guide === "pipe" || guide === "tee") && (
                    <line x1="8" y1="0" x2="8" y2="40" />
                  )}
                  {guide === "corner" && (
                    <line x1="8" y1="0" x2="8" y2="20" />
                  )}
                  {(guide === "tee" || guide === "corner") && (
                    <line x1="8" y1="20" x2="24" y2="20" />
                  )}
                </svg>
              ) : null,
            )}
            {hasChildren && hierarchyMode === "nested" ? (
              <button
                onClick={(e) => {
                  e.stopPropagation();
                  onToggleNode(issue.id);
                }}
                className="w-6 h-6 -m-1 shrink-0 flex items-center justify-center rounded cursor-pointer text-[var(--color-text-muted)] hover:text-[var(--color-text-secondary)] hover:bg-white/10"
              >
                <svg
                  className={`w-3 h-3 transition-transform duration-100 ${isNodeExpanded ? "rotate-90" : ""}`}
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                  strokeWidth={2.5}
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    d="M9 5l7 7-7 7"
                  />
                </svg>
              </button>
            ) : (
              <span className="w-4 shrink-0" />
            )}
            <div
              className="relative shrink-0"
              onClick={(e) => e.stopPropagation()}
            >
              <button
                onClick={() =>
                  popoverType === "priority"
                    ? onClosePopover()
                    : onOpenPopover(rowIndex, "priority")
                }
                className="w-6 h-6 -m-1 flex items-center justify-center rounded cursor-pointer hover:bg-white/10 transition-colors"
              >
                <PriorityIcon priority={issue.priority || 0} size={16} />
              </button>
              {popoverType === "priority" && (
                <Popover onClose={onClosePopover}>
                  <PriorityPicker
                    current={issue.priority || 0}
                    onSelect={(v) => onQuickPriority(issue.id, v)}
                    onClose={onClosePopover}
                  />
                </Popover>
              )}
            </div>
            <CopyableId
              id={issue.id}
              className="text-xs text-left shrink-0 tabular-nums"
            />
            <div
              className="relative shrink-0"
              onClick={(e) => e.stopPropagation()}
            >
              <button
                onClick={() =>
                  popoverType === "status"
                    ? onClosePopover()
                    : onOpenPopover(rowIndex, "status")
                }
                className="w-6 h-6 -m-1 flex items-center justify-center rounded cursor-pointer hover:bg-white/10 transition-colors"
              >
                <StatusIcon
                  status={issue.status}
                  size={14}
                  isInferred={issue.is_inferred}
                />
              </button>
              {popoverType === "status" && (
                <Popover onClose={onClosePopover}>
                  <StatusPicker
                    current={issue.status}
                    onSelect={(v) => onQuickStatus(issue.id, v)}
                    onClose={onClosePopover}
                  />
                </Popover>
              )}
            </div>
            {parentBreadcrumb && (
              <span className="text-sm text-[var(--color-text-muted)] truncate shrink-0 max-w-38">
                {parentBreadcrumb}
              </span>
            )}
            {parentBreadcrumb && (
              <ChevronRight className="w-3 h-3 text-[var(--color-text-muted)] shrink-0" />
            )}
            <span
              className={`text-sm truncate min-w-0 ${isGhostRow ? "text-[var(--color-text-muted)]" : "text-[var(--color-text-primary)]"}`}
            >
              {issue.title}
            </span>
            {dragBatchCount > 1 && (
              <span className="flex items-center justify-center w-5 h-5 rounded-full bg-[var(--color-accent-primary)] text-white text-xs font-medium shrink-0">
                {dragBatchCount}
              </span>
            )}
            {hasChildren && (
              <span className="flex items-center gap-2 text-xs text-[var(--color-text-muted)] shrink-0">
                <SubProgress done={childDone} total={childTotal} />
                {childDone}/{childTotal}
              </span>
            )}
            {issue.branch_stats && (
              <BranchBadge stats={issue.branch_stats} />
            )}
            {issue.artifacts && issue.artifacts.length > 0 && (
              <span className="inline-flex items-center gap-1 h-6 px-2 rounded-md border border-[var(--color-border-label)] text-xs text-[var(--color-text-secondary)] tabular-nums shrink-0">
                <Paperclip size={12} strokeWidth={1.5} />
                {issue.artifacts.length}
              </span>
            )}
            <div className="flex-1" />
            {issue.is_pending && (
              <span className="w-2 h-2 rounded-full bg-[var(--color-warning)] shrink-0" />
            )}
            <div
              className="relative flex items-center gap-3 shrink-0"
              onClick={(e) => e.stopPropagation()}
            >
              <button
                onClick={() =>
                  popoverType === "labels"
                    ? onClosePopover()
                    : onOpenPopover(rowIndex, "labels")
                }
                className="flex items-center gap-3 hover:opacity-70 transition-opacity"
              >
                {issue.labels && issue.labels.length > 0 ? (
                  <OverflowLabels labels={issue.labels} />
                ) : (
                  <span className="text-xs text-[var(--color-text-muted)] opacity-0 group-hover:opacity-100 transition-opacity">
                    + label
                  </span>
                )}
              </button>
              {popoverType === "labels" && (
                <Popover onClose={onClosePopover}>
                  <LabelPicker
                    allLabels={allKnownLabels}
                    selected={issue.labels || []}
                    onToggle={(label) => onQuickLabelToggle(issue, label)}
                    onConfigLabelsChange={onConfigLabelsChange}
                    onClose={onClosePopover}
                  />
                </Popover>
              )}
            </div>
            {issue.cycle_id && (
              <span
                className="flex items-center gap-1 text-xs text-[var(--color-text-muted)] shrink-0"
                title={`Cycle ${cycleNumber ?? issue.cycle_id}`}
              >
                <svg
                  className="w-3.5 h-3.5"
                  viewBox="0 0 20 20"
                  fill="none"
                >
                  <circle
                    cx="10"
                    cy="10"
                    r="9"
                    stroke="currentColor"
                    strokeWidth="1.5"
                  />
                  <path
                    d="M7.5 5.5v9l7-4.5z"
                    fill="currentColor"
                  />
                </svg>
                {cycleNumber ?? ""}
              </span>
            )}
            <div
              className="relative shrink-0"
              onClick={(e) => e.stopPropagation()}
            >
              <button
                onClick={() =>
                  popoverType === "estimate"
                    ? onClosePopover()
                    : onOpenPopover(rowIndex, "estimate")
                }
                className="flex items-center w-10 justify-end hover:opacity-70 transition-opacity"
              >
                <EstimateBadge
                  value={
                    hasChildren
                      ? isTerminal(issue.status)
                        ? childPointsDone
                        : childPointsTotal - childPointsDone
                      : issue.estimate
                  }
                />
              </button>
              {popoverType === "estimate" && (
                <Popover onClose={onClosePopover}>
                  <EstimatePicker
                    current={issue.estimate || 0}
                    onSelect={(v) => onQuickEstimate(issue.id, v)}
                    onClose={onClosePopover}
                  />
                </Popover>
              )}
            </div>
            {issue.assignee && (
              <Avatar name={issue.assignee} size="sm" />
            )}
            <span className="text-xs text-[var(--color-text-muted)] tabular-nums shrink-0 w-16 text-right">
              {formatShortDate(issue.created_at)}
            </span>
          </div>
        )}
      </IssueRowDnd>
    </div>
  );
});
