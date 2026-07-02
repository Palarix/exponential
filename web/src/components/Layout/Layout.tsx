import { useEffect, useRef, useState, type ReactNode } from "react";
import {
  fetchInstances,
  fetchUser,
  type Instance,
  type User,
} from "../../api/client";
import {
  Avatar,
  BellIcon,
  BoardIcon,
  ChevronUpIcon,
  CyclesIcon,
  DashboardIcon,
  EditIcon,
  FolderIcon,
  LinkIcon,
  ListIcon,
  SearchIcon,
  SidebarIcon,
  TagIcon,
  TimelineIcon,
  UserIcon,
} from "../ui";

type View = "dashboard" | "inbox" | "backlog" | "board" | "cycles" | "dependencies" | "labels" | "my-issues" | "timeline";

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


function ProjectSelector({
  instances,
  collapsed,
}: {
  instances: Instance[];
  collapsed: boolean;
}) {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);
  const current = instances.find((i) => i.is_current);

  useEffect(() => {
    if (!open) return;
    const handleClick = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node))
        setOpen(false);
    };
    const handleKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") setOpen(false);
    };
    document.addEventListener("mousedown", handleClick);
    document.addEventListener("keydown", handleKey);
    return () => {
      document.removeEventListener("mousedown", handleClick);
      document.removeEventListener("keydown", handleKey);
    };
  }, [open]);

  if (instances.length === 0) return null;

  return (
    <div ref={ref} className={`relative ${collapsed ? "px-1" : "px-2"}`}>
      {open && (
        <div
          className="absolute bottom-full left-0 right-0 mb-1 mx-0 py-1 rounded-[var(--radius-md)] border border-[var(--color-border-subtle)] bg-[var(--color-surface)] shadow-lg z-50"
          style={{ minWidth: collapsed ? "200px" : undefined }}
        >
          <div className="text-xs uppercase tracking-wider text-[var(--color-text-muted)] px-3 py-1.5">
            Projects
          </div>
          {instances.map((inst) => {
            const itemClass = `flex items-center gap-3 w-full px-3 py-2 text-sm transition-colors duration-[var(--duration-fast)]`;
            if (inst.is_current) {
              return (
                <div
                  key={`${inst.pid}-${inst.port}`}
                  title={inst.root_dir}
                  className={`${itemClass} bg-[var(--color-hover-surface)] text-[var(--color-text-primary)] font-medium cursor-default`}
                >
                  <span className="text-[var(--color-text-primary)] shrink-0">
                    <FolderIcon />
                  </span>
                  <span className="truncate flex-1">{inst.name}</span>
                  <span className="text-xs text-[var(--color-text-muted)] tabular-nums shrink-0">
                    :{inst.port}
                  </span>
                </div>
              );
            }
            return (
              <a
                key={`${inst.pid}-${inst.port}`}
                href={`http://localhost:${inst.port}/`}
                title={inst.root_dir}
                className={`${itemClass} text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-hover-surface)]`}
              >
                <span className="text-[var(--color-text-muted)] shrink-0">
                  <FolderIcon />
                </span>
                <span className="truncate flex-1">{inst.name}</span>
                <span className="text-xs text-[var(--color-text-muted)] tabular-nums shrink-0">
                  :{inst.port}
                </span>
              </a>
            );
          })}
        </div>
      )}
      <button
        onClick={() => setOpen(!open)}
        title={current?.root_dir ?? "Switch project"}
        className={`
          flex items-center w-full rounded-[var(--radius-md)]
          text-sm transition-colors duration-[var(--duration-fast)]
          text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-hover-surface)]
          ${collapsed ? "justify-center px-0 py-2" : "gap-3 px-3 py-2"}
        `
          .trim()
          .replace(/\s+/g, " ")}
      >
        <span className="text-[var(--color-text-muted)] shrink-0">
          <FolderIcon />
        </span>
        {!collapsed && (
          <>
            <span className="truncate flex-1 text-left">
              {current?.name ?? "Projects"}
            </span>
            <ChevronUpIcon
              className={`w-3 h-3 shrink-0 text-[var(--color-text-muted)] transition-transform ${open ? "rotate-180" : ""}`}
            />
          </>
        )}
      </button>
    </div>
  );
}

function NavItem({
  item,
  isActive,
  onClick,
  badge,
  collapsed,
}: {
  item: { id: View; label: string; icon: ReactNode };
  isActive: boolean;
  onClick: () => void;
  badge?: number;
  collapsed?: boolean;
}) {
  return (
    <button
      onClick={onClick}
      title={collapsed ? item.label : undefined}
      className={`
        flex items-center w-full rounded-[var(--radius-md)]
        text-sm transition-colors duration-[var(--duration-fast)]
        ${collapsed ? "justify-center px-0 py-2" : "gap-3 px-3 py-2"}
        ${
          isActive
            ? "bg-[var(--color-hover-surface)] text-[var(--color-text-primary)] font-medium"
            : "text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-hover-surface)]"
        }
      `
        .trim()
        .replace(/\s+/g, " ")}
    >
      <span
        className={`relative shrink-0 ${
          isActive
            ? "text-[var(--color-text-primary)]"
            : "text-[var(--color-text-muted)]"
        }`}
      >
        {item.icon}
        {collapsed && badge != null && badge > 0 && (
          <span className="absolute -top-1 -right-1 w-2 h-2 rounded-full bg-[var(--color-accent-primary)]" />
        )}
      </span>
      {!collapsed && item.label}
      {!collapsed && badge != null && badge > 0 && (
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
  const [collapsed, setCollapsed] = useState(() => {
    try { return localStorage.getItem("exponential-sidebar-collapsed") === "true"; } catch { return false; }
  });
  const toggleSidebar = () => {
    setCollapsed(prev => {
      const next = !prev;
      localStorage.setItem("exponential-sidebar-collapsed", String(next));
      return next;
    });
  };

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
    <div className="flex h-screen overflow-hidden bg-[var(--color-chrome)]">
      {/* Sidebar */}
      <aside className={`${collapsed ? "w-12" : "w-62"} flex-shrink-0 flex flex-col select-none transition-[width] duration-200`}>
        {/* Workspace header */}
        <div className={`flex items-center h-13 ${collapsed ? "justify-center px-2 pt-2" : "gap-2 px-4 pt-2"}`}>
          {collapsed ? (
            <button
              onClick={toggleSidebar}
              className="p-1 rounded-[var(--radius-sm)] text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-hover-surface)] transition-colors"
              title="Expand sidebar"
            >
              <img src="/xpo.svg" alt="Exponential" className="w-5 h-5 opacity-80" />
            </button>
          ) : (
            <>
              <img
                src="/xpo.svg"
                alt="Exponential"
                className="w-5 h-5 opacity-80"
              />
              <div className="flex-1" />
              <button
                onClick={onSearch}
                className="p-1 rounded-[var(--radius-sm)] text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-hover-surface)] transition-colors"
                title="Search issues"
              >
                <SearchIcon />
              </button>
              <button
                onClick={onNewIssue}
                className="p-1 rounded-[var(--radius-sm)] text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-hover-surface)] transition-colors"
                title="New issue"
              >
                <EditIcon />
              </button>
              <button
                onClick={toggleSidebar}
                className="p-1 rounded-[var(--radius-sm)] text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-hover-surface)] transition-colors"
                title="Collapse sidebar"
              >
                <SidebarIcon />
              </button>
            </>
          )}
        </div>

        {/* Nav */}
        <nav className={`pt-1 space-y-1 ${collapsed ? "px-1" : "px-2"}`}>
          <NavItem
            item={{ id: "dashboard", label: "Overview", icon: <DashboardIcon /> }}
            isActive={currentView === "dashboard"}
            onClick={() => onViewChange("dashboard")}
            collapsed={collapsed}
          />
          <NavItem
            item={{ id: "my-issues" as View, label: "My Issues", icon: <UserIcon /> }}
            isActive={currentView === "my-issues"}
            onClick={() => onViewChange("my-issues" as View)}
            collapsed={collapsed}
          />
          <NavItem
            item={{ id: "inbox" as View, label: "Notifications", icon: <BellIcon /> }}
            isActive={currentView === "inbox"}
            onClick={() => onViewChange("inbox" as View)}
            badge={inboxUnread}
            collapsed={collapsed}
          />
          <NavItem
            item={{ id: "backlog", label: "Issues", icon: <ListIcon /> }}
            isActive={currentView === "backlog"}
            onClick={() => onViewChange("backlog")}
            collapsed={collapsed}
          />
          <NavItem
            item={{ id: "board", label: "Board", icon: <BoardIcon /> }}
            isActive={currentView === "board"}
            onClick={() => onViewChange("board")}
            collapsed={collapsed}
          />
          {cyclesEnabled && (
            <NavItem
              item={{ id: "cycles" as View, label: "Cycles", icon: <CyclesIcon /> }}
              isActive={currentView === "cycles"}
              onClick={() => onViewChange("cycles" as View)}
              collapsed={collapsed}
            />
          )}
          <NavItem
            item={{ id: "labels", label: "Labels", icon: <TagIcon /> }}
            isActive={currentView === "labels"}
            onClick={() => onViewChange("labels")}
            collapsed={collapsed}
          />
          <NavItem
            item={{ id: "dependencies", label: "Dependencies", icon: <LinkIcon /> }}
            isActive={currentView === "dependencies"}
            onClick={() => onViewChange("dependencies")}
            collapsed={collapsed}
          />
          <NavItem
            item={{ id: "timeline" as View, label: "Timeline", icon: <TimelineIcon /> }}
            isActive={currentView === "timeline"}
            onClick={() => onViewChange("timeline" as View)}
            collapsed={collapsed}
          />
        </nav>

        {/* Spacer */}
        <div className="flex-1" />

        {/* Project selector + User */}
        <ProjectSelector instances={instances} collapsed={collapsed} />
        {user && (
          <div className={`flex items-center py-3 ${collapsed ? "justify-center px-2" : "gap-3 px-4"}`}>
            <Avatar name={`${user.name} <${user.email}>`} size={collapsed ? "sm" : "md"} />
            {!collapsed && (
              <span className="text-xs text-[var(--color-text-secondary)] truncate">
                {user.name}
              </span>
            )}
          </div>
        )}
      </aside>

      {/* Main */}
      <div className="flex-1 pt-2 pr-2 overflow-hidden flex flex-col">
        <main className="flex-1 overflow-hidden rounded-[var(--radius-lg)] border border-[var(--color-border-subtle)] border-b-0" style={{ background: "linear-gradient(160deg, #1A1B1C 0%, var(--color-surface) 20%)" }}>
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
                Exponential {version && `v${version}`}
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
