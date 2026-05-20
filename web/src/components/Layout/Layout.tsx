import type { ReactNode } from 'react';

type View = 'dashboard' | 'backlog' | 'board' | 'dependencies';

interface LayoutProps {
  children: ReactNode;
  currentView: View;
  onViewChange: (view: View) => void;
  onSearch: () => void;
  onNewIssue: () => void;
  version: string;
  connected: boolean;
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
    label: 'Analytics',
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
        text-sm transition-colors duration-[var(--duration-fast)]
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
  onSearch,
  onNewIssue,
  version,
  connected,
}: LayoutProps) {
  return (
    <div className="flex h-screen overflow-hidden bg-[var(--color-bg-sidebar)]">
      {/* Sidebar */}
      <aside className="w-[245px] flex-shrink-0 bg-[var(--color-bg-sidebar)] flex flex-col select-none">
        {/* Workspace header */}
        <div className="flex items-center gap-2 px-3.5 pt-2 h-[52px]">
          <img src="/logo-light.svg" alt="Beats" className="w-[18px] h-[18px] opacity-80" />
          <span className="font-semibold text-[var(--color-text-primary)] text-base tracking-tight flex-1">
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

        {/* Nav */}
        <nav className="px-2 pt-1 space-y-0.5">
          {PRIMARY_NAV.map((item) => (
            <NavItem key={item.id} item={item} isActive={currentView === item.id} onClick={() => onViewChange(item.id)} />
          ))}
          {SECONDARY_NAV.map((item) => (
            <NavItem key={item.id} item={item} isActive={currentView === item.id} onClick={() => onViewChange(item.id)} />
          ))}
        </nav>

      </aside>

      {/* Main */}
      <div className="flex-1 pt-2 pr-2 overflow-hidden flex flex-col">
        <main className="flex-1 overflow-hidden bg-[var(--color-bg-primary)] rounded-t-[var(--radius-lg)] border border-[var(--color-border-subtle)] border-b-0">
          {children}
        </main>
        <div className="flex items-center justify-end px-4 py-2 shrink-0">
          <div className="flex items-center gap-1.5 text-xs">
            <span className={`w-1.5 h-1.5 rounded-full ${connected ? 'bg-[var(--color-success)]' : 'bg-[var(--color-error)]'}`} />
            {connected ? (
              <span className="text-[var(--color-text-muted)]">Beats {version && `v${version}`}</span>
            ) : (
              <span className="text-[var(--color-error)]">Disconnected</span>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
