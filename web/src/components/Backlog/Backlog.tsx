import { useState, useRef, useEffect, useCallback, useMemo } from 'react';
import { generateKeyBetween } from 'fractional-indexing';
import { createIssue, addDraft } from '../../api/client';
import type { Issue } from '../../api/client';
import { LabelBadge, StatusIcon, CopyableId } from '../ui';
import { sortGroup, getEffectiveKeys, SORT_OPTIONS } from '../../utils/sort';
import type { SortKey } from '../../utils/sort';
import { isEditableTarget } from '../../utils/keyboard';

interface BacklogProps {
  issues: Issue[];
  onRefresh: () => void;
  onIssueClick?: (issue: Issue) => void;
  searchFocused?: boolean;
  onSearchBlur?: () => void;
  sortKey: SortKey;
  onSortChange: (key: SortKey) => void;
  onNavigationOrderChange?: (ids: string[]) => void;
}

type Tab = 'all' | 'active' | 'backlog';

const TAB_CONFIGS: Record<Tab, { label: string; statuses: string[] }> = {
  all: { label: 'All Issues', statuses: ['BACKLOG', 'PLANNED', 'DOING', 'BLOCKED', 'DONE'] },
  active: { label: 'Active', statuses: ['DOING', 'BLOCKED'] },
  backlog: { label: 'Backlog', statuses: ['BACKLOG', 'PLANNED'] },
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
  | { kind: 'issue'; issue: Issue; depth: number; hasChildren: boolean; childDone: number; childTotal: number };

export default function Backlog({ issues, onRefresh, onIssueClick, searchFocused, onSearchBlur, sortKey, onSortChange, onNavigationOrderChange }: BacklogProps) {
  const [activeTab, setActiveTab] = useState<Tab>('all');
  const [search, setSearch] = useState('');
  const [expandedGroups, setExpandedGroups] = useState<Set<string>>(() => new Set());
  const [expandedNodes, setExpandedNodes] = useState<Set<string>>(() => new Set());
  const [focusedIndex, setFocusedIndex] = useState(-1);
  const [inlineCreateStatus, setInlineCreateStatus] = useState<string | null>(null);
  const [inlineTitle, setInlineTitle] = useState('');
  const [toast, setToast] = useState<string | null>(null);
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
    for (const status of visibleStatuses) {
      const groupIssues = filteredIssues.filter(i => i.status === status);
      if (groupIssues.length === 0 && activeTab !== 'all') continue;

      result.push({ kind: 'group', status, label: STATUS_META[status]?.label || status, count: groupIssues.length, isEmpty: groupIssues.length === 0 });

      if (expandedGroups.has(status) && groupIssues.length > 0) {
        const topLevel = sortGroup(
          groupIssues.filter(i => !i.parent_id || !issues.some(p => p.id === i.parent_id)),
          sortKey
        );
        const addTree = (issue: Issue, depth: number) => {
          const children = childrenByParent.get(issue.id) || [];
          const doneCount = children.filter(c => c.status === 'DONE').length;
          result.push({ kind: 'issue', issue, depth, hasChildren: children.length > 0, childDone: doneCount, childTotal: children.length });
          if (children.length > 0 && expandedNodes.has(issue.id)) {
            for (const child of children) addTree(child, depth + 1);
          }
        };
        for (const issue of topLevel) addTree(issue, 0);
      }
    }
    return result;
  }, [visibleStatuses, filteredIssues, activeTab, expandedGroups, expandedNodes, issues, childrenByParent, sortKey]);

  // Report navigation order to parent
  useEffect(() => {
    const ids = rows.filter(r => r.kind === 'issue').map(r => (r as { kind: 'issue'; issue: Issue }).issue.id);
    onNavigationOrderChange?.(ids);
  }, [rows, onNavigationOrderChange]);

  // Drag-and-drop for manual reordering
  const isDndEnabled = sortKey === 'manual';
  const [draggedId, setDraggedId] = useState<string | null>(null);
  const [dropIndicator, setDropIndicator] = useState<{ rowIndex: number; position: 'above' | 'below' } | null>(null);

  const getRowStatusGroup = useCallback((rowIndex: number): string | null => {
    for (let j = rowIndex; j >= 0; j--) {
      const r = rows[j];
      if (r.kind === 'group') return r.status;
    }
    return null;
  }, [rows]);

  const handleDragStart = useCallback((e: React.DragEvent, issueId: string) => {
    setDraggedId(issueId);
    e.dataTransfer.effectAllowed = 'move';
    e.dataTransfer.setData('text/plain', issueId);
    if (e.currentTarget instanceof HTMLElement) {
      e.currentTarget.style.opacity = '0.4';
    }
  }, []);

  const handleDragEnd = useCallback((e: React.DragEvent) => {
    if (e.currentTarget instanceof HTMLElement) {
      e.currentTarget.style.opacity = '';
    }
    setDraggedId(null);
    setDropIndicator(null);
  }, []);

  const handleDragOver = useCallback((e: React.DragEvent, rowIndex: number) => {
    e.preventDefault();
    if (!draggedId) return;

    const row = rows[rowIndex];
    if (row.kind !== 'issue' || row.depth > 0) return;
    if (row.issue.id === draggedId) { setDropIndicator(null); return; }

    const draggedStatus = issues.find(i => i.id === draggedId)?.status;
    const targetStatus = getRowStatusGroup(rowIndex);
    if (draggedStatus !== targetStatus) { setDropIndicator(null); return; }

    const rect = (e.currentTarget as HTMLElement).getBoundingClientRect();
    const position = e.clientY < rect.top + rect.height / 2 ? 'above' : 'below';
    setDropIndicator({ rowIndex, position });
  }, [draggedId, rows, issues, getRowStatusGroup]);

  const handleDrop = useCallback(async (e: React.DragEvent) => {
    e.preventDefault();
    if (!draggedId || !dropIndicator) { setDraggedId(null); setDropIndicator(null); return; }

    const targetRow = rows[dropIndicator.rowIndex];
    if (targetRow.kind !== 'issue') { setDraggedId(null); setDropIndicator(null); return; }

    const status = getRowStatusGroup(dropIndicator.rowIndex);
    const groupTopLevel = rows
      .filter((r): r is typeof r & { kind: 'issue' } => r.kind === 'issue' && r.depth === 0)
      .filter(r => {
        const idx = rows.indexOf(r);
        return getRowStatusGroup(idx) === status;
      })
      .map(r => r.issue);

    const effectiveKeys = getEffectiveKeys(groupTopLevel);
    const withoutDragged = groupTopLevel.filter(i => i.id !== draggedId);
    let insertIdx = withoutDragged.findIndex(i => i.id === targetRow.issue.id);
    if (insertIdx === -1) insertIdx = withoutDragged.length;
    if (dropIndicator.position === 'below') insertIdx++;

    const prev = withoutDragged[insertIdx - 1];
    const next = withoutDragged[insertIdx];
    const prevKey = prev ? (effectiveKeys.get(prev.id) || null) : null;
    const nextKey = next ? (effectiveKeys.get(next.id) || null) : null;
    const newKey = generateKeyBetween(prevKey, nextKey);

    await addDraft(draggedId, 'UPDATE', { sort_order: newKey });

    setDraggedId(null);
    setDropIndicator(null);
    onRefresh();
  }, [draggedId, dropIndicator, rows, getRowStatusGroup, onRefresh]);

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
            onClick={() => setActiveTab(id)}
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
        {rows.map((row, i) => {
          const isFocused = i === focusedIndex;
          if (row.kind === 'group') {
            const isExpanded = expandedGroups.has(row.status);
            const isInlineActive = inlineCreateStatus === row.status;
            return (
              <div key={`g-${row.status}`}>
                <div
                  data-row={i}
                  onClick={() => !row.isEmpty && toggleGroup(row.status)}
                  onMouseEnter={() => setFocusedIndex(i)}
                  className={`
                    flex items-center gap-2 w-full px-5 py-2 border-b border-[var(--color-border-subtle)]
                    transition-colors duration-[var(--duration-fast)] select-none
                    ${row.isEmpty ? 'opacity-40 cursor-default' : 'cursor-pointer'}
                    ${isFocused ? 'bg-[var(--color-bg-hover)]' : 'bg-[var(--color-bg-secondary)]'}
                  `.trim().replace(/\s+/g, ' ')}
                >
                  <svg
                    className={`w-3 h-3 text-[var(--color-text-muted)] transition-transform duration-100 ${isExpanded && !row.isEmpty ? 'rotate-90' : ''}`}
                    fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}
                  >
                    <path strokeLinecap="round" strokeLinejoin="round" d="M9 5l7 7-7 7" />
                  </svg>
                  <StatusIcon status={row.status} size={14} />
                  <span className="text-sm font-medium text-[var(--color-text-primary)]">{row.label}</span>
                  <span className="text-sm text-[var(--color-text-muted)] tabular-nums">{row.count}</span>
                  <button
                    onClick={(e) => { e.stopPropagation(); startInlineCreate(row.status); }}
                    className="ml-auto p-0.5 rounded-[var(--radius-sm)] text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-bg-hover)] transition-colors"
                    title={`New ${row.label} issue`}
                  >
                    <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                      <path strokeLinecap="round" strokeLinejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
                    </svg>
                  </button>
                </div>
                {isInlineActive && (
                  <div className="flex items-center gap-2.5 px-5 h-[38px] border-b border-[var(--color-border-subtle)] bg-[var(--color-bg-tertiary)]">
                    <StatusIcon status={row.status} size={14} className="shrink-0 ml-7" />
                    <input
                      ref={inlineRef}
                      value={inlineTitle}
                      onChange={(e) => setInlineTitle(e.target.value)}
                      onKeyDown={(e) => {
                        if (e.key === 'Enter' && inlineTitle.trim()) {
                          handleInlineCreate(row.status, inlineTitle);
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
              </div>
            );
          }

          const { issue, depth, hasChildren, childDone, childTotal } = row;
          const isNodeExpanded = expandedNodes.has(issue.id);
          const indent = depth * 24;
          const canDrag = isDndEnabled && depth === 0;
          const showDropAbove = dropIndicator?.rowIndex === i && dropIndicator.position === 'above';
          const showDropBelow = dropIndicator?.rowIndex === i && dropIndicator.position === 'below';
          return (
            <div key={issue.id} className="relative">
              {showDropAbove && <div className="absolute top-0 left-5 right-5 h-[2px] bg-[var(--color-accent-primary)] z-10 rounded-full" />}
              <div
                data-row={i}
                draggable={canDrag}
                onDragStart={canDrag ? (e) => handleDragStart(e, issue.id) : undefined}
                onDragEnd={canDrag ? handleDragEnd : undefined}
                onDragOver={canDrag ? (e) => handleDragOver(e, i) : undefined}
                onDrop={canDrag ? handleDrop : undefined}
                onClick={() => onIssueClick?.(issue)}
                onMouseEnter={() => setFocusedIndex(i)}
                className={`flex items-center gap-2.5 px-5 h-[38px] border-b border-[var(--color-border-subtle)] cursor-pointer transition-colors duration-[var(--duration-fast)] group ${isFocused ? 'bg-[var(--color-bg-hover)]' : 'hover:bg-[var(--color-bg-hover)]'} ${draggedId === issue.id ? 'opacity-40' : ''}`}
                style={{ paddingLeft: `${20 + indent}px` }}
              >
              {/* Tree toggle or connector */}
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
                <span className="w-4 shrink-0 flex items-center justify-center text-[var(--color-border-default)]">
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

              <CopyableId id={issue.id} className="text-xs w-[88px] shrink-0 truncate tabular-nums" />
              <StatusIcon status={issue.status} size={14} className="shrink-0" />
              <span className="text-sm font-medium text-[var(--color-text-primary)] truncate flex-1 min-w-0">{issue.title}</span>

              {hasChildren && (
                <span className="flex items-center gap-1.5 text-xs text-[var(--color-text-muted)] shrink-0">
                  <SubProgress done={childDone} total={childTotal} />
                  {childDone}/{childTotal}
                </span>
              )}
              {issue.is_pending && <span className="w-1.5 h-1.5 rounded-full bg-[var(--color-warning)] shrink-0" />}
              {issue.priority > 0 && (
                <span className={`text-xs font-medium shrink-0 ${issue.priority === 1 ? 'text-[var(--color-error)]' : issue.priority === 2 ? 'text-[var(--color-warning)]' : 'text-[var(--color-text-muted)]'}`}>
                  {issue.priority === 1 ? '!!!' : issue.priority === 2 ? '!!' : issue.priority === 3 ? '!' : ''}
                </span>
              )}
              <div className="flex items-center gap-2.5 shrink-0">
                {issue.labels?.map((label) => <LabelBadge key={label} label={label} />)}
              </div>
              {issue.estimate > 0 && <span className="text-xs text-[var(--color-text-muted)] tabular-nums shrink-0">{issue.estimate}</span>}
              <span className="text-xs text-[var(--color-text-muted)] tabular-nums shrink-0 w-12 text-right">{formatShortDate(issue.created_at)}</span>
              </div>
              {showDropBelow && <div className="absolute bottom-0 left-5 right-5 h-[2px] bg-[var(--color-accent-primary)] z-10 rounded-full" />}
            </div>
          );
        })}
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
