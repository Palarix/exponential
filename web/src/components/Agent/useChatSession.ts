import { useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { ApiError } from "../../api/client";
import { cancelChat, createChat, sendChatMessage, stopBackground, type ChatContext } from "../../api/chat";
import { useToast } from "../ui";
import { agentQueryKeys, useAgentChat } from "./agent-context";
import { backgroundBanner, isRunning } from "./chat-thread-utils";
import type { BackgroundBannerInfo } from "./ChatComposer";
import { useChatThread } from "./useChatThread";

/** The error text of a failed API call: its JSON `error`, or its message. */
export function errorMessage(err: unknown): string {
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
 * One chat as the popup and the Agent view use it: its thread, send/stop,
 * and the banner for the agent's background tasks. Sending in a new chat
 * (chatId null) creates it first and reports the new ID through onCreated.
 */
export function useChatSession(chatId: string | null, onCreated: (id: string) => void) {
  const { thread, missing, expireApproval } = useChatThread(chatId);
  const { prefsFor } = useAgentChat();
  const qc = useQueryClient();
  const showToast = useToast();

  const send = async (text: string, context: ChatContext): Promise<boolean> => {
    try {
      const prefs = prefsFor(chatId);
      let id = chatId;
      if (!id) {
        id = (await createChat()).id;
        onCreated(id);
      }
      await sendChatMessage(id, { text, context, provider: prefs.provider, model: prefs.model, effort: prefs.effort });
      qc.invalidateQueries({ queryKey: agentQueryKeys.chats });
      return true;
    } catch (err) {
      // The chat may have changed elsewhere, e.g. archived in another tab:
      // refresh it, so the view shows why.
      qc.invalidateQueries({ queryKey: agentQueryKeys.chats });
      showToast(errorMessage(err), { variant: "error" });
      return false;
    }
  };

  const stop = () => {
    if (chatId) cancelChat(chatId).catch(() => {});
  };

  // "Stopping…" holds until the server reports no background task.
  const [stoppingChat, setStoppingChat] = useState<string | null>(null);
  const title = backgroundBanner(thread.background);
  if (stoppingChat && !title) setStoppingChat(null);
  const background: BackgroundBannerInfo | undefined = chatId && title
    ? {
        title,
        working: thread.background.some((t) => t.kind === "agent"),
        tasks: thread.background,
        stopping: stoppingChat === chatId,
        onStop: () => {
          setStoppingChat(chatId);
          stopBackground(chatId).catch((err: unknown) => {
            setStoppingChat(null);
            showToast(errorMessage(err), { variant: "error" });
          });
        },
      }
    : undefined;

  return { thread, missing, running: isRunning(thread), send, stop, expireApproval, background };
}
