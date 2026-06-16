import { useEffect, useState, type ReactNode } from "react";
import {
  fetchInstances,
  fetchUser,
  type Instance,
  type User,
} from "../../api/client";
import { Avatar } from "../ui";

type View = "dashboard" | "inbox" | "backlog" | "board" | "cycles" | "dependencies" | "labels";

interface LayoutProps {
  children: ReactNode;
  currentView: View;
  onViewChange: (view: View) => void;
  onSearch: () => void;
  onNewIssue: () => void;
  version: string;
  connected: boolean;
  cyclesEnabled?: boolean;
  inboxUnread?: number;
  statusBarLeft?: ReactNode;
}

const PRIMARY_NAV: { id: View; label: string; icon: ReactNode }[] = [
  {
    id: "dashboard",
    label: "Overview",
    icon: (
      <svg
        className="w-4 h-4"
        fill="none"
        viewBox="0 0 24 24"
        stroke="currentColor"
        strokeWidth={1.5}
      >
        <path
          strokeLinecap="round"
          strokeLinejoin="round"
          d="M4 5.5h16v3H4zM4 10.5h16v3H4zM4 15.5h16v3H4z"
        />
      </svg>
    ),
  },
  {
    id: "backlog",
    label: "Issues",
    icon: (
      <svg
        className="w-4 h-4"
        fill="none"
        viewBox="0 0 24 24"
        stroke="currentColor"
        strokeWidth={1.5}
      >
        <path
          strokeLinecap="round"
          strokeLinejoin="round"
          d="M8.25 6.75h12M8.25 12h12m-12 5.25h12M3.75 6.75h.007v.008H3.75V6.75zm.375 0a.375.375 0 11-.75 0 .375.375 0 01.75 0zM3.75 12h.007v.008H3.75V12zm.375 0a.375.375 0 11-.75 0 .375.375 0 01.75 0zm-.375 5.25h.007v.008H3.75v-.008zm.375 0a.375.375 0 11-.75 0 .375.375 0 01.75 0z"
        />
      </svg>
    ),
  },
  {
    id: "board",
    label: "Board",
    icon: (
      <svg
        className="w-4 h-4"
        fill="none"
        viewBox="0 0 24 24"
        stroke="currentColor"
        strokeWidth={1.5}
      >
        <path
          strokeLinecap="round"
          strokeLinejoin="round"
          d="M9 4.5v15m6-15v15m-10.875 0h15.75c.621 0 1.125-.504 1.125-1.125V5.625c0-.621-.504-1.125-1.125-1.125H4.125C3.504 4.5 3 5.004 3 5.625v12.75c0 .621.504 1.125 1.125 1.125z"
        />
      </svg>
    ),
  },
];

const SECONDARY_NAV: { id: View; label: string; icon: ReactNode }[] = [
  {
    id: "labels",
    label: "Labels",
    icon: (
      <svg
        className="w-4 h-4"
        fill="none"
        viewBox="0 0 24 24"
        stroke="currentColor"
        strokeWidth={1.5}
      >
        <path
          strokeLinecap="round"
          strokeLinejoin="round"
          d="M9.568 3H5.25A2.25 2.25 0 003 5.25v4.318c0 .597.237 1.17.659 1.591l9.581 9.581c.699.699 1.78.872 2.607.33a18.095 18.095 0 005.223-5.223c.542-.827.369-1.908-.33-2.607L11.16 3.66A2.25 2.25 0 009.568 3z"
        />
        <path
          strokeLinecap="round"
          strokeLinejoin="round"
          d="M6 6h.008v.008H6V6z"
        />
      </svg>
    ),
  },
  {
    id: "dependencies",
    label: "Dependencies",
    icon: (
      <svg
        className="w-4 h-4"
        fill="none"
        viewBox="0 0 24 24"
        stroke="currentColor"
        strokeWidth={1.5}
      >
        <path
          strokeLinecap="round"
          strokeLinejoin="round"
          d="M13.19 8.688a4.5 4.5 0 011.242 7.244l-4.5 4.5a4.5 4.5 0 01-6.364-6.364l1.757-1.757m13.35-.622l1.757-1.757a4.5 4.5 0 00-6.364-6.364l-4.5 4.5a4.5 4.5 0 001.242 7.244"
        />
      </svg>
    ),
  },
];

function ProjectItem({ instance }: { instance: Instance }) {
  const icon = (
    <svg
      className="w-4 h-4"
      fill="none"
      viewBox="0 0 24 24"
      stroke="currentColor"
      strokeWidth={1.5}
    >
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        d="M2.25 12.75V12A2.25 2.25 0 014.5 9.75h15A2.25 2.25 0 0121.75 12v.75m-8.69-6.44l-2.12-2.12a1.5 1.5 0 00-1.061-.44H4.5A2.25 2.25 0 002.25 6v12a2.25 2.25 0 002.25 2.25h15A2.25 2.25 0 0021.75 18V9a2.25 2.25 0 00-2.25-2.25h-5.379a1.5 1.5 0 01-1.06-.44z"
      />
    </svg>
  );
  const baseClass = `flex items-center gap-3 w-full px-3 py-2 rounded-[var(--radius-md)] text-sm transition-colors duration-[var(--duration-fast)]`;
  if (instance.is_current) {
    return (
      <div
        title={instance.root_dir}
        className={`${baseClass} bg-[var(--color-bg-hover)] text-[var(--color-text-primary)] font-medium cursor-default`}
      >
        <span className="text-[var(--color-text-primary)]">{icon}</span>
        <span className="truncate flex-1">{instance.name}</span>
        <span className="text-xs text-[var(--color-text-muted)] tabular-nums shrink-0">
          :{instance.port}
        </span>
      </div>
    );
  }
  return (
    <a
      href={`http://localhost:${instance.port}/`}
      title={instance.root_dir}
      className={`${baseClass} text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-bg-hover)]`}
    >
      <span className="text-[var(--color-text-muted)]">{icon}</span>
      <span className="truncate flex-1">{instance.name}</span>
      <span className="text-xs text-[var(--color-text-muted)] tabular-nums shrink-0">
        :{instance.port}
      </span>
    </a>
  );
}

function NavItem({
  item,
  isActive,
  onClick,
  badge,
}: {
  item: { id: View; label: string; icon: ReactNode };
  isActive: boolean;
  onClick: () => void;
  badge?: number;
}) {
  return (
    <button
      onClick={onClick}
      className={`
        flex items-center gap-3 w-full px-3 py-2 rounded-[var(--radius-md)]
        text-sm transition-colors duration-[var(--duration-fast)]
        ${
          isActive
            ? "bg-[var(--color-bg-hover)] text-[var(--color-text-primary)] font-medium"
            : "text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-bg-hover)]"
        }
      `
        .trim()
        .replace(/\s+/g, " ")}
    >
      <span
        className={
          isActive
            ? "text-[var(--color-text-primary)]"
            : "text-[var(--color-text-muted)]"
        }
      >
        {item.icon}
      </span>
      {item.label}
      {badge != null && badge > 0 && (
        <span className="ml-auto px-1.5 py-0.5 text-xs font-medium leading-none bg-[var(--color-accent-primary)] text-white rounded-full">
          {badge > 99 ? "99+" : badge}
        </span>
      )}
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
  cyclesEnabled,
  inboxUnread,
  statusBarLeft,
}: LayoutProps) {
  const [instances, setInstances] = useState<Instance[]>([]);
  const [user, setUser] = useState<User | null>(null);

  useEffect(() => {
    const load = () =>
      fetchInstances()
        .then(setInstances)
        .catch(() => {});
    load();
    const interval = setInterval(load, 10000);
    const onFocus = () => load();
    window.addEventListener("focus", onFocus);
    return () => {
      clearInterval(interval);
      window.removeEventListener("focus", onFocus);
    };
  }, []);

  useEffect(() => {
    fetchUser()
      .then(setUser)
      .catch(() => {});
  }, []);

  return (
    <div className="flex h-screen overflow-hidden bg-[var(--color-bg-sidebar)]">
      {/* Sidebar */}
      <aside className="w-62 flex-shrink-0 bg-[var(--color-bg-sidebar)] flex flex-col select-none">
        {/* Workspace header */}
        <div className="flex items-center gap-2 px-4 pt-2 h-13">
          <img
            src="/logo-light.svg"
            alt="Beats"
            className="w-5 h-5 opacity-80"
          />
          <span className="font-semibold text-[var(--color-text-primary)] text-base tracking-tight flex-1">
            Beats
          </span>
          <button
            onClick={onSearch}
            className="p-1 rounded-[var(--radius-sm)] text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-bg-hover)] transition-colors"
            title="Search issues"
          >
            <svg
              className="w-4 h-4"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
              strokeWidth={1.5}
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                d="M21 21l-5.197-5.197m0 0A7.5 7.5 0 105.196 5.196a7.5 7.5 0 0010.607 10.607z"
              />
            </svg>
          </button>
          <button
            onClick={onNewIssue}
            className="p-1 rounded-[var(--radius-sm)] text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-bg-hover)] transition-colors"
            title="New issue"
          >
            <svg
              className="w-4 h-4"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
              strokeWidth={1.5}
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                d="M16.862 4.487l1.687-1.688a1.875 1.875 0 112.652 2.652L10.582 16.07a4.5 4.5 0 01-1.897 1.13L6 18l.8-2.685a4.5 4.5 0 011.13-1.897l8.932-8.931zm0 0L19.5 7.125M18 14v4.75A2.25 2.25 0 0115.75 21H5.25A2.25 2.25 0 013 18.75V8.25A2.25 2.25 0 015.25 6H10"
              />
            </svg>
          </button>
        </div>

        {/* Nav */}
        <nav className="px-2 pt-1 space-y-1">
          {PRIMARY_NAV.map((item) => (
            <NavItem
              key={item.id}
              item={item}
              isActive={currentView === item.id}
              onClick={() => onViewChange(item.id)}
            />
          ))}
          {cyclesEnabled && (
            <NavItem
              item={{
                id: "cycles" as View,
                label: "Cycles",
                icon: (
                  <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
                    <path strokeLinecap="round" strokeLinejoin="round" d="M16.023 9.348h4.992v-.001M2.985 19.644v-4.992m0 0h4.992m-4.993 0l3.181 3.183a8.25 8.25 0 0013.803-3.7M4.031 9.865a8.25 8.25 0 0113.803-3.7l3.181 3.182M21.016 5.176v4.993" />
                  </svg>
                ),
              }}
              isActive={currentView === "cycles"}
              onClick={() => onViewChange("cycles" as View)}
            />
          )}
          <NavItem
            item={{
              id: "inbox" as View,
              label: "Inbox",
              icon: (
                <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M2.25 13.5h3.86a2.25 2.25 0 012.012 1.244l.256.512a2.25 2.25 0 002.013 1.244h3.218a2.25 2.25 0 002.013-1.244l.256-.512a2.25 2.25 0 012.013-1.244h3.859M12 3v8.25m0 0l-3-3m3 3l3-3" />
                </svg>
              ),
            }}
            isActive={currentView === "inbox"}
            onClick={() => onViewChange("inbox" as View)}
            badge={inboxUnread}
          />
          {SECONDARY_NAV.map((item) => (
            <NavItem
              key={item.id}
              item={item}
              isActive={currentView === item.id}
              onClick={() => onViewChange(item.id)}
            />
          ))}
        </nav>

        {/* Projects */}
        {instances.length > 0 && (
          <>
            <div className="mx-4 my-2 border-t border-[var(--color-border-subtle)]" />
            <div className="px-2 space-y-1">
              <div className="text-xs uppercase tracking-wider text-[var(--color-text-muted)] px-3 mb-1 mt-1">
                Projects
              </div>
              {instances.map((p) => (
                <ProjectItem key={`${p.pid}-${p.port}`} instance={p} />
              ))}
            </div>
          </>
        )}

        {/* Spacer + User */}
        <div className="flex-1" />
        {user && (
          <>
            <div className="flex items-center gap-3 px-4 py-3">
              <Avatar name={`${user.name} <${user.email}>`} size="md" />
              <span className="text-xs text-[var(--color-text-secondary)] truncate">
                {user.name}
              </span>
            </div>
          </>
        )}
      </aside>

      {/* Main */}
      <div className="flex-1 pt-2 pr-2 overflow-hidden flex flex-col">
        <main className="flex-1 overflow-hidden bg-[var(--color-bg-primary)] rounded-[var(--radius-lg)] border border-[var(--color-border-subtle)] border-b-0">
          {children}
        </main>
        <div className="flex items-center px-4 py-2 shrink-0 gap-3">
          {statusBarLeft && <div className="flex-1 min-w-0">{statusBarLeft}</div>}
          {!statusBarLeft && <div className="flex-1" />}
          <div className="flex items-center gap-2 text-xs shrink-0">
            <span
              className={`w-2 h-2 rounded-full ${connected ? "bg-[var(--color-success)]" : "bg-[var(--color-error)]"}`}
            />
            {connected ? (
              <span className="text-[var(--color-text-muted)]">
                Beats {version && `v${version}`}
              </span>
            ) : (
              <span className="text-[var(--color-error)]">Disconnected</span>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
