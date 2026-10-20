import { IssueChatStateIcon } from "../Agent/IssueChats";
import { chatState } from "../Agent/issue-chats";
import { formatRelativeTime } from "../../utils/format";
import type { ChatMatch } from "./chat-search";
import type { CommandItem } from "./types";

/** The palette rows for matched chats; choosing one calls onOpen with its ID. */
export function chatCommandItems(matches: ChatMatch[], query: string, onOpen: (chatId: string) => void): CommandItem[] {
  return matches.map(({ chat, issueId }) => ({
    id: `chat:${chat.id}`,
    group: query.trim() ? "Chats" : "Recent Chats",
    icon: <IssueChatStateIcon state={chatState(chat, chat.work?.issue_id ?? "")} />,
    label: chat.title || "Untitled chat",
    meta: (
      <span className="flex items-center gap-2 shrink-0">
        {issueId && <span className="font-mono text-xs text-[var(--color-text-muted)]">{issueId}</span>}
        <span className="text-xs text-[var(--color-text-muted)] whitespace-nowrap">{formatRelativeTime(chat.updated_at)}</span>
      </span>
    ),
    action: () => onOpen(chat.id),
  }));
}
