import { useState } from "react";
import { useDroppable } from "@dnd-kit/core";
import { SortableContext, verticalListSortingStrategy } from "@dnd-kit/sortable";
import type { Issue } from "../../api/client";
import { StatusIcon } from "../ui";
import { SortableBoardCard, type CardMeta } from "./BoardCard";

const DONE_VISIBLE_COUNT = 5;

export default function BoardColumn({
  column,
  itemIds,
  getIssue,
  getCardMeta,
  activeId,
  focusedId,
  collapsed,
  onToggleCollapse,
  onIssueClick,
  onIssueContextMenu,
}: {
  column: { id: string; label: string; shortcut?: string; isBacklog?: boolean };
  itemIds: string[];
  getIssue: (id: string) => Issue | undefined;
  getCardMeta: (issue: Issue) => CardMeta;
  activeId: string | null;
  focusedId?: string | null;
  collapsed?: boolean;
  onToggleCollapse?: () => void;
  onIssueClick?: (issue: Issue) => void;
  onIssueContextMenu?: (issue: Issue, x: number, y: number) => void;
}) {
  const { setNodeRef, isOver } = useDroppable({ id: `column-${column.id}` });
  const [showAll, setShowAll] = useState(false);
  const isDone = column.id === "DONE";
  const shouldTruncate = isDone && itemIds.length > DONE_VISIBLE_COUNT && !showAll;
  const visibleIds = shouldTruncate ? itemIds.slice(0, DONE_VISIBLE_COUNT) : itemIds;
  const hiddenCount = itemIds.length - DONE_VISIBLE_COUNT;

  const containsActive = activeId !== null && itemIds.includes(activeId);
  const showHighlight = isOver || containsActive;

  const borderStyle = column.isBacklog ? "border-dashed" : "";

  if (collapsed) {
    return (
      <button
        onClick={onToggleCollapse}
        className={`flex flex-col items-center gap-2 w-10 shrink-0 rounded-[var(--radius-md)] bg-[var(--color-bg-secondary)]/40 border border-[var(--color-border-subtle)] ${borderStyle} py-3 hover:bg-[var(--color-bg-hover)] transition-colors cursor-pointer`}
      >
        <StatusIcon status={column.id} size={14} />
        <span className="text-xs font-medium text-[var(--color-text-muted)] [writing-mode:vertical-lr] rotate-180">
          {column.label}
        </span>
        <span className="text-[10px] text-[var(--color-text-muted)] tabular-nums">{itemIds.length}</span>
      </button>
    );
  }

  return (
    <div
      className={`flex flex-col flex-1 min-w-0 rounded-[var(--radius-md)] bg-[var(--color-bg-secondary)]/40 transition-colors duration-[var(--duration-fast)] ${showHighlight ? "ring-2 ring-inset ring-[var(--color-accent-primary)] bg-[var(--color-accent-primary)]/5" : ""} ${column.isBacklog ? "border border-dashed border-[var(--color-border-default)]" : ""}`}
    >
      <div className="flex items-center justify-between px-3 py-2 border-b border-[var(--color-border-subtle)]">
        <div className="flex items-center gap-2">
          <StatusIcon status={column.id} size={14} />
          <span className="text-sm font-medium text-[var(--color-text-primary)]">{column.label}</span>
          {column.shortcut && (
            <kbd className="inline-flex items-center justify-center min-w-4 h-4 px-1 text-[10px] font-medium text-[var(--color-text-muted)] bg-[var(--color-bg-tertiary)] border border-[var(--color-border-default)] rounded-[var(--radius-sm)]">
              {column.shortcut}
            </kbd>
          )}
        </div>
        <div className="flex items-center gap-2">
          <span className="flex items-center gap-1 text-xs text-[var(--color-text-muted)] tabular-nums">
            <svg className="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M5.25 8.25h15m-16.5 7.5h15m-1.8-13.5l-3.9 19.5m-2.1-19.5l-3.9 19.5" />
            </svg>
            {itemIds.length}
          </span>
          <button
            onClick={onToggleCollapse}
            className="text-[var(--color-text-muted)] hover:text-[var(--color-text-secondary)] transition-colors"
            title="Collapse column"
          >
            <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M18.75 19.5l-7.5-7.5 7.5-7.5m-6 15L5.25 12l7.5-7.5" />
            </svg>
          </button>
        </div>
      </div>

      <SortableContext items={visibleIds} strategy={verticalListSortingStrategy}>
        <div ref={setNodeRef} className="flex-1 p-2 space-y-2 overflow-y-auto">
          {visibleIds.map((id) => {
            const issue = getIssue(id);
            if (!issue) return null;
            return (
              <SortableBoardCard
                key={id}
                issue={issue}
                meta={getCardMeta(issue)}
                isFocused={id === focusedId}
                onClick={() => onIssueClick?.(issue)}
                onContextMenu={(x, y) => onIssueContextMenu?.(issue, x, y)}
              />
            );
          })}
          {shouldTruncate && (
            <button
              onClick={() => setShowAll(true)}
              className="w-full py-2 text-xs text-[var(--color-text-muted)] hover:text-[var(--color-text-secondary)] transition-colors"
            >
              + {hiddenCount} more
            </button>
          )}
          {itemIds.length === 0 && (
            <div className="flex items-center justify-center h-16 text-[var(--color-text-muted)] text-xs">
              No issues
            </div>
          )}
        </div>
      </SortableContext>
    </div>
  );
}
