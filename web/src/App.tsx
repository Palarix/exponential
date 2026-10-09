import { useState, useEffect, useCallback, useRef, useMemo } from 'react';
import Layout from './components/Layout/Layout';
import ViewRouter from './components/Layout/ViewRouter';
import CommandPalette from './components/CommandPalette/CommandPalette';
import NewIssueModal from './components/NewIssueModal/NewIssueModal';
import KeyboardHelp from './components/KeyboardHelp/KeyboardHelp';
import { LabelColorsContext, HideDefaultLabelsContext, DefaultLabelsContext, useToast } from './components/ui';
import { useSSE } from './hooks/useSSE';
import { useKeyboardShortcuts } from './keyboard';
import { useConfig, useInbox, useInvalidateOnServerEvent, useIssueList, useIssues, useRefreshIssues, useSetConfigLabels } from './api/queries';
import { effectiveLabelColors } from './api/query-utils';
import { AppNavContext, ViewStateContext, type AppNav, type NavIntent } from './app/contexts';
import { createViewStateStore } from './app/view-state-utils';
import { VIEW_BY_GO_KEY, VIEW_BY_ID, formatRoute, parseRoute, type Route, type ViewId } from './views';

function safeLocalStorage(): Storage | null {
  try { return window.localStorage; } catch { return null; }
}

function sameRoute(a: Route, b: Route): boolean {
  return a.view === b.view && a.issueId === b.issueId && a.cycleId === b.cycleId && a.depFocusId === b.depFocusId;
}

function App() {
  const [route, setRoute] = useState<Route>(() => parseRoute(window.location.hash));
  const routeRef = useRef(route);
  useEffect(() => { routeRef.current = route; }, [route]);

  // The active view publishes its visible order into the ref; it's snapshotted
  // into state when an issue opens, so prev/next follows the originating view.
  const publishedOrderRef = useRef<string[] | null>(null);
  const [issueNavOrder, setIssueNavOrder] = useState<string[] | null>(null);
  const intentsRef = useRef(new Map<ViewId, NavIntent>());
  const [viewStateStore] = useState(() => createViewStateStore(safeLocalStorage()));

  const [showNewIssue, setShowNewIssue] = useState(false);
  const [showPalette, setShowPalette] = useState(false);
  const [showKeyboardHelp, setShowKeyboardHelp] = useState(false);

  const issuesQuery = useIssues();
  const issues = useIssueList();
  const refresh = useRefreshIssues();
  const config = useConfig();
  const setConfigLabels = useSetConfigLabels();
  const { unread: inboxUnread } = useInbox();
  const showToast = useToast();

  useSSE({ onEvent: useInvalidateOnServerEvent() });

  const openIssue = useCallback((issueId: string) => {
    setIssueNavOrder(publishedOrderRef.current);
    setRoute(r => ({ ...r, issueId }));
  }, []);

  const navigate = useCallback((view: ViewId, intent?: NavIntent) => {
    publishedOrderRef.current = null;
    if (intent) intentsRef.current.set(view, intent);
    setRoute({ view, issueId: null, cycleId: null, depFocusId: null });
  }, []);

  const nav = useMemo<AppNav>(() => ({
    view: route.view,
    navigate,
    openIssue,
    closeIssue: () => setRoute(r => ({ ...r, issueId: null })),
    newIssue: () => setShowNewIssue(true),
    cycleId: route.cycleId,
    setCycleId: (cycleId) => setRoute(r => ({ ...r, cycleId })),
    depFocusId: route.depFocusId,
    setDepFocusId: (depFocusId) => setRoute(r => ({ ...r, depFocusId })),
    publishNavOrder: (ids) => { publishedOrderRef.current = ids; },
    takeIntent: (view) => {
      const intent = intentsRef.current.get(view);
      intentsRef.current.delete(view);
      return intent;
    },
  }), [route.view, route.cycleId, route.depFocusId, navigate, openIssue]);

  useEffect(() => {
    const hash = formatRoute(route);
    if (window.location.hash !== hash) window.location.hash = hash;
  }, [route]);

  useEffect(() => {
    const label = route.issueId || VIEW_BY_ID[route.view].title;
    document.title = config.projectName ? `${config.projectName} ❯ ${label}` : label;
  }, [route.view, route.issueId, config.projectName]);

  useEffect(() => {
    const onHashChange = () => {
      const prev = routeRef.current;
      const next = parseRoute(window.location.hash);
      if (sameRoute(prev, next)) return;
      if (next.view !== prev.view) publishedOrderRef.current = null;
      if (next.issueId && !prev.issueId) setIssueNavOrder(publishedOrderRef.current);
      setRoute(next);
    };
    window.addEventListener('hashchange', onHashChange);
    return () => window.removeEventListener('hashchange', onHashChange);
  }, []);

  useKeyboardShortcuts({
    scope: 'global.palette',
    priority: 'global',
    shortcuts: [
      { id: 'palette.meta', key: 'k', label: 'Open command palette', group: 'Global', modifiers: { meta: true }, showInHelp: navigator.platform.includes('Mac'), run: () => setShowPalette(true) },
      { id: 'palette.ctrl', key: 'k', label: 'Open command palette', group: 'Global', modifiers: { ctrl: true }, showInHelp: !navigator.platform.includes('Mac'), run: () => setShowPalette(true) },
    ],
  });

  useKeyboardShortcuts({
    scope: 'global.navigation',
    priority: 'global',
    enabled: !showPalette && !showNewIssue,
    shortcuts: [
      { id: 'issue.create', key: 'c', label: 'Create new issue', group: 'Global', run: () => setShowNewIssue(true) },
      ...Object.entries(VIEW_BY_GO_KEY).map(([key, target]) => ({
        id: `navigate.${target}`,
        key: ['g', key] as const,
        label: `Go to ${VIEW_BY_ID[target].title}`,
        group: 'Global',
        run: () => navigate(target),
      })),
    ],
  });

  const labelColors = useMemo(() => effectiveLabelColors(config), [config]);

  return (
    <DefaultLabelsContext.Provider value={config.defaultLabels}>
    <HideDefaultLabelsContext.Provider value={config.hideDefaultLabels}>
    <LabelColorsContext.Provider value={labelColors}>
    <ViewStateContext.Provider value={viewStateStore}>
    <AppNavContext.Provider value={nav}>
      <Layout
        sidebar={{
          activeView: route.view,
          onViewChange: navigate,
          onSearch: () => setShowPalette(true),
          onNewIssue: () => setShowNewIssue(true),
          cyclesEnabled: config.cyclesEnabled,
          inboxUnread,
        }}
        version={config.version}
        connected={!issuesQuery.isError}
      >
        <ViewRouter
          issueId={route.issueId}
          publishedOrder={issueNavOrder}
          onIssueChange={(issueId) => setRoute(r => ({ ...r, issueId }))}
        />
      </Layout>

      <NewIssueModal
        isOpen={showNewIssue}
        onClose={() => setShowNewIssue(false)}
        onCreated={async () => {
          setShowNewIssue(false);
          await refresh();
          showToast("Issue created");
        }}
        issues={issues}
        contributors={config.contributors}
        onConfigLabelsChange={setConfigLabels}
      />

      <CommandPalette
        isOpen={showPalette}
        onClose={() => setShowPalette(false)}
        issues={issues}
        onIssueSelect={(issue) => openIssue(issue.id)}
        onViewChange={navigate}
        onNewIssue={() => setShowNewIssue(true)}
      />
      <KeyboardHelp
        isOpen={showKeyboardHelp}
        onToggle={() => setShowKeyboardHelp((visible) => !visible)}
        onClose={() => setShowKeyboardHelp(false)}
      />
    </AppNavContext.Provider>
    </ViewStateContext.Provider>
    </LabelColorsContext.Provider>
    </HideDefaultLabelsContext.Provider>
    </DefaultLabelsContext.Provider>
  );
}

export default App;
