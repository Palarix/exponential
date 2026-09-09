import { useRef, useEffect, useState, useLayoutEffect, useCallback, useMemo } from "react";
import { createPortal } from "react-dom";
import type { Issue, Cycle } from "../../api/client";
import { addDraft, fetchCycles } from "../../api/client";
import LabelPicker from "./LabelPicker";
import StatusPicker from "./StatusPicker";
import PriorityPicker from "./PriorityPicker";
import EstimatePicker from "./EstimatePicker";
import Avatar from "./Avatar";
import { UserRound, Triangle, RefreshCw, ChevronRight, Trash2, Check, Tag, Unlink } from "lucide-react";
import { toggleLabel } from "../../utils/labels";
import { collectKnownPeople } from "../../utils/issues";

type SubMenu = "status" | "priority" | "assignee" | "labels" | "estimate" | "cycle" | null;

interface ContextMenuProps {
  issue: Issue;
  issues: Issue[];
  x: number;
  y: number;
  onClose: () => void;
  onRefresh: () => void;
  allLabels: string[];
  contributors: string[];
  onConfigLabelsChange?: (labels: Record<string, string>) => void;
  patchIssue?: (issueId: string, patch: Partial<Issue>) => void;
}

interface MenuItem {
  id: SubMenu & string;
  label: string;
  shortcut: string;
  icon: React.ReactNode;
}

const MENU_ITEMS: MenuItem[] = [
  {
    id: "status",
    label: "Status",
    shortcut: "S",
    icon: (
      <svg className="w-4 h-4" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5">
        <circle cx="8" cy="8" r="5.5" />
        <path d="M8 2.5A5.5 5.5 0 0113.5 8" strokeLinecap="round" />
      </svg>
    ),
  },
  {
    id: "priority",
    label: "Priority",
    shortcut: "P",
    icon: (
      <svg className="w-4 h-4" viewBox="0 0 16 16" fill="currentColor">
        <rect x="1" y="8" width="4" height="3" rx="1" />
        <rect x="6" y="5" width="4" height="6" rx="1" opacity="0.5" />
        <rect x="11" y="2" width="4" height="9" rx="1" opacity="0.2" />
      </svg>
    ),
  },
  {
    id: "assignee",
    label: "Assignee",
    shortcut: "A",
    icon: <UserRound size={16} />,
  },
  {
    id: "labels",
    label: "Labels",
    shortcut: "L",
    icon: <Tag size={16} />,
  },
  {
    id: "estimate",
    label: "Estimate",
    shortcut: "E",
    icon: <Triangle size={16} />,
  },
  {
    id: "cycle",
    label: "Cycle",
    shortcut: "C",
    icon: <RefreshCw size={16} />,
  },
];

const Chevron = () => (
  <ChevronRight className="w-3 h-3 ml-auto text-[var(--color-text-muted)]" />
);

export default function ContextMenu({
  issue,
  issues,
  x,
  y,
  onClose,
  onRefresh,
  allLabels,
  contributors,
  onConfigLabelsChange,
  patchIssue,
}: ContextMenuProps) {
  const menuRef = useRef<HTMLDivElement>(null);
  const subMenuRef = useRef<HTMLDivElement>(null);
  const itemRefs = useRef<Map<string, HTMLButtonElement>>(new Map());
  const [subMenu, setSubMenu] = useState<SubMenu>(null);
  const [focusIndex, setFocusIndex] = useState(-1);
  const [confirmDelete, setConfirmDelete] = useState<false | "confirm" | "choose">(false);
  const [filterText, setFilterText] = useState("");
  const [subMenuOffset, setSubMenuOffset] = useState(0);
  const [cycles, setCycles] = useState<Cycle[]>([]);
  const onCloseRef = useRef(onClose);
  useEffect(() => {
    onCloseRef.current = onClose;
  }, [onClose]);

  useEffect(() => {
    fetchCycles().then(data => {
      if (data.enabled && data.cycles) setCycles(data.cycles);
    }).catch(() => {});
  }, []);

  const hasChildren = useMemo(() => issues.some(i => i.parent_id === issue.id), [issues, issue.id]);

  const closeAll = useCallback(() => {
    setSubMenu(null);
    setConfirmDelete(false);
    onCloseRef.current();
  }, []);

  useLayoutEffect(() => {
    const el = menuRef.current;
    if (!el) return;
    const rect = el.getBoundingClientRect();
    const pad = 8;
    let top = y;
    let left = x;
    if (top + rect.height > window.innerHeight - pad) top = window.innerHeight - rect.height - pad;
    if (left + rect.width > window.innerWidth - pad) left = window.innerWidth - rect.width - pad;
    if (top < pad) top = pad;
    if (left < pad) left = pad;
    el.style.top = `${top}px`;
    el.style.left = `${left}px`;
  }, [x, y]);

  useLayoutEffect(() => {
    const sub = subMenuRef.current;
    const menu = menuRef.current;
    if (!sub || !menu) return;
    const pad = 8;
    const menuRect = menu.getBoundingClientRect();
    const subRect = sub.getBoundingClientRect();
    const gap = 4;
    let top = menuRect.top + Math.max(0, subMenuOffset - 30);
    let left = menuRect.right + gap;
    if (left + subRect.width > window.innerWidth - pad) {
      left = menuRect.left - subRect.width - gap;
    }
    if (left < pad) left = pad;
    if (top + subRect.height > window.innerHeight - pad) {
      top = window.innerHeight - subRect.height - pad;
    }
    if (top < pad) top = pad;
    sub.style.top = `${top}px`;
    sub.style.left = `${left}px`;
    sub.style.visibility = "visible";
  }, [subMenu, subMenuOffset]);

  // Click-outside: armed after first frame to avoid closing on the triggering right-click
  useEffect(() => {
    let armed = false;
    const armTimer = requestAnimationFrame(() => { armed = true; });
    const handleClick = (e: MouseEvent) => {
      if (!armed) return;
      if (
        menuRef.current && !menuRef.current.contains(e.target as Node) &&
        (!subMenuRef.current || !subMenuRef.current.contains(e.target as Node))
      ) closeAll();
    };
    document.addEventListener("mousedown", handleClick);
    return () => {
      cancelAnimationFrame(armTimer);
      document.removeEventListener("mousedown", handleClick);
    };
  }, [closeAll]);

  const handleAction = useCallback((type: string, payload: Record<string, unknown>) => {
    if (type === "UPDATE" && patchIssue) patchIssue(issue.id, payload as Partial<Issue>);
    closeAll();
    addDraft(issue.id, type, payload).then(() => onRefresh());
  }, [issue.id, onRefresh, closeAll, patchIssue]);

  const handleLabelToggle = useCallback((label: string) => {
    const labels = toggleLabel(issue.labels || [], label);
    if (patchIssue) patchIssue(issue.id, { labels });
    addDraft(issue.id, "UPDATE", { labels }).then(() => onRefresh());
  }, [issue.id, issue.labels, onRefresh, patchIssue]);

  const knownPeople = useMemo(() => {
    const all = collectKnownPeople(issues, contributors);
    const q = filterText.toLowerCase();
    if (!q) return all;
    return all.filter(p => p.toLowerCase().includes(q));
  }, [issues, contributors, filterText]);

  const openSubMenu = useCallback((id: SubMenu) => {
    if (id && itemRefs.current.has(id)) {
      const btn = itemRefs.current.get(id)!;
      const menuRect = menuRef.current?.getBoundingClientRect();
      if (menuRect) {
        setSubMenuOffset(btn.getBoundingClientRect().top - menuRect.top);
      }
    }
    setSubMenu(id);
    setFilterText("");
  }, []);

  // Keyboard handler — capture phase + stopImmediatePropagation to prevent Backlog shortcuts
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        e.preventDefault();
        e.stopImmediatePropagation();
        if (subMenu) { setSubMenu(null); } else { closeAll(); }
        return;
      }
      if (subMenu) return;
      e.stopImmediatePropagation();
      const key = e.key.toLowerCase();
      if (key === "s") { e.preventDefault(); openSubMenu("status"); return; }
      if (key === "p") { e.preventDefault(); openSubMenu("priority"); return; }
      if (key === "a") { e.preventDefault(); openSubMenu("assignee"); return; }
      if (key === "l") { e.preventDefault(); openSubMenu("labels"); return; }
      if (key === "e") { e.preventDefault(); openSubMenu("estimate"); return; }
      if (key === "c") { e.preventDefault(); openSubMenu("cycle"); return; }
      if (key === "backspace" && (e.metaKey || e.ctrlKey)) { e.preventDefault(); setConfirmDelete(hasChildren ? "choose" : "confirm"); return; }
      if (key === "arrowdown") { e.preventDefault(); setFocusIndex(i => Math.min(i + 1, MENU_ITEMS.length)); return; }
      if (key === "arrowup") { e.preventDefault(); setFocusIndex(i => Math.max(i - 1, 0)); return; }
      if (key === "arrowright" && focusIndex >= 0 && focusIndex < MENU_ITEMS.length) {
        e.preventDefault();
        openSubMenu(MENU_ITEMS[focusIndex].id);
        return;
      }
      if (key === "enter" && focusIndex >= 0 && focusIndex < MENU_ITEMS.length) {
        e.preventDefault();
        openSubMenu(MENU_ITEMS[focusIndex].id);
        return;
      }
      if (key === "enter" && focusIndex === MENU_ITEMS.length) {
        e.preventDefault();
        setConfirmDelete(hasChildren ? "choose" : "confirm");
        return;
      }
    };
    document.addEventListener("keydown", handler, true);
    return () => document.removeEventListener("keydown", handler, true);
  }, [subMenu, closeAll, focusIndex, openSubMenu, hasChildren]);

  const q = filterText.toLowerCase();

  const filterInput = (placeholder: string) => (
    <>
      <div className="px-3 py-2">
        <input
          autoFocus
          value={filterText}
          onChange={e => setFilterText(e.target.value)}
          onKeyDown={e => { if (e.key === "Escape") { e.stopPropagation(); setSubMenu(null); setFilterText(""); } }}
          placeholder={placeholder}
          className="w-full text-sm bg-transparent text-[var(--color-text-primary)] placeholder:text-[var(--color-text-muted)] outline-none"
        />
      </div>
      <div className="border-t border-[var(--color-border-subtle)]" />
    </>
  );

  const renderSubMenuPanel = () => {
    if (subMenu === "status") {
      return <StatusPicker current={issue.status} onSelect={v => handleAction("UPDATE", { status: v })} onClose={() => setSubMenu(null)} />;
    }
    if (subMenu === "priority") {
      return <PriorityPicker current={issue.priority || 0} onSelect={v => handleAction("UPDATE", { priority: v })} onClose={() => setSubMenu(null)} />;
    }
    if (subMenu === "estimate") {
      return <EstimatePicker current={issue.estimate || 0} onSelect={v => handleAction("UPDATE", { estimate: v })} onClose={() => setSubMenu(null)} />;
    }
    if (subMenu === "assignee") {
      return (
        <>
          {filterInput("Set assignee...")}
          <div className="max-h-60 overflow-y-auto">
            {issue.assignee && !q && (
              <button onClick={() => handleAction("UPDATE", { assignee: "" })} className="flex items-center gap-2 w-full px-3 py-2 text-sm text-[var(--color-text-muted)] hover:bg-[var(--color-hover-surface-3)] transition-colors">
                Remove assignee
              </button>
            )}
            {knownPeople.map(person => {
              const isCurrent = person === issue.assignee;
              return (
                <button key={person} onClick={() => handleAction("UPDATE", { assignee: person })} className={`flex items-center gap-2 w-full px-3 py-2 text-sm transition-colors hover:bg-[var(--color-hover-surface-3)] ${isCurrent ? "text-[var(--color-accent-primary)]" : "text-[var(--color-text-primary)]"}`}>
                  <Avatar name={person} size="sm" />
                  <span className="truncate">{person.split(" <")[0]}</span>
                  {isCurrent && <Check className="w-4 h-4 ml-auto shrink-0" />}
                </button>
              );
            })}
            {knownPeople.length === 0 && (
              <div className="px-3 py-2 text-sm text-[var(--color-text-muted)]">No matching people</div>
            )}
          </div>
        </>
      );
    }
    if (subMenu === "cycle") {
      if (cycles.length === 0) {
        return <div className="px-3 py-2 text-sm text-[var(--color-text-muted)]">Cycles not configured</div>;
      }
      return (
        <>
          <div className="px-3 py-2 text-xs font-medium text-[var(--color-text-muted)]">Move to cycle...</div>
          <div className="border-t border-[var(--color-border-subtle)]" />
          {issue.cycle_id && (
            <button onClick={() => handleAction("UPDATE", { cycle_id: "" })} className="flex items-center gap-2 w-full px-3 py-2 text-sm text-[var(--color-text-muted)] hover:bg-[var(--color-hover-surface-3)] transition-colors">
              No cycle
            </button>
          )}
          {cycles.filter(c => c.status !== 'completed').map(c => {
            const isCurrent = c.id === issue.cycle_id;
            return (
              <button key={c.id} onClick={() => handleAction("UPDATE", { cycle_id: c.id })} className={`flex items-center gap-2 w-full px-3 py-2 text-sm transition-colors hover:bg-[var(--color-hover-surface-3)] ${isCurrent ? "text-[var(--color-accent-primary)]" : "text-[var(--color-text-primary)]"}`}>
                <span>Cycle {c.number}</span>
                <span className="text-xs text-[var(--color-text-muted)] capitalize">{c.status}</span>
                {isCurrent && <Check className="w-4 h-4 ml-auto shrink-0" />}
              </button>
            );
          })}
        </>
      );
    }
    if (subMenu === "labels") {
      return (
        <LabelPicker
          allLabels={allLabels}
          selected={issue.labels || []}
          onToggle={handleLabelToggle}
          onConfigLabelsChange={onConfigLabelsChange}
          onClose={closeAll}
        />
      );
    }
    return null;
  };

  if (confirmDelete === "choose") {
    return createPortal(
      <div ref={menuRef} style={{ position: "fixed", top: y, left: x }} className="z-[100] min-w-55 bg-[var(--color-surface-3)] border border-[var(--color-border-default)] rounded-[var(--radius-lg)] shadow-[var(--shadow-popover)] p-3">
        <p className="text-sm text-[var(--color-text-primary)] mb-3">This issue has sub-issues. What should happen to them?</p>
        <div className="flex flex-col gap-2">
          <button onClick={() => handleAction("DELETE", { cascade: false })} className="px-3 py-2 text-sm font-medium rounded-[var(--radius-md)] bg-[var(--color-error)] text-white hover:opacity-90 transition-opacity text-left">
            Keep sub-issues
          </button>
          <button onClick={() => handleAction("DELETE", { cascade: true })} className="px-3 py-2 text-sm font-medium rounded-[var(--radius-md)] border border-[var(--color-error)] text-[var(--color-error)] hover:bg-[var(--color-error)] hover:text-white transition-colors text-left">
            Delete sub-issues too
          </button>
          <button onClick={() => setConfirmDelete(false)} className="px-3 py-2 text-sm font-medium rounded-[var(--radius-md)] text-[var(--color-text-secondary)] hover:bg-[var(--color-hover-surface-3)] transition-colors text-left">
            Cancel
          </button>
        </div>
      </div>,
      document.body,
    );
  }

  if (confirmDelete === "confirm") {
    return createPortal(
      <div ref={menuRef} style={{ position: "fixed", top: y, left: x }} className="z-[100] min-w-55 bg-[var(--color-surface-3)] border border-[var(--color-border-default)] rounded-[var(--radius-lg)] shadow-[var(--shadow-popover)] p-3">
        <p className="text-sm text-[var(--color-text-primary)] mb-3">Delete this issue?</p>
        <div className="flex items-center gap-2">
          <button onClick={() => handleAction("DELETE", {})} className="px-3 py-2 text-sm font-medium rounded-[var(--radius-md)] bg-[var(--color-error)] text-white hover:opacity-90 transition-opacity">
            Delete
          </button>
          <button onClick={() => setConfirmDelete(false)} className="px-3 py-2 text-sm font-medium rounded-[var(--radius-md)] text-[var(--color-text-secondary)] hover:bg-[var(--color-hover-surface-3)] transition-colors">
            Cancel
          </button>
        </div>
      </div>,
      document.body,
    );
  }

  return createPortal(
    <>
    <div ref={menuRef} style={{ position: "fixed", top: y, left: x }} className="z-[100]">
      {/* Main menu */}
      <div className="min-w-50 bg-[var(--color-surface-3)] border border-[var(--color-border-default)] rounded-[var(--radius-lg)] shadow-[var(--shadow-popover)] overflow-hidden">
        {MENU_ITEMS.map((item, i) => (
          <button
            key={item.id}
            ref={el => { if (el) itemRefs.current.set(item.id, el); }}
            onClick={() => openSubMenu(item.id)}
            onMouseEnter={() => { setFocusIndex(i); openSubMenu(item.id); }}
            className={`flex items-center gap-3 w-full px-3 py-2 text-sm text-[var(--color-text-primary)] transition-colors hover:bg-[var(--color-hover-surface-3)] ${focusIndex === i ? "bg-[var(--color-hover-surface-3)]" : ""}`}
          >
            <span className="text-[var(--color-text-muted)] w-4 shrink-0 flex items-center justify-center">{item.icon}</span>
            <span>{item.label}</span>
            <span className="ml-auto flex items-center gap-2">
              <span className="text-xs text-[var(--color-text-muted)]">{item.shortcut}</span>
              <Chevron />
            </span>
          </button>
        ))}
        {issue.parent_id && (
          <>
            <div className="my-1 border-t border-[var(--color-border-subtle)]" />
            <button
              onClick={() => handleAction("UPDATE", { parent_id: "" })}
              onMouseEnter={() => { setFocusIndex(-1); setSubMenu(null); }}
              className="flex items-center gap-3 w-full px-3 py-2 text-sm text-[var(--color-text-primary)] transition-colors hover:bg-[var(--color-hover-surface-3)]"
            >
              <span className="text-[var(--color-text-muted)] w-4 shrink-0 flex items-center justify-center">
                <Unlink size={16} />
              </span>
              <span>Remove from parent</span>
            </button>
          </>
        )}
        <div className="my-1 border-t border-[var(--color-border-subtle)]" />
        <button
          onClick={() => setConfirmDelete(hasChildren ? "choose" : "confirm")}
          onMouseEnter={() => { setFocusIndex(MENU_ITEMS.length); setSubMenu(null); }}
          className={`flex items-center gap-3 w-full px-3 py-2 text-sm text-[var(--color-error)] transition-colors hover:bg-[var(--color-hover-surface-3)] ${focusIndex === MENU_ITEMS.length ? "bg-[var(--color-hover-surface-3)]" : ""}`}
        >
          <span className="w-4 shrink-0 flex items-center justify-center">
            <Trash2 size={16} />
          </span>
          <span>Delete</span>
          <span className="ml-auto text-xs opacity-70">{navigator.platform.includes("Mac") ? "⌘" : "Ctrl"}{"⌫"}</span>
        </button>
      </div>

    </div>
    {/* Flyout sub-menu */}
    {subMenu && (
      <div
        ref={subMenuRef}
        style={{ position: "fixed", visibility: "hidden" }}
        className="z-[100] min-w-50 max-w-70 bg-[var(--color-surface-3)] border border-[var(--color-border-default)] rounded-[var(--radius-lg)] shadow-[var(--shadow-popover)] overflow-hidden"
      >
        {renderSubMenuPanel()}
      </div>
    )}
    </>,
    document.body,
  );
}
