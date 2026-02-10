import { useState } from 'react';
import type { ReactNode } from 'react';
import { Button, Modal, ModalFooter } from '../ui';

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

  const views: { id: View; label: string; icon: ReactNode }[] = [
    {
      id: 'dashboard',
      label: 'Dashboard',
      icon: (
        <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
          <path strokeLinecap="round" strokeLinejoin="round" d="M4 5a1 1 0 011-1h4a1 1 0 011 1v5a1 1 0 01-1 1H5a1 1 0 01-1-1V5zM14 5a1 1 0 011-1h4a1 1 0 011 1v5a1 1 0 01-1 1h-4a1 1 0 01-1-1V5zM4 15a1 1 0 011-1h4a1 1 0 011 1v4a1 1 0 01-1 1H5a1 1 0 01-1-1v-4zM14 15a1 1 0 011-1h4a1 1 0 011 1v4a1 1 0 01-1 1h-4a1 1 0 01-1-1v-4z" />
        </svg>
      )
    },
    {
      id: 'backlog',
      label: 'Backlog',
      icon: (
        <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
          <path strokeLinecap="round" strokeLinejoin="round" d="M4 6h16M4 10h16M4 14h16M4 18h16" />
        </svg>
      )
    },
    {
      id: 'board',
      label: 'Board',
      icon: (
        <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
          <path strokeLinecap="round" strokeLinejoin="round" d="M9 17V7m0 10a2 2 0 01-2 2H5a2 2 0 01-2-2V7a2 2 0 012-2h2a2 2 0 012 2m0 10a2 2 0 002 2h2a2 2 0 002-2M9 7a2 2 0 012-2h2a2 2 0 012 2m0 10V7m0 10a2 2 0 002 2h2a2 2 0 002-2V7a2 2 0 00-2-2h-2a2 2 0 00-2 2" />
        </svg>
      )
    },
    {
      id: 'dependencies',
      label: 'Dependencies',
      icon: (
        <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
          <path strokeLinecap="round" strokeLinejoin="round" d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1" />
        </svg>
      )
    },
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
    <div className="min-h-screen">
      {/* Top Navigation */}
      <header className="sticky top-0 z-40 border-b border-[var(--color-border-subtle)]">
        <div className="glass-heavy">
          <div className="flex items-center justify-between px-6 h-16">
            {/* Logo */}
            <div className="flex items-center gap-3">
              <img
                src="/logo-light.svg"
                alt="Beats"
                className="w-8 h-8"
              />
              <span className="font-semibold text-[var(--color-text-primary)] text-lg tracking-tight">
                Beats
              </span>
            </div>

            {/* View Tabs */}
            <nav className="flex gap-1 bg-[var(--color-bg-tertiary)]/50 p-1 rounded-[var(--radius-lg)]">
              {views.map((v) => (
                <button
                  key={v.id}
                  onClick={() => onViewChange(v.id)}
                  className={`
                    flex items-center gap-2 px-4 py-2 rounded-[var(--radius-md)] 
                    text-sm font-medium transition-all duration-[var(--duration-fast)]
                    ${currentView === v.id
                      ? 'bg-[var(--color-accent-primary)] text-white shadow-[var(--shadow-glow-sm)]'
                      : 'text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-bg-hover)]'
                    }
                  `.trim().replace(/\s+/g, ' ')}
                >
                  {v.icon}
                  <span className="hidden sm:inline">{v.label}</span>
                </button>
              ))}
            </nav>

            {/* Actions */}
            <div className="flex items-center gap-3">
              {pendingCount > 0 && (
                <>
                  <div className="flex items-center gap-2 px-3 py-1.5 rounded-[var(--radius-md)] bg-[var(--color-warning-bg)] border border-[var(--color-warning)]/30">
                    <span className="relative flex h-2 w-2">
                      <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-[var(--color-warning)] opacity-75"></span>
                      <span className="relative inline-flex rounded-full h-2 w-2 bg-[var(--color-warning)]"></span>
                    </span>
                    <span className="text-sm font-medium text-[var(--color-warning)]">
                      {pendingCount} unsaved
                    </span>
                  </div>
                  <Button variant="ghost" size="sm" onClick={onDiscard}>
                    Discard
                  </Button>
                  <Button size="sm" onClick={handleSave}>
                    Save & Sync
                  </Button>
                </>
              )}
              {pendingCount === 0 && (
                <div className="flex items-center gap-2 text-sm text-[var(--color-text-muted)]">
                  <svg className="w-4 h-4 text-[var(--color-success)]" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                    <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
                  </svg>
                  All changes synced
                </div>
              )}
            </div>
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="p-6 max-w-[1800px] mx-auto">{children}</main>

      {/* Save Modal */}
      <Modal
        isOpen={showSaveModal}
        onClose={() => setShowSaveModal(false)}
        title="Commit Changes"
        size="md"
      >
        <div className="space-y-4">
          <p className="text-sm text-[var(--color-text-secondary)]">
            Enter a commit message to describe your changes.
          </p>
          <input
            type="text"
            value={commitMessage}
            onChange={(e) => setCommitMessage(e.target.value)}
            className="
              w-full px-4 py-3 
              bg-[var(--color-bg-tertiary)] 
              border border-[var(--color-border-default)]
              rounded-[var(--radius-md)]
              text-[var(--color-text-primary)]
              placeholder:text-[var(--color-text-muted)]
              focus:outline-none focus:border-[var(--color-accent-primary)]
              focus:ring-2 focus:ring-[var(--color-accent-primary)]/20
              transition-all duration-[var(--duration-fast)]
            "
            placeholder="e.g., Daily standup updates..."
            autoFocus
            onKeyDown={(e) => {
              if (e.key === 'Enter' && commitMessage.trim()) {
                confirmSave();
              }
            }}
          />
        </div>
        <ModalFooter>
          <Button variant="ghost" onClick={() => setShowSaveModal(false)}>
            Cancel
          </Button>
          <Button onClick={confirmSave} disabled={!commitMessage.trim()}>
            Save & Commit
          </Button>
        </ModalFooter>
      </Modal>
    </div>
  );
}
