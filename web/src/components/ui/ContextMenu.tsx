import { useRef, useEffect, useState, useLayoutEffect, useCallback, useMemo } from "react";
import { createPortal } from "react-dom";
import type { Issue } from "../../api/client";
import { useCycles } from "../../api/queries";
import { useDeleteIssue, useUpdateIssue } from "../../api/mutations";
import LabelPicker from "./LabelPicker";
import StatusPicker from "./StatusPicker";
import PriorityPicker from "./PriorityPicker";
import EstimatePicker from "./EstimatePicker";
import CyclePicker from "./CyclePicker";
import AssigneePicker from "./AssigneePicker";
import DeleteIssueDialog from "./DeleteIssueDialog";
import { UserRound, Triangle, RefreshCw, Trash2, Tag, Unlink, MessageSquare } from "lucide-react";
import { toggleLabel } from "../../utils/labels";
import { collectKnownPeople } from "../../utils/issues";
import { useKeyboardShortcuts } from "../../keyboard";
import { Menu, MenuItem, MenuDivider } from "./Menu";
import { SubMenu } from "./SubMenu";
import { useOptionalAgentChat } from "../Agent/agent-context";
import { IssueChatsList } from "../Agent/IssueChats";
import { useIssueChats } from "../Agent/useIssueChats";

interface ContextMenuProps {
  issue: Issue;
  issues: Issue[];
  x: number;
  y: number;
  onClose: () => void;
  allLabels: string[];
  contributors: string[];
  /** Offer "Create label" for labels without a configured color. */
  canCreateLabels?: boolean;
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
  issue, issues, contributors, onUpdate, onClose,
}: {
  issue: Issue; issues: Issue[]; contributors: string[];
  onUpdate: (patch: Partial<Issue>) => void;
  onClose: () => void;
}) {
  const people = useMemo(() => collectKnownPeople(issues, contributors), [issues, contributors]);
  return (
    <AssigneePicker
      people={people}
      current={issue.assignee}
      onSelect={person => onUpdate({ assignee: person ?? "" })}
      onClose={onClose}
    />
  );
}

export default function ContextMenu({
  issue, issues, x, y, onClose, allLabels, contributors, canCreateLabels,
}: ContextMenuProps) {
  const menuRef = useRef<HTMLDivElement>(null);
  const [confirmDelete, setConfirmDelete] = useState(false);
  const { cycles } = useCycles();
  const onCloseRef = useRef(onClose);
  useEffect(() => { onCloseRef.current = onClose; }, [onClose]);

  const subIssueCount = useMemo(() => issues.filter(i => i.parent_id === issue.id).length, [issues, issue.id]);
  const agent = useOptionalAgentChat();
  const { chats, holder } = useIssueChats(issue.id);

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

  const updateIssue = useUpdateIssue();
  const { mutate: deleteIssue } = useDeleteIssue();

  const handleUpdate = useCallback((patch: Partial<Issue>) => {
    closeAll();
    updateIssue(issue.id, patch);
  }, [issue.id, closeAll, updateIssue]);

  const handleDelete = useCallback((cascade: boolean) => {
    closeAll();
    deleteIssue({ issueId: issue.id, cascade });
  }, [issue.id, closeAll, deleteIssue]);

  const handleLabelToggle = useCallback((label: string) => {
    const labels = toggleLabel(issue.labels || [], label);
    updateIssue(issue.id, { labels });
  }, [issue.id, issue.labels, updateIssue]);

  useKeyboardShortcuts({
    scope: "context-menu-delete",
    priority: "overlay",
    shortcuts: [
      { id: "context-menu.delete.meta", key: "Backspace", label: "Delete", showInHelp: false, modifiers: { meta: true }, allowInEditable: true, run: () => setConfirmDelete(true) },
      { id: "context-menu.delete.ctrl", key: "Backspace", label: "Delete", showInHelp: false, modifiers: { ctrl: true }, allowInEditable: true, run: () => setConfirmDelete(true) },
    ],
  });

  // The dialog replaces the menu: with no menuRef, the outside-click handler can't close it.
  if (confirmDelete) {
    return <DeleteIssueDialog issue={issue} subIssueCount={subIssueCount} onDelete={handleDelete} onCancel={closeAll} />;
  }

  return createPortal(
    <div ref={menuRef} style={{ position: "fixed", top: y, left: x }} className="z-[100]">
      <Menu onClose={closeAll} aria-label="Issue actions">
        <SubMenu label="Status" icon={<StatusIcon />} shortcut="S"
          renderPanel={(close) => <StatusPicker current={issue.status} onSelect={v => handleUpdate({ status: v })} onClose={close} />}
        />
        <SubMenu label="Priority" icon={<PriorityIcon />} shortcut="P"
          renderPanel={(close) => <PriorityPicker current={issue.priority || 0} onSelect={v => handleUpdate({ priority: v })} onClose={close} />}
        />
        <SubMenu label="Assignee" icon={<UserRound size={16} />} shortcut="A"
          renderPanel={(close) => <AssigneePanel issue={issue} issues={issues} contributors={contributors} onUpdate={handleUpdate} onClose={close} />}
        />
        <SubMenu label="Labels" icon={<Tag size={16} />} shortcut="L"
          renderPanel={() => (
            <LabelPicker allLabels={allLabels} selected={issue.labels || []} onToggle={handleLabelToggle} canCreateLabels={canCreateLabels} onClose={closeAll} />
          )}
        />
        <SubMenu label="Estimate" icon={<Triangle size={16} />} shortcut="E"
          renderPanel={(close) => <EstimatePicker current={issue.estimate || 0} onSelect={v => handleUpdate({ estimate: v })} onClose={close} />}
        />
        <SubMenu label="Cycle" icon={<RefreshCw size={16} />} shortcut="C"
          renderPanel={(close) => <CyclePicker cycles={cycles} current={issue.cycle_id} onSelect={id => handleUpdate({ cycle_id: id })} onClose={close} />}
        />
        {chats.length > 0 && <MenuDivider />}
        {holder && agent && (
          <MenuItem label="Open work chat" icon={<MessageSquare size={16} />} onClick={() => agent.openChat(holder.id)} />
        )}
        {chats.length > 0 && (
          <SubMenu label="Chats" icon={<MessageSquare size={16} />}
            suffix={<span className="text-xs text-[var(--color-text-muted)] tabular-nums">{chats.length}</span>}
            renderPanel={() => <IssueChatsList issueId={issue.id} onClose={closeAll} bare />}
          />
        )}
        {issue.parent_id && (
          <>
            <MenuDivider />
            <MenuItem label="Remove from parent" icon={<Unlink size={16} />} onClick={() => handleUpdate({ parent_id: "" })} />
          </>
        )}
        <MenuDivider />
        <MenuItem
          label="Delete" icon={<Trash2 size={16} />} destructive keepOpen
          onClick={() => setConfirmDelete(true)}
          suffix={<span className="text-xs opacity-70">{navigator.platform.includes("Mac") ? "⌘" : "Ctrl"}{"⌫"}</span>}
        />
      </Menu>
    </div>,
    document.body,
  );
}
