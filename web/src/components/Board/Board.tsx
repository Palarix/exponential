import { useState, useMemo, useCallback, useEffect, useRef } from 'react';
import { Settings2 } from 'lucide-react';
import { TopBar, IconButton } from '../ui';
import {
  DndContext,
  DragOverlay,
  PointerSensor,
  KeyboardSensor,
  useSensor,
  useSensors,
  type DragStartEvent,
  type DragOverEvent,
  type DragEndEvent,
} from '@dnd-kit/core';
import { sortableKeyboardCoordinates } from '@dnd-kit/sortable';
import { generateKeyBetween } from 'fractional-indexing';
import { addDraft } from '../../api/client';
import type { Issue } from '../../api/client';
import { EmptyState, Popover, StatusPicker, EstimatePicker, ContextMenu, StatusIcon } from '../ui';
import { LabelPicker } from '../ui';
import { sortGroup } from '../../utils/sort';
import { useAllLabels } from '../../hooks/useLabels';
import { toggleLabel } from '../../utils/labels';
import { isTerminal, isCompleted } from '../../constants';
import { boardCollision } from './boardCollision';
import { buildChildrenByParent } from '../../utils/issues';
import { isEditableTarget } from '../../utils/keyboard';
import { useKeyboardHandler } from '../../keyboard';
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
  { id: 'CANCELED', label: 'Canceled', shortcut: '6', isBacklog: false },
  { id: 'DUPLICATE', label: 'Duplicate', shortcut: '7', isBacklog: false },
];

type Containers = Record<string, string[]>;

function buildContainers(issues: Issue[]): Containers {
  const result: Containers = {};
  const byStatus = new Map<string, Issue[]>();
  for (const col of COLUMNS) {
    result[col.id] = [];
    byStatus.set(col.id, []);
  }
  for (const issue of issues) byStatus.get(issue.status)?.push(issue);
  for (const col of COLUMNS) {
    const key = isTerminal(col.id) ? ('updated' as const) : ('manual' as const);
    result[col.id] = sortGroup(byStatus.get(col.id)!, key).map((i) => i.id);
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
    } catch { /* corrupt stored data — use default */ }
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
  const [hiddenColumns, setHiddenColumns] = useState<Set<string>>(() => {
    try {
      const stored = localStorage.getItem("exponential-board-hidden-cols");
      if (stored) return new Set(JSON.parse(stored));
    } catch { /* corrupt stored data — use default */ }
    return new Set(["BACKLOG", "CANCELED", "DUPLICATE"]);
  });
  const toggleColumnVisible = useCallback((colId: string) => {
    setHiddenColumns(prev => {
      const next = new Set(prev);
      if (next.has(colId)) next.delete(colId); else next.add(colId);
      localStorage.setItem("exponential-board-hidden-cols", JSON.stringify([...next]));
      return next;
    });
  }, []);
  const [showViewMenu, setShowViewMenu] = useState(false);
  const viewMenuRef = useRef<HTMLDivElement>(null);
  useEffect(() => {
    if (!showViewMenu) return;
    const handler = (e: MouseEvent) => {
      if (viewMenuRef.current && !viewMenuRef.current.contains(e.target as Node)) setShowViewMenu(false);
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, [showViewMenu]);
  const visibleColumns = useMemo(() => COLUMNS.filter(c => !hiddenColumns.has(c.id)), [hiddenColumns]);
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

  const childrenByParent = useMemo(() => buildChildrenByParent(issues), [issues]);

  const getCardMeta = useCallback(
    (issue: Issue): CardMeta => {
      const parent = issue.parent_id ? issuesById.get(issue.parent_id) : undefined;
      const children = childrenByParent.get(issue.id) || [];
      return {
        parentTitle: parent?.title,
        childDone: children.filter((c) => isCompleted(c.status)).length,
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

  const handleQuickUpdate = useCallback((issueId: string, payload: Record<string, unknown>) => {
    if (patchIssue) patchIssue(issueId, payload as Partial<Issue>);
    setOpenPopover(null);
    addDraft(issueId, "UPDATE", payload).then(() => onRefresh());
  }, [onRefresh, patchIssue]);

  const handleKeyboard = useCallback((e: KeyboardEvent) => {
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
    }, [focusCol, containers, focusedIssueId, issuesById, onIssueClick, collapsedCols, handleQuickUpdate]);

  useKeyboardHandler({
    scope: "board",
    priority: "view",
    handler: handleKeyboard,
    shortcuts: [
      ...[
        ["j", "Next card"], ["ArrowDown", "Next card"], ["k", "Previous card"], ["ArrowUp", "Previous card"],
        ["ArrowRight", "Next column"], ["ArrowLeft", "Previous column"], ["Enter", "Open issue"], [".", "Copy issue ID"],
        ["s", "Set status"], ["l", "Set labels"], ["e", "Set estimate"], ["1", "Move to Backlog"],
        ["2", "Move to Planned"], ["3", "Move to In Progress"], ["4", "Move to Blocked"], ["5", "Move to Done"],
        ["6", "Move to Canceled"], ["7", "Move to Duplicate"],
      ].map(([key, label]) => ({
        id: `board.${key}`, key, label,
        group: "Board",
        preventDefault: false,
      })),
    ],
  });

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
      <TopBar
        left={<span className="text-sm font-medium text-[var(--color-text-primary)]">Board</span>}
        right={
          <span className="flex items-center gap-2">
            <div className="relative" ref={viewMenuRef}>
              <IconButton
                onClick={() => setShowViewMenu(v => !v)}
                icon={<Settings2 size={14} />}
              />
              {showViewMenu && (
                <div className="absolute right-0 top-full mt-1 z-50 min-w-44 bg-[var(--color-surface-3)] border border-[var(--color-border-default)] rounded-[var(--radius-md)] shadow-[var(--shadow-popover)] py-1">
                  <div className="px-3 py-1.5 text-xs text-[var(--color-text-muted)] font-medium uppercase tracking-wider">
                    Columns
                  </div>
                  {COLUMNS.map((col) => (
                    <button
                      key={col.id}
                      onClick={() => toggleColumnVisible(col.id)}
                      className="flex items-center gap-2 w-full h-7 px-3 text-sm text-[var(--color-text-secondary)] hover:bg-[var(--color-hover-surface)] transition-colors"
                    >
                      <StatusIcon status={col.id} size={14} />
                      {col.label}
                      {!hiddenColumns.has(col.id) && (
                        <svg
                          className="w-3 h-3 ml-auto text-[var(--color-accent-primary)]"
                          fill="none"
                          viewBox="0 0 24 24"
                          stroke="currentColor"
                          strokeWidth={2.5}
                        >
                          <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
                        </svg>
                      )}
                    </button>
                  ))}
                </div>
              )}
            </div>
            <span className="text-xs text-[var(--color-text-muted)] tabular-nums">
              {(() => { const n = issues.filter(i => !hiddenColumns.has(i.status)).length; return `${n} issue${n !== 1 ? 's' : ''}`; })()}
            </span>
          </span>
        }
      />

      <DndContext
        sensors={sensors}
        collisionDetection={boardCollision}
        onDragStart={handleDragStart}
        onDragOver={handleDragOver}
        onDragEnd={handleDragEnd}
        onDragCancel={handleDragCancel}
      >
        <div className="flex-1 overflow-hidden p-3">
          <div className="flex gap-3 h-full">
            {visibleColumns.map((column) => (
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
