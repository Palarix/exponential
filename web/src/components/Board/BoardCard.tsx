import { useSortable } from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import type { Issue } from "../../api/client";
import { Avatar, LabelBadge, SubProgress } from "../ui";
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
}: {
  issue: Issue;
  meta: CardMeta;
  isFocused?: boolean;
  onClick?: () => void;
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
      className={`px-3 py-2 rounded-[var(--radius-sm)] bg-[var(--color-bg-elevated)] border cursor-pointer transition-colors duration-[var(--duration-fast)] ${isFocused ? "border-[var(--color-accent-primary)] ring-1 ring-[var(--color-accent-primary)]" : "border-[var(--color-border-subtle)] hover:border-[var(--color-border-default)] hover:bg-[var(--color-bg-hover)]"}`}
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
      className={`px-3 py-2 rounded-[var(--radius-sm)] bg-[var(--color-bg-elevated)] border border-[var(--color-border-default)] ${isOverlay ? "shadow-lg cursor-grabbing" : ""}`}
    >
      <BoardCardContent issue={issue} meta={meta} />
    </div>
  );
}

function BoardCardContent({ issue, meta }: { issue: Issue; meta: CardMeta }) {
  const hasChildren = meta.childTotal > 0;
  return (
    <>
      <div className="flex items-center gap-2 mb-1 min-w-0">
        <span className="font-mono text-xs text-[var(--color-text-muted)] shrink-0">
          {issue.id}
        </span>
        {meta.parentTitle && (
          <>
            <svg className="w-3 h-3 text-[var(--color-text-muted)] shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={3}>
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
        {issue.assignee && <Avatar name={issue.assignee} size="xs" />}
      </div>

      <p className="text-sm text-[var(--color-text-primary)] leading-snug line-clamp-2 mb-2">
        {issue.title}
      </p>

      {(hasChildren || (issue.labels && issue.labels.length > 0)) && (
        <div className="flex items-center gap-2 text-xs flex-wrap mb-2">
          {hasChildren && (
            <span className="flex items-center gap-1 text-[var(--color-text-muted)] shrink-0">
              <SubProgress done={meta.childDone} total={meta.childTotal} />
              {meta.childDone}/{meta.childTotal}
            </span>
          )}
          {issue.labels?.map((label) => (
            <LabelBadge key={label} label={label} />
          ))}
        </div>
      )}

      <div className="flex items-center gap-2 text-xs text-[var(--color-text-muted)]">
        {issue.priority > 0 && issue.priority <= 3 && (
          <span className={`font-medium shrink-0 ${issue.priority === 1 ? "text-[var(--color-error)]" : issue.priority === 2 ? "text-[var(--color-warning)]" : "text-[var(--color-text-muted)]"}`}>
            {issue.priority === 1 ? "!!!" : issue.priority === 2 ? "!!" : "!"}
          </span>
        )}
        <span className="tabular-nums">{formatShortDate(issue.created_at)}</span>
        <div className="flex-1" />
        {issue.estimate > 0 && (
          <span className="tabular-nums shrink-0">{issue.estimate}pt</span>
        )}
      </div>
    </>
  );
}
