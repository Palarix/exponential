import type { ReactNode } from "react";
import { MessageSquare, ShieldQuestion } from "lucide-react";
import type { ChatMeta } from "../../api/chat";
import { formatRelativeTime } from "../../utils/format";
import { MenuItem } from "../ui";

/** How many chats the switcher and the status-bar history list. */
export const RECENT_CHAT_LIMIT = 20;

/**
 * A chat's state in a chat list: waiting for an approval, running, or idle.
 * size is a Tailwind size class such as "w-3 h-3".
 */
export function chatStateIcon(c: ChatMeta, size: string, idle: ReactNode): ReactNode {
  if (c.waiting) {
    return (
      <span title="Waiting for your approval" className="shrink-0 inline-flex">
        <ShieldQuestion className={`${size} text-[var(--color-warning)]`} aria-label="Waiting for your approval" />
      </span>
    );
  }
  if (c.state === "running") {
    return <span className={`shrink-0 inline-block ${size} border-[1.5px] border-[var(--color-text-muted)] border-t-transparent rounded-full animate-spin`} />;
  }
  return idle;
}

/** Menu items for the most recent chats (chats arrive newest first). */
export function recentChatItems(chats: ChatMeta[], activeId: string | null, onPick: (id: string) => void) {
  return chats.slice(0, RECENT_CHAT_LIMIT).map((c) => (
    <MenuItem
      key={c.id}
      label={c.title || "Untitled chat"}
      truncate
      active={c.id === activeId}
      icon={chatStateIcon(c, "w-3 h-3", <MessageSquare size={14} />)}
      suffix={<span className="whitespace-nowrap text-xs text-[var(--color-text-muted)]">{formatRelativeTime(c.updated_at)}</span>}
      onClick={() => onPick(c.id)}
    />
  ));
}
