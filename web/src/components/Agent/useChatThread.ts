import { useEffect, useReducer, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { chatStreamURL, fetchChat, type ChatEvent } from "../../api/chat";
import { ApiError } from "../../api/client";
import { agentQueryKeys } from "./agent-context";
import { chatThreadReducer, emptyThread, type ChatThread } from "./chat-thread-utils";

// Events that change a chat's title or state in the chat list.
const META_EVENTS = new Set<ChatEvent["type"]>(["user_message", "turn_done", "turn_stopped", "error"]);

/**
 * Loads a chat's settled events, then follows its SSE stream. A null chatId
 * is a new chat: an empty thread. `missing` names a chat ID the server does
 * not know.
 */
export function useChatThread(chatId: string | null): { thread: ChatThread; missing: string | null } {
  const [thread, dispatch] = useReducer(chatThreadReducer, emptyThread);
  const [missing, setMissing] = useState<string | null>(null);
  const qc = useQueryClient();

  useEffect(() => {
    dispatch({ type: "reset" });
    if (!chatId) return;

    let es: EventSource | null = null;
    let cancelled = false;
    const onEvent = (event: ChatEvent) => {
      dispatch({ type: "event", event });
      if (META_EVENTS.has(event.type)) qc.invalidateQueries({ queryKey: agentQueryKeys.chats });
      // A finished turn teaches the board the model's full name and context window.
      if (event.type === "turn_done") qc.invalidateQueries({ queryKey: agentQueryKeys.status });
    };

    fetchChat(chatId)
      .then(({ events, running }) => {
        if (cancelled) return;
        events.forEach((event) => dispatch({ type: "event", event }));
        if (!running) dispatch({ type: "interrupted" });
        const after = events.length > 0 ? events[events.length - 1].seq ?? 0 : 0;
        // A reconnecting EventSource sends Last-Event-ID, which the server
        // prefers over ?after=, so replays never duplicate.
        es = new EventSource(`${chatStreamURL(chatId)}?after=${after}`);
        es.onmessage = (msg) => {
          try {
            onEvent(JSON.parse(msg.data) as ChatEvent);
          } catch {
            // Ignore a malformed frame; the next settled event resyncs.
          }
        };
      })
      .catch((err: unknown) => {
        // A 404 means the chat is gone; anything else shows the empty thread.
        if (!cancelled && err instanceof ApiError && err.status === 404) setMissing(chatId);
      });

    return () => {
      cancelled = true;
      es?.close();
    };
  }, [chatId, qc]);

  return { thread, missing: missing === chatId ? missing : null };
}
