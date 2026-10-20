import { useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { ApiError } from "../../api/client";
import { cancelChat, createChat, removeQueued, sendChatMessage, steerQueued, stopBackground, type ChatAttachment, type ChatContext, type QueuedMessage } from "../../api/chat";
import { useToast } from "../ui";
import { agentQueryKeys, useAgentChat } from "./agent-context";
import { attachmentUrl, toSendAttachment } from "./attachment-utils";
import { backgroundBanner, editTarget, isRunning, pendingApprovals } from "./chat-thread-utils";
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

/** The queue as the composer shows it: chips with Edit, Steer and remove. */
export interface ComposerQueue {
  items: QueuedMessage[];
  /** Steer is off while a card waits for the user, and while a steer is on its way. */
  canSteer: boolean;
  steer(id: string): void;
  remove(id: string): void;
  /** Takes the message off the board's queue and puts it in this composer. */
  edit(id: string): void;
}

/** A sent message's attachments as files again, for the composer. */
async function attachmentFiles(chatId: string, list: ChatAttachment[] = []): Promise<File[]> {
  return Promise.all(
    list.map(async (a) => {
      const res = await fetch(attachmentUrl(chatId, a.id));
      if (!res.ok) throw new Error(`Could not load ${a.name ?? "an attachment"}`);
      return new File([await res.blob()], a.name ?? "attachment", { type: a.mime });
    }),
  );
}

/**
 * One chat as the popup and the Agent view use it: its thread, send/stop,
 * the queue of messages sent during a turn, editing the last message, and
 * the banner for the agent's background tasks. Sending in a new chat
 * (chatId null) creates it first and reports the new ID through onCreated.
 */
export function useChatSession(chatId: string | null, onCreated: (id: string) => void) {
  const { thread, missing, expireApproval } = useChatThread(chatId);
  const { prefsFor, adoptNewChatPrefs, drafts, allChats } = useAgentChat();
  const qc = useQueryClient();
  const showToast = useToast();
  const running = isRunning(thread);
  const meta = allChats.find((c) => c.id === chatId);
  const toastError = (err: unknown) => showToast(errorMessage(err), { variant: "error" });

  /** Sends a message. While a turn runs, the board queues it instead. */
  const send = async (text: string, context: ChatContext, files: File[] = [], opts: { edit?: boolean } = {}): Promise<boolean> => {
    try {
      const prefs = prefsFor(chatId);
      let id = chatId;
      if (!id) {
        id = (await createChat()).id;
        adoptNewChatPrefs(id);
        onCreated(id);
      }
      const attachments = await Promise.all(files.map(toSendAttachment));
      await sendChatMessage(id, { text, context, provider: prefs.provider, model: prefs.model, effort: prefs.effort, ...(attachments.length ? { attachments } : {}), ...(opts.edit && { edit: true }) });
      qc.invalidateQueries({ queryKey: agentQueryKeys.chats });
      return true;
    } catch (err) {
      // The chat may have changed elsewhere, e.g. archived in another tab:
      // refresh it, so the view shows why.
      qc.invalidateQueries({ queryKey: agentQueryKeys.chats });
      toastError(err);
      return false;
    }
  };

  const [steering, setSteering] = useState(false);
  const queue: ComposerQueue | undefined = chatId
    ? {
        items: thread.queue,
        canSteer: running && !steering && !thread.turns.some((t) => pendingApprovals(t).length > 0),
        steer: (id) => {
          setSteering(true);
          steerQueued(chatId, id)
            .catch(toastError)
            .finally(() => setSteering(false));
        },
        remove: (id) => {
          removeQueued(chatId, id).catch(toastError);
        },
        edit: (id) => {
          removeQueued(chatId, id)
            .then(async (m: QueuedMessage) => {
              const files = await attachmentFiles(chatId, m.attachments);
              drafts.setText(chatId, m.text ?? "");
              drafts.addFiles(chatId, files);
            })
            .catch(toastError);
        },
      }
    : undefined;

  // The user message the Edit action and ↑ put back in the composer.
  const target = chatId && !meta?.archived_at ? editTarget(thread, meta?.agent ?? "") : null;
  const editLast =
    target === null || !chatId
      ? undefined
      : () => {
          const user = thread.turns[target].user;
          attachmentFiles(chatId, user.attachments)
            .then((files) => drafts.startEdit(chatId, user.text, files))
            .catch(toastError);
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

  return { thread, missing, running, send, stop, expireApproval, background, queue, editTarget: target, editLast };
}
