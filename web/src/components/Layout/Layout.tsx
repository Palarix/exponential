import { useState } from 'react';
import type { ReactNode } from 'react';

type View = 'dashboard' | 'backlog' | 'board' | 'dependencies';

interface LayoutProps {
  children: ReactNode;
  currentView: View;
  onViewChange: (view: View) => void;
  pendingCount: number;
  onSave: (commitMessage: string) => void;
  onDiscard: () => void;
}

export default function Layout({
  children,
  currentView,
  onViewChange,
  pendingCount,
  onSave,
  onDiscard,
}: LayoutProps) {
  const [showSaveModal, setShowSaveModal] = useState(false);
  const [commitMessage, setCommitMessage] = useState('');

  const views: { id: View; label: string }[] = [
    { id: 'dashboard', label: 'Dashboard' },
    { id: 'backlog', label: 'Backlog' },
    { id: 'board', label: 'Board' },
    { id: 'dependencies', label: 'Dependencies' },
  ];

  const handleSave = () => {
    const now = new Date().toISOString().split('T')[0];
    setCommitMessage(`Update ${now}`);
    setShowSaveModal(true);
  };

  const confirmSave = () => {
    onSave(commitMessage);
    setShowSaveModal(false);
    setCommitMessage('');
  };

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Top Navigation */}
      <header className="bg-white border-b border-gray-200 sticky top-0 z-50">
        <div className="flex items-center justify-between px-4 h-14">
          {/* Logo */}
          <div className="flex items-center gap-2">
            <span className="text-xl">🎵</span>
            <span className="font-semibold text-gray-900">Beats</span>
          </div>

          {/* View Tabs */}
          <nav className="flex gap-1">
            {views.map((v) => (
              <button
                key={v.id}
                onClick={() => onViewChange(v.id)}
                className={`px-4 py-2 rounded-lg text-sm font-medium transition-colors ${currentView === v.id
                  ? 'bg-blue-100 text-blue-700'
                  : 'text-gray-600 hover:bg-gray-100'
                  }`}
              >
                {v.label}
              </button>
            ))}
          </nav>

          {/* Actions */}
          <div className="flex items-center gap-2">
            {pendingCount > 0 && (
              <>
                <span className="text-sm text-amber-600 bg-amber-50 px-2 py-1 rounded">
                  {pendingCount} unsaved change{pendingCount !== 1 ? 's' : ''}
                </span>
                <button
                  onClick={onDiscard}
                  className="px-3 py-1.5 text-sm text-gray-600 hover:bg-gray-100 rounded-lg"
                >
                  Discard
                </button>
                <button
                  onClick={handleSave}
                  className="px-3 py-1.5 text-sm bg-blue-600 text-white rounded-lg hover:bg-blue-700"
                >
                  Save & Sync
                </button>
              </>
            )}
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="p-6">{children}</main>

      {/* Save Modal */}
      {showSaveModal && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg p-6 w-full max-w-md">
            <h2 className="text-lg font-semibold mb-4">Commit Message</h2>
            <input
              type="text"
              value={commitMessage}
              onChange={(e) => setCommitMessage(e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg mb-4 focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="Enter commit message..."
              autoFocus
            />
            <div className="flex justify-end gap-2">
              <button
                onClick={() => setShowSaveModal(false)}
                className="px-4 py-2 text-gray-600 hover:bg-gray-100 rounded-lg"
              >
                Cancel
              </button>
              <button
                onClick={confirmSave}
                className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
              >
                Save
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
