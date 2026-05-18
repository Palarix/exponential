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
  onSearch: () => void;
  onNewIssue: () => void;
  onPendingClick: () => void;
}

const PRIMARY_NAV: { id: View; label: string; icon: ReactNode }[] = [
  {
    id: 'backlog',
    label: 'Issues',
    icon: (
      <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
        <path strokeLinecap="round" strokeLinejoin="round" d="M8.25 6.75h12M8.25 12h12m-12 5.25h12M3.75 6.75h.007v.008H3.75V6.75zm.375 0a.375.375 0 11-.75 0 .375.375 0 01.75 0zM3.75 12h.007v.008H3.75V12zm.375 0a.375.375 0 11-.75 0 .375.375 0 01.75 0zm-.375 5.25h.007v.008H3.75v-.008zm.375 0a.375.375 0 11-.75 0 .375.375 0 01.75 0z" />
      </svg>
    ),
  },
  {
    id: 'board',
    label: 'Board',
    icon: (
      <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
        <path strokeLinecap="round" strokeLinejoin="round" d="M9 4.5v15m6-15v15m-10.875 0h15.75c.621 0 1.125-.504 1.125-1.125V5.625c0-.621-.504-1.125-1.125-1.125H4.125C3.504 4.5 3 5.004 3 5.625v12.75c0 .621.504 1.125 1.125 1.125z" />
      </svg>
    ),
  },
];

const SECONDARY_NAV: { id: View; label: string; icon: ReactNode }[] = [
  {
    id: 'dashboard',
    label: 'Dashboard',
    icon: (
      <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
        <path strokeLinecap="round" strokeLinejoin="round" d="M3 13.125C3 12.504 3.504 12 4.125 12h2.25c.621 0 1.125.504 1.125 1.125v6.75C7.5 20.496 6.996 21 6.375 21h-2.25A1.125 1.125 0 013 19.875v-6.75zM9.75 8.625c0-.621.504-1.125 1.125-1.125h2.25c.621 0 1.125.504 1.125 1.125v11.25c0 .621-.504 1.125-1.125 1.125h-2.25a1.125 1.125 0 01-1.125-1.125V8.625zM16.5 4.125c0-.621.504-1.125 1.125-1.125h2.25C20.496 3 21 3.504 21 4.125v15.75c0 .621-.504 1.125-1.125 1.125h-2.25a1.125 1.125 0 01-1.125-1.125V4.125z" />
      </svg>
    ),
  },
  {
    id: 'dependencies',
    label: 'Dependencies',
    icon: (
      <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
        <path strokeLinecap="round" strokeLinejoin="round" d="M13.19 8.688a4.5 4.5 0 011.242 7.244l-4.5 4.5a4.5 4.5 0 01-6.364-6.364l1.757-1.757m13.35-.622l1.757-1.757a4.5 4.5 0 00-6.364-6.364l-4.5 4.5a4.5 4.5 0 001.242 7.244" />
      </svg>
    ),
  },
];

function NavItem({ item, isActive, onClick }: { item: { id: View; label: string; icon: ReactNode }; isActive: boolean; onClick: () => void }) {
  return (
    <button
      onClick={onClick}
      className={`
        flex items-center gap-2.5 w-full px-2.5 py-1.5 rounded-[var(--radius-md)]
        text-[13px] transition-colors duration-[var(--duration-fast)]
        ${isActive
          ? 'bg-[var(--color-bg-hover)] text-[var(--color-text-primary)] font-medium'
          : 'text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-bg-hover)]'
        }
      `.trim().replace(/\s+/g, ' ')}
    >
      <span className={isActive ? 'text-[var(--color-text-primary)]' : 'text-[var(--color-text-muted)]'}>
        {item.icon}
      </span>
      {item.label}
    </button>
  );
}

export default function Layout({
  children,
  currentView,
  onViewChange,
  pendingCount,
  onSave,
  onDiscard,
  onSearch,
  onNewIssue,
  onPendingClick,
}: LayoutProps) {
  const [showSaveModal, setShowSaveModal] = useState(false);
  const [commitMessage, setCommitMessage] = useState('');

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
    <div className="flex h-screen overflow-hidden">
      {/* Sidebar */}
      <aside className="w-[220px] flex-shrink-0 bg-[var(--color-bg-sidebar)] border-r border-[var(--color-border-subtle)] flex flex-col select-none">
        {/* Workspace header */}
        <div className="flex items-center gap-2 px-3.5 h-12">
          <img src="/logo-light.svg" alt="Beats" className="w-[18px] h-[18px] opacity-80" />
          <span className="font-semibold text-[var(--color-text-primary)] text-[14px] tracking-tight flex-1">
            Beats
          </span>
          <button
            onClick={onSearch}
            className="p-1 rounded-[var(--radius-sm)] text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-bg-hover)] transition-colors"
            title="Search issues"
          >
            <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M21 21l-5.197-5.197m0 0A7.5 7.5 0 105.196 5.196a7.5 7.5 0 0010.607 10.607z" />
            </svg>
          </button>
          <button
            onClick={onNewIssue}
            className="p-1 rounded-[var(--radius-sm)] text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-bg-hover)] transition-colors"
            title="New issue"
          >
            <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M16.862 4.487l1.687-1.688a1.875 1.875 0 112.652 2.652L10.582 16.07a4.5 4.5 0 01-1.897 1.13L6 18l.8-2.685a4.5 4.5 0 011.13-1.897l8.932-8.931zm0 0L19.5 7.125M18 14v4.75A2.25 2.25 0 0115.75 21H5.25A2.25 2.25 0 013 18.75V8.25A2.25 2.25 0 015.25 6H10" />
            </svg>
          </button>
        </div>

        {/* Primary nav */}
        <nav className="px-2 space-y-0.5">
          {PRIMARY_NAV.map((item) => (
            <NavItem key={item.id} item={item} isActive={currentView === item.id} onClick={() => onViewChange(item.id)} />
          ))}
        </nav>

        {/* Section label */}
        <div className="px-4 pt-5 pb-1">
          <span className="text-[11px] font-medium text-[var(--color-text-muted)] uppercase tracking-wider">Insights</span>
        </div>

        {/* Secondary nav */}
        <nav className="px-2 space-y-0.5">
          {SECONDARY_NAV.map((item) => (
            <NavItem key={item.id} item={item} isActive={currentView === item.id} onClick={() => onViewChange(item.id)} />
          ))}
        </nav>

        {/* Spacer */}
        <div className="flex-1" />

        {/* Pending footer */}
        <div className="px-3 py-3 border-t border-[var(--color-border-subtle)]">
          {pendingCount > 0 ? (
            <div className="space-y-2">
              <button
                onClick={onPendingClick}
                className="flex items-center gap-2 px-0.5 w-full text-left hover:opacity-80 transition-opacity"
              >
                <span className="w-1.5 h-1.5 rounded-full bg-[var(--color-warning)]" />
                <span className="text-[11px] text-[var(--color-warning)]">
                  {pendingCount} unsaved change{pendingCount !== 1 ? 's' : ''}
                </span>
              </button>
              <div className="flex gap-1.5">
                <button
                  onClick={onDiscard}
                  className="flex-1 px-2 py-1 text-[11px] text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-bg-hover)] rounded-[var(--radius-sm)] transition-colors"
                >
                  Discard
                </button>
                <button
                  onClick={handleSave}
                  className="flex-1 px-2 py-1 text-[11px] text-white bg-[var(--color-accent-primary)] hover:bg-[var(--color-accent-primary-hover)] rounded-[var(--radius-sm)] transition-colors"
                >
                  Save
                </button>
              </div>
            </div>
          ) : (
            <div className="flex items-center gap-1.5 px-0.5 text-[11px] text-[var(--color-text-muted)]">
              <svg className="w-3 h-3 text-[var(--color-success)]" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
              </svg>
              All synced
            </div>
          )}
        </div>
      </aside>

      {/* Main */}
      <main className="flex-1 overflow-hidden bg-[var(--color-bg-primary)]">
        {children}
      </main>

      {/* Save modal */}
      <Modal
        isOpen={showSaveModal}
        onClose={() => setShowSaveModal(false)}
        title="Commit Changes"
        size="md"
      >
        <div className="space-y-3">
          <p className="text-[13px] text-[var(--color-text-secondary)]">
            Enter a commit message for your changes.
          </p>
          <input
            type="text"
            value={commitMessage}
            onChange={(e) => setCommitMessage(e.target.value)}
            className="w-full h-9 px-3 bg-[var(--color-bg-tertiary)] border border-[var(--color-border-default)] rounded-[var(--radius-md)] text-[13px] text-[var(--color-text-primary)] placeholder:text-[var(--color-text-muted)] focus:outline-none focus:border-[var(--color-border-focus)] transition-colors"
            placeholder="e.g., Daily standup updates..."
            autoFocus
            onKeyDown={(e) => { if (e.key === 'Enter' && commitMessage.trim()) confirmSave(); }}
          />
        </div>
        <ModalFooter>
          <Button variant="ghost" size="sm" onClick={() => setShowSaveModal(false)}>Cancel</Button>
          <Button size="sm" onClick={confirmSave} disabled={!commitMessage.trim()}>Commit</Button>
        </ModalFooter>
      </Modal>
    </div>
  );
}
