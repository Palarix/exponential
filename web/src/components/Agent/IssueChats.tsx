import { useContext, useEffect, useRef, useState, type RefObject } from "react";
import { Archive, Ellipsis, Hammer, MessageSquare, MessageSquareDot, MessageSquareMore, MessageSquareWarning } from "lucide-react";
import type { ChatMeta } from "../../api/chat";
import { AppNavContext } from "../../app/contexts";
import { cn } from "../../utils/cn";
import { formatRelativeTime } from "../../utils/format";
import { indicatorPill, Menu, MenuDivider, MenuFilter, MenuItem, MenuLabel, Popover } from "../ui";
import { useOptionalAgentChat } from "./agent-context";
import { chatState, enterWorkBlocker, filterChats, type IssueChatState } from "./issue-chats";
import { chatCount, useIssueChats } from "./useIssueChats";
import { useStartWorkChat } from "./useWorkMode";

// Muted like every row indicator: the shape alone tells the state.
const STATE_ICONS: Record<IssueChatState, { Icon: typeof MessageSquare; label: string }> = {
  waiting: { Icon: MessageSquareDot, label: "Waiting for your approval" },
  running: { Icon: MessageSquareMore, label: "Running" },
  interrupted: { Icon: MessageSquareWarning, label: "Interrupted" },
  work: { Icon: Hammer, label: "In Work Mode" },
  idle: { Icon: MessageSquare, label: "Idle" },
};

/** A chat state as a message-square icon. */
export function IssueChatStateIcon({ state, size = 14 }: { state: IssueChatState; size?: number }) {
  const { Icon, label } = STATE_ICONS[state];
  return <Icon size={size} strokeWidth={1.75} className="shrink-0 text-[var(--color-text-muted)]" aria-label={label} />;
}

/**
 * The chat icon on a backlog row or board card. It opens the chat in Work
 * Mode on the issue, or else the chats popover. Clicks never reach the row.
 */
export function IssueChatIcon({ issueId }: { issueId: string }) {
  const agent = useOptionalAgentChat();
  const { chats, holder, state } = useIssueChats(issueId);
  const ref = useRef<HTMLButtonElement>(null);
  const [open, setOpen] = useState(false);
  if (!agent || !state) return null;

  const label = holder ? `Open work chat: ${holder.title || "Untitled chat"}` : `${chatCount(chats.length)} on ${issueId}`;
  return (
    // Portaled popover events bubble through React to here; keep them off the row.
    <span className="contents" onClick={(e) => e.stopPropagation()} onPointerDown={(e) => e.stopPropagation()}>
      <button
        ref={ref}
        type="button"
        aria-label={label}
        title={label}
        aria-haspopup={holder ? undefined : "menu"}
        aria-expanded={holder ? undefined : open}
        onClick={() => (holder ? agent.openChat(holder.id) : setOpen((o) => !o))}
        // The row indicator pill: it marks that chats exist; the tooltip has the count.
        className={`${indicatorPill} hover:bg-[var(--color-hover-surface-3)] transition-colors`}
      >
        <IssueChatStateIcon state={state} size={12} />
        {chats.length}
      </button>
      {open && <IssueChatsPopover issueId={issueId} anchorRef={ref} onClose={() => setOpen(false)} />}
    </span>
  );
}

/** The chats list in a popover, for the row icon and the issue sidebar. */
export function IssueChatsPopover({ issueId, anchorRef, onClose }: { issueId: string; anchorRef: RefObject<HTMLElement | null>; onClose(): void }) {
  return (
    <Popover anchorRef={anchorRef} placement="bottom-start" onClose={onClose}>
      <IssueChatsList issueId={issueId} onClose={onClose} />
    </Popover>
  );
}

/**
 * An issue's chats as a searchable menu, newest first. A row opens its
 * chat. With no chat in Work Mode on the issue, a row's actions (right-click,
 * its ⋯ button, or Shift+F10) offer "Enter work mode" under the row.
 */
export function IssueChatsList({ issueId, onClose, bare }: { issueId: string; onClose(): void; bare?: boolean }) {
  const agent = useOptionalAgentChat();
  const { chats, holder, archived } = useIssueChats(issueId);
  const nav = useContext(AppNavContext);
  const { startWork, noGitRoot } = useStartWorkChat();
  const [query, setQuery] = useState("");
  const [actionsFor, setActionsFor] = useState<string | null>(null);
  if (!agent) return null;

  const shown = filterChats(chats, query);
  const items = [<MenuFilter key="filter" value={query} onChange={setQuery} placeholder="Search chats…" />];
  if (shown.length === 0) items.push(<MenuLabel key="none" className="normal-case tracking-normal font-normal">No matching chats</MenuLabel>);
  for (const c of shown) {
    items.push(
      <ChatRow
        key={c.id}
        chat={c}
        issueId={issueId}
        hasActions={!holder}
        actionsOpen={actionsFor === c.id}
        onToggleActions={() => setActionsFor((id) => (id === c.id ? null : c.id))}
        onClick={() => agent.openChat(c.id)}
      />,
    );
    if (actionsFor === c.id && !holder) {
      const blocker = enterWorkBlocker(c, noGitRoot);
      items.push(
        <MenuItem key={`${c.id}-work`} label="Enter work mode" icon={<Hammer size={14} className="ml-5" />} disabled={!!blocker} onClick={() => startWork(issueId, c.id)} />,
      );
      if (blocker) items.push(<MenuLabel key={`${c.id}-why`} className="pl-12 normal-case tracking-normal font-normal">{blocker}</MenuLabel>);
    }
  }
  if (archived > 0 && nav) {
    items.push(<MenuDivider key="archived-divider" />);
    items.push(
      <MenuItem
        key="archived"
        label={`${archived} archived ${archived === 1 ? "chat" : "chats"}`}
        icon={<Archive size={14} />}
        onClick={() => {
          onClose();
          nav.openChatHistory(issueId);
        }}
      />,
    );
  }
  return (
    // Bare, it fills a submenu flyout, which is at most 280px wide.
    <Menu onClose={onClose} aria-label={`Chats on ${issueId}`} className={bare ? "w-70" : "w-96"} maxHeight="24rem" bare={bare}>
      {items}
    </Menu>
  );
}

interface ChatRowProps {
  chat: ChatMeta;
  issueId: string;
  hasActions: boolean;
  actionsOpen: boolean;
  onToggleActions(): void;
  onClick(): void;
  // Injected by Menu.
  _menuIndex?: number;
  _focused?: boolean;
  _onMouseEnter?: (index: number) => void;
  _onClick?: (index: number) => void;
}

/** One chat in IssueChatsList: a menu row with a ⋯ button for its actions. */
function ChatRow({ chat, issueId, hasActions, actionsOpen, onToggleActions, _menuIndex = 0, _focused, _onMouseEnter, _onClick }: ChatRowProps) {
  // Shift+F10 and the context-menu key open the focused row's actions.
  useEffect(() => {
    if (!_focused || !hasActions) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "ContextMenu" || (e.shiftKey && e.key === "F10")) {
        e.preventDefault();
        onToggleActions();
      }
    };
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [_focused, hasActions, onToggleActions]);

  const holder = chat.mode === "work" && chat.work?.issue_id === issueId;
  return (
    <div
      data-menu-index={_menuIndex}
      onMouseEnter={() => _onMouseEnter?.(_menuIndex)}
      onContextMenu={(e) => {
        if (!hasActions) return;
        e.preventDefault();
        onToggleActions();
      }}
      className={cn(
        "group flex items-center rounded-md transition-colors",
        _focused ? "bg-[var(--color-hover-surface-3)] text-[var(--color-text-primary)]" : "text-[var(--color-text-secondary)] hover:bg-[var(--color-hover-surface-3)] hover:text-[var(--color-text-primary)]",
      )}
    >
      <button
        type="button"
        role="menuitem"
        tabIndex={_focused ? 0 : -1}
        onClick={() => _onClick?.(_menuIndex)}
        className="flex flex-1 items-center gap-2 min-w-0 px-2 py-1.5 text-sm text-left outline-none"
      >
        <IssueChatStateIcon state={chatState(chat, issueId)} />
        <span className="flex-1 min-w-0 truncate">{chat.title || "Untitled chat"}</span>
        {holder && (
          <span className="shrink-0 px-1.5 rounded-sm text-[11px] font-medium bg-[var(--color-surface-4)] text-[var(--color-text-secondary)]">Work</span>
        )}
        <span className="shrink-0 whitespace-nowrap text-xs text-[var(--color-text-muted)]">{formatRelativeTime(chat.updated_at)}</span>
      </button>
      {hasActions && (
        <button
          type="button"
          tabIndex={-1}
          aria-label="Chat actions"
          aria-expanded={actionsOpen}
          onClick={onToggleActions}
          className={cn(
            "shrink-0 mr-1 p-1 rounded text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-hover-surface-4)]",
            _focused || actionsOpen ? "opacity-100" : "opacity-0 group-hover:opacity-100",
          )}
        >
          <Ellipsis size={14} />
        </button>
      )}
    </div>
  );
}
