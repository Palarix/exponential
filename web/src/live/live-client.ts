import { onVisibleAgain, watchDocumentVisibility } from "./visibility";

/**
 * The browser side of the board's one event stream (GET /api/events, see
 * internal/live). Topics are either signals ("issues changed, refetch") or
 * logs ("chat:<id>", ordered events with seq ids).
 *
 * After every hello, which is any (re)connect, it calls every signal
 * handler and re-adds every followed log topic from its last seq, so
 * nothing missed while away stays missed.
 */

/** The part of EventSource that the live client uses. */
export interface LiveSourceLike {
  readonly readyState: number;
  addEventListener(type: string, listener: (msg: { data?: unknown }) => void): void;
  close(): void;
}

/** The body of POST /api/events/{stream}/topics. */
export interface TopicChange {
  add?: { topic: string; after: number }[];
  remove?: string[];
}

interface LiveOptions {
  create: () => LiveSourceLike;
  /** Changes the stream's topics. A rejection with status 404 means the stream is gone. */
  post: (stream: string, change: TopicChange) => Promise<void>;
  schedule?: (fn: () => void, ms: number) => unknown;
  cancel?: (timer: unknown) => void;
  /** Calls fn with the tab's visibility on each change; returns an unsubscribe. */
  watchVisibility?: (fn: (visible: boolean) => void) => () => void;
}

export interface LiveClient {
  start(): void;
  stop(): void;
  /** Calls fn on each signal of topic, and after each (re)connect. Returns an unsubscribe. */
  onSignal(topic: string, fn: () => void): () => void;
  /**
   * Follows a log topic from seq after: replayed events, then live ones.
   * onGone runs when the server dropped the topic (it no longer exists, or
   * it could not be read): reload it from scratch. Returns a stop function.
   */
  follow(topic: string, after: number, onEvent: (data: unknown) => void, onGone?: () => void): () => void;
}

interface Follower {
  last: number;
  onEvent: (data: unknown) => void;
  onGone?: () => void;
}

const CLOSED = 2;
const MIN_RETRY_MS = 1000;
const MAX_RETRY_MS = 30000;

export function createLive({
  create,
  post,
  schedule = setTimeout,
  cancel = (t) => clearTimeout(t as ReturnType<typeof setTimeout>),
  watchVisibility = watchDocumentVisibility,
}: LiveOptions): LiveClient {
  const signals = new Map<string, Set<() => void>>();
  const logs = new Map<string, Set<Follower>>();
  let es: LiveSourceLike | null = null;
  let stream: string | null = null;
  let retry: unknown;
  let delay = 0;
  let unwatch: (() => void) | null = null;

  const send = (change: TopicChange) => {
    if (!stream) return; // the next hello sends the full set
    post(stream, change).catch((err: unknown) => {
      if (err && typeof err === "object" && "status" in err && err.status === 404) connect();
    });
  };

  const fromSeq = (followers: Set<Follower>) => Math.min(...[...followers].map((f) => f.last));

  const parse = (msg: { data?: unknown }): { topic?: string; data?: unknown; stream?: string } | null => {
    try {
      return JSON.parse(String(msg.data)) as { topic?: string; data?: unknown; stream?: string };
    } catch {
      return null;
    }
  };

  const connect = () => {
    cancel(retry);
    es?.close();
    stream = null;
    const source = create();
    es = source;

    source.addEventListener("hello", (msg) => {
      const hello = parse(msg);
      if (es !== source || !hello?.stream) return;
      stream = hello.stream;
      delay = 0;
      for (const fns of signals.values()) for (const fn of fns) fn();
      if (logs.size > 0) send({ add: [...logs].map(([topic, followers]) => ({ topic, after: fromSeq(followers) })) });
    });
    source.addEventListener("signal", (msg) => {
      const topic = parse(msg)?.topic;
      if (topic) for (const fn of signals.get(topic) ?? []) fn();
    });
    source.addEventListener("log", (msg) => {
      const frame = parse(msg);
      if (!frame?.topic) return;
      const seq = (frame.data as { seq?: unknown } | null)?.seq;
      for (const f of logs.get(frame.topic) ?? []) {
        if (typeof seq === "number" && seq > 0) {
          if (seq <= f.last) continue;
          f.last = seq;
        }
        f.onEvent(frame.data);
      }
    });
    source.addEventListener("gone", (msg) => {
      const topic = parse(msg)?.topic;
      if (!topic) return;
      const followers = logs.get(topic);
      logs.delete(topic);
      for (const f of followers ?? []) f.onGone?.();
    });
    // While CONNECTING, the browser retries by itself, and its next hello
    // resyncs. Once CLOSED, it has given up.
    source.addEventListener("error", () => {
      if (es !== source) return;
      stream = null;
      if (source.readyState !== CLOSED) return;
      source.close();
      es = null;
      delay = Math.min(delay === 0 ? MIN_RETRY_MS : delay * 2, MAX_RETRY_MS);
      retry = schedule(connect, delay);
    });
  };

  return {
    start() {
      if (es) return;
      connect();
      unwatch = onVisibleAgain(watchVisibility, connect);
    },
    stop() {
      cancel(retry);
      unwatch?.();
      unwatch = null;
      es?.close();
      es = null;
      stream = null;
    },
    onSignal(topic, fn) {
      let fns = signals.get(topic);
      if (!fns) signals.set(topic, (fns = new Set()));
      fns.add(fn);
      return () => {
        fns.delete(fn);
      };
    },
    follow(topic, after, onEvent, onGone) {
      const follower: Follower = { last: after, onEvent, onGone };
      let followers = logs.get(topic);
      if (!followers) logs.set(topic, (followers = new Set()));
      const from = followers.size > 0 ? fromSeq(followers) : Infinity;
      followers.add(follower);
      // Re-add only when this follower needs events the topic doesn't
      // replay yet.
      if (after < from) send({ add: [{ topic, after }] });
      return () => {
        const current = logs.get(topic);
        if (!current?.delete(follower) || current.size > 0) return;
        logs.delete(topic);
        send({ remove: [topic] });
      };
    },
  };
}
