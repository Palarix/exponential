import { useState, useEffect, useCallback, useRef, useMemo, lazy, Suspense } from 'react';
import { fetchIssues, fetchConfig, fetchInbox, fetchInboxStatus, markInboxRead, ApiError } from './api/client';
import type { Issue, InboxItem } from './api/client';
import Layout from './components/Layout/Layout';
import Backlog from './components/Backlog/Backlog';
import type { Tab } from './components/Backlog/Backlog';
import CommandPalette from './components/CommandPalette/CommandPalette';
import NewIssueModal from './components/NewIssueModal/NewIssueModal';
import KeyboardHelp from './components/KeyboardHelp/KeyboardHelp';

const Dashboard = lazy(() => import('./components/Dashboard/Dashboard'));
const Board = lazy(() => import('./components/Board/Board'));
const Dependencies = lazy(() => import('./components/Dependencies/Dependencies'));
const Labels = lazy(() => import('./components/Labels/Labels'));
const Cycles = lazy(() => import('./components/Cycles/Cycles'));
const Inbox = lazy(() => import('./components/Inbox/Inbox'));
const MyIssues = lazy(() => import('./components/MyIssues/MyIssues'));
const Timeline = lazy(() => import('./components/Timeline/Timeline'));
import type { MyIssuesTab } from './components/MyIssues/MyIssues';
const IssueDetail = lazy(() => import('./components/IssueDetail/IssueDetail'));
import { LabelColorsContext, HideDefaultLabelsContext, DefaultLabelsContext, ErrorBoundary, useToast } from './components/ui';
import { useSSE } from './hooks/useSSE';
import { sortIssuesWithinGroups } from './utils/sort';
import type { SortKey } from './utils/sort';
import { isEditableTarget } from './utils/keyboard';
import { type BacklogFilters, EMPTY_FILTERS, hasActiveFilters } from './components/Backlog/filters';
import FilterChips from './components/Backlog/FilterChips';

type View = 'dashboard' | 'inbox' | 'backlog' | 'board' | 'cycles' | 'dependencies' | 'labels' | 'my-issues' | 'timeline';

const VIEW_LABELS: Record<View, string> = {
  dashboard: 'Dashboard',
  inbox: 'Inbox',
  backlog: 'Issues',
  board: 'Board',
  cycles: 'Cycles',
  dependencies: 'Dependencies',
  labels: 'Labels',
  'my-issues': 'My Issues',
  timeline: 'Timeline',
};

const VIEW_ROUTES: Record<string, View> = {
  'issues': 'backlog',
  'board': 'board',
  'dashboard': 'dashboard',
  'inbox': 'inbox',
  'cycles': 'cycles',
  'dependencies': 'dependencies',
  'labels': 'labels',
  'my-issues': 'my-issues',
  'timeline': 'timeline',
};
const ROUTE_VIEWS: Record<View, string> = {
  backlog: 'issues',
  board: 'board',
  dashboard: 'dashboard',
  inbox: 'inbox',
  cycles: 'cycles',
  dependencies: 'dependencies',
  labels: 'labels',
  'my-issues': 'my-issues',
  timeline: 'timeline',
};

function parseHash(): { view: View; issueId: string | null; cycleId: string | null; depFocusId: string | null } {
  const hash = window.location.hash.replace(/^#\/?/, '');
  const parts = hash.split('/');
  if (parts[0] === 'issues' && parts[1]) {
    return { view: 'backlog', issueId: parts[1], cycleId: null, depFocusId: null };
  }
  if (parts[0] === 'cycles' && parts[1]) {
    return { view: 'cycles', issueId: null, cycleId: parts[1], depFocusId: null };
  }
  if (parts[0] === 'dependencies' && parts[1]) {
    return { view: 'dependencies', issueId: null, cycleId: null, depFocusId: parts[1] };
  }
  const view = VIEW_ROUTES[parts[0]];
  return { view: view || 'dashboard', issueId: null, cycleId: null, depFocusId: null };
}

function setHash(view: View, issueId: string | null, cycleId?: string | null, depFocusId?: string | null) {
  let route: string;
  if (issueId) {
    route = `issues/${issueId}`;
  } else if (view === 'cycles' && cycleId) {
    route = `cycles/${cycleId}`;
  } else if (view === 'dependencies' && depFocusId) {
    route = `dependencies/${depFocusId}`;
  } else {
    route = ROUTE_VIEWS[view];
  }
  const newHash = `#/${route}`;
  if (window.location.hash !== newHash) {
    window.location.hash = newHash;
  }
}

const GO_TARGETS: Record<string, View> = {
  o: 'dashboard',
  i: 'backlog',
  b: 'board',
  n: 'inbox',
  d: 'dependencies',
  l: 'labels',
  c: 'cycles',
  m: 'my-issues',
  t: 'timeline',
};

function App() {
  const initial = parseHash();
  const [view, setView] = useState<View>(initial.view);
  const [issues, setIssues] = useState<Issue[]>([]);
  const patchIssue = useCallback((issueId: string, patch: Partial<Issue>) => {
    setIssues(prev => prev.map(i => i.id === issueId ? { ...i, ...patch } : i));
  }, []);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selectedIssueId, setSelectedIssueId] = useState<string | null>(initial.issueId);
  const [selectedCycleId, setSelectedCycleId] = useState<string | null>(initial.cycleId);
  const [depFocusId, setDepFocusId] = useState<string | null>(initial.depFocusId);

  const [showNewIssue, setShowNewIssue] = useState(false);
  const [showPalette, setShowPalette] = useState(false);
  const [showKeyboardHelp, setShowKeyboardHelp] = useState(false);
  const [prefix, setPrefix] = useState('issue-');
  const [version, setVersion] = useState('');
  const [projectName, setProjectName] = useState('');
  const [configLabels, setConfigLabels] = useState<Record<string, string>>({});
  const [contributors, setContributors] = useState<string[]>([]);
  const [hideDefaultLabels, setHideDefaultLabels] = useState(false);
  const [cyclesEnabled, setCyclesEnabled] = useState(false);
  const [defaultLabels, setDefaultLabels] = useState<{ name: string; color: string }[]>([]);
  const [sortKey, setSortKey] = useState<SortKey>(() =>
    (localStorage.getItem('exponential-sort') as SortKey) || 'manual'
  );
  const [backlogNavOrder, setBacklogNavOrder] = useState<string[]>([]);
  const [backlogTab, setBacklogTab] = useState<Tab>('all');
  const [backlogFilters, setBacklogFilters] = useState<BacklogFilters>(() => {
    const stored = localStorage.getItem(`exponential-backlog-filters-${backlogTab}`);
    if (stored) { try { return JSON.parse(stored); } catch { /* corrupt stored filter — use default */ } }
    return EMPTY_FILTERS;
  });
  const [myIssuesTab, setMyIssuesTab] = useState<MyIssuesTab>(() =>
    (localStorage.getItem('exponential-my-issues-tab') as MyIssuesTab) || 'assigned'
  );
  const [myIssuesFilters, setMyIssuesFilters] = useState<BacklogFilters>(() => {
    const stored = localStorage.getItem(`exponential-my-issues-filters-${myIssuesTab}`);
    if (stored) { try { return JSON.parse(stored); } catch { /* corrupt stored filter — use default */ } }
    return EMPTY_FILTERS;
  });
  const [inboxItems, setInboxItems] = useState<InboxItem[]>([]);
  const [inboxLastRead, setInboxLastRead] = useState('');
  const [inboxUnread, setInboxUnread] = useState(0);
  const [inboxFilters, setInboxFilters] = useState<BacklogFilters>(() => {
    const stored = localStorage.getItem('exponential-inbox-filters');
    if (stored) { try { return JSON.parse(stored); } catch { /* corrupt stored filter — use default */ } }
    return EMPTY_FILTERS;
  });
  const showToast = useToast();

  const handleInboxFiltersChange = useCallback((f: BacklogFilters) => {
    setInboxFilters(f);
    localStorage.setItem('exponential-inbox-filters', JSON.stringify(f));
  }, []);

  const handleFiltersChange = useCallback((f: BacklogFilters) => {
    setBacklogFilters(f);
    localStorage.setItem(`exponential-backlog-filters-${backlogTab}`, JSON.stringify(f));
  }, [backlogTab]);

  const handleMyIssuesTabChange = useCallback((t: MyIssuesTab) => {
    setMyIssuesTab(t);
    localStorage.setItem('exponential-my-issues-tab', t);
    const stored = localStorage.getItem(`exponential-my-issues-filters-${t}`);
    if (stored) { try { setMyIssuesFilters(JSON.parse(stored)); } catch { /* corrupt stored filter — use default */ } }
    else setMyIssuesFilters(EMPTY_FILTERS);
  }, []);

  const handleMyIssuesFiltersChange = useCallback((f: BacklogFilters) => {
    setMyIssuesFilters(f);
    localStorage.setItem(`exponential-my-issues-filters-${myIssuesTab}`, JSON.stringify(f));
  }, [myIssuesTab]);

  const selectedIssue = selectedIssueId ? (issues.find(i => i.id === selectedIssueId) ?? issues.find(i => i.id.endsWith(selectedIssueId)) ?? null) : null;

  const defaultNavOrder = useMemo(() =>
    sortIssuesWithinGroups(issues, sortKey).map(i => i.id),
    [issues, sortKey]
  );
  const navigationOrder = backlogNavOrder.length > 0 ? backlogNavOrder : defaultNavOrder;

  useEffect(() => {
    setHash(view, selectedIssueId, selectedCycleId, depFocusId);
  }, [view, selectedIssueId, selectedCycleId, depFocusId]);

  useEffect(() => {
    const label = selectedIssueId || VIEW_LABELS[view];
    document.title = projectName ? `${projectName} ❯ ${label}` : label;
  }, [view, selectedIssueId, projectName]);

  useEffect(() => {
    const onHashChange = () => {
      const { view: v, issueId, cycleId, depFocusId: dfId } = parseHash();
      setView(v);
      setSelectedIssueId(issueId);
      setSelectedCycleId(cycleId);
      setDepFocusId(dfId);
    };
    window.addEventListener('hashchange', onHashChange);
    return () => window.removeEventListener('hashchange', onHashChange);
  }, []);

  const gPendingRef = useRef(false);
  const gTimerRef = useRef<ReturnType<typeof setTimeout>>(undefined);

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if ((e.key === '?' || (e.key === '/' && e.shiftKey)) && !showPalette && !showNewIssue && !isEditableTarget(e)) {
        e.preventDefault();
        setShowKeyboardHelp(v => !v);
        return;
      }
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault();
        setShowPalette(true);
        return;
      }

      if (isEditableTarget(e)) return;
      if (e.metaKey || e.ctrlKey) return;

      if (gPendingRef.current) {
        gPendingRef.current = false;
        clearTimeout(gTimerRef.current);
        const target = GO_TARGETS[e.key.toLowerCase()];
        if (target) {
          e.preventDefault();
          handleViewChange(target);
        }
        return;
      }

      if (e.key === 'g' && !showPalette && !showNewIssue && !showKeyboardHelp) {
        e.preventDefault();
        gPendingRef.current = true;
        clearTimeout(gTimerRef.current);
        gTimerRef.current = setTimeout(() => { gPendingRef.current = false; }, 1000);
        return;
      }

      if (e.key === 'c' && !showNewIssue && !showPalette && !showKeyboardHelp) {
        e.preventDefault();
        setShowNewIssue(true);
      }
    };
    document.addEventListener('keydown', handler);
    return () => {
      document.removeEventListener('keydown', handler);
      clearTimeout(gTimerRef.current);
    };
  }, [showNewIssue, showPalette, showKeyboardHelp]);

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
    } catch { /* inbox fetch is best-effort */ }
  }, []);

  const handleMarkAllRead = useCallback(async () => {
    try {
      const res = await markInboxRead();
      setInboxLastRead(res.last_read);
      setInboxUnread(0);
      showToast("Notifications marked as read");
    } catch (err) {
      showToast(err instanceof Error ? err.message : "Failed to mark as read", { variant: "error" });
    }
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
      setProjectName(c.name || '');
    }).catch(() => { /* config fetch is best-effort */ });
  }, [fetchData, fetchInboxData]);

  const handleSSEEvent = useCallback(() => {
    fetchData();
    fetchInboxData();
  }, [fetchData, fetchInboxData]);

  useSSE({ onEvent: handleSSEEvent });

  const handleIssueClick = (issue: Issue) => {
    setSelectedIssueId(issue.id);
  };

  const handleViewChange = (v: View) => {
    setSelectedIssueId(null);
    setSelectedCycleId(null);
    setDepFocusId(null);
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
    localStorage.setItem('exponential-sort', key);
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
              Unable to reach the Exponential server. Make sure <code className="text-xs font-mono bg-[var(--color-surface-1)] px-2 py-1 rounded-[var(--radius-sm)]">xpo board</code> is running in your terminal.
            </p>
          </div>

          <button
            onClick={fetchData}
            className="px-4 py-2 text-sm bg-[var(--color-surface-1)] border border-[var(--color-border-default)] rounded-[var(--radius-md)] text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-hover-surface)] transition-colors"
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

    if (selectedIssueId) {
      return (
        <div className="flex flex-col items-center justify-center h-full gap-6 px-8">
          <div className="opacity-25">
            <svg className="w-16 h-16" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M21 21l-5.197-5.197m0 0A7.5 7.5 0 105.196 5.196a7.5 7.5 0 0010.607 10.607z" />
            </svg>
          </div>
          <div className="flex flex-col items-center gap-2 max-w-96 text-center">
            <h2 className="text-lg font-semibold text-[var(--color-text-primary)]">Issue not found</h2>
            <p className="text-sm text-[var(--color-text-secondary)] leading-relaxed">
              No issue matching <span className="font-mono text-[var(--color-text-primary)]">{selectedIssueId}</span> was found. It may have been deleted or the link may be incorrect.
            </p>
          </div>
          <button onClick={() => setSelectedIssueId(null)} className="px-4 py-2 text-sm bg-[var(--color-accent-primary)] text-white rounded-[var(--radius-md)] hover:opacity-90 transition-opacity">
            Back to Backlog
          </button>
        </div>
      );
    }

    switch (view) {
      case 'inbox':
        return (
          <Inbox
            items={inboxItems}
            lastRead={inboxLastRead}
            issues={issues}
            onMarkAllRead={handleMarkAllRead}
            onRefresh={fetchData}
            prefix={prefix}
            contributors={contributors}
            onConfigLabelsChange={setConfigLabels}
            filters={inboxFilters}
            onFiltersChange={handleInboxFiltersChange}
          />
        );
      case 'dashboard':
        return <Dashboard issues={issues} onIssueClick={handleIssueClick} onNewIssue={() => setShowNewIssue(true)} />;
      case 'backlog':
        return (
          <Backlog
            issues={issues}
            onRefresh={fetchData}
            onIssueClick={handleIssueClick}
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
            patchIssue={patchIssue}
          />
        );
      case 'board':
        return <Board issues={issues} onRefresh={fetchData} onIssueClick={handleIssueClick} onNewIssue={() => setShowNewIssue(true)} contributors={contributors} onConfigLabelsChange={setConfigLabels} patchIssue={patchIssue} />;
      case 'dependencies':
        return <Dependencies issues={issues} onIssueClick={handleIssueClick} focusIssueId={depFocusId} onFocusChange={setDepFocusId} />;
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
      case 'my-issues':
        return (
          <MyIssues
            issues={issues}
            onIssueClick={handleIssueClick}
            activeTab={myIssuesTab}
            onTabChange={handleMyIssuesTabChange}
            filters={myIssuesFilters}
            onFiltersChange={handleMyIssuesFiltersChange}
          />
        );
      case 'timeline':
        return <Timeline issues={issues} onIssueClick={handleIssueClick} />;
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
        statusBarLeft={
          view === 'backlog' && hasActiveFilters(backlogFilters) ? (
            <FilterChips filters={backlogFilters} onChange={handleFiltersChange} />
          ) : view === 'my-issues' && hasActiveFilters(myIssuesFilters) ? (
            <FilterChips filters={myIssuesFilters} onChange={handleMyIssuesFiltersChange} />
          ) : view === 'inbox' && hasActiveFilters(inboxFilters) ? (
            <FilterChips filters={inboxFilters} onChange={handleInboxFiltersChange} />
          ) : undefined
        }
      >
        <ErrorBoundary onReset={fetchData}>
          <Suspense fallback={null}>
            {renderContent()}
          </Suspense>
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
      <KeyboardHelp isOpen={showKeyboardHelp} onClose={() => setShowKeyboardHelp(false)} />
    </LabelColorsContext.Provider>
    </HideDefaultLabelsContext.Provider>
    </DefaultLabelsContext.Provider>
  );
}

export default App;
