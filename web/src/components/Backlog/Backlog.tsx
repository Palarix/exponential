import { useState, useRef, useEffect, useCallback, useMemo } from 'react';
import { generateKeyBetween } from 'fractional-indexing';
import { createIssue, addDraft } from '../../api/client';
import type { Issue } from '../../api/client';
import { LabelBadge, StatusIcon, CopyableId, Popover, PopoverHeader } from '../ui';
import { sortGroup, getEffectiveKeys, SORT_OPTIONS } from '../../utils/sort';
import type { SortKey } from '../../utils/sort';
import { isEditableTarget } from '../../utils/keyboard';
import { STATUS_OPTIONS, ESTIMATE_OPTIONS } from '../../constants';

export type Tab = 'all' | 'active' | 'backlog';

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
}

const TAB_CONFIGS: Record<Tab, { label: string; statuses: string[] }> = {
  all: { label: 'All Issues', statuses: ['BACKLOG', 'PLANNED', 'DOING', 'BLOCKED', 'DONE'] },
  active: { label: 'Active', statuses: ['PLANNED', 'DOING', 'BLOCKED'] },
  backlog: { label: 'Backlog', statuses: ['BACKLOG'] },
};

const STATUS_META: Record<string, { label: string }> = {
  BACKLOG: { label: 'Backlog' },
  PLANNED: { label: 'Planned' },
  DOING: { label: 'In Progress' },
  BLOCKED: { label: 'Blocked' },
  DONE: { label: 'Done' },
};

type RowItem =
  | { kind: 'group'; status: string; label: string; count: number; isEmpty: boolean }
  | { kind: 'issue'; issue: Issue; depth: number; hasChildren: boolean; childDone: number; childTotal: number; parentBreadcrumb?: string; isGhostParent?: boolean };

export default function Backlog({ issues, onRefresh, onIssueClick, searchFocused, onSearchBlur, sortKey, onSortChange, onNavigationOrderChange, activeTab, onTabChange }: BacklogProps) {
  const [search, setSearch] = useState('');
  const [expandedGroups, setExpandedGroups] = useState<Set<string>>(() => new Set());
  const [expandedNodes, setExpandedNodes] = useState<Set<string>>(() => new Set());
  const [focusedIndex, setFocusedIndex] = useState(-1);
  const [inlineCreateStatus, setInlineCreateStatus] = useState<string | null>(null);
  const [inlineTitle, setInlineTitle] = useState('');
  const [toast, setToast] = useState<string | null>(null);
  const [openPopover, setOpenPopover] = useState<{ issueId: string; type: 'status' | 'estimate' | 'labels' } | null>(null);
  const searchRef = useRef<HTMLInputElement>(null);
  const listRef = useRef<HTMLDivElement>(null);
  const inlineRef = useRef<HTMLInputElement>(null);

  const handleInlineCreate = useCallback(async (status: string, title: string) => {
    if (!title.trim()) return;
    const issueId = await createIssue({ title: title.trim(), labels: ['feature'] });
    if (status !== 'BACKLOG') {
      await addDraft(issueId, 'UPDATE', { status });
    }
    setInlineTitle('');
    onRefresh();
  }, [onRefresh]);

  const handleQuickStatus = useCallback(async (issueId: string, status: string) => {
    await addDraft(issueId, 'UPDATE', { status });
    setOpenPopover(null);
    onRefresh();
  }, [onRefresh]);

  const handleQuickEstimate = useCallback(async (issueId: string, estimate: number) => {
    await addDraft(issueId, 'UPDATE', { estimate });
    setOpenPopover(null);
    onRefresh();
  }, [onRefresh]);

  const handleQuickLabelToggle = useCallback(async (issue: Issue, label: string) => {
    const current = issue.labels || [];
    const labels = current.includes(label) ? current.filter(l => l !== label) : [...current, label];
    await addDraft(issue.id, 'UPDATE', { labels });
    onRefresh();
  }, [onRefresh]);

  const allKnownLabels = useMemo(() =>
    Array.from(new Set(issues.flatMap(i => i.labels || []))).sort(),
    [issues]
  );

  const startInlineCreate = useCallback((status: string) => {
    setInlineCreateStatus(status);
    setInlineTitle('');
    if (!expandedGroups.has(status)) {
      setExpandedGroups(prev => new Set(prev).add(status));
    }
    setTimeout(() => inlineRef.current?.focus(), 0);
  }, [expandedGroups]);

  useEffect(() => {
    if (searchFocused && searchRef.current) {
      searchRef.current.focus();
    }
  }, [searchFocused]);

  const visibleStatuses = TAB_CONFIGS[activeTab].statuses;
  const query = search.toLowerCase();
  const filteredIssues = issues.filter((i) =>
    visibleStatuses.includes(i.status) &&
    (!query || i.title.toLowerCase().includes(query) || i.id.toLowerCase().includes(query) || i.labels?.some(l => l.toLowerCase().includes(query)))
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

  // Initialize expanded groups based on content
  useEffect(() => {
    const next = new Set<string>();
    for (const status of visibleStatuses) {
      const count = issues.filter(i => i.status === status).length;
      if (count > 0 && status !== 'DONE') next.add(status);
    }
    setExpandedGroups(next);
    setExpandedNodes(new Set());
    // Default: all parent nodes expanded
    for (const [parentId] of childrenByParent) {
      next.add(`node:${parentId}`);
    }
    setExpandedNodes(new Set(Array.from(childrenByParent.keys())));
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [activeTab]);

  const toggleGroup = useCallback((status: string) => {
    setExpandedGroups(prev => {
      const next = new Set(prev);
      if (next.has(status)) next.delete(status);
      else next.add(status);
      return next;
    });
  }, []);

  const toggleNode = useCallback((id: string) => {
    setExpandedNodes(prev => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }, []);

  // Build flat row list
  const rows = useMemo(() => {
    const result: RowItem[] = [];
    const isAllTab = activeTab === 'all';

    for (const status of visibleStatuses) {
      const groupIssues = filteredIssues.filter(i => i.status === status);
      if (groupIssues.length === 0 && !isAllTab) continue;

      result.push({ kind: 'group', status, label: STATUS_META[status]?.label || status, count: groupIssues.length, isEmpty: groupIssues.length === 0 });

      if (expandedGroups.has(status) && groupIssues.length > 0) {
        const groupIssueIds = new Set(groupIssues.map(i => i.id));

        if (isAllTab) {
          // All Issues tab: flat with breadcrumbs for orphaned children
          const topLevel = sortGroup(
            groupIssues.filter(i => !i.parent_id || !groupIssueIds.has(i.parent_id)),
            sortKey
          );
          const addTree = (issue: Issue, depth: number, breadcrumb?: string) => {
            const allChildren = childrenByParent.get(issue.id) || [];
            const doneCount = allChildren.filter(c => c.status === 'DONE').length;
            result.push({ kind: 'issue', issue, depth, hasChildren: allChildren.length > 0, childDone: doneCount, childTotal: allChildren.length, parentBreadcrumb: breadcrumb });
            const visibleChildren = sortGroup(allChildren.filter(c => groupIssueIds.has(c.id)), sortKey);
            if (visibleChildren.length > 0 && expandedNodes.has(issue.id)) {
              for (const child of visibleChildren) addTree(child, depth + 1);
            }
          };
          for (const issue of topLevel) {
            const parent = issue.parent_id ? issues.find(i => i.id === issue.parent_id) : null;
            addTree(issue, 0, parent ? parent.title : undefined);
          }
        } else {
          // Active/Backlog tabs: tree with ghost parents
          const topLevel = sortGroup(
            groupIssues.filter(i => !i.parent_id),
            sortKey
          );
          // Find children whose parent is NOT in this group → need ghost parents
          const orphanedChildren = groupIssues.filter(i => i.parent_id && !groupIssueIds.has(i.parent_id));
          const ghostParentIds = new Set(orphanedChildren.map(i => i.parent_id!));

          const addTree = (issue: Issue, depth: number, isGhost?: boolean) => {
            const allChildren = childrenByParent.get(issue.id) || [];
            const doneCount = allChildren.filter(c => c.status === 'DONE').length;
            result.push({ kind: 'issue', issue, depth, hasChildren: allChildren.length > 0, childDone: doneCount, childTotal: allChildren.length, isGhostParent: isGhost });
            const visibleChildren = sortGroup(allChildren.filter(c => groupIssueIds.has(c.id)), sortKey);
            if (visibleChildren.length > 0 && (isGhost || expandedNodes.has(issue.id))) {
              for (const child of visibleChildren) addTree(child, depth + 1);
            }
          };

          for (const issue of topLevel) addTree(issue, 0);
          // Add ghost parent rows for orphaned children
          for (const parentId of ghostParentIds) {
            const parent = issues.find(i => i.id === parentId);
            if (parent) addTree(parent, 0, true);
          }
        }
      }
    }
    return result;
  }, [visibleStatuses, filteredIssues, activeTab, expandedGroups, expandedNodes, issues, childrenByParent, sortKey]);

  // Report navigation order to parent
  const prevNavOrder = useRef<string>('');
  useEffect(() => {
    const ids = rows.filter(r => r.kind === 'issue').map(r => (r as { kind: 'issue'; issue: Issue }).issue.id);
    const key = ids.join(',');
    if (key !== prevNavOrder.current) {
      prevNavOrder.current = key;
      onNavigationOrderChange?.(ids);
    }
  }, [rows, onNavigationOrderChange]);

  // Drag-and-drop for manual reordering
  const isDndEnabled = sortKey === 'manual';
  const [draggedId, setDraggedId] = useState<string | null>(null);
  const [dropIndicator, _setDropIndicator] = useState<{ rowIndex: number; position: 'above' | 'below' } | null>(null);
  const [dropGroupStatus, _setDropGroupStatus] = useState<string | null>(null);
  const dropIndicatorRef = useRef(dropIndicator);
  const dropGroupStatusRef = useRef(dropGroupStatus);
  const [dropNestTargetId, _setDropNestTargetId] = useState<string | null>(null);
  const dropNestTargetIdRef = useRef(dropNestTargetId);
  const setDropIndicator = useCallback((v: typeof dropIndicator) => { dropIndicatorRef.current = v; _setDropIndicator(v); }, []);
  const setDropGroupStatus = useCallback((v: typeof dropGroupStatus) => { dropGroupStatusRef.current = v; _setDropGroupStatus(v); }, []);
  const setDropNestTargetId = useCallback((v: typeof dropNestTargetId) => { dropNestTargetIdRef.current = v; _setDropNestTargetId(v); }, []);

  const getRowStatusGroup = useCallback((rowIndex: number): string | null => {
    for (let j = rowIndex; j >= 0; j--) {
      const r = rows[j];
      if (r.kind === 'group') return r.status;
    }
    return null;
  }, [rows]);

  const getDragGroup = useCallback((issueId: string): string[] => {
    const issue = issues.find(i => i.id === issueId);
    if (!issue) return [issueId];
    const children = (childrenByParent.get(issueId) || []).filter(c => c.status === issue.status);
    if (children.length === 0) return [issueId];
    return [issueId, ...children.map(c => c.id)];
  }, [issues, childrenByParent]);

  const dragGroupRef = useRef<string[]>([]);

  const handleDragStart = useCallback((e: React.DragEvent, issueId: string) => {
    dropHandledRef.current = false;
    setDropNestTargetId(null);
    const group = getDragGroup(issueId);
    dragGroupRef.current = group;
    setDraggedId(issueId);
    e.dataTransfer.effectAllowed = 'all';
    e.dataTransfer.setData('text/plain', issueId);
    if (e.currentTarget instanceof HTMLElement) {
      e.currentTarget.style.opacity = '0.4';
    }
  }, [getDragGroup]);

  const handleDragOver = useCallback((e: React.DragEvent, rowIndex: number) => {
    e.preventDefault();
    e.dataTransfer.dropEffect = 'move';
    if (!draggedId) return;

    const row = rows[rowIndex];
    if (row.kind !== 'issue') return;
    if (row.issue.id === draggedId) { setDropIndicator(null); setDropGroupStatus(null); setDropNestTargetId(null); return; }

    // ALT+drag: nest as child
    if (e.altKey) {
      const targetId = row.issue.id;
      const isDescendant = (parentId: string, childId: string): boolean => {
        for (const i of issues) {
          if (i.parent_id === parentId) {
            if (i.id === childId) return true;
            if (isDescendant(i.id, childId)) return true;
          }
        }
        return false;
      };
      if (!row.issue.parent_id && !isDescendant(draggedId, targetId)) {
        setDropIndicator(null);
        setDropGroupStatus(null);
        setDropNestTargetId(targetId);
        return;
      }
    }

    setDropNestTargetId(null);
    const draggedStatus = issues.find(i => i.id === draggedId)?.status;
    const targetStatus = getRowStatusGroup(rowIndex);
    const isCrossGroup = draggedStatus !== targetStatus;

    if (isCrossGroup && !e.metaKey) {
      setDropIndicator(null);
      setDropGroupStatus(targetStatus);
      return;
    }

    setDropGroupStatus(null);
    const rect = (e.currentTarget as HTMLElement).getBoundingClientRect();
    let position: 'above' | 'below' = e.clientY < rect.top + rect.height / 2 ? 'above' : 'below';

    // "Below" a parent with expanded children → redirect to "above first child"
    if (position === 'below' && row.hasChildren && expandedNodes.has(row.issue.id)) {
      const firstChildIdx = rows.findIndex((r, j) => j > rowIndex && r.kind === 'issue' && r.depth > row.depth);
      if (firstChildIdx !== -1) {
        setDropIndicator({ rowIndex: firstChildIdx, position: 'above' });
        return;
      }
    }

    setDropIndicator({ rowIndex, position });
  }, [draggedId, rows, issues, getRowStatusGroup, expandedNodes]);

  const dropHandledRef = useRef(false);

  const performDrop = useCallback(async (
    droppedId: string,
    groupTarget: string | null,
    indicatorTarget: { rowIndex: number; position: 'above' | 'below' } | null,
    nestTarget: string | null,
  ) => {
    if (dropHandledRef.current) return;
    dropHandledRef.current = true;

    const batchIds = dragGroupRef.current;

    if (nestTarget) {
      await addDraft(droppedId, 'UPDATE', { parent_id: nestTarget });
      onRefresh();
      return;
    }

    if (groupTarget) {
      const groupIssues = rows
        .filter((r): r is typeof r & { kind: 'issue' } => r.kind === 'issue' && r.depth === 0)
        .filter(r => {
          const idx = rows.indexOf(r);
          return getRowStatusGroup(idx) === groupTarget;
        })
        .map(r => r.issue);

      const effectiveKeys = getEffectiveKeys(groupIssues);
      const last = groupIssues[groupIssues.length - 1];
      const lastKey = last ? (effectiveKeys.get(last.id) || null) : null;
      const newKey = generateKeyBetween(lastKey, null);

      for (const id of batchIds) {
        const issue = issues.find(i => i.id === id);
        const update: Record<string, unknown> = { status: groupTarget };
        if (id === droppedId) {
          update.sort_order = newKey;
          if (issue?.parent_id) update.parent_id = '';
        }
        await addDraft(id, 'UPDATE', update);
      }
      onRefresh();
      return;
    }

    if (!indicatorTarget) return;

    const targetRow = rows[indicatorTarget.rowIndex];
    if (targetRow.kind !== 'issue') return;

    const status = getRowStatusGroup(indicatorTarget.rowIndex);
    const draggedStatus = issues.find(i => i.id === droppedId)?.status;
    const update: Record<string, unknown> = {};
    if (draggedStatus !== status) update.status = status;

    if (targetRow.depth > 0) {
      // Dropping between children → nest under that parent
      const targetParentId = targetRow.issue.parent_id!;
      update.parent_id = targetParentId;
      const siblings = issues.filter(i => i.parent_id === targetParentId);
      const effectiveKeys = getEffectiveKeys(siblings);
      const withoutDragged = siblings.filter(i => i.id !== droppedId);
      let insertIdx = withoutDragged.findIndex(i => i.id === targetRow.issue.id);
      if (insertIdx === -1) insertIdx = withoutDragged.length;
      if (indicatorTarget.position === 'below') insertIdx++;
      const prev = withoutDragged[insertIdx - 1];
      const next = withoutDragged[insertIdx];
      const prevKey = prev ? (effectiveKeys.get(prev.id) || null) : null;
      const nextKey = next ? (effectiveKeys.get(next.id) || null) : null;
      update.sort_order = generateKeyBetween(prevKey, nextKey);
    } else {
      // Dropping between top-level issues → un-nest
      const dragged = issues.find(i => i.id === droppedId);
      if (dragged?.parent_id) update.parent_id = '';
      const groupTopLevel = rows
        .filter((r): r is typeof r & { kind: 'issue' } => r.kind === 'issue' && r.depth === 0)
        .filter(r => {
          const idx = rows.indexOf(r);
          return getRowStatusGroup(idx) === status;
        })
        .map(r => r.issue);
      const effectiveKeys = getEffectiveKeys(groupTopLevel);
      const withoutDragged = groupTopLevel.filter(i => i.id !== droppedId);
      let insertIdx = withoutDragged.findIndex(i => i.id === targetRow.issue.id);
      if (insertIdx === -1) insertIdx = withoutDragged.length;
      if (indicatorTarget.position === 'below') insertIdx++;
      const prev = withoutDragged[insertIdx - 1];
      const next = withoutDragged[insertIdx];
      const prevKey = prev ? (effectiveKeys.get(prev.id) || null) : null;
      const nextKey = next ? (effectiveKeys.get(next.id) || null) : null;
      update.sort_order = generateKeyBetween(prevKey, nextKey);
    }

    await addDraft(droppedId, 'UPDATE', update);
    // Batch-move same-status children
    for (const id of batchIds) {
      if (id === droppedId) continue;
      const childUpdate: Record<string, unknown> = {};
      if (update.status) childUpdate.status = update.status;
      if (Object.keys(childUpdate).length > 0) await addDraft(id, 'UPDATE', childUpdate);
    }
    onRefresh();
  }, [rows, getRowStatusGroup, issues, onRefresh]);

  const handleDrop = useCallback(async (e: React.DragEvent) => {
    e.preventDefault();
    if (!draggedId) { setDraggedId(null); setDropIndicator(null); setDropGroupStatus(null); setDropNestTargetId(null); return; }
    await performDrop(draggedId, dropGroupStatusRef.current, dropIndicatorRef.current, dropNestTargetIdRef.current);
    setDraggedId(null);
    setDropIndicator(null);
    setDropGroupStatus(null);
    setDropNestTargetId(null);
  }, [draggedId, performDrop]);

  const handleDragEnd = useCallback((e: React.DragEvent) => {
    if (e.currentTarget instanceof HTMLElement) {
      e.currentTarget.style.opacity = '';
    }
    const currentDropGroup = dropGroupStatusRef.current;
    const currentDropIndicator = dropIndicatorRef.current;
    const currentNestTarget = dropNestTargetIdRef.current;
    if (draggedId && (currentDropGroup || currentDropIndicator || currentNestTarget)) {
      performDrop(draggedId, currentDropGroup, currentDropIndicator, currentNestTarget);
    }
    setDraggedId(null);
    setDropIndicator(null);
    setDropGroupStatus(null);
    setDropNestTargetId(null);
  }, [draggedId, performDrop]);

  const [showSortMenu, setShowSortMenu] = useState(false);
  const sortBtnRef = useRef<HTMLButtonElement>(null);
  const sortMenuRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!showSortMenu) return;
    const handler = (e: MouseEvent) => {
      if (sortMenuRef.current && !sortMenuRef.current.contains(e.target as Node) &&
          sortBtnRef.current && !sortBtnRef.current.contains(e.target as Node)) {
        setShowSortMenu(false);
      }
    };
    document.addEventListener('mousedown', handler);
    return () => document.removeEventListener('mousedown', handler);
  }, [showSortMenu]);

  // Keyboard nav
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (isEditableTarget(e)) return;
      if (e.metaKey || e.ctrlKey) return;

      if (e.key === 'ArrowDown' || e.key === 'j') {
        e.preventDefault();
        setFocusedIndex(i => Math.min(i + 1, rows.length - 1));
        return;
      }
      if (e.key === 'ArrowUp' || e.key === 'k') {
        e.preventDefault();
        setFocusedIndex(i => Math.max(i - 1, 0));
        return;
      }

      const row = rows[focusedIndex];
      if (!row) return;

      if (e.key === 'Enter') {
        e.preventDefault();
        if (row.kind === 'group' && !row.isEmpty) toggleGroup(row.status);
        else if (row.kind === 'issue') onIssueClick?.(row.issue);
        return;
      }
      if (e.key === 'ArrowRight') {
        e.preventDefault();
        if (row.kind === 'group' && !row.isEmpty && !expandedGroups.has(row.status)) toggleGroup(row.status);
        else if (row.kind === 'issue' && row.hasChildren && !expandedNodes.has(row.issue.id)) toggleNode(row.issue.id);
        return;
      }
      if (e.key === 'ArrowLeft') {
        e.preventDefault();
        if (row.kind === 'group' && expandedGroups.has(row.status)) toggleGroup(row.status);
        else if (row.kind === 'issue' && row.hasChildren && expandedNodes.has(row.issue.id)) toggleNode(row.issue.id);
        return;
      }
      if (e.key === '.' && row.kind === 'issue') {
        navigator.clipboard.writeText(row.issue.id);
        setToast("Copied issue ID");
        setTimeout(() => setToast(null), 1500);
        return;
      }
    };
    document.addEventListener('keydown', handler);
    return () => document.removeEventListener('keydown', handler);
  }, [rows, focusedIndex, expandedGroups, expandedNodes, toggleGroup, toggleNode, onIssueClick]);

  // Scroll focused row into view
  useEffect(() => {
    const el = listRef.current?.querySelector(`[data-row="${focusedIndex}"]`);
    el?.scrollIntoView({ block: 'nearest' });
  }, [focusedIndex]);

  return (
    <div className="h-full flex flex-col relative">
      {/* Toast */}
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
            className={`
              text-sm font-medium h-full border-b-[1.5px] -mb-px transition-colors duration-[var(--duration-fast)]
              ${activeTab === id
                ? 'text-[var(--color-text-primary)] border-[var(--color-text-primary)]'
                : 'text-[var(--color-text-muted)] border-transparent hover:text-[var(--color-text-secondary)]'
              }
            `.trim().replace(/\s+/g, ' ')}
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
                onKeyDown={(e) => { if (e.key === 'Escape') { setSearch(''); onSearchBlur?.(); } }}
                placeholder="Filter issues..."
                className="bg-transparent text-sm text-[var(--color-text-primary)] placeholder:text-[var(--color-text-muted)] outline-none w-36"
              />
            </div>
          )}
          <div className="relative">
            <button
              ref={sortBtnRef}
              onClick={() => setShowSortMenu(v => !v)}
              className="flex items-center gap-1 h-6 px-1.5 rounded-[var(--radius-sm)] text-xs text-[var(--color-text-muted)] hover:text-[var(--color-text-secondary)] hover:bg-[var(--color-bg-hover)] transition-colors"
            >
              <svg className="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M3 7h6M3 12h10M3 17h14" />
              </svg>
              {SORT_OPTIONS.find(o => o.value === sortKey)?.label}
            </button>
            {showSortMenu && (
              <div ref={sortMenuRef} className="absolute right-0 top-full mt-1 z-50 min-w-[140px] bg-[var(--color-bg-elevated)] border border-[var(--color-border-default)] rounded-[var(--radius-md)] shadow-[var(--shadow-popover)] py-1">
                {SORT_OPTIONS.map(opt => (
                  <button
                    key={opt.value}
                    onClick={() => { onSortChange(opt.value); setShowSortMenu(false); }}
                    className={`flex items-center gap-2 w-full h-7 px-3 text-sm transition-colors ${
                      opt.value === sortKey
                        ? 'text-[var(--color-text-primary)] bg-[var(--color-bg-hover)]'
                        : 'text-[var(--color-text-secondary)] hover:bg-[var(--color-bg-hover)]'
                    }`}
                  >
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
          <span className="text-xs text-[var(--color-text-muted)] tabular-nums">
            {filteredIssues.length} issue{filteredIssues.length !== 1 ? 's' : ''}
          </span>
        </div>
      </div>

      {/* Rows */}
      <div ref={listRef} className="flex-1 overflow-y-auto">
        {(() => {
          const sections: { groupRow: RowItem & { kind: 'group' }; groupIndex: number; issueRows: { row: RowItem & { kind: 'issue' }; index: number }[] }[] = [];
          for (let i = 0; i < rows.length; i++) {
            const row = rows[i];
            if (row.kind === 'group') {
              sections.push({ groupRow: row, groupIndex: i, issueRows: [] });
            } else if (sections.length > 0) {
              sections[sections.length - 1].issueRows.push({ row, index: i });
            }
          }
          return sections.map(({ groupRow, groupIndex, issueRows }) => {
            const isFocused = groupIndex === focusedIndex;
            const isExpanded = expandedGroups.has(groupRow.status);
            const isInlineActive = inlineCreateStatus === groupRow.status;
            const isDropGroup = dropGroupStatus === groupRow.status;
            const handleGroupDragOver = isDndEnabled && draggedId ? (e: React.DragEvent) => {
              e.preventDefault();
              const draggedStatus = issues.find(ii => ii.id === draggedId)?.status;
              const isCrossGroup = draggedStatus !== groupRow.status;
              if (isCrossGroup && !e.metaKey) {
                setDropIndicator(null);
                setDropGroupStatus(groupRow.status);
                return;
              }
              setDropGroupStatus(null);
              const firstIssueIdx = rows.findIndex((r, j) => j > groupIndex && r.kind === 'issue' && r.depth === 0);
              if (firstIssueIdx !== -1) setDropIndicator({ rowIndex: firstIssueIdx, position: 'above' });
            } : undefined;
            return (
              <div
                key={`g-${groupRow.status}`}
                className={`${isDropGroup ? 'ring-2 ring-inset ring-[var(--color-accent-primary)] bg-[var(--color-accent-primary)]/5' : ''}`}
              >
                <div
                  data-row={groupIndex}
                  onClick={() => !groupRow.isEmpty && toggleGroup(groupRow.status)}
                  onMouseEnter={() => setFocusedIndex(groupIndex)}
                  onDragOver={handleGroupDragOver}
                  onDrop={handleGroupDragOver ? handleDrop : undefined}
                  className={`
                    flex items-center gap-2 w-full px-5 py-2 border-b border-[var(--color-border-subtle)]
                    transition-colors duration-[var(--duration-fast)] select-none
                    ${groupRow.isEmpty ? 'opacity-40 cursor-default' : 'cursor-pointer'}
                    ${!isDropGroup && isFocused ? 'bg-[var(--color-bg-hover)]' : !isDropGroup ? 'bg-[var(--color-bg-secondary)]' : ''}
                  `.trim().replace(/\s+/g, ' ')}
                >
                  <svg
                    className={`w-3 h-3 text-[var(--color-text-muted)] transition-transform duration-100 ${isExpanded && !groupRow.isEmpty ? 'rotate-90' : ''}`}
                    fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}
                  >
                    <path strokeLinecap="round" strokeLinejoin="round" d="M9 5l7 7-7 7" />
                  </svg>
                  <StatusIcon status={groupRow.status} size={14} />
                  <span className="text-sm font-medium text-[var(--color-text-primary)]">{groupRow.label}</span>
                  <span className="text-sm text-[var(--color-text-muted)] tabular-nums">{groupRow.count}</span>
                  <button
                    onClick={(e) => { e.stopPropagation(); startInlineCreate(groupRow.status); }}
                    className="ml-auto p-0.5 rounded-[var(--radius-sm)] text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-bg-hover)] transition-colors"
                    title={`New ${groupRow.label} issue`}
                  >
                    <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                      <path strokeLinecap="round" strokeLinejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
                    </svg>
                  </button>
                </div>
                {isInlineActive && (
                  <div className="flex items-center gap-2.5 px-5 h-[38px] border-b border-[var(--color-border-subtle)] bg-[var(--color-bg-tertiary)]">
                    <StatusIcon status={groupRow.status} size={14} className="shrink-0 ml-7" />
                    <input
                      ref={inlineRef}
                      value={inlineTitle}
                      onChange={(e) => setInlineTitle(e.target.value)}
                      onKeyDown={(e) => {
                        if (e.key === 'Enter' && inlineTitle.trim()) {
                          handleInlineCreate(groupRow.status, inlineTitle);
                        }
                        if (e.key === 'Escape') {
                          setInlineCreateStatus(null);
                          setInlineTitle('');
                        }
                      }}
                      onBlur={() => { if (!inlineTitle.trim()) { setInlineCreateStatus(null); setInlineTitle(''); } }}
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
                  const isDropTarget = isDndEnabled && !!draggedId;
                  const showDropAbove = dropIndicator?.rowIndex === i && dropIndicator.position === 'above';
                  const showDropBelow = dropIndicator?.rowIndex === i && dropIndicator.position === 'below';
                  const isNestTarget = dropNestTargetId === issue.id;
                  const isDraggedOrBatch = draggedId !== null && dragGroupRef.current.includes(issue.id);
                  const dragBatchCount = draggedId === issue.id ? dragGroupRef.current.length : 0;
                  return (
                    <div key={issue.id} className="relative">
                      {showDropAbove && <div className="absolute top-0 right-5 h-[2px] bg-[var(--color-accent-primary)] z-10 rounded-full" style={{ left: `${20 + depth * 24}px` }} />}
                      <div
                        data-row={i}
                        draggable={canDrag}
                        onDragStart={canDrag ? (e) => handleDragStart(e, issue.id) : undefined}
                        onDragEnd={canDrag ? handleDragEnd : undefined}
                        onDragOver={isDropTarget ? (e) => handleDragOver(e, i) : undefined}
                        onDrop={isDropTarget ? handleDrop : undefined}
                        onClick={() => onIssueClick?.(issue)}
                        onMouseEnter={() => setFocusedIndex(i)}
                        className={`flex items-center gap-2.5 px-5 h-[38px] border-b border-[var(--color-border-subtle)] cursor-pointer transition-colors duration-[var(--duration-fast)] group ${isGhostParent ? 'opacity-50' : ''} ${isNestTarget ? 'ring-2 ring-inset ring-[var(--color-accent-primary)] bg-[var(--color-accent-primary)]/10' : isRowFocused ? 'bg-[var(--color-bg-hover)]' : 'hover:bg-[var(--color-bg-hover)]'} ${isDraggedOrBatch ? 'opacity-40' : ''}`}
                        style={{ paddingLeft: `${20 + indent}px` }}
                      >
                      {hasChildren ? (
                        <button
                          onClick={(e) => { e.stopPropagation(); toggleNode(issue.id); }}
                          className="w-4 shrink-0 flex items-center justify-center text-[var(--color-text-muted)] hover:text-[var(--color-text-secondary)]"
                        >
                          <svg
                            className={`w-3 h-3 transition-transform duration-100 ${isNodeExpanded ? 'rotate-90' : ''}`}
                            fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}
                          >
                            <path strokeLinecap="round" strokeLinejoin="round" d="M9 5l7 7-7 7" />
                          </svg>
                        </button>
                      ) : depth > 0 ? (
                        <span className={`w-4 shrink-0 flex items-center justify-center text-[var(--color-border-default)] ${canDrag ? 'cursor-grab active:cursor-grabbing' : ''}`}>
                          <svg width="12" height="16" viewBox="0 0 12 16" fill="none" stroke="currentColor" strokeWidth="1.5">
                            <path d="M1 0v9h10" />
                          </svg>
                        </span>
                      ) : (
                        <span className={`text-[var(--color-text-muted)] transition-opacity w-4 shrink-0 flex items-center justify-center ${canDrag ? 'opacity-30 cursor-grab active:cursor-grabbing' : 'opacity-0 group-hover:opacity-30'}`}>
                          <svg width="6" height="10" viewBox="0 0 6 10" fill="currentColor">
                            <circle cx="1" cy="1" r="1" /><circle cx="5" cy="1" r="1" />
                            <circle cx="1" cy="5" r="1" /><circle cx="5" cy="5" r="1" />
                            <circle cx="1" cy="9" r="1" /><circle cx="5" cy="9" r="1" />
                          </svg>
                        </span>
                      )}

                      <CopyableId id={issue.id} className="text-xs w-[110px] shrink-0 truncate tabular-nums" />
                      <div className="relative shrink-0" onClick={(e) => e.stopPropagation()}>
                        <button onClick={() => setOpenPopover(openPopover?.issueId === issue.id && openPopover?.type === 'status' ? null : { issueId: issue.id, type: 'status' })} className="hover:opacity-70 transition-opacity">
                          <StatusIcon status={issue.status} size={14} />
                        </button>
                        {openPopover?.issueId === issue.id && openPopover?.type === 'status' && (
                          <Popover onClose={() => setOpenPopover(null)}>
                            <PopoverHeader>Set status...</PopoverHeader>
                            {STATUS_OPTIONS.map((opt) => (
                              <button key={opt.value} onClick={() => handleQuickStatus(issue.id, opt.value)} className={`flex items-center gap-2 w-full px-3 py-1.5 text-sm transition-colors hover:bg-[var(--color-bg-hover)] ${opt.value === issue.status ? 'text-[var(--color-accent-primary)]' : 'text-[var(--color-text-primary)]'}`}>
                                <StatusIcon status={opt.value} size={14} />
                                <span>{opt.label}</span>
                                {opt.value === issue.status && <svg className="w-3.5 h-3.5 ml-auto" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}><path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" /></svg>}
                              </button>
                            ))}
                          </Popover>
                        )}
                      </div>
                      {parentBreadcrumb && (
                        <span className="text-sm text-[var(--color-text-muted)] truncate shrink-0 max-w-[150px]">{parentBreadcrumb}</span>
                      )}
                      {parentBreadcrumb && (
                        <svg className="w-3 h-3 text-[var(--color-text-muted)] shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                          <path strokeLinecap="round" strokeLinejoin="round" d="M9 5l7 7-7 7" />
                        </svg>
                      )}
                      <span className={`text-sm font-medium truncate min-w-0 ${isGhostParent ? 'text-[var(--color-text-muted)]' : 'text-[var(--color-text-primary)]'}`}>{issue.title}</span>
                      {dragBatchCount > 1 && (
                        <span className="flex items-center justify-center w-5 h-5 rounded-full bg-[var(--color-accent-primary)] text-white text-xs font-medium shrink-0">{dragBatchCount}</span>
                      )}

                      {hasChildren && (
                        <span className="flex items-center gap-1.5 text-xs text-[var(--color-text-muted)] shrink-0">
                          <SubProgress done={childDone} total={childTotal} />
                          {childDone}/{childTotal}
                        </span>
                      )}
                      <div className="flex-1" />
                      {issue.is_pending && <span className="w-1.5 h-1.5 rounded-full bg-[var(--color-warning)] shrink-0" />}
                      {issue.priority > 0 && (
                        <span className={`text-xs font-medium shrink-0 ${issue.priority === 1 ? 'text-[var(--color-error)]' : issue.priority === 2 ? 'text-[var(--color-warning)]' : 'text-[var(--color-text-muted)]'}`}>
                          {issue.priority === 1 ? '!!!' : issue.priority === 2 ? '!!' : issue.priority === 3 ? '!' : ''}
                        </span>
                      )}
                      <div className="relative flex items-center gap-2.5 shrink-0" onClick={(e) => e.stopPropagation()}>
                        <button onClick={() => setOpenPopover(openPopover?.issueId === issue.id && openPopover?.type === 'labels' ? null : { issueId: issue.id, type: 'labels' })} className="flex items-center gap-2.5 hover:opacity-70 transition-opacity">
                          {issue.labels?.map((label) => <LabelBadge key={label} label={label} />)}
                          {(!issue.labels || issue.labels.length === 0) && <span className="text-xs text-[var(--color-text-muted)] opacity-0 group-hover:opacity-100 transition-opacity">+ label</span>}
                        </button>
                        {openPopover?.issueId === issue.id && openPopover?.type === 'labels' && (
                          <Popover onClose={() => setOpenPopover(null)}>
                            <PopoverHeader>Toggle labels...</PopoverHeader>
                            {allKnownLabels.map((label) => {
                              const isActive = (issue.labels || []).includes(label);
                              return (
                                <button key={label} onClick={() => handleQuickLabelToggle(issue, label)} className="flex items-center gap-2 w-full px-3 py-1.5 text-sm text-[var(--color-text-primary)] transition-colors hover:bg-[var(--color-bg-hover)]">
                                  <span className={`w-3.5 h-3.5 rounded-[3px] border flex items-center justify-center shrink-0 ${isActive ? 'bg-[var(--color-accent-primary)] border-[var(--color-accent-primary)]' : 'border-[var(--color-border-default)]'}`}>
                                    {isActive && <svg className="w-2.5 h-2.5 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={3}><path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" /></svg>}
                                  </span>
                                  <LabelBadge label={label} />
                                </button>
                              );
                            })}
                          </Popover>
                        )}
                      </div>
                      <div className="relative shrink-0" onClick={(e) => e.stopPropagation()}>
                        <button onClick={() => setOpenPopover(openPopover?.issueId === issue.id && openPopover?.type === 'estimate' ? null : { issueId: issue.id, type: 'estimate' })} className="flex items-center gap-1 text-xs text-[var(--color-text-muted)] tabular-nums w-10 justify-end hover:opacity-70 transition-opacity">
                          {issue.estimate > 0 ? (<>
                            <svg className="w-3 h-3" viewBox="0 0 16 16" fill="none"><path d="M8 2L14 14H2L8 2Z" stroke="currentColor" strokeWidth="1.5" strokeLinejoin="round" /></svg>
                            {issue.estimate}
                          </>) : (
                            <span className="opacity-0 group-hover:opacity-100 transition-opacity">
                              <svg className="w-3 h-3" viewBox="0 0 16 16" fill="none"><path d="M8 2L14 14H2L8 2Z" stroke="currentColor" strokeWidth="1.5" strokeLinejoin="round" /></svg>
                            </span>
                          )}
                        </button>
                        {openPopover?.issueId === issue.id && openPopover?.type === 'estimate' && (
                          <Popover onClose={() => setOpenPopover(null)}>
                            <PopoverHeader>Set estimate...</PopoverHeader>
                            {ESTIMATE_OPTIONS.map((est) => (
                              <button key={est} onClick={() => handleQuickEstimate(issue.id, est)} className={`flex items-center gap-2 w-full px-3 py-1.5 text-sm transition-colors hover:bg-[var(--color-bg-hover)] ${est === (issue.estimate || 0) ? 'text-[var(--color-accent-primary)]' : 'text-[var(--color-text-primary)]'}`}>
                                <span>{est === 0 ? 'No estimate' : `${est} Point${est !== 1 ? 's' : ''}`}</span>
                                {est === (issue.estimate || 0) && <svg className="w-3.5 h-3.5 ml-auto" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}><path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" /></svg>}
                              </button>
                            ))}
                          </Popover>
                        )}
                      </div>
                      <span className="text-xs text-[var(--color-text-muted)] tabular-nums shrink-0 w-16 text-right">{formatShortDate(issue.created_at)}</span>
                      </div>
                      {showDropBelow && <div className="absolute bottom-0 right-5 h-[2px] bg-[var(--color-accent-primary)] z-10 rounded-full" style={{ left: `${20 + depth * 24}px` }} />}
                    </div>
                  );
                })}
              </div>
            );
          });
        })()}
      </div>
    </div>
  );
}

function formatShortDate(dateStr: string): string {
  const d = new Date(dateStr);
  const months = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];
  return `${months[d.getMonth()]} ${d.getDate()}`;
}

function SubProgress({ done, total }: { done: number; total: number }) {
  const pct = total > 0 ? (done / total) * 100 : 0;
  return (
    <svg width="14" height="14" viewBox="0 0 16 16" className="shrink-0">
      <circle cx="8" cy="8" r="6" fill="none" stroke="var(--color-bg-tertiary)" strokeWidth="2" />
      <circle
        cx="8" cy="8" r="6" fill="none"
        stroke={pct === 100 ? 'var(--color-success)' : 'var(--color-accent-primary)'}
        strokeWidth="2"
        strokeLinecap="round"
        strokeDasharray={`${pct * 0.377} 100`}
        transform="rotate(-90 8 8)"
      />
    </svg>
  );
}
