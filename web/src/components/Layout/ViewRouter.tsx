import { lazy, Suspense, useMemo, type ComponentType, type LazyExoticComponent } from "react";
import { useConfig, useIssueList, useIssues, useRefreshIssues } from "../../api/queries";
import { useAppNav, useViewState } from "../../app/hooks";
import { resolveNavOrder } from "../../app/nav-order-utils";
import { ErrorBoundary } from "../ui";
import { sortIssuesWithinGroups, SORT_CODEC, SORT_STATE_KEY } from "../../utils/sort";
import type { ViewId } from "../../views";
import Backlog from "../Backlog/Backlog";

const IssueDetail = lazy(() => import("../IssueDetail/IssueDetail"));

const VIEW_COMPONENTS: Record<ViewId, ComponentType | LazyExoticComponent<ComponentType>> = {
  dashboard: lazy(() => import("../Dashboard/Dashboard")),
  inbox: lazy(() => import("../Inbox/Inbox")),
  backlog: Backlog,
  board: lazy(() => import("../Board/Board")),
  cycles: lazy(() => import("../Cycles/Cycles")),
  dependencies: lazy(() => import("../Dependencies/Dependencies")),
  labels: lazy(() => import("../Labels/Labels")),
  "my-issues": lazy(() => import("../MyIssues/MyIssues")),
  timeline: lazy(() => import("../Timeline/Timeline")),
};

interface ViewRouterProps {
  issueId: string | null;
  /** Order published by the view the issue was opened from, if any. */
  publishedOrder: string[] | null;
  onIssueChange: (issueId: string) => void;
}

/** Renders the global loading/error states, IssueDetail, or the active view. */
export default function ViewRouter({ issueId, publishedOrder, onIssueChange }: ViewRouterProps) {
  const { view, closeIssue } = useAppNav();
  const issuesQuery = useIssues();
  const refresh = useRefreshIssues();

  return (
    <ErrorBoundary onReset={refresh}>
      <Suspense fallback={null}>
        {issuesQuery.isPending ? (
          <Loading />
        ) : issuesQuery.isError ? (
          <ServerOffline onRetry={() => issuesQuery.refetch()} />
        ) : issueId ? (
          <IssueRoute issueId={issueId} publishedOrder={publishedOrder} onIssueChange={onIssueChange} onClose={closeIssue} />
        ) : (
          <ActiveView view={view} />
        )}
      </Suspense>
    </ErrorBoundary>
  );
}

function ActiveView({ view }: { view: ViewId }) {
  const View = VIEW_COMPONENTS[view];
  return <View />;
}

function IssueRoute({ issueId, publishedOrder, onIssueChange, onClose }: {
  issueId: string;
  publishedOrder: string[] | null;
  onIssueChange: (issueId: string) => void;
  onClose: () => void;
}) {
  const issues = useIssueList();
  const { prefix, contributors } = useConfig();
  const [sortKey] = useViewState(SORT_STATE_KEY, SORT_CODEC);
  const fallbackOrder = useMemo(() => sortIssuesWithinGroups(issues, sortKey).map(i => i.id), [issues, sortKey]);
  const navigationOrder = resolveNavOrder(publishedOrder, fallbackOrder, issueId);

  const issue = issues.find(i => i.id === issueId) ?? issues.find(i => i.id.endsWith(issueId)) ?? null;
  if (!issue) return <IssueNotFound issueId={issueId} onBack={onClose} />;

  const currentIndex = navigationOrder.indexOf(issueId);
  const navigate = (direction: "prev" | "next") => {
    if (currentIndex === -1) return;
    const next = direction === "prev" ? currentIndex - 1 : currentIndex + 1;
    if (next >= 0 && next < navigationOrder.length) onIssueChange(navigationOrder[next]);
  };

  return (
    <IssueDetail
      issue={issue}
      issues={issues}
      currentIndex={currentIndex}
      totalCount={navigationOrder.length}
      onClose={onClose}
      onNavigate={navigate}
      prefix={prefix}
      contributors={contributors}
    />
  );
}

function Loading() {
  return (
    <div className="flex flex-col items-center justify-center h-full gap-3">
      <div className="w-5 h-5 border-2 border-[var(--color-text-muted)] border-t-transparent rounded-full animate-spin" />
      <p className="text-[var(--color-text-muted)] text-sm">Loading...</p>
    </div>
  );
}

function ServerOffline({ onRetry }: { onRetry: () => void }) {
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
        onClick={onRetry}
        className="px-4 py-2 text-sm bg-[var(--color-surface-1)] border border-[var(--color-border-default)] rounded-[var(--radius-md)] text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-hover-surface)] transition-colors"
      >
        Retry Connection
      </button>
    </div>
  );
}

function IssueNotFound({ issueId, onBack }: { issueId: string; onBack: () => void }) {
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
          No issue matching <span className="font-mono text-[var(--color-text-primary)]">{issueId}</span> was found. It may have been deleted or the link may be incorrect.
        </p>
      </div>
      <button onClick={onBack} className="px-4 py-2 text-sm bg-[var(--color-accent-primary)] text-white rounded-[var(--radius-md)] hover:opacity-90 transition-opacity">
        Back to Backlog
      </button>
    </div>
  );
}
