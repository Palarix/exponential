import { useDraggable, useDroppable, closestCenter, type CollisionDetection } from "@dnd-kit/core";
import { CSS } from "@dnd-kit/utilities";
import type { Issue } from "../../api/client";
import { CopyableId, LabelBadge, StatusIcon } from "../ui";

export const backlogCollision: CollisionDetection = (args) => {
  const all = closestCenter(args);
  const rows = all.filter((c) => !String(c.id).startsWith("group-"));
  return rows.length > 0 ? rows : all;
};

export function GroupHeaderDnd({
  status,
  enabled,
  children,
}: {
  status: string;
  enabled: boolean;
  children: (setNodeRef: (el: HTMLElement | null) => void) => React.ReactElement;
}) {
  const { setNodeRef } = useDroppable({ id: `group-${status}`, disabled: !enabled });
  return children(setNodeRef);
}

export function IssueRowDnd({
  id,
  canDrag,
  enabled,
  children,
}: {
  id: string;
  canDrag: boolean;
  enabled: boolean;
  children: (
    setNodeRef: (el: HTMLElement | null) => void,
    dragProps: {
      attributes: React.HTMLAttributes<HTMLElement>;
      listeners: React.HTMLAttributes<HTMLElement>;
    },
  ) => React.ReactElement;
}) {
  const draggable = useDraggable({ id, disabled: !canDrag });
  const droppable = useDroppable({ id, disabled: !enabled });
  const setNodeRef = (el: HTMLElement | null) => {
    draggable.setNodeRef(el);
    droppable.setNodeRef(el);
  };
  return children(setNodeRef, {
    attributes: draggable.attributes as React.HTMLAttributes<HTMLElement>,
    listeners: (draggable.listeners ?? {}) as React.HTMLAttributes<HTMLElement>,
  });
}

export function DragOverlayCard({ issue, batchCount }: { issue: Issue; batchCount: number }) {
  return (
    <div
      style={{ transform: CSS.Translate.toString({ x: 0, y: 0, scaleX: 1, scaleY: 1 }) }}
      className="flex items-center gap-3 px-5 h-[38px] border border-[var(--color-border-default)] bg-[var(--color-surface-elevated)] rounded-[var(--radius-sm)] shadow-lg pointer-events-none"
    >
      <CopyableId id={issue.id} className="text-xs w-[110px] shrink-0 truncate tabular-nums" />
      <StatusIcon status={issue.status} size={14} />
      <span className="text-sm text-[var(--color-text-primary)] truncate">{issue.title}</span>
      {batchCount > 1 && (
        <span className="flex items-center justify-center w-5 h-5 rounded-full bg-[var(--color-accent-primary)] text-white text-xs font-medium shrink-0">
          {batchCount}
        </span>
      )}
      {issue.labels?.map((label) => <LabelBadge key={label} label={label} />)}
    </div>
  );
}
