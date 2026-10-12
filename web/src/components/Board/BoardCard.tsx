import { useSortable } from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import { useContext } from "react";
import type { Issue } from "../../api/client";
import { Avatar, BranchBadge, EstimateBadge, Heading, LabelBadge, DefaultLabelsContext, PriorityIcon, SubProgress, Text } from "../ui";
import { RefreshCw, Paperclip } from "lucide-react";
import { formatShortDate } from "../../utils/format";
import { splitLabels } from "../../utils/labels";
import { popoverTriggerProps } from "../../utils/popover-trigger";

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
  assigneeOpen = false,
  onAssigneeOpen,
}: {
  issue: Issue;
  meta: CardMeta;
  isFocused?: boolean;
  onClick?: () => void;
  onContextMenu?: (x: number, y: number) => void;
  assigneeOpen?: boolean;
  onAssigneeOpen?: () => void;
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
      <BoardCardContent
        issue={issue}
        meta={meta}
        assignee={onAssigneeOpen ? { open: assigneeOpen, onOpen: onAssigneeOpen } : undefined}
      />
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

function AssigneeAvatar({ assignee }: { assignee?: string }) {
  return assignee ? (
    <Avatar name={assignee} size="sm" />
  ) : (
    <span className="flex opacity-50">
      <Avatar size="sm" />
    </span>
  );
}

function BoardCardContent({
  issue,
  meta,
  assignee,
}: {
  issue: Issue;
  meta: CardMeta;
  /** Makes the avatar a picker trigger; omitted on the drag overlay. */
  assignee?: { open: boolean; onOpen: () => void };
}) {
  const defaultLabels = useContext(DefaultLabelsContext);
  const { primary: primaryLabels, metadata: metaLabels } = splitLabels(issue.labels || [], defaultLabels);
  const hasChildren = meta.childTotal > 0;
  const hasLabels = issue.labels && issue.labels.length > 0;
  const hasBranch = !!issue.branch_stats;
  const cycleId = issue.effective_cycle_id || issue.cycle_id;
  const artifactCount = issue.artifacts?.length ?? 0;
  const hasBottom = hasChildren || hasLabels || hasBranch || cycleId || artifactCount > 0;
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
            <Text size="xs" truncate>
              {meta.parentTitle}
            </Text>
          </>
        )}
        <div className="flex-1" />
        {issue.is_pending && (
          <span className="w-2 h-2 rounded-full bg-[var(--color-warning)] shrink-0" />
        )}
        <EstimateBadge value={issue.estimate} />
        {issue.priority > 0 && <PriorityIcon priority={issue.priority} size={14} />}
        {assignee ? (
          <button
            data-assignee-trigger
            onClick={(e) => {
              e.stopPropagation();
              assignee.onOpen();
            }}
            aria-label={`${issue.assignee ? issue.assignee.split(" <")[0] : "No assignee"}, change assignee`}
            {...popoverTriggerProps("assignee", assignee.open ? "assignee" : null)}
            className="w-6 h-6 -m-1 flex items-center justify-center rounded cursor-pointer hover:bg-white/10 transition-colors shrink-0"
          >
            <AssigneeAvatar assignee={issue.assignee} />
          </button>
        ) : (
          <AssigneeAvatar assignee={issue.assignee} />
        )}
      </div>

      {/* Title */}
      <Heading as="p" leading="snug" clamp={2} title={issue.title} />

      {/* SW: labels, sub-progress | SE: cycle, date */}
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
            {artifactCount > 0 && (
              <span className="inline-flex items-center gap-1 h-6 px-2 rounded-md border border-[var(--color-border-label)] text-xs text-[var(--color-text-secondary)] tabular-nums">
                <Paperclip size={12} strokeWidth={1.5} />
                {artifactCount}
              </span>
            )}
            {(() => {
              const all = [...metaLabels, ...primaryLabels];
              const extra = Math.max(0, all.length - 2);
              return (
                <>
                  {all.slice(0, 2).map((label) => (
                    <LabelBadge key={label} label={label} />
                  ))}
                  {extra > 0 && (
                    <span className="text-xs text-[var(--color-text-muted)] shrink-0">+{extra}</span>
                  )}
                </>
              );
            })()}
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
