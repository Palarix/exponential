import { useQueryClient } from "@tanstack/react-query";
import { ApiError } from "../../api/client";
import { cancelChat, createChat, sendChatMessage, type ChatContext } from "../../api/chat";
import { useToast } from "../ui";
import { agentQueryKeys, useAgentChat } from "./agent-context";
import { isRunning } from "./chat-thread-utils";
import { useChatThread } from "./useChatThread";

function errorMessage(err: unknown): string {
  if (err instanceof ApiError) {
    try {
      return (JSON.parse(err.body) as { error?: string }).error ?? err.message;
    } catch {
      return err.message;
    }
  }
  return err instanceof Error ? err.message : String(err);
}

/**
 * One chat as the popup and the Agent view use it: its thread, and send/stop.
 * Sending in a new chat (chatId null) creates it first and reports the new ID
 * through onCreated.
 */
export function useChatSession(chatId: string | null, onCreated: (id: string) => void) {
  const { thread, missing } = useChatThread(chatId);
  const { prefs } = useAgentChat();
  const qc = useQueryClient();
  const showToast = useToast();

  const send = async (text: string, context: ChatContext): Promise<boolean> => {
    try {
      let id = chatId;
      if (!id) {
        id = (await createChat()).id;
        onCreated(id);
      }
      await sendChatMessage(id, { text, context, provider: prefs.provider, model: prefs.model, effort: prefs.effort });
      qc.invalidateQueries({ queryKey: agentQueryKeys.chats });
      return true;
    } catch (err) {
      showToast(errorMessage(err), { variant: "error" });
      return false;
    }
  };

  const stop = () => {
    if (chatId) cancelChat(chatId).catch(() => {});
  };

  return { thread, missing, running: isRunning(thread), send, stop };
}
