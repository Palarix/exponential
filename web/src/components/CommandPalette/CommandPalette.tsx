import { useState, useEffect, useRef, useMemo } from 'react';
import { createPortal } from 'react-dom';
import type { Issue } from '../../api/client';
import { StatusIcon, LabelBadge } from '../ui';
import { Search, Link, Bell, Plus, List, Columns3, Clock, LayoutDashboard } from "lucide-react";

type View = 'dashboard' | 'inbox' | 'backlog' | 'board' | 'dependencies' | 'timeline';

interface CommandPaletteProps {
  isOpen: boolean;
  onClose: () => void;
  issues: Issue[];
  onIssueSelect: (issue: Issue) => void;
  onViewChange: (view: View) => void;
  onNewIssue: () => void;
}

interface CommandItem {
  id: string;
  group: string;
  icon: React.ReactNode;
  label: string;
  meta?: React.ReactNode;
  shortcut?: string;
  action: () => void;
}

const NAV_ICONS: Record<string, React.ReactNode> = {
  backlog: <List size={16} />,
  board: <Columns3 size={16} />,
  dashboard: <LayoutDashboard size={16} />,
  dependencies: <Link size={16} />,
  inbox: <Bell size={16} />,
  timeline: <Clock size={16} />,
};

function fuzzyMatch(text: string, query: string): boolean {
  let qi = 0;
  const tl = text.toLowerCase();
  const ql = query.toLowerCase();
  for (let ti = 0; ti < tl.length && qi < ql.length; ti++) {
    if (tl[ti] === ql[qi]) qi++;
  }
  return qi === ql.length;
}

export default function CommandPalette({ isOpen, onClose, issues, onIssueSelect, onViewChange, onNewIssue }: CommandPaletteProps) {
  const [query, setQuery] = useState('');
  const [selectedIndex, setSelectedIndex] = useState(0);
  const inputRef = useRef<HTMLInputElement>(null);
  const listRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (isOpen) {
      setQuery('');
      setSelectedIndex(0);
      setTimeout(() => inputRef.current?.focus(), 0);
    }
  }, [isOpen]);

  const items = useMemo(() => {
    const results: CommandItem[] = [];

    // Actions (always available)
    if (!query || fuzzyMatch('create new issue', query)) {
      results.push({
        id: 'action:new',
        group: 'Actions',
        icon: <Plus size={16} />,
        label: 'Create new issue',
        shortcut: 'C',
        action: () => { onClose(); onNewIssue(); },
      });
    }

    // Navigation
    const navItems: { id: View; label: string }[] = [
      { id: 'backlog', label: 'Issues' },
      { id: 'board', label: 'Board' },
      { id: 'dashboard', label: 'Dashboard' },
      { id: 'dependencies', label: 'Dependencies' },
      { id: 'inbox', label: 'Notifications' },
      { id: 'timeline', label: 'Timeline' },
    ];
    for (const nav of navItems) {
      if (!query || fuzzyMatch(nav.label, query)) {
        results.push({
          id: `nav:${nav.id}`,
          group: 'Navigation',
          icon: NAV_ICONS[nav.id],
          label: nav.label,
          action: () => { onClose(); onViewChange(nav.id); },
        });
      }
    }

    // Issues (only when there's a query, or show recent)
    const matchedIssues = query
      ? issues.filter(i => { const q = query.toLowerCase(); return i.title.toLowerCase().includes(q) || i.id.includes(q) || i.labels?.some(l => l.toLowerCase().includes(q)); })
      : issues.slice(0, 8);

    for (const issue of matchedIssues.slice(0, 12)) {
      results.push({
        id: `issue:${issue.id}`,
        group: query ? 'Issues' : 'Recent Issues',
        icon: <StatusIcon status={issue.status} size={14} isInferred={issue.is_inferred} />,
        label: issue.title,
        meta: (
          <span className="flex items-center gap-2">
            <span className="font-mono text-xs text-[var(--color-text-muted)]">{issue.id.replace('issue-', '')}</span>
            {issue.labels?.map(l => <LabelBadge key={l} label={l} />)}
          </span>
        ),
        action: () => { onClose(); onIssueSelect(issue); },
      });
    }

    return results;
  }, [query, issues, onClose, onIssueSelect, onViewChange, onNewIssue]);

  // Group items for display
  const groups = useMemo(() => {
    const map = new Map<string, CommandItem[]>();
    for (const item of items) {
      const list = map.get(item.group) || [];
      list.push(item);
      map.set(item.group, list);
    }
    return Array.from(map.entries());
  }, [items]);

  useEffect(() => {
    setSelectedIndex(0);
  }, [query]);

  // Scroll selected into view
  useEffect(() => {
    const el = listRef.current?.querySelector(`[data-index="${selectedIndex}"]`);
    el?.scrollIntoView({ block: 'nearest' });
  }, [selectedIndex]);

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'ArrowDown') {
      e.preventDefault();
      setSelectedIndex(i => Math.min(i + 1, items.length - 1));
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      setSelectedIndex(i => Math.max(i - 1, 0));
    } else if (e.key === 'Enter') {
      e.preventDefault();
      items[selectedIndex]?.action();
    } else if (e.key === 'Escape') {
      onClose();
    }
  };

  if (!isOpen) return null;

  let flatIndex = 0;

  return createPortal(
    <div className="fixed inset-0 z-[200] flex items-start justify-center pt-[15vh]">
      <div className="absolute inset-0 bg-black/50" onClick={onClose} />
      <div
        className="relative w-full max-w-[860px] bg-[var(--color-surface-3)] border border-[var(--color-border-default)] rounded-[var(--radius-xl)] shadow-[var(--shadow-lg)] overflow-hidden animate-fade-in"
        onKeyDown={handleKeyDown}
      >
        {/* Search input */}
        <div className="flex items-center gap-3 px-4 h-12 border-b border-[var(--color-border-subtle)]">
          <Search className="w-4 h-4 text-[var(--color-text-muted)] shrink-0" />
          <input
            ref={inputRef}
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Search issues, actions, navigation..."
            className="flex-1 bg-transparent text-base text-[var(--color-text-primary)] placeholder:text-[var(--color-text-muted)] outline-none"
          />
          <kbd className="text-xs text-[var(--color-text-muted)] bg-[var(--color-surface-1)] px-2 py-1 rounded-[var(--radius-sm)]">ESC</kbd>
        </div>

        {/* Results */}
        <div ref={listRef} className="max-h-100 overflow-y-auto py-1">
          {items.length === 0 ? (
            <div className="px-4 py-8 text-center text-sm text-[var(--color-text-muted)]">
              No results for "{query}"
            </div>
          ) : (
            groups.map(([group, groupItems]) => (
              <div key={group}>
                <div className="px-4 pt-2 pb-1 text-xs font-medium text-[var(--color-text-muted)] uppercase tracking-wider">
                  {group}
                </div>
                {groupItems.map((item) => {
                  const idx = flatIndex++;
                  const isSelected = idx === selectedIndex;
                  return (
                    <button
                      key={item.id}
                      data-index={idx}
                      onClick={item.action}
                      onMouseEnter={() => setSelectedIndex(idx)}
                      className={`flex items-center gap-3 w-full px-4 h-10 text-left transition-colors ${isSelected ? 'bg-[var(--color-hover-surface-3)]' : ''}`}
                    >
                      <span className="text-[var(--color-text-muted)] shrink-0">{item.icon}</span>
                      <span className="text-sm text-[var(--color-text-primary)] truncate flex-1">{item.label}</span>
                      {item.meta}
                      {item.shortcut && (
                        <kbd className="text-xs text-[var(--color-text-muted)] bg-[var(--color-surface-1)] px-2 py-1 rounded-[var(--radius-sm)] ml-auto shrink-0">
                          {item.shortcut}
                        </kbd>
                      )}
                    </button>
                  );
                })}
              </div>
            ))
          )}
        </div>

        {/* Footer hints */}
        <div className="flex items-center gap-4 px-4 h-8 border-t border-[var(--color-border-subtle)] text-xs text-[var(--color-text-muted)]">
          <span className="flex items-center gap-1">
            <kbd className="bg-[var(--color-surface-1)] px-1 rounded">↑↓</kbd> navigate
          </span>
          <span className="flex items-center gap-1">
            <kbd className="bg-[var(--color-surface-1)] px-1 rounded">↵</kbd> select
          </span>
          <span className="flex items-center gap-1">
            <kbd className="bg-[var(--color-surface-1)] px-1 rounded">esc</kbd> close
          </span>
        </div>
      </div>
    </div>,
    document.body
  );
}
