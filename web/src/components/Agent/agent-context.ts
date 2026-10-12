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
  prefs: ChatPrefs;
  setPrefs(prefs: ChatPrefs): void;
  openNew(): void;
  openChat(id: string): void;
  toggle(): void;
  minimize(): void;
  close(): void;
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

export const agentQueryKeys = {
  status: ["agent", "status"] as const,
  chats: ["agent", "chats"] as const,
};
