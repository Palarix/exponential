import { useState, useEffect, useCallback, useRef, useMemo } from 'react';
import { createPortal } from 'react-dom';
import { fetchIssues, fetchConfig, createIssue } from './api/client';
import type { Issue } from './api/client';
import Layout from './components/Layout/Layout';
import Dashboard from './components/Dashboard/Dashboard';
import Backlog from './components/Backlog/Backlog';
import Board from './components/Board/Board';
import Dependencies from './components/Dependencies/Dependencies';
import IssueDetail from './components/IssueDetail/IssueDetail';
import CommandPalette from './components/CommandPalette/CommandPalette';
import { Modal, Button } from './components/ui';
import { sortIssuesWithinGroups } from './utils/sort';
import type { SortKey } from './utils/sort';

type View = 'dashboard' | 'backlog' | 'board' | 'dependencies';

const VIEW_ROUTES: Record<string, View> = {
  'issues': 'backlog',
  'board': 'board',
  'dashboard': 'dashboard',
  'dependencies': 'dependencies',
};
const ROUTE_VIEWS: Record<View, string> = {
  backlog: 'issues',
  board: 'board',
  dashboard: 'dashboard',
  dependencies: 'dependencies',
};

function parseHash(): { view: View; issueId: string | null } {
  const hash = window.location.hash.replace(/^#\/?/, '');
  const parts = hash.split('/');
  if (parts[0] === 'issues' && parts[1]) {
    return { view: 'backlog', issueId: parts[1] };
  }
  const view = VIEW_ROUTES[parts[0]];
  return { view: view || 'backlog', issueId: null };
}

function setHash(view: View, issueId: string | null) {
  const route = issueId ? `issues/${issueId}` : ROUTE_VIEWS[view];
  const newHash = `#/${route}`;
  if (window.location.hash !== newHash) {
    window.location.hash = newHash;
  }
}

function App() {
  const initial = parseHash();
  const [view, setView] = useState<View>(initial.view);
  const [issues, setIssues] = useState<Issue[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selectedIssueId, setSelectedIssueId] = useState<string | null>(initial.issueId);
  const [searchFocused, setSearchFocused] = useState(false);
  const [showNewIssue, setShowNewIssue] = useState(false);
  const [showPalette, setShowPalette] = useState(false);
  const [prefix, setPrefix] = useState('beats-');
  const [version, setVersion] = useState('');
  const [sortKey, setSortKey] = useState<SortKey>(() =>
    (localStorage.getItem('beats-sort') as SortKey) || 'manual'
  );
  const [backlogNavOrder, setBacklogNavOrder] = useState<string[]>([]);

  const selectedIssue = selectedIssueId ? issues.find(i => i.id === selectedIssueId) ?? null : null;

  const defaultNavOrder = useMemo(() =>
    sortIssuesWithinGroups(issues, sortKey).map(i => i.id),
    [issues, sortKey]
  );
  const navigationOrder = backlogNavOrder.length > 0 ? backlogNavOrder : defaultNavOrder;

  useEffect(() => {
    setHash(view, selectedIssueId);
  }, [view, selectedIssueId]);

  useEffect(() => {
    const onHashChange = () => {
      const { view: v, issueId } = parseHash();
      setView(v);
      setSelectedIssueId(issueId);
    };
    window.addEventListener('hashchange', onHashChange);
    return () => window.removeEventListener('hashchange', onHashChange);
  }, []);

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (e.target instanceof HTMLInputElement || e.target instanceof HTMLTextAreaElement) return;
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault();
        setShowPalette(true);
        return;
      }
      if (e.key === 'c' && !e.metaKey && !e.ctrlKey && !showNewIssue && !showPalette) {
        e.preventDefault();
        setShowNewIssue(true);
      }
    };
    document.addEventListener('keydown', handler);
    return () => document.removeEventListener('keydown', handler);
  }, [showNewIssue, showPalette]);

  const fetchData = useCallback(async () => {
    try {
      const issuesData = await fetchIssues();
      setIssues(issuesData);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch data');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchData();
    fetchConfig().then(c => { setPrefix(c.prefix); setVersion(c.version || ''); }).catch(() => {});
    const interval = setInterval(fetchData, 5000);
    return () => clearInterval(interval);
  }, [fetchData]);

  const handleIssueClick = (issue: Issue) => {
    setSelectedIssueId(issue.id);
  };

  const handleViewChange = (v: View) => {
    setSelectedIssueId(null);
    setView(v);
  };

  const selectedNavIndex = selectedIssueId ? navigationOrder.indexOf(selectedIssueId) : -1;

  const navigateIssue = (direction: 'prev' | 'next') => {
    if (selectedNavIndex === -1) return;
    const nextIndex = direction === 'prev' ? selectedNavIndex - 1 : selectedNavIndex + 1;
    if (nextIndex >= 0 && nextIndex < navigationOrder.length) {
      setSelectedIssueId(navigationOrder[nextIndex]);
    }
  };

  const handleSortChange = useCallback((key: SortKey) => {
    setSortKey(key);
    localStorage.setItem('beats-sort', key);
  }, []);

  const renderContent = () => {
    if (loading) {
      return (
        <div className="flex flex-col items-center justify-center h-full gap-3">
          <div className="w-5 h-5 border-[1.5px] border-[var(--color-text-muted)] border-t-transparent rounded-full animate-spin" />
          <p className="text-[var(--color-text-muted)] text-sm">Loading...</p>
        </div>
      );
    }

    if (error) {
      return (
        <div className="flex flex-col items-center justify-center h-full gap-6 px-8">
          {/* Illustration */}
          <svg width="160" height="120" viewBox="0 0 160 120" fill="none" className="opacity-30">
            <rect x="30" y="20" width="100" height="70" rx="8" stroke="var(--color-text-muted)" strokeWidth="1.5" strokeDasharray="4 3" />
            <circle cx="80" cy="48" r="12" stroke="var(--color-text-muted)" strokeWidth="1.5" />
            <path d="M76 48h8M80 44v8" stroke="var(--color-text-muted)" strokeWidth="1.5" strokeLinecap="round" />
            <path d="M50 100h60" stroke="var(--color-text-muted)" strokeWidth="1.5" strokeLinecap="round" strokeDasharray="3 4" />
            <path d="M55 106h50" stroke="var(--color-text-muted)" strokeWidth="1.5" strokeLinecap="round" strokeDasharray="3 4" />
            <circle cx="80" cy="72" r="2" fill="var(--color-text-muted)" />
          </svg>

          <div className="flex flex-col items-center gap-2 max-w-[320px] text-center">
            <h2 className="text-lg font-semibold text-[var(--color-text-primary)]">Server Offline</h2>
            <p className="text-sm text-[var(--color-text-secondary)] leading-relaxed">
              Unable to reach the Beats server. Make sure <code className="text-xs font-mono bg-[var(--color-bg-secondary)] px-1.5 py-0.5 rounded-[var(--radius-sm)]">beats board</code> is running in your terminal.
            </p>
          </div>

          <button
            onClick={fetchData}
            className="px-4 py-2 text-sm bg-[var(--color-bg-secondary)] border border-[var(--color-border-default)] rounded-[var(--radius-md)] text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-bg-hover)] transition-colors"
          >
            Retry Connection
          </button>
        </div>
      );
    }

    if (selectedIssue) {
      return (
        <IssueDetail
          issue={selectedIssue}
          issues={issues}
          currentIndex={selectedNavIndex}
          totalCount={navigationOrder.length}
          onClose={() => setSelectedIssueId(null)}
          onNavigate={navigateIssue}
          onRefresh={fetchData}
          prefix={prefix}
        />
      );
    }

    switch (view) {
      case 'dashboard':
        return <Dashboard issues={issues} />;
      case 'backlog':
        return (
          <Backlog
            issues={issues}
            onRefresh={fetchData}
            onIssueClick={handleIssueClick}
            searchFocused={searchFocused}
            onSearchBlur={() => setSearchFocused(false)}
            sortKey={sortKey}
            onSortChange={handleSortChange}
            onNavigationOrderChange={setBacklogNavOrder}
          />
        );
      case 'board':
        return <Board issues={issues} onRefresh={fetchData} onIssueClick={handleIssueClick} />;
      case 'dependencies':
        return <Dependencies issues={issues} onIssueClick={handleIssueClick} />;
    }
  };

  return (
    <>
      <Layout
        currentView={view}
        onViewChange={handleViewChange}
        onSearch={() => setShowPalette(true)}
        onNewIssue={() => setShowNewIssue(true)}
        version={version}
        connected={!error}
      >
        {renderContent()}
      </Layout>

      <NewIssueModal
        isOpen={showNewIssue}
        onClose={() => setShowNewIssue(false)}
        onCreated={async () => {
          setShowNewIssue(false);
          await fetchData();
        }}
      />

      <CommandPalette
        isOpen={showPalette}
        onClose={() => setShowPalette(false)}
        issues={issues}
        onIssueSelect={(issue) => setSelectedIssueId(issue.id)}
        onViewChange={handleViewChange}
        onNewIssue={() => setShowNewIssue(true)}
      />
    </>
  );
}

type DropdownOption = { value: string; label: string; dot?: string };

const LABEL_OPTIONS: DropdownOption[] = [
  { value: 'bug', label: 'Bug', dot: 'var(--color-label-bug)' },
  { value: 'feature', label: 'Feature', dot: 'var(--color-label-feature)' },
  { value: 'epic', label: 'Epic', dot: 'var(--color-label-epic)' },
  { value: 'improvement', label: 'Improvement', dot: 'var(--color-label-improvement)' },
];

const STATUS_OPTIONS: DropdownOption[] = [
  { value: 'BACKLOG', label: 'Backlog' },
  { value: 'PLANNED', label: 'Planned' },
  { value: 'DOING', label: 'In Progress' },
];

const ESTIMATE_OPTIONS: DropdownOption[] = [
  { value: '0', label: 'No estimate' },
  { value: '1', label: '1 Point' },
  { value: '2', label: '2 Points' },
  { value: '3', label: '3 Points' },
  { value: '5', label: '5 Points' },
  { value: '8', label: '8 Points' },
];

function NewIssueModal({ isOpen, onClose, onCreated }: { isOpen: boolean; onClose: () => void; onCreated: () => void }) {
  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [label, setLabel] = useState('feature');
  const [status, setStatus] = useState('BACKLOG');
  const [estimate, setEstimate] = useState('0');
  const [saving, setSaving] = useState(false);

  const canCreate = title.trim() && label;

  const handleCreate = async () => {
    if (!canCreate) return;
    setSaving(true);
    try {
      await createIssue({
        title: title.trim(),
        description: description.trim() || undefined,
        labels: [label],
      });
      setTitle('');
      setDescription('');
      setLabel('feature');
      setStatus('BACKLOG');
      setEstimate('0');
      onCreated();
    } catch (err) {
      console.error('Failed to create issue:', err);
    } finally {
      setSaving(false);
    }
  };

  const handleModalKeyDown = (e: React.KeyboardEvent) => {
    if ((e.metaKey || e.ctrlKey) && e.key === 'Enter' && canCreate) {
      e.preventDefault();
      handleCreate();
    }
  };

  return (
    <Modal isOpen={isOpen} onClose={onClose} title="New Issue" size="2xl">
      <div className="space-y-3" onKeyDown={handleModalKeyDown}>
        {/* Properties row: label, status, estimate */}
        <div className="flex items-center gap-2">
          <InlineDropdown
            placeholder="Label *"
            options={LABEL_OPTIONS}
            value={label}
            onChange={setLabel}
            required
          />
          <InlineDropdown
            placeholder="Status"
            options={STATUS_OPTIONS}
            value={status}
            onChange={setStatus}
          />
          <div className="flex-1" />
          <InlineDropdown
            placeholder="Estimate"
            options={ESTIMATE_OPTIONS}
            value={estimate}
            onChange={setEstimate}
          />
        </div>

        {/* Title */}
        <input
          autoFocus
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          placeholder="Issue title *"
          className="w-full h-10 px-3 bg-[var(--color-bg-tertiary)] border border-[var(--color-border-default)] rounded-[var(--radius-md)] text-lg text-[var(--color-text-primary)] placeholder:text-[var(--color-text-muted)] focus:outline-none focus:border-[var(--color-border-focus)] transition-colors"
          onKeyDown={(e) => { if (e.key === 'Enter' && !e.shiftKey && canCreate) handleCreate(); }}
        />

        {/* Description */}
        <textarea
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          placeholder="Add description (markdown supported)..."
          rows={8}
          className="w-full px-3 py-2 bg-[var(--color-bg-tertiary)] border border-[var(--color-border-default)] rounded-[var(--radius-md)] text-base text-[var(--color-text-primary)] placeholder:text-[var(--color-text-muted)] focus:outline-none focus:border-[var(--color-border-focus)] transition-colors resize-y"
        />

        {/* Actions — inline, no divider */}
        <div className="flex items-center justify-end gap-2 pt-2">
          <Button variant="ghost" size="sm" onClick={onClose}>Cancel</Button>
          <Button size="sm" onClick={handleCreate} disabled={!canCreate || saving} loading={saving}>
            Create Issue
          </Button>
        </div>
      </div>
    </Modal>
  );
}

function InlineDropdown({ placeholder, options, value, onChange, required }: {
  placeholder: string;
  options: DropdownOption[];
  value: string;
  onChange: (v: string) => void;
  required?: boolean;
}) {
  const [open, setOpen] = useState(false);
  const btnRef = useRef<HTMLButtonElement>(null);
  const menuRef = useRef<HTMLDivElement>(null);
  const [pos, setPos] = useState({ top: 0, left: 0 });
  const selected = options.find(o => o.value === value);

  useEffect(() => {
    if (!open) return;
    const handler = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node) &&
          btnRef.current && !btnRef.current.contains(e.target as Node)) {
        setOpen(false);
      }
    };
    document.addEventListener('mousedown', handler);
    return () => document.removeEventListener('mousedown', handler);
  }, [open]);

  const handleOpen = () => {
    if (btnRef.current) {
      const rect = btnRef.current.getBoundingClientRect();
      setPos({ top: rect.bottom + 4, left: rect.left });
    }
    setOpen(!open);
  };

  return (
    <div>
      <button
        ref={btnRef}
        type="button"
        onClick={handleOpen}
        className={`
          flex items-center gap-1.5 h-8 px-3 rounded-[var(--radius-md)] text-sm transition-colors
          border border-[var(--color-border-default)] hover:border-[var(--color-border-focus)]
          ${!selected && required ? 'border-[var(--color-error)]/40' : ''}
        `.trim().replace(/\s+/g, ' ')}
      >
        {selected ? (
          <>
            {selected.dot && <span className="w-2 h-2 rounded-full" style={{ background: selected.dot }} />}
            <span className="text-[var(--color-text-primary)]">{selected.label}</span>
          </>
        ) : (
          <span className="text-[var(--color-text-muted)]">{placeholder}</span>
        )}
        <svg className="w-3 h-3 text-[var(--color-text-muted)]" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
          <path strokeLinecap="round" strokeLinejoin="round" d="M19 9l-7 7-7-7" />
        </svg>
      </button>
      {open && createPortal(
        <div
          ref={menuRef}
          className="fixed z-[100] min-w-[160px] bg-[var(--color-bg-elevated)] border border-[var(--color-border-default)] rounded-[var(--radius-lg)] shadow-[var(--shadow-popover)] py-1"
          style={{ top: pos.top, left: pos.left }}
        >
          {options.map((opt) => (
            <button
              key={opt.value}
              onClick={() => { onChange(opt.value); setOpen(false); }}
              className="flex items-center gap-2 w-full h-8 px-3 text-sm hover:bg-[var(--color-bg-hover)] transition-colors"
            >
              {opt.dot && <span className="w-2 h-2 rounded-full shrink-0" style={{ background: opt.dot }} />}
              <span className="text-[var(--color-text-primary)]">{opt.label}</span>
              {opt.value === value && (
                <svg className="w-3.5 h-3.5 ml-auto text-[var(--color-accent-primary)]" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
                </svg>
              )}
            </button>
          ))}
        </div>,
        document.body
      )}
    </div>
  );
}

export default App;
