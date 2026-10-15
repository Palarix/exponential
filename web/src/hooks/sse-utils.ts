/** The part of EventSource that connectEvents uses. */
export interface EventSourceLike {
  addEventListener(type: string, listener: () => void): void;
  close(): void;
}

/** Events that change issue data. */
export const ISSUE_EVENTS = [
  "issue_created",
  "issue_updated",
  "issue_deleted",
  "issue_merged",
  "issue_commented",
  "refresh",
  "draft_changed",
] as const;

const RETRY_MS = 5000;

interface ConnectOptions {
  create: () => EventSourceLike;
  /** An issue event arrived. */
  onEvent: () => void;
  /** A chat changed, or the stream (re)opened and may have missed changes. */
  onChatEvent: () => void;
  schedule?: (fn: () => void, ms: number) => unknown;
  cancel?: (timer: unknown) => void;
}

/**
 * Connects to the server's event stream and reconnects 5s after an error.
 * Returns a function that closes it.
 */
export function connectEvents({
  create,
  onEvent,
  onChatEvent,
  schedule = setTimeout,
  cancel = (t) => clearTimeout(t as ReturnType<typeof setTimeout>),
}: ConnectOptions): () => void {
  let es: EventSourceLike | null = null;
  let retry: unknown;

  const connect = () => {
    es = create();
    for (const type of ISSUE_EVENTS) es.addEventListener(type, onEvent);
    es.addEventListener("chat_changed", onChatEvent);
    es.addEventListener("open", onChatEvent);
    es.addEventListener("error", () => {
      es?.close();
      es = null;
      retry = schedule(connect, RETRY_MS);
    });
  };

  connect();

  return () => {
    cancel(retry);
    es?.close();
  };
}
