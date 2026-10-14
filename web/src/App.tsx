import { useState, useEffect, useCallback, useRef, useMemo } from 'react';
import Layout from './components/Layout/Layout';
import ViewRouter from './components/Layout/ViewRouter';
import CommandPalette from './components/CommandPalette/CommandPalette';
import NewIssueModal from './components/NewIssueModal/NewIssueModal';
import KeyboardHelp from './components/KeyboardHelp/KeyboardHelp';
import AgentChatProvider from './components/Agent/AgentChatProvider';
import TabStatus from './components/Agent/TabStatus';
import { chatContextFor } from './components/Agent/agent-state-utils';
import { LabelColorsContext, HideDefaultLabelsContext, DefaultLabelsContext, useToast } from './components/ui';
import { useSSE } from './hooks/useSSE';
import { useKeyboardShortcuts } from './keyboard';
import { useConfig, useInbox, useInvalidateOnServerEvent, useIssueDrafts, useIssueList, useIssues } from './api/queries';
import type { IssueDraft } from './api/types';
import { effectiveLabelColors } from './api/query-utils';
import { AppNavContext, ViewStateContext, type AppNav, type NavIntent } from './app/contexts';
import { createViewStateStore } from './app/view-state-utils';
import { VIEW_BY_GO_KEY, VIEW_BY_ID, formatRoute, parseRoute, type Route, type ViewId } from './views';

function safeLocalStorage(): Storage | null {
  try { return window.localStorage; } catch { return null; }
}

function sameRoute(a: Route, b: Route): boolean {
  return a.view === b.view && a.issueId === b.issueId && a.cycleId === b.cycleId && a.depFocusId === b.depFocusId && a.chatId === b.chatId;
}

function App() {
  const [route, setRoute] = useState<Route>(() => parseRoute(window.location.hash));
  const routeRef = useRef(route);
  // The last route outside the Agent view, where docking a chat returns to.
  const outsideAgentRef = useRef<Route | null>(route.view === 'agent' ? null : route);
  useEffect(() => {
    routeRef.current = route;
    if (route.view !== 'agent') outsideAgentRef.current = route;
  }, [route]);

  // The active view publishes its visible order into the ref; it's snapshotted
  // into state when an issue opens, so prev/next follows the originating view.
  const publishedOrderRef = useRef<string[] | null>(null);
  const [issueNavOrder, setIssueNavOrder] = useState<string[] | null>(null);
  const intentsRef = useRef(new Map<ViewId, NavIntent>());
  const [viewStateStore] = useState(() => createViewStateStore(safeLocalStorage()));

  const [showNewIssue, setShowNewIssue] = useState(false);
  // Each open remounts the modal, so its form starts from the draft (or empty).
  const [newIssueSession, setNewIssueSession] = useState<{ key: number; draft: IssueDraft | null }>({ key: 0, draft: null });
  const openNewIssue = useCallback((draft: IssueDraft | null = null) => {
    setNewIssueSession(s => ({ key: s.key + 1, draft }));
    setShowNewIssue(true);
  }, []);
  const [showPalette, setShowPalette] = useState(false);
  const [showKeyboardHelp, setShowKeyboardHelp] = useState(false);

  const issuesQuery = useIssues();
  const issues = useIssueList();
  const config = useConfig();
  const { unread: inboxUnread } = useInbox();
  const draftCount = useIssueDrafts().length;
  const showToast = useToast();

  useSSE({ onEvent: useInvalidateOnServerEvent() });

  const openIssue = useCallback((issueId: string) => {
    setIssueNavOrder(publishedOrderRef.current);
    setRoute(r => ({ ...r, issueId }));
  }, []);

  const navigate = useCallback((view: ViewId, intent?: NavIntent) => {
    publishedOrderRef.current = null;
    if (intent) intentsRef.current.set(view, intent);
    setRoute({ view, issueId: null, cycleId: null, depFocusId: null, chatId: null });
  }, []);

  const nav = useMemo<AppNav>(() => ({
    view: route.view,
    navigate,
    openIssue,
    closeIssue: () => setRoute(r => ({ ...r, issueId: null })),
    newIssue: () => openNewIssue(),
    openDraft: (draft) => openNewIssue(draft),
    cycleId: route.cycleId,
    setCycleId: (cycleId) => setRoute(r => ({ ...r, cycleId })),
    depFocusId: route.depFocusId,
    setDepFocusId: (depFocusId) => setRoute(r => ({ ...r, depFocusId })),
    chatId: route.chatId,
    openChatView: (chatId) => {
      publishedOrderRef.current = null;
      setRoute({ view: 'agent', issueId: null, cycleId: null, depFocusId: null, chatId });
    },
    leaveChatView: () => {
      publishedOrderRef.current = null;
      setRoute(outsideAgentRef.current ?? parseRoute(''));
    },
    publishNavOrder: (ids) => { publishedOrderRef.current = ids; },
    takeIntent: (view) => {
      const intent = intentsRef.current.get(view);
      intentsRef.current.delete(view);
      return intent;
    },
  }), [route.view, route.cycleId, route.depFocusId, route.chatId, navigate, openIssue, openNewIssue]);

  useEffect(() => {
    const hash = formatRoute(route);
    if (window.location.hash !== hash) window.location.hash = hash;
  }, [route]);

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
      { id: 'issue.create', key: 'c', label: 'Create new issue', group: 'Global', run: () => openNewIssue() },
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
  const tabLabel = route.issueId || VIEW_BY_ID[route.view].title;
  const tabTitle = config.projectName ? `${config.projectName} ❯ ${tabLabel}` : tabLabel;
  const chatContext = useMemo(() => chatContextFor(route.view, route.issueId, issues), [route.view, route.issueId, issues]);

  return (
    <DefaultLabelsContext.Provider value={config.defaultLabels}>
    <HideDefaultLabelsContext.Provider value={config.hideDefaultLabels}>
    <LabelColorsContext.Provider value={labelColors}>
    <ViewStateContext.Provider value={viewStateStore}>
    <AppNavContext.Provider value={nav}>
    <AgentChatProvider context={chatContext}>
      <TabStatus title={tabTitle} />
      <Layout
        sidebar={{
          activeView: route.view,
          onViewChange: navigate,
          onSearch: () => setShowPalette(true),
          onNewIssue: () => openNewIssue(),
          cyclesEnabled: config.cyclesEnabled,
          inboxUnread,
          draftCount,
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
        key={newIssueSession.key}
        draft={newIssueSession.draft}
        isOpen={showNewIssue}
        onClose={() => setShowNewIssue(false)}
        onCreated={({ draftDeleted }) => {
          setShowNewIssue(false);
          if (draftDeleted) showToast("Issue created");
          else showToast("Issue created, but its draft could not be deleted", { variant: "error" });
        }}
        onDraftSaved={() => {
          setShowNewIssue(false);
          showToast("Draft saved");
        }}
        issues={issues}
        contributors={config.contributors}
      />

      <CommandPalette
        isOpen={showPalette}
        onClose={() => setShowPalette(false)}
        issues={issues}
        onIssueSelect={(issue) => openIssue(issue.id)}
        onViewChange={navigate}
        onNewIssue={() => openNewIssue()}
      />
      <KeyboardHelp
        isOpen={showKeyboardHelp}
        onToggle={() => setShowKeyboardHelp((visible) => !visible)}
        onClose={() => setShowKeyboardHelp(false)}
      />
    </AgentChatProvider>
    </AppNavContext.Provider>
    </ViewStateContext.Provider>
    </LabelColorsContext.Provider>
    </HideDefaultLabelsContext.Provider>
    </DefaultLabelsContext.Provider>
  );
}

export default App;
