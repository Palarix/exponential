import { useSortable } from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import type { Issue } from "../../api/client";
import { Avatar, BranchBadge, EstimateBadge, LabelBadge, PriorityIcon, SubProgress } from "../ui";
import { RefreshCw } from "lucide-react";
import { formatShortDate } from "../../utils/format";

export interface CardMeta {
  parentTitle?: string;
  childDone: number;
  childTotal: number;
}

export function SortableBoardCard({
  issue,
  meta,
  isFocused,
  onClick,
  onContextMenu,
}: {
  issue: Issue;
  meta: CardMeta;
  isFocused?: boolean;
  onClick?: () => void;
  onContextMenu?: (x: number, y: number) => void;
}) {
  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({ id: issue.id });

  const style: React.CSSProperties = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.4 : undefined,
  };

  return (
    <div
      ref={setNodeRef}
      style={style}
      data-board-card={issue.id}
      {...attributes}
      {...listeners}
      onClick={onClick}
      onContextMenu={(e) => {
        if (onContextMenu) {
          e.preventDefault();
          onContextMenu(e.clientX, e.clientY);
        }
      }}
      className={`px-3 py-2 rounded-[var(--radius-sm)] bg-[var(--color-surface-2)] border cursor-pointer transition-colors duration-[var(--duration-fast)] ${isFocused ? "border-[var(--color-accent-primary)] ring-1 ring-[var(--color-accent-primary)]" : "border-[var(--color-border-subtle)] hover:border-[var(--color-border-default)] hover:bg-[var(--color-hover-surface-2)]"}`}
    >
      <BoardCardContent issue={issue} meta={meta} />
    </div>
  );
}

export function BoardCard({
  issue,
  meta,
  isOverlay = false,
}: {
  issue: Issue;
  meta: CardMeta;
  isOverlay?: boolean;
}) {
  return (
    <div
      className={`px-3 py-2 rounded-[var(--radius-sm)] bg-[var(--color-surface-2)] border border-[var(--color-border-default)] ${isOverlay ? "shadow-lg cursor-grabbing" : ""}`}
    >
      <BoardCardContent issue={issue} meta={meta} />
    </div>
  );
}

function BoardCardContent({ issue, meta }: { issue: Issue; meta: CardMeta }) {
  const hasChildren = meta.childTotal > 0;
  const hasLabels = issue.labels && issue.labels.length > 0;
  const hasBranch = !!issue.branch_stats;
  const cycleId = issue.effective_cycle_id || issue.cycle_id;
  const hasBottom = hasChildren || hasLabels || hasBranch || cycleId;
  return (
    <>
      {/* NW: ID + parent | NE: pending, priority, assignee */}
      <div className="flex items-center gap-2 mb-1.5 min-w-0">
        <span className="font-mono text-xs text-[var(--color-text-muted)] shrink-0">
          {issue.id}
        </span>
        {meta.parentTitle && (
          <>
            <svg className="w-2.5 h-2.5 text-[var(--color-text-muted)] shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={3}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M9 5l7 7-7 7" />
            </svg>
            <span className="text-xs text-[var(--color-text-muted)] truncate">
              {meta.parentTitle}
            </span>
          </>
        )}
        <div className="flex-1" />
        {issue.is_pending && (
          <span className="w-2 h-2 rounded-full bg-[var(--color-warning)] shrink-0" />
        )}
        <EstimateBadge value={issue.estimate} />
        {issue.priority > 0 && <PriorityIcon priority={issue.priority} size={14} />}
        {issue.assignee && <Avatar name={issue.assignee} size="sm" />}
      </div>

      {/* Title */}
      <p className="text-sm font-medium text-[var(--color-text-primary)] leading-snug line-clamp-2">
        {issue.title}
      </p>

      {/* SW: labels, sub-progress | SE: estimate, date */}
      {hasBottom && (
        <div className="flex items-center gap-2 mt-2 min-w-0">
          <div className="flex items-center gap-1.5 flex-wrap flex-1 min-w-0">
            {hasChildren && (
              <span className="flex items-center gap-1 text-xs text-[var(--color-text-muted)] shrink-0">
                <SubProgress done={meta.childDone} total={meta.childTotal} />
                {meta.childDone}/{meta.childTotal}
              </span>
            )}
            {hasBranch && <BranchBadge stats={issue.branch_stats!} />}
            {issue.labels?.map((label) => (
              <LabelBadge key={label} label={label} />
            ))}
          </div>
          <div className="flex items-center gap-2 text-xs text-[var(--color-text-muted)] shrink-0">
            {cycleId && (
              <span className="flex items-center gap-0.5 tabular-nums">
                <RefreshCw className="w-3 h-3" />
                {cycleId}
              </span>
            )}
            <span className="tabular-nums">{formatShortDate(issue.created_at)}</span>
          </div>
        </div>
      )}
    </>
  );
}
