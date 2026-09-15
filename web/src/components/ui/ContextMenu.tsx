import { useRef, useEffect, useState, useLayoutEffect, useCallback, useMemo } from "react";
import { createPortal } from "react-dom";
import type { Issue, Cycle } from "../../api/client";
import { addDraft, fetchCycles } from "../../api/client";
import LabelPicker from "./LabelPicker";
import StatusPicker from "./StatusPicker";
import PriorityPicker from "./PriorityPicker";
import EstimatePicker from "./EstimatePicker";
import Avatar from "./Avatar";
import { UserRound, Triangle, RefreshCw, Trash2, Check, Tag, Unlink } from "lucide-react";
import { toggleLabel } from "../../utils/labels";
import { collectKnownPeople } from "../../utils/issues";
import { useKeyboardShortcuts } from "../../keyboard";
import { Menu, MenuItem, MenuDivider } from "./Menu";
import { SubMenu } from "./SubMenu";

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

const StatusIcon = () => (
  <svg className="w-4 h-4" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5">
    <circle cx="8" cy="8" r="5.5" />
    <path d="M8 2.5A5.5 5.5 0 0113.5 8" strokeLinecap="round" />
  </svg>
);

const PriorityIcon = () => (
  <svg className="w-4 h-4" viewBox="0 0 16 16" fill="currentColor">
    <rect x="1" y="8" width="4" height="3" rx="1" />
    <rect x="6" y="5" width="4" height="6" rx="1" opacity="0.5" />
    <rect x="11" y="2" width="4" height="9" rx="1" opacity="0.2" />
  </svg>
);

function AssigneePanel({
  issue, issues, contributors, onAction, onClose,
}: {
  issue: Issue; issues: Issue[]; contributors: string[];
  onAction: (type: string, payload: Record<string, unknown>) => void;
  onClose: () => void;
}) {
  const [filterText, setFilterText] = useState("");
  const knownPeople = useMemo(() => {
    const all = collectKnownPeople(issues, contributors);
    const q = filterText.toLowerCase();
    if (!q) return all;
    return all.filter(p => p.toLowerCase().includes(q));
  }, [issues, contributors, filterText]);
  const q = filterText.toLowerCase();

  return (
    <div onKeyDown={e => { if (e.key === "Escape") { e.stopPropagation(); if (filterText) setFilterText(""); else onClose(); } }}>
      <div className="px-3 py-1.5">
        <input
          autoFocus
          value={filterText}
          onChange={e => setFilterText(e.target.value)}
          placeholder="Set assignee..."
          className="w-full text-sm bg-transparent text-[var(--color-text-primary)] placeholder:text-[var(--color-text-muted)] outline-none"
        />
      </div>
      <div className="border-t border-[var(--color-border-subtle)]" />
      <div className="max-h-60 overflow-y-auto">
        {issue.assignee && !q && (
          <button onClick={() => onAction("UPDATE", { assignee: "" })} className="flex items-center gap-2 w-full px-3 py-1.5 text-sm text-[var(--color-text-muted)] hover:bg-[var(--color-hover-surface-3)] transition-colors">
            Remove assignee
          </button>
        )}
        {knownPeople.map(person => {
          const isCurrent = person === issue.assignee;
          return (
            <button key={person} onClick={() => onAction("UPDATE", { assignee: person })} className={`flex items-center gap-2 w-full px-3 py-1.5 text-sm transition-colors hover:bg-[var(--color-hover-surface-3)] ${isCurrent ? "text-[var(--color-accent-primary)]" : "text-[var(--color-text-primary)]"}`}>
              <Avatar name={person} size="sm" />
              <span className="truncate">{person.split(" <")[0]}</span>
              {isCurrent && <Check className="w-4 h-4 ml-auto shrink-0" />}
            </button>
          );
        })}
        {knownPeople.length === 0 && (
          <div className="px-3 py-1.5 text-sm text-[var(--color-text-muted)]">No matching people</div>
        )}
      </div>
    </div>
  );
}

function CyclePanel({
  issue, cycles, onAction, onClose,
}: {
  issue: Issue; cycles: Cycle[];
  onAction: (type: string, payload: Record<string, unknown>) => void;
  onClose: () => void;
}) {
  if (cycles.length === 0) {
    return <div className="px-3 py-1.5 text-sm text-[var(--color-text-muted)]">Cycles not configured</div>;
  }
  return (
    <div onKeyDown={e => { if (e.key === "Escape") { e.stopPropagation(); onClose(); } }}>
      <div className="px-3 py-1 text-[11px] font-medium text-[var(--color-text-muted)]">Move to cycle...</div>
      <div className="border-t border-[var(--color-border-subtle)]" />
      {issue.cycle_id && (
        <button onClick={() => onAction("UPDATE", { cycle_id: "" })} className="flex items-center gap-2 w-full px-3 py-1.5 text-sm text-[var(--color-text-muted)] hover:bg-[var(--color-hover-surface-3)] transition-colors">
          No cycle
        </button>
      )}
      {cycles.filter(c => c.status !== 'completed').map(c => {
        const isCurrent = c.id === issue.cycle_id;
        return (
          <button key={c.id} onClick={() => onAction("UPDATE", { cycle_id: c.id })} className={`flex items-center gap-2 w-full px-3 py-1.5 text-sm transition-colors hover:bg-[var(--color-hover-surface-3)] ${isCurrent ? "text-[var(--color-accent-primary)]" : "text-[var(--color-text-primary)]"}`}>
            <span>Cycle {c.number}</span>
            <span className="text-xs text-[var(--color-text-muted)] capitalize">{c.status}</span>
            {isCurrent && <Check className="w-4 h-4 ml-auto shrink-0" />}
          </button>
        );
      })}
    </div>
  );
}

export default function ContextMenu({
  issue, issues, x, y, onClose, onRefresh, allLabels, contributors, onConfigLabelsChange, patchIssue,
}: ContextMenuProps) {
  const menuRef = useRef<HTMLDivElement>(null);
  const [confirmDelete, setConfirmDelete] = useState<false | "confirm" | "choose">(false);
  const [cycles, setCycles] = useState<Cycle[]>([]);
  const onCloseRef = useRef(onClose);
  useEffect(() => { onCloseRef.current = onClose; }, [onClose]);

  useEffect(() => {
    fetchCycles().then(data => {
      if (data.enabled && data.cycles) setCycles(data.cycles);
    }).catch(() => {});
  }, []);

  const hasChildren = useMemo(() => issues.some(i => i.parent_id === issue.id), [issues, issue.id]);

  const closeAll = useCallback(() => {
    setConfirmDelete(false);
    onCloseRef.current();
  }, [setConfirmDelete]);

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

  useEffect(() => {
    let armed = false;
    const armTimer = requestAnimationFrame(() => { armed = true; });
    const handleClick = (e: MouseEvent) => {
      if (!armed) return;
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        const flyout = document.querySelector("[data-submenu-flyout]");
        if (flyout?.contains(e.target as Node)) return;
        closeAll();
      }
    };
    document.addEventListener("mousedown", handleClick);
    return () => { cancelAnimationFrame(armTimer); document.removeEventListener("mousedown", handleClick); };
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

  useKeyboardShortcuts({
    scope: "context-menu-delete",
    priority: "overlay",
    shortcuts: [
      { id: "context-menu.delete.meta", key: "Backspace", label: "Delete", showInHelp: false, modifiers: { meta: true }, allowInEditable: true, run: () => setConfirmDelete(hasChildren ? "choose" : "confirm") },
      { id: "context-menu.delete.ctrl", key: "Backspace", label: "Delete", showInHelp: false, modifiers: { ctrl: true }, allowInEditable: true, run: () => setConfirmDelete(hasChildren ? "choose" : "confirm") },
    ],
  });

  if (confirmDelete === "choose") {
    return createPortal(
      <div ref={menuRef} style={{ position: "fixed", top: y, left: x }} className="z-[100] min-w-55 bg-[var(--color-surface-3)] border border-[var(--color-border-default)] rounded-[var(--radius-lg)] shadow-[var(--shadow-popover)] p-3">
        <p className="text-sm text-[var(--color-text-primary)] mb-3">This issue has sub-issues. What should happen to them?</p>
        <div className="flex flex-col gap-2">
          <button onClick={() => handleAction("DELETE", { cascade: false })} className="px-3 py-2 text-sm font-medium rounded-[var(--radius-md)] bg-[var(--color-error)] text-white hover:opacity-90 transition-opacity text-left">Keep sub-issues</button>
          <button onClick={() => handleAction("DELETE", { cascade: true })} className="px-3 py-2 text-sm font-medium rounded-[var(--radius-md)] border border-[var(--color-error)] text-[var(--color-error)] hover:bg-[var(--color-error)] hover:text-white transition-colors text-left">Delete sub-issues too</button>
          <button onClick={() => setConfirmDelete(false)} className="px-3 py-2 text-sm font-medium rounded-[var(--radius-md)] text-[var(--color-text-secondary)] hover:bg-[var(--color-hover-surface-3)] transition-colors text-left">Cancel</button>
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
          <button onClick={() => handleAction("DELETE", {})} className="px-3 py-2 text-sm font-medium rounded-[var(--radius-md)] bg-[var(--color-error)] text-white hover:opacity-90 transition-opacity">Delete</button>
          <button onClick={() => setConfirmDelete(false)} className="px-3 py-2 text-sm font-medium rounded-[var(--radius-md)] text-[var(--color-text-secondary)] hover:bg-[var(--color-hover-surface-3)] transition-colors">Cancel</button>
        </div>
      </div>,
      document.body,
    );
  }

  return createPortal(
    <div ref={menuRef} style={{ position: "fixed", top: y, left: x }} className="z-[100]">
      <Menu onClose={closeAll} aria-label="Issue actions">
        <SubMenu label="Status" icon={<StatusIcon />} shortcut="S"
          renderPanel={(close) => <StatusPicker current={issue.status} onSelect={v => handleAction("UPDATE", { status: v })} onClose={close} />}
        />
        <SubMenu label="Priority" icon={<PriorityIcon />} shortcut="P"
          renderPanel={(close) => <PriorityPicker current={issue.priority || 0} onSelect={v => handleAction("UPDATE", { priority: v })} onClose={close} />}
        />
        <SubMenu label="Assignee" icon={<UserRound size={16} />} shortcut="A"
          renderPanel={(close) => <AssigneePanel issue={issue} issues={issues} contributors={contributors} onAction={handleAction} onClose={close} />}
        />
        <SubMenu label="Labels" icon={<Tag size={16} />} shortcut="L"
          renderPanel={() => (
            <LabelPicker allLabels={allLabels} selected={issue.labels || []} onToggle={handleLabelToggle} onConfigLabelsChange={onConfigLabelsChange} onClose={closeAll} />
          )}
        />
        <SubMenu label="Estimate" icon={<Triangle size={16} />} shortcut="E"
          renderPanel={(close) => <EstimatePicker current={issue.estimate || 0} onSelect={v => handleAction("UPDATE", { estimate: v })} onClose={close} />}
        />
        <SubMenu label="Cycle" icon={<RefreshCw size={16} />} shortcut="C"
          renderPanel={(close) => <CyclePanel issue={issue} cycles={cycles} onAction={handleAction} onClose={close} />}
        />
        {issue.parent_id && (
          <>
            <MenuDivider />
            <MenuItem label="Remove from parent" icon={<Unlink size={16} />} onClick={() => handleAction("UPDATE", { parent_id: "" })} />
          </>
        )}
        <MenuDivider />
        <MenuItem
          label="Delete" icon={<Trash2 size={16} />} destructive
          onClick={() => setConfirmDelete(hasChildren ? "choose" : "confirm")}
          suffix={<span className="text-xs opacity-70">{navigator.platform.includes("Mac") ? "⌘" : "Ctrl"}{"⌫"}</span>}
        />
      </Menu>
    </div>,
    document.body,
  );
}
