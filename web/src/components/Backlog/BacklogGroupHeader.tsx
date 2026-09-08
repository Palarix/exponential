import { memo } from "react";
import { StatusIcon } from "../ui";
import { GroupHeaderDnd } from "./DndComponents";

interface Props {
  status: string;
  label: string;
  isEmpty: boolean;
  isExpanded: boolean;
  count: number;
  storyPoints: number;
  groupIndex: number;
  isFocused: boolean;
  keyboardNav: boolean;
  isDndEnabled: boolean;
  hasActiveId: boolean;
  showStoryPoints: boolean;
  onToggle: (status: string) => void;
  onToggleStoryPoints: () => void;
  onStartInlineCreate: (status: string) => void;
  onMouseEnter: (index: number) => void;
}

export const BacklogGroupHeader = memo(function BacklogGroupHeader({
  status,
  label,
  isEmpty,
  isExpanded,
  count,
  storyPoints,
  groupIndex,
  isFocused,
  keyboardNav,
  isDndEnabled,
  hasActiveId,
  showStoryPoints,
  onToggle,
  onToggleStoryPoints,
  onStartInlineCreate,
  onMouseEnter,
}: Props) {
  return (
    <GroupHeaderDnd status={status} enabled={isDndEnabled && hasActiveId}>
      {(setHeaderRef) => (
        <div
          ref={setHeaderRef}
          data-row={groupIndex}
          data-group-header
          onClick={() => !isEmpty && onToggle(status)}
          onMouseEnter={() => onMouseEnter(groupIndex)}
          className={`flex items-center gap-3 w-full px-5 py-2 border-b border-[var(--color-border-subtle)] transition-colors duration-[var(--duration-fast)] select-none ${isEmpty ? "opacity-40 cursor-default" : "cursor-pointer"} ${isFocused && keyboardNav ? "bg-[var(--color-hover-surface)] ring-1 ring-inset ring-[var(--color-accent-primary)]/40" : isFocused ? "bg-[var(--color-hover-surface)]" : "bg-[var(--color-surface-1)]"}`}
        >
          <span className="w-4 shrink-0 flex items-center justify-center">
            <svg
              className={`w-3 h-3 text-[var(--color-text-muted)] transition-transform duration-100 ${isExpanded && !isEmpty ? "rotate-90" : ""}`}
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
          </span>
          <StatusIcon status={status} size={14} />
          <span className="text-sm font-medium text-[var(--color-text-primary)]">
            {label}
          </span>
          <span
            className="text-sm text-[var(--color-text-muted)] tabular-nums cursor-pointer hover:text-[var(--color-text-secondary)] transition-colors inline-flex items-center gap-1 h-5"
            onClick={(e) => {
              e.stopPropagation();
              onToggleStoryPoints();
            }}
            title={
              showStoryPoints
                ? "Story points — click for issue count"
                : "Issue count — click for story points"
            }
          >
            <span className="w-3.5 shrink-0 inline-flex items-center justify-center">
              {showStoryPoints ? (
                <svg
                  className="w-3 h-3"
                  viewBox="0 0 12 12"
                  fill="currentColor"
                >
                  <path d="M6 1L11 11H1z" />
                </svg>
              ) : (
                <span className="font-medium">#</span>
              )}
            </span>
            {showStoryPoints ? storyPoints : count}
          </span>
          <button
            onClick={(e) => {
              e.stopPropagation();
              onStartInlineCreate(status);
            }}
            className="ml-auto p-1 rounded-[var(--radius-sm)] text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-hover-surface)] transition-colors"
            title={`New ${label} issue`}
          >
            <svg
              className="w-4 h-4"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
              strokeWidth={2}
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                d="M12 4.5v15m7.5-7.5h-15"
              />
            </svg>
          </button>
        </div>
      )}
    </GroupHeaderDnd>
  );
});
