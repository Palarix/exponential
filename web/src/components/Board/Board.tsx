import { useState, useMemo, useCallback, useEffect, useRef } from 'react';
import {
  DndContext,
  DragOverlay,
  PointerSensor,
  KeyboardSensor,
  useSensor,
  useSensors,
  useDroppable,
  closestCorners,
  type DragStartEvent,
  type DragOverEvent,
  type DragEndEvent,
} from '@dnd-kit/core';
import {
  SortableContext,
  useSortable,
  verticalListSortingStrategy,
  sortableKeyboardCoordinates,
} from '@dnd-kit/sortable';
import { CSS } from '@dnd-kit/utilities';
import { generateKeyBetween } from 'fractional-indexing';
import { addDraft } from '../../api/client';
import type { Issue } from '../../api/client';
import { Avatar, LabelBadge, StatusIcon } from '../ui';
import { sortGroup, getEffectiveKeys } from '../../utils/sort';

interface BoardProps {
  issues: Issue[];
  onRefresh: () => void;
  onIssueClick?: (issue: Issue) => void;
}

const COLUMNS = [
  { id: 'PLANNED', label: 'Planned' },
  { id: 'DOING', label: 'In Progress' },
  { id: 'BLOCKED', label: 'Blocked' },
  { id: 'DONE', label: 'Done' },
];

const DONE_VISIBLE_COUNT = 5;

type Containers = Record<string, string[]>;

function buildContainers(issues: Issue[]): Containers {
  const sorted = sortGroup(
    issues.filter((i) => i.status !== 'BACKLOG'),
    'manual',
  );
  const result: Containers = {};
  for (const col of COLUMNS) result[col.id] = [];
  for (const issue of sorted) {
    result[issue.status]?.push(issue.id);
  }
  return result;
}

function findContainer(id: string, state: Containers): string | null {
  if (id.startsWith('column-')) {
    const status = id.slice('column-'.length);
    return state[status] !== undefined ? status : null;
  }
  for (const [k, v] of Object.entries(state)) {
    if (v.includes(id)) return k;
  }
  return null;
}

export default function Board({ issues, onRefresh, onIssueClick }: BoardProps) {
  // Mirror state in a ref so handleDragEnd can read the latest optimistic
  // arrangement synchronously — React state updates from handleDragOver are
  // not guaranteed to be flushed by the time the drop fires, and a closure
  // snapshot would lag behind.
  const containersRef = useRef<Containers>(buildContainers(issues));
  const [containers, setContainersState] = useState<Containers>(
    containersRef.current,
  );
  const setContainers = useCallback(
    (updater: Containers | ((prev: Containers) => Containers)) => {
      const next =
        typeof updater === 'function'
          ? (updater as (prev: Containers) => Containers)(containersRef.current)
          : updater;
      containersRef.current = next;
      setContainersState(next);
    },
    [],
  );
  const [activeId, setActiveId] = useState<string | null>(null);
  const isDraggingRef = useRef(false);
  const pointerYRef = useRef<number>(0);

  useEffect(() => {
    if (!activeId) return;
    const handler = (e: PointerEvent) => {
      pointerYRef.current = e.clientY;
    };
    window.addEventListener('pointermove', handler);
    return () => window.removeEventListener('pointermove', handler);
  }, [activeId]);

  // Sync from server data when not actively dragging. During a drag, the
  // local containers state holds the optimistic arrangement.
  useEffect(() => {
    if (isDraggingRef.current) return;
    setContainers(buildContainers(issues));
  }, [issues, setContainers]);

  const issuesById = useMemo(
    () => new Map(issues.map((i) => [i.id, i])),
    [issues],
  );

  const childrenByParent = useMemo(() => {
    const map = new Map<string, Issue[]>();
    for (const issue of issues) {
      if (issue.parent_id) {
        const siblings = map.get(issue.parent_id) || [];
        siblings.push(issue);
        map.set(issue.parent_id, siblings);
      }
    }
    return map;
  }, [issues]);

  const getCardMeta = useCallback(
    (issue: Issue) => {
      const parent = issue.parent_id ? issuesById.get(issue.parent_id) : undefined;
      const children = childrenByParent.get(issue.id) || [];
      return {
        parentTitle: parent?.title,
        childDone: children.filter((c) => c.status === 'DONE').length,
        childTotal: children.length,
      };
    },
    [issuesById, childrenByParent],
  );

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 3 } }),
    useSensor(KeyboardSensor, {
      coordinateGetter: sortableKeyboardCoordinates,
    }),
  );

  const handleDragStart = useCallback((event: DragStartEvent) => {
    isDraggingRef.current = true;
    setActiveId(String(event.active.id));
    const orig = event.activatorEvent as PointerEvent;
    if (typeof orig.clientY === 'number') {
      pointerYRef.current = orig.clientY;
    }
  }, []);

  const handleDragOver = useCallback((event: DragOverEvent) => {
    const draggedId = String(event.active.id);
    const overId = event.over ? String(event.over.id) : null;
    if (!overId || overId === draggedId) return;

    setContainers((prev) => {
      const activeContainer = findContainer(draggedId, prev);
      let overContainer: string | null;
      if (overId.startsWith('column-')) {
        overContainer = overId.slice('column-'.length);
      } else {
        overContainer = findContainer(overId, prev);
      }
      if (
        !activeContainer ||
        !overContainer ||
        activeContainer === overContainer
      ) {
        return prev;
      }

      // Cross-container move: pull active out of its current container and
      // splice it into the target container at the right index.
      const activeItems = prev[activeContainer].filter((id) => id !== draggedId);
      const overItems = [...prev[overContainer]];
      let insertIdx: number;
      if (overId.startsWith('column-')) {
        // Dropped onto the column body itself → end of column.
        insertIdx = overItems.length;
      } else {
        const overIdx = overItems.indexOf(overId);
        if (overIdx === -1) {
          insertIdx = overItems.length;
        } else {
          // Pattern from dnd-kit's multi-container example: use the active's
          // translated bottom vs over's bottom to decide above/below. The
          // SortableContext will then keep re-resolving as the user drags
          // further, so a slight initial offset self-corrects.
          const activeTranslated = event.active.rect.current.translated;
          const overRect = event.over?.rect;
          let isBelow = false;
          if (activeTranslated && overRect) {
            isBelow =
              activeTranslated.top + activeTranslated.height >
              overRect.top + overRect.height / 2;
          }
          insertIdx = isBelow ? overIdx + 1 : overIdx;
        }
      }
      overItems.splice(insertIdx, 0, draggedId);
      return {
        ...prev,
        [activeContainer]: activeItems,
        [overContainer]: overItems,
      };
    });
  }, [setContainers]);

  const handleDragEnd = useCallback(
    async (event: DragEndEvent) => {
      const draggedId = String(event.active.id);
      const overId = event.over ? String(event.over.id) : null;
      setActiveId(null);

      const dragged = issuesById.get(draggedId);
      if (!dragged || !overId) {
        isDraggingRef.current = false;
        setContainers(buildContainers(issues));
        return;
      }

      // Determine which column the item landed in.
      // For cross-column drags handleDragOver already moved the item in
      // containersRef, so look up the container there.
      let targetStatus: string;
      if (overId.startsWith('column-')) {
        targetStatus = overId.slice('column-'.length);
      } else {
        const overContainer = findContainer(overId, containersRef.current);
        if (!overContainer) {
          isDraggingRef.current = false;
          setContainers(buildContainers(issues));
          return;
        }
        targetStatus = overContainer;
      }

      // Compute effective keys from ALL non-BACKLOG issues — this matches
      // the key space used by buildContainers → sortGroup on render.
      // Using only the column's issues would generate a different virtual-key
      // space and the saved sort_order would land in the wrong slot.
      const allNonBacklog = issues.filter((i) => i.status !== 'BACKLOG');
      const effectiveKeys = getEffectiveKeys(allNonBacklog);

      // Sort column neighbours by those global keys, excluding the dragged item.
      const columnIssues = allNonBacklog
        .filter((i) => i.status === targetStatus && i.id !== draggedId)
        .sort((a, b) => {
          const ka = effectiveKeys.get(a.id) || '';
          const kb = effectiveKeys.get(b.id) || '';
          return ka < kb ? -1 : ka > kb ? 1 : 0;
        });

      let insertIdx: number;
      if (overId.startsWith('column-') || overId === draggedId) {
        insertIdx = columnIssues.length;
      } else {
        const targetIdx = columnIssues.findIndex((i) => i.id === overId);
        if (targetIdx === -1) {
          insertIdx = columnIssues.length;
        } else if (dragged.status === targetStatus) {
          // Within-column: derive direction from original sort position.
          // SortableContext swaps items when the active passes them, so
          // overId is the last item the active displaced.  Comparing keys
          // tells us the drag direction without relying on pointer rects.
          const draggedKey = effectiveKeys.get(draggedId) || '';
          const overKey = effectiveKeys.get(overId) || '';
          insertIdx = draggedKey < overKey ? targetIdx + 1 : targetIdx;
        } else {
          // Cross-column: use actual pointer Y vs over item center.
          const overRect = event.over?.rect;
          let isBelow = false;
          if (overRect) {
            isBelow =
              pointerYRef.current > overRect.top + overRect.height / 2;
          }
          insertIdx = isBelow ? targetIdx + 1 : targetIdx;
        }
      }

      const prev = columnIssues[insertIdx - 1];
      const next = columnIssues[insertIdx];
      const prevKey = prev ? effectiveKeys.get(prev.id) ?? null : null;
      const nextKey = next ? effectiveKeys.get(next.id) ?? null : null;
      const newKey = generateKeyBetween(prevKey, nextKey);

      const update: Record<string, unknown> = { sort_order: newKey };
      if (dragged.status !== targetStatus) update.status = targetStatus;
      try {
        await addDraft(draggedId, 'UPDATE', update);
      } finally {
        isDraggingRef.current = false;
        onRefresh();
      }
    },
    [issues, issuesById, onRefresh, setContainers],
  );

  const handleDragCancel = useCallback(() => {
    setActiveId(null);
    isDraggingRef.current = false;
    setContainers(buildContainers(issues));
  }, [issues, setContainers]);

  const activeIssue = activeId ? issuesById.get(activeId) ?? null : null;

  return (
    <div className="h-full flex flex-col">
      <div className="flex items-center gap-3 px-5 h-11 border-b border-[var(--color-border-subtle)] shrink-0">
        <span className="text-sm font-medium text-[var(--color-text-primary)]">Board</span>
        <span className="text-xs text-[var(--color-text-muted)] tabular-nums">
          {issues.filter((i) => i.status !== 'BACKLOG').length} issue
          {issues.filter((i) => i.status !== 'BACKLOG').length !== 1 ? 's' : ''}
        </span>
      </div>

      <DndContext
        sensors={sensors}
        collisionDetection={closestCorners}
        onDragStart={handleDragStart}
        onDragOver={handleDragOver}
        onDragEnd={handleDragEnd}
        onDragCancel={handleDragCancel}
      >
        <div className="flex-1 overflow-hidden p-3">
          <div className="flex gap-2.5 h-full">
            {COLUMNS.map((column) => (
              <BoardColumn
                key={column.id}
                column={column}
                itemIds={containers[column.id] ?? []}
                getIssue={(id) => issuesById.get(id)}
                getCardMeta={getCardMeta}
                activeId={activeId}
                onIssueClick={onIssueClick}
              />
            ))}
          </div>
        </div>
        <DragOverlay dropAnimation={null}>
          {activeIssue ? <BoardCard issue={activeIssue} meta={getCardMeta(activeIssue)} isOverlay /> : null}
        </DragOverlay>
      </DndContext>
    </div>
  );
}

interface CardMeta {
  parentTitle?: string;
  childDone: number;
  childTotal: number;
}

function BoardColumn({
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
  const isDone = column.id === 'DONE';
  const shouldTruncate = isDone && itemIds.length > DONE_VISIBLE_COUNT && !showAll;
  const visibleIds = shouldTruncate ? itemIds.slice(0, DONE_VISIBLE_COUNT) : itemIds;
  const hiddenCount = itemIds.length - DONE_VISIBLE_COUNT;

  const containsActive = activeId !== null && itemIds.includes(activeId);
  const showHighlight = isOver || containsActive;

  return (
    <div
      className={`flex flex-col flex-1 min-w-0 rounded-[var(--radius-md)] bg-[var(--color-bg-secondary)]/40 transition-colors duration-[var(--duration-fast)] ${showHighlight ? 'ring-2 ring-inset ring-[var(--color-accent-primary)] bg-[var(--color-accent-primary)]/5' : ''}`}
    >
      <div className="flex items-center justify-between px-3 py-2 border-b border-[var(--color-border-subtle)]">
        <div className="flex items-center gap-2">
          <StatusIcon status={column.id} size={14} />
          <span className="text-sm font-medium text-[var(--color-text-primary)]">{column.label}</span>
        </div>
        <span className="text-xs text-[var(--color-text-muted)] tabular-nums">{itemIds.length}</span>
      </div>

      <SortableContext items={visibleIds} strategy={verticalListSortingStrategy}>
        <div ref={setNodeRef} className="flex-1 p-1.5 space-y-2 overflow-y-auto">
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

function SortableBoardCard({
  issue,
  meta,
  onClick,
}: {
  issue: Issue;
  meta: CardMeta;
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
      {...attributes}
      {...listeners}
      onClick={onClick}
      className="px-2.5 py-2 rounded-[var(--radius-sm)] bg-[var(--color-bg-elevated)] border border-[var(--color-border-subtle)] hover:border-[var(--color-border-default)] hover:bg-[var(--color-bg-hover)] cursor-pointer transition-colors duration-[var(--duration-fast)]"
    >
      <BoardCardContent issue={issue} meta={meta} />
    </div>
  );
}

function BoardCard({
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
      className={`px-2.5 py-2 rounded-[var(--radius-sm)] bg-[var(--color-bg-elevated)] border border-[var(--color-border-default)] ${isOverlay ? 'shadow-lg cursor-grabbing' : ''}`}
    >
      <BoardCardContent issue={issue} meta={meta} />
    </div>
  );
}

function BoardCardContent({ issue, meta }: { issue: Issue; meta: CardMeta }) {
  const hasChildren = meta.childTotal > 0;
  return (
    <>
      {/* Top row: ID + parent breadcrumb | pending + avatar */}
      <div className="flex items-center gap-1.5 mb-1 min-w-0">
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
          <span className="w-1.5 h-1.5 rounded-full bg-[var(--color-warning)] shrink-0" />
        )}
        {issue.assignee && <Avatar name={issue.assignee} size="xs" />}
      </div>

      {/* Title */}
      <p className="text-sm text-[var(--color-text-primary)] leading-snug line-clamp-2 mb-1.5">
        {issue.title}
      </p>

      {/* Middle: sub-progress + labels */}
      {(hasChildren || (issue.labels && issue.labels.length > 0)) && (
        <div className="flex items-center gap-2 text-xs flex-wrap mb-1.5">
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

      {/* Bottom row: priority + date | estimate */}
      <div className="flex items-center gap-2 text-xs text-[var(--color-text-muted)]">
        {issue.priority > 0 && issue.priority <= 3 && (
          <span className={`font-medium shrink-0 ${issue.priority === 1 ? 'text-[var(--color-error)]' : issue.priority === 2 ? 'text-[var(--color-warning)]' : 'text-[var(--color-text-muted)]'}`}>
            {issue.priority === 1 ? '!!!' : issue.priority === 2 ? '!!' : '!'}
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

function SubProgress({ done, total }: { done: number; total: number }) {
  const pct = total > 0 ? (done / total) * 100 : 0;
  return (
    <svg width="14" height="14" viewBox="0 0 16 16" className="shrink-0">
      <circle cx="8" cy="8" r="6" fill="none" stroke="var(--color-bg-tertiary)" strokeWidth="2" />
      <circle
        cx="8" cy="8" r="6" fill="none"
        stroke={pct === 100 ? 'var(--color-success)' : 'var(--color-accent-primary)'}
        strokeWidth="2" strokeLinecap="round"
        strokeDasharray={`${pct * 0.377} 100`}
        transform="rotate(-90 8 8)"
      />
    </svg>
  );
}

function formatShortDate(dateStr: string): string {
  const d = new Date(dateStr);
  const months = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];
  return `${months[d.getMonth()]} ${d.getDate()}`;
}
