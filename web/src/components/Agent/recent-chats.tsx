import { MessageSquare } from "lucide-react";
import type { ChatMeta } from "../../api/chat";
import { formatRelativeTime } from "../../utils/format";
import { MenuItem } from "../ui";

/** How many chats the switcher and the status-bar history list. */
export const RECENT_CHAT_LIMIT = 20;

const runningDot = <span className="inline-block w-3 h-3 border-[1.5px] border-[var(--color-text-muted)] border-t-transparent rounded-full animate-spin" />;

/** Menu items for the most recent chats (chats arrive newest first). */
export function recentChatItems(chats: ChatMeta[], activeId: string | null, onPick: (id: string) => void) {
  return chats.slice(0, RECENT_CHAT_LIMIT).map((c) => (
    <MenuItem
      key={c.id}
      label={c.title || "Untitled chat"}
      truncate
      active={c.id === activeId}
      icon={c.state === "running" ? runningDot : <MessageSquare size={14} />}
      suffix={<span className="whitespace-nowrap text-xs text-[var(--color-text-muted)]">{formatRelativeTime(c.updated_at)}</span>}
      onClick={() => onPick(c.id)}
    />
  ));
}
