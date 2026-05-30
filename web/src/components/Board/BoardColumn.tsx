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
  onIssueClick,
}: {
  column: { id: string; label: string };
  itemIds: string[];
  getIssue: (id: string) => Issue | undefined;
  getCardMeta: (issue: Issue) => CardMeta;
  activeId: string | null;
  onIssueClick?: (issue: Issue) => void;
}) {
  const { setNodeRef, isOver } = useDroppable({ id: `column-${column.id}` });
  const [showAll, setShowAll] = useState(false);
  const isDone = column.id === "DONE";
  const shouldTruncate = isDone && itemIds.length > DONE_VISIBLE_COUNT && !showAll;
  const visibleIds = shouldTruncate ? itemIds.slice(0, DONE_VISIBLE_COUNT) : itemIds;
  const hiddenCount = itemIds.length - DONE_VISIBLE_COUNT;

  const containsActive = activeId !== null && itemIds.includes(activeId);
  const showHighlight = isOver || containsActive;

  return (
    <div
      className={`flex flex-col flex-1 min-w-0 rounded-[var(--radius-md)] bg-[var(--color-bg-secondary)]/40 transition-colors duration-[var(--duration-fast)] ${showHighlight ? "ring-2 ring-inset ring-[var(--color-accent-primary)] bg-[var(--color-accent-primary)]/5" : ""}`}
    >
      <div className="flex items-center justify-between px-3 py-2 border-b border-[var(--color-border-subtle)]">
        <div className="flex items-center gap-2">
          <StatusIcon status={column.id} size={14} />
          <span className="text-sm font-medium text-[var(--color-text-primary)]">{column.label}</span>
        </div>
        <span className="text-xs text-[var(--color-text-muted)] tabular-nums">{itemIds.length}</span>
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
                onClick={() => onIssueClick?.(issue)}
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
