import { useState, useEffect, useRef, useMemo } from 'react';
import { createPortal } from 'react-dom';
import type { Issue } from '../../api/client';
import { StatusIcon, LabelBadge } from '../ui';

type View = 'dashboard' | 'backlog' | 'board' | 'dependencies';

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
  backlog: <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}><path strokeLinecap="round" strokeLinejoin="round" d="M8.25 6.75h12M8.25 12h12m-12 5.25h12M3.75 6.75h.007v.008H3.75V6.75zm.375 0a.375.375 0 11-.75 0 .375.375 0 01.75 0zM3.75 12h.007v.008H3.75V12zm.375 0a.375.375 0 11-.75 0 .375.375 0 01.75 0zm-.375 5.25h.007v.008H3.75v-.008zm.375 0a.375.375 0 11-.75 0 .375.375 0 01.75 0z" /></svg>,
  board: <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}><path strokeLinecap="round" strokeLinejoin="round" d="M9 4.5v15m6-15v15m-10.875 0h15.75c.621 0 1.125-.504 1.125-1.125V5.625c0-.621-.504-1.125-1.125-1.125H4.125C3.504 4.5 3 5.004 3 5.625v12.75c0 .621.504 1.125 1.125 1.125z" /></svg>,
  dashboard: <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}><path strokeLinecap="round" strokeLinejoin="round" d="M3 13.125C3 12.504 3.504 12 4.125 12h2.25c.621 0 1.125.504 1.125 1.125v6.75C7.5 20.496 6.996 21 6.375 21h-2.25A1.125 1.125 0 013 19.875v-6.75zM9.75 8.625c0-.621.504-1.125 1.125-1.125h2.25c.621 0 1.125.504 1.125 1.125v11.25c0 .621-.504 1.125-1.125 1.125h-2.25a1.125 1.125 0 01-1.125-1.125V8.625zM16.5 4.125c0-.621.504-1.125 1.125-1.125h2.25C20.496 3 21 3.504 21 4.125v15.75c0 .621-.504 1.125-1.125 1.125h-2.25a1.125 1.125 0 01-1.125-1.125V4.125z" /></svg>,
  dependencies: <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}><path strokeLinecap="round" strokeLinejoin="round" d="M13.19 8.688a4.5 4.5 0 011.242 7.244l-4.5 4.5a4.5 4.5 0 01-6.364-6.364l1.757-1.757m13.35-.622l1.757-1.757a4.5 4.5 0 00-6.364-6.364l-4.5 4.5a4.5 4.5 0 001.242 7.244" /></svg>,
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
        icon: <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}><path strokeLinecap="round" strokeLinejoin="round" d="M12 4.5v15m7.5-7.5h-15" /></svg>,
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
      ? issues.filter(i => fuzzyMatch(i.title, query) || i.id.includes(query.toLowerCase()) || i.labels?.some(l => fuzzyMatch(l, query)))
      : issues.slice(0, 8);

    for (const issue of matchedIssues.slice(0, 12)) {
      results.push({
        id: `issue:${issue.id}`,
        group: query ? 'Issues' : 'Recent Issues',
        icon: <StatusIcon status={issue.status} size={14} />,
        label: issue.title,
        meta: (
          <span className="flex items-center gap-2">
            <span className="font-mono text-xs text-[var(--color-text-muted)]">{issue.id.replace('beats-', '')}</span>
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
        className="relative w-full max-w-140 bg-[var(--color-bg-elevated)] border border-[var(--color-border-default)] rounded-[var(--radius-xl)] shadow-[var(--shadow-lg)] overflow-hidden animate-fade-in"
        onKeyDown={handleKeyDown}
      >
        {/* Search input */}
        <div className="flex items-center gap-3 px-4 h-12 border-b border-[var(--color-border-subtle)]">
          <svg className="w-4 h-4 text-[var(--color-text-muted)] shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M21 21l-5.197-5.197m0 0A7.5 7.5 0 105.196 5.196a7.5 7.5 0 0010.607 10.607z" />
          </svg>
          <input
            ref={inputRef}
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="Search issues, actions, navigation..."
            className="flex-1 bg-transparent text-base text-[var(--color-text-primary)] placeholder:text-[var(--color-text-muted)] outline-none"
          />
          <kbd className="text-xs text-[var(--color-text-muted)] bg-[var(--color-bg-tertiary)] px-2 py-1 rounded-[var(--radius-sm)]">ESC</kbd>
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
                      className={`flex items-center gap-3 w-full px-4 h-10 text-left transition-colors ${isSelected ? 'bg-[var(--color-bg-hover)]' : ''}`}
                    >
                      <span className="text-[var(--color-text-muted)] shrink-0">{item.icon}</span>
                      <span className="text-sm text-[var(--color-text-primary)] truncate flex-1">{item.label}</span>
                      {item.meta}
                      {item.shortcut && (
                        <kbd className="text-xs text-[var(--color-text-muted)] bg-[var(--color-bg-tertiary)] px-2 py-1 rounded-[var(--radius-sm)] ml-auto shrink-0">
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
            <kbd className="bg-[var(--color-bg-tertiary)] px-1 rounded">↑↓</kbd> navigate
          </span>
          <span className="flex items-center gap-1">
            <kbd className="bg-[var(--color-bg-tertiary)] px-1 rounded">↵</kbd> select
          </span>
          <span className="flex items-center gap-1">
            <kbd className="bg-[var(--color-bg-tertiary)] px-1 rounded">esc</kbd> close
          </span>
        </div>
      </div>
    </div>,
    document.body
  );
}
