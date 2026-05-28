import { useState, useRef, useEffect, useCallback, useMemo } from "react";
import { generateKeyBetween } from "fractional-indexing";
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
} from "@dnd-kit/core";
import { createIssue, addDraft } from "../../api/client";
import type { Issue } from "../../api/client";
import {
  Avatar,
  LabelBadge,
  StatusIcon,
  CopyableId,
  Popover,
  PopoverHeader,
  LabelPicker,
  SubProgress,
} from "../ui";
import { formatShortDate } from "../../utils/format";
import { getEffectiveKeys, SORT_OPTIONS } from "../../utils/sort";
import type { SortKey } from "../../utils/sort";
import { isEditableTarget } from "../../utils/keyboard";
import { STATUS_OPTIONS, ESTIMATE_OPTIONS } from "../../constants";
import { backlogCollision, GroupHeaderDnd, IssueRowDnd, DragOverlayCard } from "./DndComponents";
import { useBacklogRows, type RowItem } from "./useBacklogRows";

export type Tab = "all" | "active" | "backlog";

const TAB_CONFIGS: Record<Tab, { label: string; statuses: string[] }> = {
  all: { label: "All Issues", statuses: ["BACKLOG", "PLANNED", "DOING", "BLOCKED", "DONE"] },
  active: { label: "Active", statuses: ["PLANNED", "DOING", "BLOCKED"] },
  backlog: { label: "Backlog", statuses: ["BACKLOG"] },
};

interface BacklogProps {
  issues: Issue[];
  onRefresh: () => void;
  onIssueClick?: (issue: Issue) => void;
  searchFocused?: boolean;
  onSearchBlur?: () => void;
  sortKey: SortKey;
  onSortChange: (key: SortKey) => void;
  onNavigationOrderChange?: (ids: string[]) => void;
  activeTab: Tab;
  onTabChange: (tab: Tab) => void;
  onConfigLabelsChange?: (labels: Record<string, string>) => void;
}

export default function Backlog({
  issues,
  onRefresh,
  onIssueClick,
  searchFocused,
  onSearchBlur,
  sortKey,
  onSortChange,
  onNavigationOrderChange,
  activeTab,
  onTabChange,
  onConfigLabelsChange,
}: BacklogProps) {
  const [search, setSearch] = useState("");
  const [expandedGroups, setExpandedGroups] = useState<Set<string>>(() => new Set());
  const [expandedNodes, setExpandedNodes] = useState<Set<string>>(() => new Set());
  const [focusedIndex, setFocusedIndex] = useState(-1);
  const [keyboardNav, setKeyboardNav] = useState(false);
  const [inlineCreateStatus, setInlineCreateStatus] = useState<string | null>(null);
  const [inlineTitle, setInlineTitle] = useState("");
  const [toast, setToast] = useState<string | null>(null);
  const [openPopover, setOpenPopover] = useState<{ issueId: string; type: "status" | "estimate" | "labels" } | null>(null);
  const searchRef = useRef<HTMLInputElement>(null);
  const listRef = useRef<HTMLDivElement>(null);
  const inlineRef = useRef<HTMLInputElement>(null);

  const handleInlineCreate = useCallback(
    async (status: string, title: string) => {
      if (!title.trim()) return;
      const issueId = await createIssue({ title: title.trim(), labels: ["feature"] });
      if (status !== "BACKLOG") await addDraft(issueId, "UPDATE", { status });
      setInlineTitle("");
      onRefresh();
    },
    [onRefresh],
  );

  const handleQuickStatus = useCallback(async (issueId: string, status: string) => { await addDraft(issueId, "UPDATE", { status }); setOpenPopover(null); onRefresh(); }, [onRefresh]);
  const handleQuickEstimate = useCallback(async (issueId: string, estimate: number) => { await addDraft(issueId, "UPDATE", { estimate }); setOpenPopover(null); onRefresh(); }, [onRefresh]);
  const handleQuickLabelToggle = useCallback(async (issue: Issue, label: string) => {
    const current = issue.labels || [];
    const labels = current.includes(label) ? current.filter((l) => l !== label) : [...current, label];
    await addDraft(issue.id, "UPDATE", { labels });
    onRefresh();
  }, [onRefresh]);

  const allKnownLabels = useMemo(() => Array.from(new Set(issues.flatMap((i) => i.labels || []))).sort(), [issues]);

  const startInlineCreate = useCallback((status: string) => {
    setInlineCreateStatus(status);
    setInlineTitle("");
    if (!expandedGroups.has(status)) setExpandedGroups((prev) => new Set(prev).add(status));
    setTimeout(() => inlineRef.current?.focus(), 0);
  }, [expandedGroups]);

  useEffect(() => { if (searchFocused && searchRef.current) searchRef.current.focus(); }, [searchFocused]);

  const visibleStatuses = TAB_CONFIGS[activeTab].statuses;
  const query = search.toLowerCase();
  const filteredIssues = issues.filter(
    (i) => visibleStatuses.includes(i.status) && (!query || i.title.toLowerCase().includes(query) || i.id.toLowerCase().includes(query) || i.labels?.some((l) => l.toLowerCase().includes(query))),
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

  useEffect(() => {
    const storageKey = `beats-backlog-expanded-${activeTab}`;
    const stored = localStorage.getItem(storageKey);
    let next: Set<string>;
    if (stored) {
      next = new Set(JSON.parse(stored) as string[]);
    } else {
      next = new Set<string>();
      for (const status of visibleStatuses) {
        const count = issues.filter((i) => i.status === status).length;
        if (count > 0 && status !== "DONE") next.add(status);
      }
    }
    setExpandedGroups(next);
    setExpandedNodes(new Set(Array.from(childrenByParent.keys())));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [activeTab]);

  const toggleGroup = useCallback((status: string) => {
    setExpandedGroups((prev) => {
      const next = new Set(prev);
      if (next.has(status)) next.delete(status); else next.add(status);
      localStorage.setItem(`beats-backlog-expanded-${activeTab}`, JSON.stringify(Array.from(next)));
      return next;
    });
  }, [activeTab]);

  const toggleNode = useCallback((id: string) => {
    setExpandedNodes((prev) => { const next = new Set(prev); if (next.has(id)) next.delete(id); else next.add(id); return next; });
  }, []);

  const rows = useBacklogRows(issues, filteredIssues, activeTab, expandedGroups, expandedNodes, sortKey);

  // Report navigation order to parent
  const prevNavOrder = useRef<string>("");
  useEffect(() => {
    const ids = rows.filter((r) => r.kind === "issue").map((r) => (r as { kind: "issue"; issue: Issue }).issue.id);
    const key = ids.join(",");
    if (key !== prevNavOrder.current) { prevNavOrder.current = key; onNavigationOrderChange?.(ids); }
  }, [rows, onNavigationOrderChange]);

  // DnD state
  const isDndEnabled = sortKey === "manual";
  const [activeId, setActiveId] = useState<string | null>(null);
  const [dropIndicator, setDropIndicator] = useState<{ rowIndex: number; position: "above" | "below" } | null>(null);
  const [dropGroupStatus, setDropGroupStatus] = useState<string | null>(null);
  const [dropNestTargetId, setDropNestTargetId] = useState<string | null>(null);
  const dragBatchRef = useRef<string[]>([]);
  const modifiersRef = useRef({ alt: false, meta: false, ctrl: false });
  const pointerYRef = useRef<number>(0);

  useEffect(() => {
    if (!activeId) return;
    const syncKeys = (e: KeyboardEvent) => { modifiersRef.current = { alt: e.altKey, meta: e.metaKey, ctrl: e.ctrlKey }; };
    const syncPointer = (e: PointerEvent) => { pointerYRef.current = e.clientY; };
    window.addEventListener("keydown", syncKeys);
    window.addEventListener("keyup", syncKeys);
    window.addEventListener("pointermove", syncPointer);
    return () => { window.removeEventListener("keydown", syncKeys); window.removeEventListener("keyup", syncKeys); window.removeEventListener("pointermove", syncPointer); };
  }, [activeId]);

  const sensors = useSensors(useSensor(PointerSensor, { activationConstraint: { distance: 3 } }), useSensor(KeyboardSensor));

  const getRowStatusGroup = useCallback((rowIndex: number): string | null => {
    for (let j = rowIndex; j >= 0; j--) { const r = rows[j]; if (r.kind === "group") return r.status; } return null;
  }, [rows]);

  const getDragGroup = useCallback((issueId: string): string[] => {
    const issue = issues.find((i) => i.id === issueId);
    if (!issue) return [issueId];
    const children = (childrenByParent.get(issueId) || []).filter((c) => c.status === issue.status);
    if (children.length === 0) return [issueId];
    return [issueId, ...children.map((c) => c.id)];
  }, [issues, childrenByParent]);

  const performDrop = useCallback(async (
    droppedId: string,
    groupTarget: string | null,
    indicatorTarget: { rowIndex: number; position: "above" | "below" } | null,
    nestTarget: string | null,
    batchIds: string[],
  ) => {
    if (nestTarget) { await addDraft(droppedId, "UPDATE", { parent_id: nestTarget }); onRefresh(); return; }
    if (groupTarget) {
      const groupIssues = rows.filter((r): r is typeof r & { kind: "issue" } => r.kind === "issue" && r.depth === 0).filter((r) => { const idx = rows.indexOf(r); return getRowStatusGroup(idx) === groupTarget; }).map((r) => r.issue);
      const effectiveKeys = getEffectiveKeys(groupIssues);
      const last = groupIssues[groupIssues.length - 1];
      const lastKey = last ? effectiveKeys.get(last.id) || null : null;
      const newKey = generateKeyBetween(lastKey, null);
      for (const id of batchIds) { const update: Record<string, unknown> = { status: groupTarget }; if (id === droppedId) update.sort_order = newKey; await addDraft(id, "UPDATE", update); }
      onRefresh(); return;
    }
    if (!indicatorTarget) return;
    const targetRow = rows[indicatorTarget.rowIndex];
    if (targetRow.kind !== "issue") return;
    const status = getRowStatusGroup(indicatorTarget.rowIndex);
    const draggedStatus = issues.find((i) => i.id === droppedId)?.status;
    const update: Record<string, unknown> = {};
    if (draggedStatus !== status) update.status = status;
    if (targetRow.depth > 0) {
      const targetParentId = targetRow.issue.parent_id!;
      update.parent_id = targetParentId;
      const siblings = issues.filter((i) => i.parent_id === targetParentId);
      const effectiveKeys = getEffectiveKeys(siblings);
      const withoutDragged = siblings.filter((i) => i.id !== droppedId);
      let insertIdx = withoutDragged.findIndex((i) => i.id === targetRow.issue.id);
      if (insertIdx === -1) insertIdx = withoutDragged.length;
      if (indicatorTarget.position === "below") insertIdx++;
      const prev = withoutDragged[insertIdx - 1];
      const next = withoutDragged[insertIdx];
      update.sort_order = generateKeyBetween(prev ? effectiveKeys.get(prev.id) || null : null, next ? effectiveKeys.get(next.id) || null : null);
    } else {
      const dragged = issues.find((i) => i.id === droppedId);
      if (dragged?.parent_id) update.parent_id = "";
      const groupTopLevel = rows.filter((r): r is typeof r & { kind: "issue" } => r.kind === "issue" && r.depth === 0).filter((r) => { const idx = rows.indexOf(r); return getRowStatusGroup(idx) === status; }).map((r) => r.issue);
      const effectiveKeys = getEffectiveKeys(groupTopLevel);
      const withoutDragged = groupTopLevel.filter((i) => i.id !== droppedId);
      let insertIdx = withoutDragged.findIndex((i) => i.id === targetRow.issue.id);
      if (insertIdx === -1) insertIdx = withoutDragged.length;
      if (indicatorTarget.position === "below") insertIdx++;
      const prev = withoutDragged[insertIdx - 1];
      const next = withoutDragged[insertIdx];
      update.sort_order = generateKeyBetween(prev ? effectiveKeys.get(prev.id) || null : null, next ? effectiveKeys.get(next.id) || null : null);
    }
    await addDraft(droppedId, "UPDATE", update);
    for (const id of batchIds) {
      if (id === droppedId) continue;
      const childUpdate: Record<string, unknown> = {};
      if (update.status) childUpdate.status = update.status;
      if (Object.keys(childUpdate).length > 0) await addDraft(id, "UPDATE", childUpdate);
    }
    onRefresh();
  }, [rows, getRowStatusGroup, issues, onRefresh]);

  const resetDropState = useCallback(() => { setActiveId(null); setDropIndicator(null); setDropGroupStatus(null); setDropNestTargetId(null); modifiersRef.current = { alt: false, meta: false, ctrl: false }; }, []);

  const handleDndStart = useCallback((event: DragStartEvent) => {
    const id = String(event.active.id);
    const orig = event.activatorEvent as MouseEvent | KeyboardEvent;
    modifiersRef.current = { alt: !!orig.altKey, meta: !!orig.metaKey, ctrl: !!orig.ctrlKey };
    if (typeof (orig as PointerEvent).clientY === "number") pointerYRef.current = (orig as PointerEvent).clientY;
    dragBatchRef.current = getDragGroup(id);
    setActiveId(id); setDropIndicator(null); setDropGroupStatus(null); setDropNestTargetId(null);
  }, [getDragGroup]);

  const handleDndOver = useCallback((event: DragOverEvent) => {
    const draggedId = String(event.active.id);
    const overRaw = event.over?.id ? String(event.over.id) : null;
    if (!overRaw) { setDropIndicator(null); setDropGroupStatus(null); setDropNestTargetId(null); return; }
    if (overRaw.startsWith("group-")) {
      const status = overRaw.slice("group-".length);
      const draggedStatus = issues.find((i) => i.id === draggedId)?.status;
      if (draggedStatus !== status) { setDropIndicator(null); setDropGroupStatus(status); setDropNestTargetId(null); } else { setDropGroupStatus(null); }
      return;
    }
    const overRowIndex = rows.findIndex((r) => r.kind === "issue" && r.issue.id === overRaw);
    if (overRowIndex < 0) return;
    const overRow = rows[overRowIndex];
    if (overRow.kind !== "issue") return;
    if (overRow.issue.id === draggedId) { setDropIndicator(null); setDropGroupStatus(null); setDropNestTargetId(null); return; }
    if (modifiersRef.current.alt) {
      const isDescendant = (parentId: string, childId: string): boolean => {
        for (const i of issues) { if (i.parent_id === parentId) { if (i.id === childId) return true; if (isDescendant(i.id, childId)) return true; } } return false;
      };
      if (!overRow.issue.parent_id && !isDescendant(draggedId, overRaw)) { setDropIndicator(null); setDropGroupStatus(null); setDropNestTargetId(overRaw); return; }
    }
    setDropNestTargetId(null);
    const draggedStatus = issues.find((i) => i.id === draggedId)?.status;
    const targetStatus = getRowStatusGroup(overRowIndex);
    const isCrossGroup = draggedStatus !== targetStatus;
    if (isCrossGroup && !(modifiersRef.current.meta || modifiersRef.current.ctrl)) { setDropIndicator(null); setDropGroupStatus(targetStatus); return; }
    setDropGroupStatus(null);
    const overRect = event.over?.rect;
    if (!overRect) return;
    const overCenterY = overRect.top + overRect.height / 2;
    const position: "above" | "below" = pointerYRef.current < overCenterY ? "above" : "below";
    if (position === "below" && overRow.hasChildren && expandedNodes.has(overRow.issue.id)) {
      const firstChildIdx = rows.findIndex((r, j) => j > overRowIndex && r.kind === "issue" && r.depth > overRow.depth);
      if (firstChildIdx !== -1) { setDropIndicator({ rowIndex: firstChildIdx, position: "above" }); return; }
    }
    setDropIndicator({ rowIndex: overRowIndex, position });
  }, [rows, issues, getRowStatusGroup, expandedNodes]);

  const handleDndEnd = useCallback(async (event: DragEndEvent) => {
    const droppedId = String(event.active.id);
    const groupTarget = dropGroupStatus; const indicatorTarget = dropIndicator; const nestTarget = dropNestTargetId; const batchIds = dragBatchRef.current;
    resetDropState();
    if (!groupTarget && !indicatorTarget && !nestTarget) return;
    await performDrop(droppedId, groupTarget, indicatorTarget, nestTarget, batchIds);
  }, [dropGroupStatus, dropIndicator, dropNestTargetId, performDrop, resetDropState]);

  const [showSortMenu, setShowSortMenu] = useState(false);
  const sortBtnRef = useRef<HTMLButtonElement>(null);
  const sortMenuRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!showSortMenu) return;
    const handler = (e: MouseEvent) => {
      if (sortMenuRef.current && !sortMenuRef.current.contains(e.target as Node) && sortBtnRef.current && !sortBtnRef.current.contains(e.target as Node)) setShowSortMenu(false);
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, [showSortMenu]);

  // Stable refs for keyboard handler
  const rowsRef = useRef(rows); rowsRef.current = rows;
  const focusedIndexRef = useRef(focusedIndex); focusedIndexRef.current = focusedIndex;
  const expandedGroupsRef = useRef(expandedGroups); expandedGroupsRef.current = expandedGroups;
  const expandedNodesRef = useRef(expandedNodes); expandedNodesRef.current = expandedNodes;
  const openPopoverRef = useRef(openPopover); openPopoverRef.current = openPopover;
  const onIssueClickRef = useRef(onIssueClick); onIssueClickRef.current = onIssueClick;
  const toggleGroupRef = useRef(toggleGroup); toggleGroupRef.current = toggleGroup;
  const toggleNodeRef = useRef(toggleNode); toggleNodeRef.current = toggleNode;

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (openPopoverRef.current) return;
      if (isEditableTarget(e)) return;
      if (e.metaKey || e.ctrlKey) return;
      if (e.key === "ArrowDown" || e.key === "j") { e.preventDefault(); setKeyboardNav(true); setFocusedIndex((i) => Math.min(i + 1, rowsRef.current.length - 1)); return; }
      if (e.key === "ArrowUp" || e.key === "k") { e.preventDefault(); setKeyboardNav(true); setFocusedIndex((i) => Math.max(i - 1, 0)); return; }
      const row = rowsRef.current[focusedIndexRef.current];
      if (!row) return;
      if (e.key === "Enter") { e.preventDefault(); if (row.kind === "group" && !row.isEmpty) toggleGroupRef.current(row.status); else if (row.kind === "issue") onIssueClickRef.current?.(row.issue); return; }
      if (e.key === "ArrowRight") { e.preventDefault(); if (row.kind === "group" && !row.isEmpty && !expandedGroupsRef.current.has(row.status)) toggleGroupRef.current(row.status); else if (row.kind === "issue" && row.hasChildren && !expandedNodesRef.current.has(row.issue.id)) toggleNodeRef.current(row.issue.id); return; }
      if (e.key === "ArrowLeft") { e.preventDefault(); if (row.kind === "group" && expandedGroupsRef.current.has(row.status)) toggleGroupRef.current(row.status); else if (row.kind === "issue" && row.hasChildren && expandedNodesRef.current.has(row.issue.id)) toggleNodeRef.current(row.issue.id); return; }
      if (e.key === "." && row.kind === "issue") { navigator.clipboard.writeText(row.issue.id); setToast("Copied issue ID"); setTimeout(() => setToast(null), 1500); return; }
      if (row.kind === "issue") {
        if (e.key === "s") { e.preventDefault(); setOpenPopover({ issueId: row.issue.id, type: "status" }); return; }
        if (e.key === "l") { e.preventDefault(); setOpenPopover({ issueId: row.issue.id, type: "labels" }); return; }
        if (e.key === "e") { e.preventDefault(); setOpenPopover({ issueId: row.issue.id, type: "estimate" }); return; }
      }
    };
    document.addEventListener("keydown", handler);
    return () => document.removeEventListener("keydown", handler);
  }, []);

  useEffect(() => {
    const el = listRef.current?.querySelector(`[data-row="${focusedIndex}"]`);
    el?.scrollIntoView({ block: "nearest" });
  }, [focusedIndex]);

  return (
    <div className="h-full flex flex-col relative">
      {toast && (
        <div className="absolute bottom-6 left-1/2 -translate-x-1/2 z-50 px-3 py-1.5 rounded-[var(--radius-md)] bg-[var(--color-bg-elevated)] border border-[var(--color-border-default)] shadow-[var(--shadow-md)] text-sm text-[var(--color-text-primary)] animate-fade-in">
          {toast}
        </div>
      )}

      {/* Tab bar */}
      <div className="flex items-center gap-4 px-5 h-11 border-b border-[var(--color-border-subtle)] shrink-0">
        {(Object.entries(TAB_CONFIGS) as [Tab, { label: string }][]).map(([id, config]) => (
          <button
            key={id}
            onClick={() => onTabChange(id)}
            className={`text-sm font-medium h-full border-b-[1.5px] -mb-px transition-colors duration-[var(--duration-fast)] ${activeTab === id ? "text-[var(--color-text-primary)] border-[var(--color-text-primary)]" : "text-[var(--color-text-muted)] border-transparent hover:text-[var(--color-text-secondary)]"}`}
          >
            {config.label}
          </button>
        ))}
        <div className="ml-auto flex items-center gap-3">
          {(search || searchFocused) && (
            <div className="flex items-center gap-1.5 bg-[var(--color-bg-tertiary)] rounded-[var(--radius-md)] px-2 py-1">
              <svg className="w-3.5 h-3.5 text-[var(--color-text-muted)] shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M21 21l-5.197-5.197m0 0A7.5 7.5 0 105.196 5.196a7.5 7.5 0 0010.607 10.607z" />
              </svg>
              <input
                ref={searchRef}
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                onBlur={() => { if (!search) onSearchBlur?.(); }}
                onKeyDown={(e) => { if (e.key === "Escape") { setSearch(""); onSearchBlur?.(); } }}
                placeholder="Filter issues..."
                className="bg-transparent text-sm text-[var(--color-text-primary)] placeholder:text-[var(--color-text-muted)] outline-none w-36"
              />
            </div>
          )}
          <div className="relative">
            <button
              ref={sortBtnRef}
              onClick={() => setShowSortMenu((v) => !v)}
              className="flex items-center gap-1 h-6 px-1.5 rounded-[var(--radius-sm)] text-xs text-[var(--color-text-muted)] hover:text-[var(--color-text-secondary)] hover:bg-[var(--color-bg-hover)] transition-colors"
            >
              <svg className="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M3 7h6M3 12h10M3 17h14" />
              </svg>
              {SORT_OPTIONS.find((o) => o.value === sortKey)?.label}
            </button>
            {showSortMenu && (
              <div ref={sortMenuRef} className="absolute right-0 top-full mt-1 z-50 min-w-[140px] bg-[var(--color-bg-elevated)] border border-[var(--color-border-default)] rounded-[var(--radius-md)] shadow-[var(--shadow-popover)] py-1">
                {SORT_OPTIONS.map((opt) => (
                  <button key={opt.value} onClick={() => { onSortChange(opt.value); setShowSortMenu(false); }} className={`flex items-center gap-2 w-full h-7 px-3 text-sm transition-colors ${opt.value === sortKey ? "text-[var(--color-text-primary)] bg-[var(--color-bg-hover)]" : "text-[var(--color-text-secondary)] hover:bg-[var(--color-bg-hover)]"}`}>
                    {opt.label}
                    {opt.value === sortKey && (
                      <svg className="w-3 h-3 ml-auto text-[var(--color-accent-primary)]" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}>
                        <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
                      </svg>
                    )}
                  </button>
                ))}
              </div>
            )}
          </div>
          <span className="text-xs text-[var(--color-text-muted)] tabular-nums">{filteredIssues.length} issue{filteredIssues.length !== 1 ? "s" : ""}</span>
        </div>
      </div>

      {/* Rows */}
      <DndContext sensors={sensors} collisionDetection={backlogCollision} onDragStart={handleDndStart} onDragOver={handleDndOver} onDragEnd={handleDndEnd} onDragCancel={resetDropState}>
        <div ref={listRef} className="flex-1 overflow-y-auto" onMouseLeave={() => { if (!keyboardNav) setFocusedIndex(-1); }}>
          {(() => {
            const sections: { groupRow: RowItem & { kind: "group" }; groupIndex: number; issueRows: { row: RowItem & { kind: "issue" }; index: number }[] }[] = [];
            for (let i = 0; i < rows.length; i++) {
              const row = rows[i];
              if (row.kind === "group") sections.push({ groupRow: row, groupIndex: i, issueRows: [] });
              else if (sections.length > 0) sections[sections.length - 1].issueRows.push({ row, index: i });
            }
            return sections.map(({ groupRow, groupIndex, issueRows }) => {
              const isFocused = groupIndex === focusedIndex;
              const isExpanded = expandedGroups.has(groupRow.status);
              const isInlineActive = inlineCreateStatus === groupRow.status;
              const isDropGroup = dropGroupStatus === groupRow.status;
              return (
                <div key={`g-${groupRow.status}`} className={`${isDropGroup ? "ring-2 ring-inset ring-[var(--color-accent-primary)] bg-[var(--color-accent-primary)]/5" : ""}`}>
                  <GroupHeaderDnd status={groupRow.status} enabled={isDndEnabled && !!activeId}>
                    {(setHeaderRef) => (
                      <div
                        ref={setHeaderRef}
                        data-row={groupIndex}
                        onClick={() => !groupRow.isEmpty && toggleGroup(groupRow.status)}
                        onMouseEnter={() => { setKeyboardNav(false); setFocusedIndex(groupIndex); }}
                        className={`flex items-center gap-2 w-full px-5 py-2 border-b border-[var(--color-border-subtle)] transition-colors duration-[var(--duration-fast)] select-none ${groupRow.isEmpty ? "opacity-40 cursor-default" : "cursor-pointer"} ${!isDropGroup && isFocused && keyboardNav ? "bg-[var(--color-bg-hover)] ring-1 ring-inset ring-[var(--color-accent-primary)]/40" : !isDropGroup && isFocused ? "bg-[var(--color-bg-hover)]" : !isDropGroup ? "bg-[var(--color-bg-secondary)]" : ""}`}
                      >
                        <svg className={`w-3 h-3 text-[var(--color-text-muted)] transition-transform duration-100 ${isExpanded && !groupRow.isEmpty ? "rotate-90" : ""}`} fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}>
                          <path strokeLinecap="round" strokeLinejoin="round" d="M9 5l7 7-7 7" />
                        </svg>
                        <StatusIcon status={groupRow.status} size={14} />
                        <span className="text-sm font-medium text-[var(--color-text-primary)]">{groupRow.label}</span>
                        <span className="text-sm text-[var(--color-text-muted)] tabular-nums">{groupRow.count}</span>
                        <button onClick={(e) => { e.stopPropagation(); startInlineCreate(groupRow.status); }} className="ml-auto p-0.5 rounded-[var(--radius-sm)] text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-bg-hover)] transition-colors" title={`New ${groupRow.label} issue`}>
                          <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}><path strokeLinecap="round" strokeLinejoin="round" d="M12 4.5v15m7.5-7.5h-15" /></svg>
                        </button>
                      </div>
                    )}
                  </GroupHeaderDnd>
                  {isInlineActive && (
                    <div className="flex items-center gap-2.5 px-5 h-[38px] border-b border-[var(--color-border-subtle)] bg-[var(--color-bg-tertiary)]">
                      <StatusIcon status={groupRow.status} size={14} className="shrink-0 ml-7" />
                      <input
                        ref={inlineRef}
                        value={inlineTitle}
                        onChange={(e) => setInlineTitle(e.target.value)}
                        onKeyDown={(e) => { if (e.key === "Enter" && inlineTitle.trim()) handleInlineCreate(groupRow.status, inlineTitle); if (e.key === "Escape") { setInlineCreateStatus(null); setInlineTitle(""); } }}
                        onBlur={() => { if (!inlineTitle.trim()) { setInlineCreateStatus(null); setInlineTitle(""); } }}
                        placeholder="New issue title... (Enter to create, Esc to cancel)"
                        className="flex-1 bg-transparent text-sm text-[var(--color-text-primary)] placeholder:text-[var(--color-text-muted)] outline-none"
                      />
                    </div>
                  )}
                  {issueRows.map(({ row, index: i }) => {
                    const { issue, depth, hasChildren, childDone, childTotal, parentBreadcrumb, isGhostParent } = row;
                    const isRowFocused = i === focusedIndex;
                    const isNodeExpanded = expandedNodes.has(issue.id);
                    const indent = depth * 24;
                    const canDrag = isDndEnabled && !isGhostParent;
                    const isDropTarget = isDndEnabled && !!activeId;
                    const showDropAbove = dropIndicator?.rowIndex === i && dropIndicator.position === "above";
                    const showDropBelow = dropIndicator?.rowIndex === i && dropIndicator.position === "below";
                    const isNestTarget = dropNestTargetId === issue.id;
                    const isDraggedOrBatch = activeId !== null && dragBatchRef.current.includes(issue.id);
                    const dragBatchCount = activeId === issue.id ? dragBatchRef.current.length : 0;
                    return (
                      <div key={issue.id} className="relative">
                        {showDropAbove && <div className="absolute top-0 right-5 h-[2px] bg-[var(--color-accent-primary)] z-10 rounded-full" style={{ left: `${20 + depth * 24}px` }} />}
                        <IssueRowDnd id={issue.id} canDrag={canDrag} enabled={isDropTarget}>
                          {(setRowRef, dragProps) => (
                            <div
                              ref={setRowRef}
                              data-row={i}
                              {...dragProps.attributes}
                              {...dragProps.listeners}
                              onClick={() => onIssueClick?.(issue)}
                              onMouseEnter={() => { setKeyboardNav(false); setFocusedIndex(i); }}
                              className={`flex items-center gap-2.5 px-5 h-[38px] border-b border-[var(--color-border-subtle)] cursor-pointer transition-colors duration-[var(--duration-fast)] group ${isGhostParent ? "opacity-50" : ""} ${isNestTarget ? "ring-2 ring-inset ring-[var(--color-accent-primary)] bg-[var(--color-accent-primary)]/10" : isRowFocused && keyboardNav ? "bg-[var(--color-bg-hover)] ring-1 ring-inset ring-[var(--color-accent-primary)]/40" : isRowFocused ? "bg-[var(--color-bg-hover)]" : keyboardNav ? "" : "hover:bg-[var(--color-bg-hover)]"} ${isDraggedOrBatch ? "opacity-40" : ""}`}
                              style={{ paddingLeft: `${20 + indent}px` }}
                            >
                              {hasChildren ? (
                                <button onClick={(e) => { e.stopPropagation(); toggleNode(issue.id); }} className="w-6 h-6 -m-1 shrink-0 flex items-center justify-center rounded cursor-pointer text-[var(--color-text-muted)] hover:text-[var(--color-text-secondary)] hover:bg-white/10">
                                  <svg className={`w-3 h-3 transition-transform duration-100 ${isNodeExpanded ? "rotate-90" : ""}`} fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}><path strokeLinecap="round" strokeLinejoin="round" d="M9 5l7 7-7 7" /></svg>
                                </button>
                              ) : depth > 0 ? (
                                <span className={`w-4 shrink-0 flex items-center justify-center text-[var(--color-border-default)] ${canDrag ? "cursor-grab active:cursor-grabbing" : ""}`}>
                                  <svg width="12" height="16" viewBox="0 0 12 16" fill="none" stroke="currentColor" strokeWidth="1.5"><path d="M1 0v9h10" /></svg>
                                </span>
                              ) : (
                                <span className={`text-[var(--color-text-muted)] transition-opacity w-4 shrink-0 flex items-center justify-center ${canDrag ? "opacity-30 cursor-grab active:cursor-grabbing" : "opacity-0 group-hover:opacity-30"}`}>
                                  <svg width="6" height="10" viewBox="0 0 6 10" fill="currentColor"><circle cx="1" cy="1" r="1" /><circle cx="5" cy="1" r="1" /><circle cx="1" cy="5" r="1" /><circle cx="5" cy="5" r="1" /><circle cx="1" cy="9" r="1" /><circle cx="5" cy="9" r="1" /></svg>
                                </span>
                              )}
                              <CopyableId id={issue.id} className="text-xs w-[110px] shrink-0 truncate tabular-nums" />
                              <div className="relative shrink-0" onClick={(e) => e.stopPropagation()}>
                                <button onClick={() => setOpenPopover(openPopover?.issueId === issue.id && openPopover?.type === "status" ? null : { issueId: issue.id, type: "status" })} className="w-6 h-6 -m-1 flex items-center justify-center rounded cursor-pointer hover:bg-white/10 transition-colors">
                                  <StatusIcon status={issue.status} size={14} />
                                </button>
                                {openPopover?.issueId === issue.id && openPopover?.type === "status" && (
                                  <Popover onClose={() => setOpenPopover(null)}>
                                    <PopoverHeader>Set status...</PopoverHeader>
                                    {STATUS_OPTIONS.map((opt) => (
                                      <button key={opt.value} onClick={() => handleQuickStatus(issue.id, opt.value)} className={`flex items-center gap-2 w-full px-3 py-1.5 text-sm transition-colors hover:bg-[var(--color-bg-hover)] ${opt.value === issue.status ? "text-[var(--color-accent-primary)]" : "text-[var(--color-text-primary)]"}`}>
                                        <StatusIcon status={opt.value} size={14} /><span>{opt.label}</span>
                                        {opt.value === issue.status && <svg className="w-3.5 h-3.5 ml-auto" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}><path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" /></svg>}
                                      </button>
                                    ))}
                                  </Popover>
                                )}
                              </div>
                              {parentBreadcrumb && <span className="text-sm text-[var(--color-text-muted)] truncate shrink-0 max-w-[150px]">{parentBreadcrumb}</span>}
                              {parentBreadcrumb && <svg className="w-3 h-3 text-[var(--color-text-muted)] shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}><path strokeLinecap="round" strokeLinejoin="round" d="M9 5l7 7-7 7" /></svg>}
                              <span className={`text-sm truncate min-w-0 ${isGhostParent ? "text-[var(--color-text-muted)]" : "text-[var(--color-text-primary)]"}`}>{issue.title}</span>
                              {dragBatchCount > 1 && <span className="flex items-center justify-center w-5 h-5 rounded-full bg-[var(--color-accent-primary)] text-white text-xs font-medium shrink-0">{dragBatchCount}</span>}
                              {hasChildren && <span className="flex items-center gap-1.5 text-xs text-[var(--color-text-muted)] shrink-0"><SubProgress done={childDone} total={childTotal} />{childDone}/{childTotal}</span>}
                              <div className="flex-1" />
                              {issue.is_pending && <span className="w-1.5 h-1.5 rounded-full bg-[var(--color-warning)] shrink-0" />}
                              {issue.priority > 0 && <span className={`text-xs font-medium shrink-0 ${issue.priority === 1 ? "text-[var(--color-error)]" : issue.priority === 2 ? "text-[var(--color-warning)]" : "text-[var(--color-text-muted)]"}`}>{issue.priority === 1 ? "!!!" : issue.priority === 2 ? "!!" : issue.priority === 3 ? "!" : ""}</span>}
                              <div className="relative flex items-center gap-2.5 shrink-0" onClick={(e) => e.stopPropagation()}>
                                <button onClick={() => setOpenPopover(openPopover?.issueId === issue.id && openPopover?.type === "labels" ? null : { issueId: issue.id, type: "labels" })} className="flex items-center gap-2.5 hover:opacity-70 transition-opacity">
                                  {issue.labels?.map((label) => <LabelBadge key={label} label={label} />)}
                                  {(!issue.labels || issue.labels.length === 0) && <span className="text-xs text-[var(--color-text-muted)] opacity-0 group-hover:opacity-100 transition-opacity">+ label</span>}
                                </button>
                                {openPopover?.issueId === issue.id && openPopover?.type === "labels" && (
                                  <Popover onClose={() => setOpenPopover(null)}>
                                    <LabelPicker allLabels={allKnownLabels} selected={issue.labels || []} onToggle={(label) => handleQuickLabelToggle(issue, label)} onConfigLabelsChange={onConfigLabelsChange} onClose={() => setOpenPopover(null)} />
                                  </Popover>
                                )}
                              </div>
                              <div className="relative shrink-0" onClick={(e) => e.stopPropagation()}>
                                <button onClick={() => setOpenPopover(openPopover?.issueId === issue.id && openPopover?.type === "estimate" ? null : { issueId: issue.id, type: "estimate" })} className="flex items-center gap-1 text-xs text-[var(--color-text-muted)] tabular-nums w-10 justify-end hover:opacity-70 transition-opacity">
                                  {issue.estimate > 0 ? (<><svg className="w-3 h-3" viewBox="0 0 16 16" fill="none"><path d="M8 2L14 14H2L8 2Z" stroke="currentColor" strokeWidth="1.5" strokeLinejoin="round" /></svg>{issue.estimate}</>) : (<span className="opacity-0 group-hover:opacity-100 transition-opacity"><svg className="w-3 h-3" viewBox="0 0 16 16" fill="none"><path d="M8 2L14 14H2L8 2Z" stroke="currentColor" strokeWidth="1.5" strokeLinejoin="round" /></svg></span>)}
                                </button>
                                {openPopover?.issueId === issue.id && openPopover?.type === "estimate" && (
                                  <Popover onClose={() => setOpenPopover(null)}>
                                    <PopoverHeader>Set estimate...</PopoverHeader>
                                    {ESTIMATE_OPTIONS.map((est) => (
                                      <button key={est} onClick={() => handleQuickEstimate(issue.id, est)} className={`flex items-center gap-2 w-full px-3 py-1.5 text-sm transition-colors hover:bg-[var(--color-bg-hover)] ${est === (issue.estimate || 0) ? "text-[var(--color-accent-primary)]" : "text-[var(--color-text-primary)]"}`}>
                                        <span>{est === 0 ? "No estimate" : `${est} Point${est !== 1 ? "s" : ""}`}</span>
                                        {est === (issue.estimate || 0) && <svg className="w-3.5 h-3.5 ml-auto" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}><path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" /></svg>}
                                      </button>
                                    ))}
                                  </Popover>
                                )}
                              </div>
                              {issue.assignee && <Avatar name={issue.assignee} size="sm" />}
                              <span className="text-xs text-[var(--color-text-muted)] tabular-nums shrink-0 w-16 text-right">{formatShortDate(issue.created_at)}</span>
                            </div>
                          )}
                        </IssueRowDnd>
                        {showDropBelow && <div className="absolute bottom-0 right-5 h-[2px] bg-[var(--color-accent-primary)] z-10 rounded-full" style={{ left: `${20 + depth * 24}px` }} />}
                      </div>
                    );
                  })}
                </div>
              );
            });
          })()}
        </div>
        <DragOverlay dropAnimation={null}>
          {activeId ? (() => { const dragged = issues.find((i) => i.id === activeId); if (!dragged) return null; return <DragOverlayCard issue={dragged} batchCount={dragBatchRef.current.length} />; })() : null}
        </DragOverlay>
      </DndContext>
    </div>
  );
}
