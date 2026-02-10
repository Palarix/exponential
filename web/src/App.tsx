import { useState, useEffect, useCallback } from 'react';
import { getIssues, getPending, save, discardPending } from './api/client';
import type { Issue, PendingState } from './api/client';
import Layout from './components/Layout/Layout';
import Dashboard from './components/Dashboard/Dashboard';
import Backlog from './components/Backlog/Backlog';
import Board from './components/Board/Board';
import Dependencies from './components/Dependencies/Dependencies';
import IssueDetailModal from './components/IssueDetailModal';
import PendingEventsPanel from './components/PendingEventsPanel';
import './index.css';

type View = 'dashboard' | 'backlog' | 'board' | 'dependencies';

function App() {
  const [view, setView] = useState<View>('board');
  const [issues, setIssues] = useState<Issue[]>([]);
  const [pending, setPending] = useState<PendingState>({ count: 0 });
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Modal states
  const [selectedIssue, setSelectedIssue] = useState<Issue | null>(null);
  const [showPendingPanel, setShowPendingPanel] = useState(false);

  const fetchData = useCallback(async () => {
    try {
      const [issuesData, pendingData] = await Promise.all([
        getIssues(),
        getPending(),
      ]);
      setIssues(issuesData);
      setPending(pendingData);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch data');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchData();
    // Poll every 5 seconds
    const interval = setInterval(fetchData, 5000);
    return () => clearInterval(interval);
  }, [fetchData]);

  const handleSave = async (commitMessage: string) => {
    try {
      await save(commitMessage);
      setShowPendingPanel(false);
      await fetchData();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save');
    }
  };

  const handleDiscard = async () => {
    try {
      await discardPending();
      setShowPendingPanel(false);
      await fetchData();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to discard');
    }
  };

  const handleIssueClick = (issue: Issue) => {
    setSelectedIssue(issue);
  };

  const handleSaveFromPanel = () => {
    const now = new Date().toISOString().split('T')[0];
    handleSave(`Update ${now}`);
  };

  const renderView = () => {
    switch (view) {
      case 'dashboard':
        return <Dashboard issues={issues} />;
      case 'backlog':
        return <Backlog issues={issues} onRefresh={fetchData} onIssueClick={handleIssueClick} />;
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
        onViewChange={setView}
        pendingCount={pending.count}
        onSave={handleSave}
        onDiscard={handleDiscard}
      >
        {loading ? (
          <div className="flex flex-col items-center justify-center h-64 gap-4">
            <div className="w-8 h-8 border-2 border-[var(--color-accent-primary)] border-t-transparent rounded-full animate-spin" />
            <p className="text-[var(--color-text-muted)] text-sm">Loading issues...</p>
          </div>
        ) : error ? (
          <div className="flex flex-col items-center justify-center h-64 gap-4">
            <div className="w-16 h-16 rounded-full bg-[var(--color-error-bg)] flex items-center justify-center">
              <svg className="w-8 h-8 text-[var(--color-error)]" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
              </svg>
            </div>
            <div className="text-center">
              <p className="text-[var(--color-error)] font-medium">Error loading data</p>
              <p className="text-[var(--color-text-muted)] text-sm mt-1">{error}</p>
            </div>
            <button
              onClick={fetchData}
              className="px-4 py-2 text-sm bg-[var(--color-surface-elevated)] rounded-[var(--radius-md)] text-[var(--color-text-primary)] hover:bg-[var(--color-bg-hover)] transition-colors"
            >
              Retry
            </button>
          </div>
        ) : (
          renderView()
        )}
      </Layout>

      {/* Issue Detail Modal */}
      <IssueDetailModal
        issue={selectedIssue}
        isOpen={selectedIssue !== null}
        onClose={() => setSelectedIssue(null)}
        onRefresh={fetchData}
      />

      {/* Pending Events Panel */}
      <PendingEventsPanel
        pending={pending}
        isOpen={showPendingPanel}
        onToggle={() => setShowPendingPanel(!showPendingPanel)}
        onSave={handleSaveFromPanel}
        onDiscard={handleDiscard}
      />
    </>
  );
}

export default App;
