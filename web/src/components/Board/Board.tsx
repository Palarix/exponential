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
import { EmptyState, Popover, StatusPicker, EstimatePicker, ContextMenu } from '../ui';
import { LabelPicker } from '../ui';
import { sortGroup } from '../../utils/sort';
import { useAllLabels } from '../../hooks/useLabels';
import { toggleLabel } from '../../utils/labels';
import { isEditableTarget } from '../../utils/keyboard';
import BoardColumn from './BoardColumn';
import { BoardCard, type CardMeta } from './BoardCard';

interface BoardProps {
  issues: Issue[];
  onRefresh: () => void;
  onIssueClick?: (issue: Issue) => void;
  onNewIssue?: () => void;
  contributors?: string[];
  onConfigLabelsChange?: (labels: Record<string, string>) => void;
  patchIssue?: (issueId: string, patch: Partial<Issue>) => void;
}

const COLUMNS = [
  { id: 'BACKLOG', label: 'Backlog', shortcut: '1', isBacklog: true },
  { id: 'PLANNED', label: 'Planned', shortcut: '2', isBacklog: false },
  { id: 'DOING', label: 'In Progress', shortcut: '3', isBacklog: false },
  { id: 'BLOCKED', label: 'Blocked', shortcut: '4', isBacklog: false },
  { id: 'DONE', label: 'Done', shortcut: '5', isBacklog: false },
];

type Containers = Record<string, string[]>;

function buildContainers(issues: Issue[]): Containers {
  const sorted = sortGroup(issues, 'manual');
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

export default function Board({ issues, onRefresh, onIssueClick, onNewIssue, contributors = [], onConfigLabelsChange, patchIssue }: BoardProps) {
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
  const [collapsedCols, setCollapsedCols] = useState<Set<string>>(() => {
    try {
      const stored = localStorage.getItem("exponential-board-collapsed");
      if (stored) return new Set(JSON.parse(stored));
    } catch {}
    return new Set(["BACKLOG"]);
  });
  const toggleCollapse = useCallback((colId: string) => {
    setCollapsedCols(prev => {
      const next = new Set(prev);
      if (next.has(colId)) next.delete(colId); else next.add(colId);
      localStorage.setItem("exponential-board-collapsed", JSON.stringify([...next]));
      return next;
    });
  }, []);
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

  // Keyboard navigation
  const [focusCol, setFocusCol] = useState(0);
  const [focusCard, setFocusCard] = useState(0);
  const [keyboardNav, setKeyboardNav] = useState(false);
  const [openPopover, setOpenPopover] = useState<{ issueId: string; type: "status" | "labels" | "estimate" } | null>(null);
  const [contextMenu, setContextMenu] = useState<{ issueId: string; x: number; y: number } | null>(null);

  const allKnownLabels = useAllLabels(issues);
  const openPopoverRef = useRef(openPopover);
  openPopoverRef.current = openPopover;

  const focusedIssueId = useMemo(() => {
    if (!keyboardNav) return null;
    const colId = COLUMNS[focusCol]?.id;
    const ids = containers[colId] ?? [];
    return ids[focusCard] ?? null;
  }, [keyboardNav, focusCol, focusCard, containers]);

  useEffect(() => {
    if (!keyboardNav) return;
    const el = document.querySelector(`[data-board-card="${focusedIssueId}"]`);
    el?.scrollIntoView({ block: "nearest" });
  }, [focusedIssueId, keyboardNav]);

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (openPopoverRef.current) return;
      if (isEditableTarget(e)) return;
      if (e.metaKey || e.ctrlKey) return;
      if (isDraggingRef.current) return;

      const colId = COLUMNS[focusCol]?.id;
      const colIds = containers[colId] ?? [];

      if (e.key === "ArrowDown" || e.key === "j") {
        e.preventDefault();
        setKeyboardNav(true);
        setFocusCard(i => Math.min(i + 1, colIds.length - 1));
        return;
      }
      if (e.key === "ArrowUp" || e.key === "k") {
        e.preventDefault();
        setKeyboardNav(true);
        setFocusCard(i => Math.max(i - 1, 0));
        return;
      }
      if (e.key === "ArrowRight") {
        e.preventDefault();
        setKeyboardNav(true);
        setFocusCol(i => {
          for (let n = i + 1; n < COLUMNS.length; n++) {
            const ids = containers[COLUMNS[n].id] ?? [];
            if (ids.length > 0 && !collapsedCols.has(COLUMNS[n].id)) {
              setFocusCard(c => Math.min(c, ids.length - 1));
              return n;
            }
          }
          return i;
        });
        return;
      }
      if (e.key === "ArrowLeft") {
        e.preventDefault();
        setKeyboardNav(true);
        setFocusCol(i => {
          for (let n = i - 1; n >= 0; n--) {
            const ids = containers[COLUMNS[n].id] ?? [];
            if (ids.length > 0 && !collapsedCols.has(COLUMNS[n].id)) {
              setFocusCard(c => Math.min(c, ids.length - 1));
              return n;
            }
          }
          return i;
        });
        return;
      }
      if (e.key === "Enter" && focusedIssueId) {
        e.preventDefault();
        const issue = issuesById.get(focusedIssueId);
        if (issue) onIssueClick?.(issue);
        return;
      }
      if (e.key === "." && focusedIssueId) {
        navigator.clipboard.writeText(focusedIssueId);
        return;
      }
      if (focusedIssueId) {
        if (e.key === "s") {
          e.preventDefault();
          setOpenPopover({ issueId: focusedIssueId, type: "status" });
          return;
        }
        if (e.key === "l") {
          e.preventDefault();
          setOpenPopover({ issueId: focusedIssueId, type: "labels" });
          return;
        }
        if (e.key === "e") {
          e.preventDefault();
          setOpenPopover({ issueId: focusedIssueId, type: "estimate" });
          return;
        }
        const col = COLUMNS.find(c => c.shortcut === e.key);
        if (col) {
          const issue = issuesById.get(focusedIssueId);
          if (issue && issue.status !== col.id) {
            e.preventDefault();
            handleQuickUpdate(focusedIssueId, { status: col.id });
          }
          return;
        }
      }
    };
    document.addEventListener("keydown", handler);
    return () => document.removeEventListener("keydown", handler);
  }, [focusCol, containers, focusedIssueId, issuesById, onIssueClick, collapsedCols]);

  const handleQuickUpdate = useCallback((issueId: string, payload: Record<string, unknown>) => {
    if (patchIssue) patchIssue(issueId, payload as Partial<Issue>);
    setOpenPopover(null);
    addDraft(issueId, "UPDATE", payload).then(() => onRefresh());
  }, [onRefresh, patchIssue]);

  if (issues.length === 0) {
    return (
      <EmptyState
        title="Your board is empty"
        description="Issues in Planned, In Progress, Blocked, and Done statuses will appear here as cards you can drag between columns."
        icon={
          <svg width="180" height="120" viewBox="0 0 180 120" fill="none">
            <rect x="10" y="20" width="35" height="80" rx="6" stroke="var(--color-text-muted)" strokeWidth="1.5" strokeDasharray="4 3" />
            <rect x="52" y="20" width="35" height="80" rx="6" stroke="var(--color-text-muted)" strokeWidth="1.5" strokeDasharray="4 3" />
            <rect x="94" y="20" width="35" height="80" rx="6" stroke="var(--color-text-muted)" strokeWidth="1.5" strokeDasharray="4 3" />
            <rect x="136" y="20" width="35" height="80" rx="6" stroke="var(--color-text-muted)" strokeWidth="1.5" strokeDasharray="4 3" />
            <rect x="16" y="10" width="23" height="4" rx="2" fill="var(--color-text-muted)" opacity="0.5" />
            <rect x="58" y="10" width="23" height="4" rx="2" fill="var(--color-text-muted)" opacity="0.5" />
            <rect x="100" y="10" width="23" height="4" rx="2" fill="var(--color-text-muted)" opacity="0.5" />
            <rect x="142" y="10" width="23" height="4" rx="2" fill="var(--color-text-muted)" opacity="0.5" />
          </svg>
        }
        actionLabel="Create an issue"
        onAction={onNewIssue}
      />
    );
  }

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
          <div className="flex gap-3 h-full">
            {COLUMNS.map((column) => (
              <BoardColumn
                key={column.id}
                column={column}
                itemIds={containers[column.id] ?? []}
                getIssue={(id) => issuesById.get(id)}
                getCardMeta={getCardMeta}
                activeId={activeId}
                focusedId={focusedIssueId}
                collapsed={collapsedCols.has(column.id)}
                onToggleCollapse={() => toggleCollapse(column.id)}
                onIssueClick={onIssueClick}
                onIssueContextMenu={(issue, x, y) => setContextMenu({ issueId: issue.id, x, y })}
              />
            ))}
          </div>
        </div>
        <DragOverlay dropAnimation={null}>
          {activeIssue ? <BoardCard issue={activeIssue} meta={getCardMeta(activeIssue)} isOverlay /> : null}
        </DragOverlay>
      </DndContext>

      {contextMenu && (() => {
        const ctxIssue = issuesById.get(contextMenu.issueId);
        if (!ctxIssue) return null;
        return (
          <ContextMenu
            issue={ctxIssue}
            issues={issues}
            x={contextMenu.x}
            y={contextMenu.y}
            onClose={() => setContextMenu(null)}
            onRefresh={onRefresh}
            allLabels={allKnownLabels}
            contributors={contributors}
            onConfigLabelsChange={onConfigLabelsChange}
            patchIssue={patchIssue}
          />
        );
      })()}

      {openPopover && (() => {
        const issue = issuesById.get(openPopover.issueId);
        if (!issue) return null;
        return (
          <>
            {openPopover.type === "status" && (
              <Popover onClose={() => setOpenPopover(null)}>
                <StatusPicker current={issue.status} onSelect={v => handleQuickUpdate(issue.id, { status: v })} onClose={() => setOpenPopover(null)} />
              </Popover>
            )}
            {openPopover.type === "estimate" && (
              <Popover onClose={() => setOpenPopover(null)}>
                <EstimatePicker current={issue.estimate || 0} onSelect={v => handleQuickUpdate(issue.id, { estimate: v })} onClose={() => setOpenPopover(null)} />
              </Popover>
            )}
            {openPopover.type === "labels" && (
              <Popover onClose={() => setOpenPopover(null)}>
                <LabelPicker
                  allLabels={allKnownLabels}
                  selected={issue.labels || []}
                  onToggle={async (label) => {
                    const labels = toggleLabel(issue.labels || [], label);
                    await addDraft(issue.id, "UPDATE", { labels });
                    onRefresh();
                  }}
                  onClose={() => setOpenPopover(null)}
                />
              </Popover>
            )}
          </>
        );
      })()}
    </div>
  );
}
