import { useState, useEffect, useCallback, useRef, useMemo } from 'react';
import { fetchIssues, fetchConfig, fetchInbox, fetchInboxStatus, markInboxRead, ApiError } from './api/client';
import type { Issue, InboxItem } from './api/client';
import Layout from './components/Layout/Layout';
import Dashboard from './components/Dashboard/Dashboard';
import Backlog from './components/Backlog/Backlog';
import type { Tab } from './components/Backlog/Backlog';
import Board from './components/Board/Board';
import Dependencies from './components/Dependencies/Dependencies';
import Labels from './components/Labels/Labels';
import Cycles from './components/Cycles/Cycles';
import Inbox from './components/Inbox/Inbox';
import IssueDetail from './components/IssueDetail/IssueDetail';
import CommandPalette from './components/CommandPalette/CommandPalette';
import NewIssueModal from './components/NewIssueModal/NewIssueModal';
import { LabelColorsContext, HideDefaultLabelsContext, DefaultLabelsContext, ErrorBoundary, useToast } from './components/ui';
import { useSSE } from './hooks/useSSE';
import { sortIssuesWithinGroups } from './utils/sort';
import type { SortKey } from './utils/sort';
import { isEditableTarget } from './utils/keyboard';
import { type BacklogFilters, EMPTY_FILTERS, hasActiveFilters } from './components/Backlog/filters';
import FilterChips from './components/Backlog/FilterChips';

type View = 'dashboard' | 'inbox' | 'backlog' | 'board' | 'cycles' | 'dependencies' | 'labels';

const VIEW_ROUTES: Record<string, View> = {
  'issues': 'backlog',
  'board': 'board',
  'dashboard': 'dashboard',
  'inbox': 'inbox',
  'cycles': 'cycles',
  'dependencies': 'dependencies',
  'labels': 'labels',
};
const ROUTE_VIEWS: Record<View, string> = {
  backlog: 'issues',
  board: 'board',
  dashboard: 'dashboard',
  inbox: 'inbox',
  cycles: 'cycles',
  dependencies: 'dependencies',
  labels: 'labels',
};

function parseHash(): { view: View; issueId: string | null; cycleId: string | null } {
  const hash = window.location.hash.replace(/^#\/?/, '');
  const parts = hash.split('/');
  if (parts[0] === 'issues' && parts[1]) {
    return { view: 'backlog', issueId: parts[1], cycleId: null };
  }
  if (parts[0] === 'cycles' && parts[1]) {
    return { view: 'cycles', issueId: null, cycleId: parts[1] };
  }
  const view = VIEW_ROUTES[parts[0]];
  return { view: view || 'dashboard', issueId: null, cycleId: null };
}

function setHash(view: View, issueId: string | null, cycleId?: string | null) {
  let route: string;
  if (issueId) {
    route = `issues/${issueId}`;
  } else if (view === 'cycles' && cycleId) {
    route = `cycles/${cycleId}`;
  } else {
    route = ROUTE_VIEWS[view];
  }
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
  const [selectedCycleId, setSelectedCycleId] = useState<string | null>(initial.cycleId);
  const [searchFocused, setSearchFocused] = useState(false);
  const [showNewIssue, setShowNewIssue] = useState(false);
  const [showPalette, setShowPalette] = useState(false);
  const [prefix, setPrefix] = useState('beats-');
  const [version, setVersion] = useState('');
  const [configLabels, setConfigLabels] = useState<Record<string, string>>({});
  const [contributors, setContributors] = useState<string[]>([]);
  const [hideDefaultLabels, setHideDefaultLabels] = useState(false);
  const [cyclesEnabled, setCyclesEnabled] = useState(false);
  const [defaultLabels, setDefaultLabels] = useState<{ name: string; color: string }[]>([]);
  const [sortKey, setSortKey] = useState<SortKey>(() =>
    (localStorage.getItem('beats-sort') as SortKey) || 'manual'
  );
  const [backlogNavOrder, setBacklogNavOrder] = useState<string[]>([]);
  const [backlogTab, setBacklogTab] = useState<Tab>('all');
  const [backlogFilters, setBacklogFilters] = useState<BacklogFilters>(() => {
    const stored = localStorage.getItem(`beats-backlog-filters-${backlogTab}`);
    if (stored) { try { return JSON.parse(stored); } catch {} }
    return EMPTY_FILTERS;
  });
  const [inboxItems, setInboxItems] = useState<InboxItem[]>([]);
  const [inboxLastRead, setInboxLastRead] = useState('');
  const [inboxUnread, setInboxUnread] = useState(0);
  const showToast = useToast();

  const handleFiltersChange = useCallback((f: BacklogFilters) => {
    setBacklogFilters(f);
    localStorage.setItem(`beats-backlog-filters-${backlogTab}`, JSON.stringify(f));
  }, [backlogTab]);

  const selectedIssue = selectedIssueId ? issues.find(i => i.id === selectedIssueId) ?? null : null;

  const defaultNavOrder = useMemo(() =>
    sortIssuesWithinGroups(issues, sortKey).map(i => i.id),
    [issues, sortKey]
  );
  const navigationOrder = backlogNavOrder.length > 0 ? backlogNavOrder : defaultNavOrder;

  useEffect(() => {
    setHash(view, selectedIssueId, selectedCycleId);
  }, [view, selectedIssueId, selectedCycleId]);

  useEffect(() => {
    const onHashChange = () => {
      const { view: v, issueId, cycleId } = parseHash();
      setView(v);
      setSelectedIssueId(issueId);
      setSelectedCycleId(cycleId);
    };
    window.addEventListener('hashchange', onHashChange);
    return () => window.removeEventListener('hashchange', onHashChange);
  }, []);

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (isEditableTarget(e)) return;
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

  const lastJsonRef = useRef('');
  const fetchData = useCallback(async () => {
    try {
      const issuesData = await fetchIssues();
      const json = JSON.stringify(issuesData);
      if (json !== lastJsonRef.current) {
        lastJsonRef.current = json;
        setIssues(issuesData);
      }
      setError(null);
    } catch (err) {
      setError(err instanceof ApiError ? `${err.status}: ${err.message}` : err instanceof Error ? err.message : 'Failed to fetch data');
    } finally {
      setLoading(false);
    }
  }, []);

  const fetchInboxData = useCallback(async () => {
    try {
      const [items, status] = await Promise.all([fetchInbox(), fetchInboxStatus()]);
      setInboxItems(items ?? []);
      setInboxLastRead(status.last_read ?? '');
      setInboxUnread(status.unread ?? 0);
    } catch {}
  }, []);

  const handleMarkAllRead = useCallback(async () => {
    try {
      const res = await markInboxRead();
      setInboxLastRead(res.last_read);
      setInboxUnread(0);
      showToast("Inbox marked as read");
    } catch {}
  }, [showToast]);

  useEffect(() => {
    fetchData();
    fetchInboxData();
    fetchConfig().then(c => {
      setPrefix(c.prefix);
      setVersion(c.version || '');
      setConfigLabels(c.labels || {});
      setContributors(c.contributors || []);
      setHideDefaultLabels(!!c.hide_default_labels);
      setCyclesEnabled(!!c.cycles?.enabled);
      if (c.default_labels) {
        const colors = c.labels || {};
        setDefaultLabels(c.default_labels.map(name => ({ name, color: colors[name] || colors[name.toLowerCase()] || '' })));
      }
      document.title = c.name ? `${c.name} | Beats` : 'Beats';
    }).catch(() => {});
  }, [fetchData]);

  const handleSSEEvent = useCallback(() => {
    fetchData();
    fetchInboxData();
  }, [fetchData, fetchInboxData]);

  useSSE({ onEvent: handleSSEEvent, fallbackInterval: 30000 });

  const handleIssueClick = (issue: Issue) => {
    setSelectedIssueId(issue.id);
  };

  const handleViewChange = (v: View) => {
    setSelectedIssueId(null);
    setSelectedCycleId(null);
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
          <div className="w-5 h-5 border-2 border-[var(--color-text-muted)] border-t-transparent rounded-full animate-spin" />
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

          <div className="flex flex-col items-center gap-2 max-w-80 text-center">
            <h2 className="text-lg font-semibold text-[var(--color-text-primary)]">Server Offline</h2>
            <p className="text-sm text-[var(--color-text-secondary)] leading-relaxed">
              Unable to reach the Beats server. Make sure <code className="text-xs font-mono bg-[var(--color-bg-secondary)] px-2 py-1 rounded-[var(--radius-sm)]">beats board</code> is running in your terminal.
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
          contributors={contributors}
          onConfigLabelsChange={setConfigLabels}
        />
      );
    }

    switch (view) {
      case 'inbox':
        return <Inbox items={inboxItems} lastRead={inboxLastRead} issues={issues} onIssueClick={handleIssueClick} onMarkAllRead={handleMarkAllRead} />;
      case 'dashboard':
        return <Dashboard issues={issues} onIssueClick={handleIssueClick} onNewIssue={() => setShowNewIssue(true)} />;
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
            activeTab={backlogTab}
            onTabChange={setBacklogTab}
            contributors={contributors}
            onConfigLabelsChange={setConfigLabels}
            onNewIssue={() => setShowNewIssue(true)}
            filters={backlogFilters}
            onFiltersChange={handleFiltersChange}
          />
        );
      case 'board':
        return <Board issues={issues} onRefresh={fetchData} onIssueClick={handleIssueClick} onNewIssue={() => setShowNewIssue(true)} />;
      case 'dependencies':
        return <Dependencies issues={issues} onIssueClick={handleIssueClick} />;
      case 'cycles':
        return (
          <Cycles
            issues={issues}
            onIssueClick={handleIssueClick}
            onRefresh={fetchData}
            selectedCycleId={selectedCycleId}
            onCycleSelect={setSelectedCycleId}
          />
        );
      case 'labels':
        return <Labels issues={issues} onConfigLabelsChange={setConfigLabels} onRefresh={fetchData} />;
    }
  };

  const effectiveLabelColors = useMemo(() => {
    if (hideDefaultLabels) return configLabels;
    const defaults = Object.fromEntries(defaultLabels.map(d => [d.name, d.color]));
    return { ...defaults, ...configLabels };
  }, [configLabels, hideDefaultLabels, defaultLabels]);

  return (
    <DefaultLabelsContext.Provider value={defaultLabels}>
    <HideDefaultLabelsContext.Provider value={hideDefaultLabels}>
    <LabelColorsContext.Provider value={effectiveLabelColors}>
      <Layout
        currentView={view}
        onViewChange={handleViewChange}
        onSearch={() => setShowPalette(true)}
        onNewIssue={() => setShowNewIssue(true)}
        version={version}
        connected={!error}
        cyclesEnabled={cyclesEnabled}
        inboxUnread={inboxUnread}
        statusBarLeft={view === 'backlog' && hasActiveFilters(backlogFilters) ? (
          <FilterChips filters={backlogFilters} onChange={handleFiltersChange} />
        ) : undefined}
      >
        <ErrorBoundary onReset={fetchData}>
          {renderContent()}
        </ErrorBoundary>
      </Layout>

      <NewIssueModal
        isOpen={showNewIssue}
        onClose={() => setShowNewIssue(false)}
        onCreated={async () => {
          setShowNewIssue(false);
          await fetchData();
          showToast("Issue created");
        }}
        issues={issues}
        contributors={contributors}
        onConfigLabelsChange={setConfigLabels}
      />

      <CommandPalette
        isOpen={showPalette}
        onClose={() => setShowPalette(false)}
        issues={issues}
        onIssueSelect={(issue) => setSelectedIssueId(issue.id)}
        onViewChange={handleViewChange}
        onNewIssue={() => setShowNewIssue(true)}
      />
    </LabelColorsContext.Provider>
    </HideDefaultLabelsContext.Provider>
    </DefaultLabelsContext.Provider>
  );
}

export default App;
