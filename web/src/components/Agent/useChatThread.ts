import { useCallback, useEffect, useReducer, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { fetchChat, type ChatEvent } from "../../api/chat";
import { ApiError } from "../../api/client";
import { agentQueryKeys } from "./agent-context";
import { changesChatMeta, chatThreadReducer, emptyThread, type ChatThread } from "./chat-thread-utils";
import { live } from "../../live";

/**
 * Loads a chat's settled events, then follows its topic on the tab's event
 * stream, from the last settled seq. If the server drops the topic (its log
 * could not be read), it loads the chat again. A null chatId
 * is a new chat: an empty thread. `missing` names a chat ID the server does
 * not know. expireApproval marks a card the server no longer has open.
 */
export function useChatThread(chatId: string | null): { thread: ChatThread; missing: string | null; expireApproval: (id: string) => void } {
  const [thread, dispatch] = useReducer(chatThreadReducer, emptyThread);
  const [missing, setMissing] = useState<string | null>(null);
  const [reloads, setReloads] = useState(0);
  const qc = useQueryClient();

  useEffect(() => {
    dispatch({ type: "reset" });
    if (!chatId) {
      dispatch({ type: "loaded" });
      return;
    }

    let stop: (() => void) | null = null;
    let cancelled = false;
    const onEvent = (event: ChatEvent) => {
      dispatch({ type: "event", event });
      if (changesChatMeta(event.type)) qc.invalidateQueries({ queryKey: agentQueryKeys.chats });
      // A finished turn teaches the board the model's full name and context window.
      if (event.type === "turn_done") qc.invalidateQueries({ queryKey: agentQueryKeys.status });
    };

    fetchChat(chatId)
      .then(({ events, running, background, queue }) => {
        if (cancelled) return;
        events.forEach((event) => dispatch({ type: "event", event }));
        if (!running) dispatch({ type: "interrupted" });
        dispatch({ type: "background", tasks: background ?? [] });
        dispatch({ type: "queue", queue: queue ?? [] });
        dispatch({ type: "loaded" });
        const after = events.length > 0 ? events[events.length - 1].seq ?? 0 : 0;
        stop = live.follow(`chat:${chatId}`, after, (data) => onEvent(data as ChatEvent), () => setReloads((n) => n + 1));
      })
      .catch((err: unknown) => {
        // A 404 means the chat is gone; anything else shows the empty thread.
        if (!cancelled && err instanceof ApiError && err.status === 404) setMissing(chatId);
      });

    return () => {
      cancelled = true;
      stop?.();
    };
  }, [chatId, qc, reloads]);

  const expireApproval = useCallback((id: string) => dispatch({ type: "approval_expired", id }), []);

  return { thread, missing: missing === chatId ? missing : null, expireApproval };
}
