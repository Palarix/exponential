import { useState, useMemo, useCallback, useEffect, useRef } from 'react';
import {
  DndContext,
  DragOverlay,
  PointerSensor,
  KeyboardSensor,
  useSensor,
  useSensors,
  closestCorners,
  type DragStartEvent,
  type DragOverEvent,
  type DragEndEvent,
} from '@dnd-kit/core';
import { sortableKeyboardCoordinates } from '@dnd-kit/sortable';
import { generateKeyBetween } from 'fractional-indexing';
import { addDraft } from '../../api/client';
import type { Issue } from '../../api/client';
import { sortGroup } from '../../utils/sort';
import BoardColumn from './BoardColumn';
import { BoardCard, type CardMeta } from './BoardCard';

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

type Containers = Record<string, string[]>;

function buildContainers(issues: Issue[]): Containers {
  const sorted = sortGroup(issues.filter((i) => i.status !== 'BACKLOG'), 'manual');
  const result: Containers = {};
  for (const col of COLUMNS) result[col.id] = [];
  for (const issue of sorted) result[issue.status]?.push(issue.id);
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
  const containersRef = useRef<Containers>(buildContainers(issues));
  const [containers, setContainersState] = useState<Containers>(containersRef.current);
  const setContainers = useCallback(
    (updater: Containers | ((prev: Containers) => Containers)) => {
      const next = typeof updater === 'function' ? (updater as (prev: Containers) => Containers)(containersRef.current) : updater;
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
    const handler = (e: PointerEvent) => { pointerYRef.current = e.clientY; };
    window.addEventListener('pointermove', handler);
    return () => window.removeEventListener('pointermove', handler);
  }, [activeId]);

  useEffect(() => {
    if (isDraggingRef.current) return;
    setContainers(buildContainers(issues));
  }, [issues, setContainers]);

  const issuesById = useMemo(() => new Map(issues.map((i) => [i.id, i])), [issues]);

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
    (issue: Issue): CardMeta => {
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
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates }),
  );

  const handleDragStart = useCallback((event: DragStartEvent) => {
    isDraggingRef.current = true;
    setActiveId(String(event.active.id));
    const orig = event.activatorEvent as PointerEvent;
    if (typeof orig.clientY === 'number') pointerYRef.current = orig.clientY;
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
      if (!activeContainer || !overContainer || activeContainer === overContainer) return prev;

      const activeItems = prev[activeContainer].filter((id) => id !== draggedId);
      const overItems = [...prev[overContainer]];
      let insertIdx: number;
      if (overId.startsWith('column-')) {
        insertIdx = overItems.length;
      } else {
        const overIdx = overItems.indexOf(overId);
        if (overIdx === -1) {
          insertIdx = overItems.length;
        } else {
          const activeTranslated = event.active.rect.current.translated;
          const overRect = event.over?.rect;
          let isBelow = false;
          if (activeTranslated && overRect) {
            isBelow = activeTranslated.top + activeTranslated.height > overRect.top + overRect.height / 2;
          }
          insertIdx = isBelow ? overIdx + 1 : overIdx;
        }
      }
      overItems.splice(insertIdx, 0, draggedId);
      return { ...prev, [activeContainer]: activeItems, [overContainer]: overItems };
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

      const columnIssues = issues
        .filter((i) => i.status === targetStatus && i.id !== draggedId)
        .sort((a, b) => {
          const ka = a.sort_order || '';
          const kb = b.sort_order || '';
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
          const draggedKey = dragged.sort_order || '';
          const overIssue = columnIssues[targetIdx];
          const overKey = overIssue?.sort_order || '';
          insertIdx = draggedKey < overKey ? targetIdx + 1 : targetIdx;
        } else {
          const overRect = event.over?.rect;
          let isBelow = false;
          if (overRect) {
            isBelow = pointerYRef.current > overRect.top + overRect.height / 2;
          }
          insertIdx = isBelow ? targetIdx + 1 : targetIdx;
        }
      }

      const prev = columnIssues[insertIdx - 1];
      const next = columnIssues[insertIdx];
      const prevKey = prev?.sort_order || null;
      const nextKey = next?.sort_order || null;
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
