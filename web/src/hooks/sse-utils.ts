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
  /** An issue event arrived, or the stream (re)opened and may have missed changes. */
  onEvent: () => void;
  /** A chat changed, or the stream (re)opened and may have missed changes. */
  onChatEvent: () => void;
  schedule?: (fn: () => void, ms: number) => unknown;
  cancel?: (timer: unknown) => void;
  watchVisibility?: (fn: (visible: boolean) => void) => () => void;
}

/** Calls fn with the tab's visibility on each change; returns an unsubscribe. */
export function watchDocumentVisibility(fn: (visible: boolean) => void): () => void {
  const onChange = () => fn(document.visibilityState === "visible");
  document.addEventListener("visibilitychange", onChange);
  return () => document.removeEventListener("visibilitychange", onChange);
}

/**
 * Calls onVisible when the tab becomes visible after being hidden. Returns
 * an unsubscribe.
 */
export function onVisibleAgain(watch: (fn: (visible: boolean) => void) => () => void, onVisible: () => void): () => void {
  let hidden = false;
  return watch((visible) => {
    if (!visible) hidden = true;
    else if (hidden) {
      hidden = false;
      onVisible();
    }
  });
}

/**
 * Connects to the server's event stream and reconnects 5s after an error.
 * The server ends the stream of a client that falls behind, and a hidden
 * tab can be frozen, so it also reconnects as soon as the tab is visible
 * again. Each (re)open refetches issues and chats, since events may have
 * been missed. Returns a function that closes it.
 */
export function connectEvents({
  create,
  onEvent,
  onChatEvent,
  schedule = setTimeout,
  cancel = (t) => clearTimeout(t as ReturnType<typeof setTimeout>),
  watchVisibility = watchDocumentVisibility,
}: ConnectOptions): () => void {
  let es: EventSourceLike | null = null;
  let retry: unknown;

  const connect = () => {
    cancel(retry);
    es?.close();
    const source = create();
    es = source;
    for (const type of ISSUE_EVENTS) source.addEventListener(type, onEvent);
    source.addEventListener("chat_changed", onChatEvent);
    source.addEventListener("open", () => {
      onEvent();
      onChatEvent();
    });
    source.addEventListener("error", () => {
      if (es !== source) return;
      source.close();
      es = null;
      retry = schedule(connect, RETRY_MS);
    });
  };

  connect();
  const unwatch = onVisibleAgain(watchVisibility, connect);

  return () => {
    cancel(retry);
    unwatch();
    es?.close();
  };
}
