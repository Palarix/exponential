import type { ChatEvent } from "../../api/chat";
import { onVisibleAgain, watchDocumentVisibility } from "../../hooks/sse-utils";

/** The part of EventSource that followChatStream uses. */
export interface ChatSourceLike {
  readonly readyState: number;
  addEventListener(type: "message" | "error", listener: (msg: { data?: unknown }) => void): void;
  close(): void;
}

const CLOSED = 2;
const RETRY_MS = 5000;

interface FollowOptions {
  /** The seq of the last settled event the thread has. */
  after: number;
  /** Opens the chat's stream with ?after=. */
  create: (after: number) => ChatSourceLike;
  onEvent: (event: ChatEvent) => void;
  schedule?: (fn: () => void, ms: number) => unknown;
  cancel?: (timer: unknown) => void;
  /** Calls fn with the tab's visibility on each change; returns an unsubscribe. */
  watchVisibility?: (fn: (visible: boolean) => void) => () => void;
}

/**
 * Follows a chat's SSE stream from a settled seq. The server ends the stream
 * when the client falls behind, and a hidden tab can be frozen; either way
 * the thread must not wait forever. So it reopens the stream from the last
 * settled seq 5s after the browser gives up, and whenever the tab becomes
 * visible again. Returns a function that stops it.
 */
export function followChatStream({
  after,
  create,
  onEvent,
  schedule = setTimeout,
  cancel = (t) => clearTimeout(t as ReturnType<typeof setTimeout>),
  watchVisibility = watchDocumentVisibility,
}: FollowOptions): () => void {
  let last = after;
  let es: ChatSourceLike | null = null;
  let retry: unknown;

  const open = () => {
    cancel(retry);
    es?.close();
    const source = create(last);
    es = source;
    source.addEventListener("message", (msg) => {
      let event: ChatEvent;
      try {
        event = JSON.parse(String(msg.data)) as ChatEvent;
      } catch {
        return; // Ignore a malformed frame; the next settled event resyncs.
      }
      if (event.seq) last = Math.max(last, event.seq);
      onEvent(event);
    });
    // While CONNECTING, the browser retries with Last-Event-ID by itself.
    source.addEventListener("error", () => {
      if (source.readyState === CLOSED && es === source) retry = schedule(open, RETRY_MS);
    });
  };

  open();
  const unwatch = onVisibleAgain(watchVisibility, open);

  return () => {
    cancel(retry);
    unwatch();
    es?.close();
  };
}
