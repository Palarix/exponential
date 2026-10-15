import type { ReactNode } from "react";
import { CircleAlert, Hammer, MessageSquare } from "lucide-react";
import type { ChatMeta } from "../../api/chat";
import { formatRelativeTime } from "../../utils/format";
import { MenuItem } from "../ui";
import { chatIndicator, type ChatIndicator, type SeenChats } from "./agent-state-utils";

/** How many chats the switcher and the status-bar history list. */
export const RECENT_CHAT_LIMIT = 20;

const DOTS: Partial<Record<ChatIndicator, { color: string; label: string }>> = {
  waiting: { color: "var(--color-warning)", label: "Waiting for your approval" },
  interrupted: { color: "var(--color-error)", label: "Interrupted" },
  unseen: { color: "var(--color-info)", label: "New reply" },
};

/** How much of a failed turn's error the label shows. */
const ERROR_LABEL_MAX = 200;

/**
 * A chat's state in a chat list: a dot for waiting, interrupted and an
 * unseen reply, an alert circle for a failed turn, a spinner while running,
 * else idle. size is a Tailwind size class such as "w-3 h-3"; the dot sits
 * centered in it.
 */
export function chatStateIcon(c: ChatMeta, seen: SeenChats, size: string, idle: ReactNode): ReactNode {
  const state = chatIndicator(c, seen);
  if (state === "running") {
    return <span className={`shrink-0 inline-block ${size} border-[1.5px] border-[var(--color-text-muted)] border-t-transparent rounded-full animate-spin`} />;
  }
  if (state === "failed") {
    const msg = c.last_error ?? "";
    const label = `Last turn failed: ${msg.length > ERROR_LABEL_MAX ? `${msg.slice(0, ERROR_LABEL_MAX)}…` : msg}`;
    return (
      <span title={label} aria-label={label} role="img" className={`shrink-0 inline-flex items-center justify-center ${size} text-[var(--color-error)]`}>
        <CircleAlert className="w-full h-full" strokeWidth={2.5} aria-hidden />
      </span>
    );
  }
  const dot = DOTS[state];
  if (!dot) return idle;
  return (
    <span title={dot.label} aria-label={dot.label} role="img" className={`shrink-0 inline-flex items-center justify-center ${size}`}>
      <span className="w-2 h-2 rounded-full" style={{ background: dot.color }} />
    </span>
  );
}

/** The idle icon of a chat: a hammer in Work Mode, like the composer, else chatIcon. */
export function idleChatIcon(c: ChatMeta, size: string, chatIcon: ReactNode): ReactNode {
  if (c.mode !== "work") return chatIcon;
  return <Hammer className={`shrink-0 ${size} text-[var(--color-text-muted)]`} aria-label="In Work Mode" role="img" />;
}

/** Menu items for the most recent chats (chats arrive newest first). */
export function recentChatItems(chats: ChatMeta[], seen: SeenChats, activeId: string | null, onPick: (id: string) => void) {
  return chats.slice(0, RECENT_CHAT_LIMIT).map((c) => (
    <MenuItem
      key={c.id}
      label={c.title || "Untitled chat"}
      truncate
      active={c.id === activeId}
      icon={chatStateIcon(c, seen, "w-3 h-3", idleChatIcon(c, "w-3.5 h-3.5", <MessageSquare size={14} />))}
      suffix={<span className="whitespace-nowrap text-xs text-[var(--color-text-muted)]">{formatRelativeTime(c.updated_at)}</span>}
      onClick={() => onPick(c.id)}
    />
  ));
}
