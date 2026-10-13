import { createContext, useContext } from "react";
import type { AgentStatus, ChatContext, ChatMeta } from "../../api/chat";
import type { ChatPrefs } from "./agent-state-utils";

export interface AgentChat {
  status: AgentStatus | undefined;
  /** All chats, newest first; used for pill titles and states. */
  chats: ChatMeta[];
  popupOpen: boolean;
  /** The chat in the popup; null is a new chat that does not exist yet. */
  chatId: string | null;
  pills: string[];
  /** What the user has open now. */
  context: ChatContext;
  /** The composer's provider, model and effort for a chat; null is a new chat. */
  prefsFor(chatId: string | null): ChatPrefs;
  /** Records a pick in a chat's composer and remembers it for new chats. */
  setPrefsFor(chatId: string | null, prefs: ChatPrefs): void;
  openNew(): void;
  /** Opens a chat in the popup; a draft prefills its composer. */
  openChat(id: string, draft?: string): void;
  /** The text waiting to prefill a chat's composer. */
  draftFor(chatId: string | null): string | undefined;
  /** Called once a draft is in the composer. */
  clearDraft(chatId: string): void;
  /** The issue a chat's next message carries as context; null is a new chat. */
  pendingIssueFor(chatId: string | null): string | undefined;
  /** Clears a chat's pending context, once sent or removed. */
  consumeContext(chatId: string | null): void;
  toggle(): void;
  minimize(): void;
  close(): void;
  /** Hands the popup's chat to the Agent view: closes the popup and drops the chat's pill. */
  maximize(): void;
  /** Called once the popup's new chat exists on the server. */
  chatCreated(id: string): void;
  /** Called after a chat is deleted: drops its pill and leaves it if open. */
  chatDeleted(id: string): void;
}

export const AgentChatContext = createContext<AgentChat | null>(null);

export function useAgentChat(): AgentChat {
  const ctx = useContext(AgentChatContext);
  if (!ctx) throw new Error("useAgentChat requires AgentChatProvider");
  return ctx;
}

/** The agent chat, or null outside AgentChatProvider. */
export function useOptionalAgentChat(): AgentChat | null {
  return useContext(AgentChatContext);
}

export const agentQueryKeys = {
  status: ["agent", "status"] as const,
  chats: ["agent", "chats"] as const,
};
