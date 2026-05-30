import { useRef, useEffect, useState, useLayoutEffect, useCallback, useMemo } from "react";
import { createPortal } from "react-dom";
import type { Issue } from "../../api/client";
import { addDraft } from "../../api/client";
import { STATUS_OPTIONS, ESTIMATE_OPTIONS, PRIORITY_OPTIONS } from "../../constants";
import LabelPicker from "./LabelPicker";
import StatusIcon from "./StatusIcon";
import Avatar from "./Avatar";

type SubMenu = "status" | "priority" | "assignee" | "labels" | "estimate" | null;

interface ContextMenuProps {
  issue: Issue;
  issues: Issue[];
  x: number;
  y: number;
  onClose: () => void;
  onRefresh: () => void;
  allLabels: string[];
  onConfigLabelsChange?: (labels: Record<string, string>) => void;
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
    icon: (
      <svg className="w-4 h-4" fill="none" viewBox="0 0 16 16" stroke="currentColor" strokeWidth="1.5">
        <circle cx="8" cy="6" r="2.5" />
        <path d="M3.5 13.5C4 11 5.8 9.5 8 9.5s4 1.5 4.5 4" strokeLinecap="round" />
      </svg>
    ),
  },
  {
    id: "labels",
    label: "Labels",
    shortcut: "L",
    icon: (
      <svg className="w-4 h-4" fill="none" viewBox="0 0 16 16" stroke="currentColor" strokeWidth="1.5">
        <path strokeLinecap="round" strokeLinejoin="round" d="M7.2 2H4a2 2 0 00-2 2v3.2c0 .4.2.8.5 1.1l5.8 5.8c.6.6 1.5.6 2.1 0l3.2-3.2c.6-.6.6-1.5 0-2.1L7.8 2.5c-.3-.3-.7-.5-1.1-.5z" />
        <circle cx="5.5" cy="5.5" r="0.75" fill="currentColor" />
      </svg>
    ),
  },
  {
    id: "estimate",
    label: "Estimate",
    shortcut: "E",
    icon: (
      <svg className="w-4 h-4" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinejoin="round">
        <path d="M8 2L14 14H2L8 2Z" />
      </svg>
    ),
  },
];

const Chevron = () => (
  <svg className="w-3 h-3 ml-auto text-[var(--color-text-muted)]" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
    <path strokeLinecap="round" strokeLinejoin="round" d="M9 5l7 7-7 7" />
  </svg>
);

export default function ContextMenu({
  issue,
  issues,
  x,
  y,
  onClose,
  onRefresh,
  allLabels,
  onConfigLabelsChange,
}: ContextMenuProps) {
  const menuRef = useRef<HTMLDivElement>(null);
  const itemRefs = useRef<Map<string, HTMLButtonElement>>(new Map());
  const [subMenu, setSubMenu] = useState<SubMenu>(null);
  const [focusIndex, setFocusIndex] = useState(-1);
  const [confirmDelete, setConfirmDelete] = useState(false);
  const [filterText, setFilterText] = useState("");
  const [subMenuOffset, setSubMenuOffset] = useState(0);
  const onCloseRef = useRef(onClose);
  onCloseRef.current = onClose;

  const closeAll = useCallback(() => {
    setSubMenu(null);
    setConfirmDelete(false);
    onCloseRef.current();
  }, []);

  const [pos, setPos] = useState({ top: y, left: x });
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
    setPos({ top, left });
  }, [x, y]);

  // Click-outside: armed after first frame to avoid closing on the triggering right-click
  useEffect(() => {
    let armed = false;
    const armTimer = requestAnimationFrame(() => { armed = true; });
    const handleClick = (e: MouseEvent) => {
      if (!armed) return;
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) closeAll();
    };
    document.addEventListener("mousedown", handleClick);
    return () => {
      cancelAnimationFrame(armTimer);
      document.removeEventListener("mousedown", handleClick);
    };
  }, [closeAll]);

  const handleAction = useCallback(async (type: string, payload: Record<string, unknown>) => {
    await addDraft(issue.id, type, payload);
    onRefresh();
    closeAll();
  }, [issue.id, onRefresh, closeAll]);

  const handleLabelToggle = useCallback(async (label: string) => {
    const current = issue.labels || [];
    const labels = current.includes(label) ? current.filter(l => l !== label) : [...current, label];
    await addDraft(issue.id, "UPDATE", { labels });
    onRefresh();
  }, [issue.id, issue.labels, onRefresh]);

  const knownPeople = useMemo(() => {
    const byEmail = new Map<string, string>();
    for (const i of issues) {
      for (const val of [i.created_by, i.assignee]) {
        if (!val) continue;
        const email = val.match(/<([^>]+)>/)?.[1]?.toLowerCase() || val;
        if (!byEmail.has(email)) byEmail.set(email, val);
      }
    }
    const all = Array.from(byEmail.values()).sort((a, b) => a.split(" <")[0].localeCompare(b.split(" <")[0]));
    const q = filterText.toLowerCase();
    if (!q) return all;
    return all.filter(p => p.toLowerCase().includes(q));
  }, [issues, filterText]);

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
      if (key === "backspace" && (e.metaKey || e.ctrlKey)) { e.preventDefault(); setConfirmDelete(true); return; }
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
        setConfirmDelete(true);
        return;
      }
    };
    document.addEventListener("keydown", handler, true);
    return () => document.removeEventListener("keydown", handler, true);
  }, [subMenu, closeAll, focusIndex, openSubMenu]);

  const q = filterText.toLowerCase();

  const filteredStatuses = useMemo(() =>
    STATUS_OPTIONS.filter(opt => !q || opt.label.toLowerCase().includes(q)),
    [q],
  );

  const filteredPriorities = useMemo(() =>
    PRIORITY_OPTIONS.filter(opt => !q || opt.label.toLowerCase().includes(q)),
    [q],
  );

  const filteredEstimates = useMemo(() =>
    ESTIMATE_OPTIONS.filter(est => {
      if (!q) return true;
      const label = est === 0 ? "no estimate" : `${est} point`;
      return label.includes(q);
    }),
    [q],
  );

  const filterInput = (placeholder: string) => (
    <>
      <div className="px-3 py-1.5">
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
      return (
        <>
          {filterInput("Change status...")}
          {filteredStatuses.map((opt, i) => {
            const isCurrent = opt.value === issue.status;
            return (
              <button key={opt.value} onClick={() => handleAction("UPDATE", { status: opt.value })} className={`flex items-center gap-2 w-full px-3 py-1.5 text-sm transition-colors hover:bg-[var(--color-bg-hover)] ${isCurrent ? "text-[var(--color-accent-primary)]" : "text-[var(--color-text-primary)]"}`}>
                <StatusIcon status={opt.value} size={14} />
                <span>{opt.label}</span>
                {isCurrent && <CheckIcon />}
                {!isCurrent && <span className="ml-auto text-xs text-[var(--color-text-muted)]">{i + 1}</span>}
              </button>
            );
          })}
          {filteredStatuses.length === 0 && <div className="px-3 py-1.5 text-sm text-[var(--color-text-muted)]">No matching statuses</div>}
        </>
      );
    }
    if (subMenu === "priority") {
      return (
        <>
          {filterInput("Set priority...")}
          {filteredPriorities.map(opt => {
            const isCurrent = opt.value === (issue.priority || 0);
            return (
              <button key={opt.value} onClick={() => handleAction("UPDATE", { priority: opt.value })} className={`flex items-center gap-2 w-full px-3 py-1.5 text-sm transition-colors hover:bg-[var(--color-bg-hover)] ${isCurrent ? "text-[var(--color-accent-primary)]" : "text-[var(--color-text-primary)]"}`}>
                <PriorityDot priority={opt.value} />
                <span>{opt.label}</span>
                {isCurrent && <CheckIcon />}
              </button>
            );
          })}
          {filteredPriorities.length === 0 && <div className="px-3 py-1.5 text-sm text-[var(--color-text-muted)]">No matching priorities</div>}
        </>
      );
    }
    if (subMenu === "estimate") {
      return (
        <>
          {filterInput("Set estimate...")}
          {filteredEstimates.map(est => {
            const isCurrent = est === (issue.estimate || 0);
            return (
              <button key={est} onClick={() => handleAction("UPDATE", { estimate: est })} className={`flex items-center gap-2 w-full px-3 py-1.5 text-sm transition-colors hover:bg-[var(--color-bg-hover)] ${isCurrent ? "text-[var(--color-accent-primary)]" : "text-[var(--color-text-primary)]"}`}>
                <span>{est === 0 ? "No estimate" : `${est} Point${est !== 1 ? "s" : ""}`}</span>
                {isCurrent && <CheckIcon />}
              </button>
            );
          })}
          {filteredEstimates.length === 0 && <div className="px-3 py-1.5 text-sm text-[var(--color-text-muted)]">No matching estimates</div>}
        </>
      );
    }
    if (subMenu === "assignee") {
      return (
        <>
          {filterInput("Set assignee...")}
          <div className="max-h-[240px] overflow-y-auto">
            {issue.assignee && !q && (
              <button onClick={() => handleAction("UPDATE", { assignee: "" })} className="flex items-center gap-2 w-full px-3 py-1.5 text-sm text-[var(--color-text-muted)] hover:bg-[var(--color-bg-hover)] transition-colors">
                Remove assignee
              </button>
            )}
            {knownPeople.map(person => {
              const isCurrent = person === issue.assignee;
              return (
                <button key={person} onClick={() => handleAction("UPDATE", { assignee: person })} className={`flex items-center gap-2 w-full px-3 py-1.5 text-sm transition-colors hover:bg-[var(--color-bg-hover)] ${isCurrent ? "text-[var(--color-accent-primary)]" : "text-[var(--color-text-primary)]"}`}>
                  <Avatar name={person} size="sm" />
                  <span className="truncate">{person.split(" <")[0]}</span>
                  {isCurrent && <CheckIcon />}
                </button>
              );
            })}
            {knownPeople.length === 0 && (
              <div className="px-3 py-1.5 text-sm text-[var(--color-text-muted)]">No matching people</div>
            )}
          </div>
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

  if (confirmDelete) {
    return createPortal(
      <div ref={menuRef} style={{ position: "fixed", top: pos.top, left: pos.left }} className="z-[100] min-w-[220px] bg-[var(--color-bg-elevated)] border border-[var(--color-border-default)] rounded-[var(--radius-lg)] shadow-[var(--shadow-popover)] p-3">
        <p className="text-sm text-[var(--color-text-primary)] mb-3">Delete this issue?</p>
        <div className="flex items-center gap-2">
          <button onClick={() => handleAction("DELETE", {})} className="px-3 py-1.5 text-sm font-medium rounded-[var(--radius-md)] bg-[var(--color-error)] text-white hover:opacity-90 transition-opacity">
            Delete
          </button>
          <button onClick={() => setConfirmDelete(false)} className="px-3 py-1.5 text-sm font-medium rounded-[var(--radius-md)] text-[var(--color-text-secondary)] hover:bg-[var(--color-bg-hover)] transition-colors">
            Cancel
          </button>
        </div>
      </div>,
      document.body,
    );
  }

  return createPortal(
    <div ref={menuRef} style={{ position: "fixed", top: pos.top, left: pos.left }} className="z-[100] flex items-start">
      {/* Main menu */}
      <div className="min-w-[200px] bg-[var(--color-bg-elevated)] border border-[var(--color-border-default)] rounded-[var(--radius-lg)] shadow-[var(--shadow-popover)] overflow-hidden">
        {MENU_ITEMS.map((item, i) => (
          <button
            key={item.id}
            ref={el => { if (el) itemRefs.current.set(item.id, el); }}
            onClick={() => openSubMenu(item.id)}
            onMouseEnter={() => { setFocusIndex(i); openSubMenu(item.id); }}
            className={`flex items-center gap-2.5 w-full px-3 py-1.5 text-sm text-[var(--color-text-primary)] transition-colors hover:bg-[var(--color-bg-hover)] ${focusIndex === i ? "bg-[var(--color-bg-hover)]" : ""}`}
          >
            <span className="text-[var(--color-text-muted)] w-4 shrink-0 flex items-center justify-center">{item.icon}</span>
            <span>{item.label}</span>
            <span className="ml-auto flex items-center gap-1.5">
              <span className="text-xs text-[var(--color-text-muted)]">{item.shortcut}</span>
              <Chevron />
            </span>
          </button>
        ))}
        <div className="my-1 border-t border-[var(--color-border-subtle)]" />
        <button
          onClick={() => setConfirmDelete(true)}
          onMouseEnter={() => { setFocusIndex(MENU_ITEMS.length); setSubMenu(null); }}
          className={`flex items-center gap-2.5 w-full px-3 py-1.5 text-sm text-[var(--color-error)] transition-colors hover:bg-[var(--color-bg-hover)] ${focusIndex === MENU_ITEMS.length ? "bg-[var(--color-bg-hover)]" : ""}`}
        >
          <span className="w-4 shrink-0 flex items-center justify-center">
            <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M14.74 9l-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 01-2.244 2.077H8.084a2.25 2.25 0 01-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 00-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 013.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 00-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 00-7.5 0" />
            </svg>
          </span>
          <span>Delete</span>
          <span className="ml-auto text-xs opacity-70">{navigator.platform.includes("Mac") ? "⌘" : "Ctrl"}{"⌫"}</span>
        </button>
      </div>

      {/* Flyout sub-menu */}
      {subMenu && (
        <div
          className="min-w-[200px] max-w-[280px] bg-[var(--color-bg-elevated)] border border-[var(--color-border-default)] rounded-[var(--radius-lg)] shadow-[var(--shadow-popover)] overflow-hidden ml-1"
          style={{ marginTop: Math.max(0, subMenuOffset - 30) }}
        >
          {renderSubMenuPanel()}
        </div>
      )}
    </div>,
    document.body,
  );
}

function CheckIcon() {
  return (
    <svg className="w-4 h-4 ml-auto shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}>
      <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
    </svg>
  );
}

function PriorityDot({ priority }: { priority: number }) {
  const colors: Record<number, string> = {
    0: "var(--color-text-muted)",
    1: "var(--color-error)",
    2: "var(--color-warning)",
    3: "var(--color-text-secondary)",
    4: "var(--color-text-muted)",
  };
  return <span className="w-3 h-3 rounded-full shrink-0" style={{ background: colors[priority] || colors[0] }} />;
}
