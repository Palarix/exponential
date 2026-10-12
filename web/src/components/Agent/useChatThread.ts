import { useEffect, useReducer } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { chatStreamURL, fetchChat, type ChatEvent } from "../../api/chat";
import { agentQueryKeys } from "./agent-context";
import { chatThreadReducer, emptyThread, type ChatThread } from "./chat-thread-utils";

// Events that change a chat's title or state in the chat list.
const META_EVENTS = new Set<ChatEvent["type"]>(["user_message", "turn_done", "turn_stopped", "error"]);

/**
 * Loads a chat's settled events, then follows its SSE stream. A null chatId
 * is a new chat: an empty thread.
 */
export function useChatThread(chatId: string | null): ChatThread {
  const [thread, dispatch] = useReducer(chatThreadReducer, emptyThread);
  const qc = useQueryClient();

  useEffect(() => {
    dispatch({ type: "reset" });
    if (!chatId) return;

    let es: EventSource | null = null;
    let cancelled = false;
    const onEvent = (event: ChatEvent) => {
      dispatch({ type: "event", event });
      if (META_EVENTS.has(event.type)) qc.invalidateQueries({ queryKey: agentQueryKeys.chats });
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
      .catch(() => {
        // The chat is gone or the board is down: show the empty thread.
      });

    return () => {
      cancelled = true;
      es?.close();
    };
  }, [chatId, qc]);

  return thread;
}
