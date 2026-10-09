import { useEffect, useRef, useState, type ReactNode } from "react";
import { fetchInstances, type Instance } from "../../api/client";
import { useUser } from "../../api/queries";
import { Avatar } from "../ui";
import {
  Bell,
  Columns3,
  ChevronUp,
  RefreshCw,
  LayoutDashboard,
  Pencil,
  Folder,
  Link,
  List,
  Search,
  PanelLeft,
  Tag,
  Clock,
  User as UserIcon,
} from "lucide-react";
import { useKeyboardShortcuts } from "../../keyboard";
import { VIEWS, type ViewDef, type ViewId } from "../../views";

const VIEW_ICONS: Record<ViewId, ReactNode> = {
  dashboard: <LayoutDashboard size={16} />,
  backlog: <List size={16} />,
  board: <Columns3 size={16} />,
  cycles: <RefreshCw size={16} />,
  timeline: <Clock size={16} />,
  labels: <Tag size={16} />,
  dependencies: <Link size={16} />,
  "my-issues": <UserIcon size={16} />,
  inbox: <Bell size={16} />,
};

interface SidebarProps {
  activeView: ViewId;
  onViewChange: (view: ViewId) => void;
  onSearch: () => void;
  onNewIssue: () => void;
  cyclesEnabled: boolean;
  inboxUnread: number;
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

  useKeyboardShortcuts({
    scope: "project-selector",
    priority: "overlay",
    enabled: open,
    shortcuts: [{ id: "project-selector.close", key: "Escape", label: "Close project selector", showInHelp: false, allowInEditable: true, run: () => setOpen(false) }],
  });

  useEffect(() => {
    if (!open) return;
    const handleClick = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node))
        setOpen(false);
    };
    document.addEventListener("mousedown", handleClick);
    return () => {
      document.removeEventListener("mousedown", handleClick);
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
                    <Folder size={16} />
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
                  <Folder size={16} />
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
          <Folder size={16} />
        </span>
        {!collapsed && (
          <>
            <span className="truncate flex-1 text-left">
              {current?.name ?? "Projects"}
            </span>
            <ChevronUp
              size={12}
              className={`shrink-0 text-[var(--color-text-muted)] transition-transform ${open ? "rotate-180" : ""}`}
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
  item: { label: string; icon: ReactNode };
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

export default function Sidebar({
  activeView,
  onViewChange,
  onSearch,
  onNewIssue,
  cyclesEnabled,
  inboxUnread,
}: SidebarProps) {
  const [instances, setInstances] = useState<Instance[]>([]);
  const user = useUser();
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

  const visible = (v: ViewDef) => v.requires !== "cycles" || cyclesEnabled;
  const renderItem = (v: ViewDef) => (
    <NavItem
      key={v.id}
      item={{ label: v.navLabel, icon: VIEW_ICONS[v.id] }}
      isActive={activeView === v.id}
      onClick={() => onViewChange(v.id)}
      badge={v.id === "inbox" ? inboxUnread : undefined}
      collapsed={collapsed}
    />
  );

  return (
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
              <Search size={16} />
            </button>
            <button
              onClick={onNewIssue}
              className="p-1 rounded-[var(--radius-sm)] text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-hover-surface)] transition-colors"
              title="New issue"
            >
              <Pencil size={16} />
            </button>
            <button
              onClick={toggleSidebar}
              className="p-1 rounded-[var(--radius-sm)] text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-hover-surface)] transition-colors"
              title="Collapse sidebar"
            >
              <PanelLeft size={16} />
            </button>
          </>
        )}
      </div>

      {/* Nav */}
      <nav className={`pt-1 space-y-1 ${collapsed ? "px-1" : "px-2"}`}>
        {VIEWS.filter(v => v.section === "main" && visible(v)).map(renderItem)}
        <div className={`my-1 ${collapsed ? "mx-1" : "mx-3"} border-t border-[var(--color-border-default)]`} />
        {VIEWS.filter(v => v.section === "personal" && visible(v)).map(renderItem)}
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
  );
}
