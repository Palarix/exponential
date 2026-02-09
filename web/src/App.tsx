import { useState, useEffect, useCallback } from 'react';
import { getIssues, getPending, save, discardPending } from './api/client';
import type { Issue, PendingState } from './api/client';
import Layout from './components/Layout/Layout';
import Dashboard from './components/Dashboard/Dashboard';
import Backlog from './components/Backlog/Backlog';
import Board from './components/Board/Board';
import Dependencies from './components/Dependencies/Dependencies';
import './index.css';

type View = 'dashboard' | 'backlog' | 'board' | 'dependencies';

function App() {
  const [view, setView] = useState<View>('board');
  const [issues, setIssues] = useState<Issue[]>([]);
  const [pending, setPending] = useState<PendingState>({ count: 0 });
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

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
      await fetchData();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save');
    }
  };

  const handleDiscard = async () => {
    try {
      await discardPending();
      await fetchData();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to discard');
    }
  };

  const renderView = () => {
    switch (view) {
      case 'dashboard':
        return <Dashboard issues={issues} />;
      case 'backlog':
        return <Backlog issues={issues} onRefresh={fetchData} />;
      case 'board':
        return <Board issues={issues} onRefresh={fetchData} />;
      case 'dependencies':
        return <Dependencies issues={issues} />;
    }
  };

  return (
    <Layout
      currentView={view}
      onViewChange={setView}
      pendingCount={pending.count}
      onSave={handleSave}
      onDiscard={handleDiscard}
    >
      {loading ? (
        <div className="flex items-center justify-center h-64">
          <div className="text-gray-500">Loading...</div>
        </div>
      ) : error ? (
        <div className="flex items-center justify-center h-64">
          <div className="text-red-500">Error: {error}</div>
        </div>
      ) : (
        renderView()
      )}
    </Layout>
  );
}

export default App;
