import { describe, expect, it, vi } from "vitest";
import { createLive, type LiveSourceLike, type TopicChange } from "./live-client";

const CONNECTING = 0;
const OPEN = 1;
const CLOSED = 2;

class FakeSource implements LiveSourceLike {
  readyState = CONNECTING;
  listeners: Record<string, ((msg: { data?: unknown }) => void)[]> = {};
  closed = false;
  addEventListener(type: string, fn: (msg: { data?: unknown }) => void) {
    (this.listeners[type] ??= []).push(fn);
  }
  close() {
    this.closed = true;
    this.readyState = CLOSED;
  }
  emit(type: string, data?: unknown) {
    for (const fn of this.listeners[type] ?? []) fn({ data: data === undefined ? undefined : JSON.stringify(data) });
  }
  hello(stream: string) {
    this.readyState = OPEN;
    this.emit("hello", { stream });
  }
  fail(state: number) {
    this.readyState = state;
    this.emit("error");
  }
}

class PostError extends Error {
  readonly status: number;
  constructor(status: number) {
    super(`status ${status}`);
    this.status = status;
  }
}

function setup(opts: { post?: (stream: string, change: TopicChange) => Promise<void> } = {}) {
  const sources: FakeSource[] = [];
  const timers: { fn: () => void; ms: number }[] = [];
  const posts: { stream: string; change: TopicChange }[] = [];
  const cancel = vi.fn();
  const unwatch = vi.fn();
  let setVisible: (visible: boolean) => void = () => {};
  const live = createLive({
    create: () => {
      const s = new FakeSource();
      sources.push(s);
      return s;
    },
    post: (stream, change) => {
      posts.push({ stream, change });
      return opts.post ? opts.post(stream, change) : Promise.resolve();
    },
    schedule: (fn, ms) => {
      timers.push({ fn, ms });
      return timers.length;
    },
    cancel,
    watchVisibility: (fn) => {
      setVisible = fn;
      return unwatch;
    },
  });
  const last = () => sources[sources.length - 1];
  return { live, sources, last, timers, posts, cancel, unwatch, setVisible: (v: boolean) => setVisible(v) };
}

const flush = () => new Promise((r) => setTimeout(r, 0));

describe("createLive", () => {
  it("opens one source on start, whatever the number of followed topics", () => {
    const { live, sources } = setup();
    live.start();
    live.follow("chat:a", 0, () => {});
    live.follow("chat:b", 0, () => {});
    expect(sources).toHaveLength(1);
  });

  it("routes signals to their topic's handlers", () => {
    const { live, last } = setup();
    const issues = vi.fn();
    const chats = vi.fn();
    live.onSignal("issues", issues);
    live.onSignal("chats", chats);
    live.start();
    last().hello("s1");
    issues.mockClear();
    chats.mockClear();

    last().emit("signal", { topic: "issues" });
    expect(issues).toHaveBeenCalledTimes(1);
    expect(chats).not.toHaveBeenCalled();
  });

  it("resyncs every signal topic on hello", () => {
    const { live, last } = setup();
    const issues = vi.fn();
    const drafts = vi.fn();
    live.onSignal("issues", issues);
    live.onSignal("drafts", drafts);
    live.start();
    last().hello("s1");
    expect(issues).toHaveBeenCalledTimes(1);
    expect(drafts).toHaveBeenCalledTimes(1);
  });

  it("removing a signal handler stops it", () => {
    const { live, last } = setup();
    const fn = vi.fn();
    const off = live.onSignal("issues", fn);
    live.start();
    off();
    last().hello("s1");
    last().emit("signal", { topic: "issues" });
    expect(fn).not.toHaveBeenCalled();
  });

  it("follow posts add once the stream is open, and hello re-adds from the last seq", () => {
    const { live, last, posts } = setup();
    live.start();
    const events: unknown[] = [];
    live.follow("chat:a", 3, (e) => events.push(e));
    expect(posts).toHaveLength(0); // no stream yet: hello adds it

    last().hello("s1");
    expect(posts).toEqual([{ stream: "s1", change: { add: [{ topic: "chat:a", after: 3 }] } }]);

    last().emit("log", { topic: "chat:a", data: { seq: 4, type: "x" } });
    last().emit("log", { topic: "chat:a", data: { type: "text_delta" } });
    expect(events).toEqual([{ seq: 4, type: "x" }, { type: "text_delta" }]);

    // The browser reconnects by itself; the new stream replays from seq 4.
    last().fail(CONNECTING);
    last().hello("s2");
    expect(posts[1]).toEqual({ stream: "s2", change: { add: [{ topic: "chat:a", after: 4 }] } });
  });

  it("follow on an open stream posts add at once, and stop posts remove", () => {
    const { live, last, posts } = setup();
    live.start();
    last().hello("s1");
    const stop = live.follow("chat:a", 7, () => {});
    expect(posts).toEqual([{ stream: "s1", change: { add: [{ topic: "chat:a", after: 7 }] } }]);
    stop();
    expect(posts[1]).toEqual({ stream: "s1", change: { remove: ["chat:a"] } });

    last().emit("log", { topic: "chat:a", data: { seq: 8 } });
    last().fail(CONNECTING);
    last().hello("s2");
    expect(posts).toHaveLength(2); // a stopped topic is not re-added
  });

  it("drops settled events a follower already has", () => {
    const { live, last } = setup();
    live.start();
    last().hello("s1");
    const events: unknown[] = [];
    live.follow("chat:a", 5, (e) => events.push(e));
    last().emit("log", { topic: "chat:a", data: { seq: 5 } });
    last().emit("log", { topic: "chat:a", data: { seq: 6 } });
    last().emit("log", { topic: "chat:a", data: { seq: 6 } });
    expect(events).toEqual([{ seq: 6 }]);
  });

  it("two followers of one topic share it: add from the lower seq, remove after the last stops", () => {
    const { live, last, posts } = setup();
    live.start();
    last().hello("s1");
    const a: unknown[] = [];
    const b: unknown[] = [];
    const stopA = live.follow("chat:x", 5, (e) => a.push(e));
    const stopB = live.follow("chat:x", 3, (e) => b.push(e));
    expect(posts[1]).toEqual({ stream: "s1", change: { add: [{ topic: "chat:x", after: 3 }] } });

    last().emit("log", { topic: "chat:x", data: { seq: 4 } });
    last().emit("log", { topic: "chat:x", data: { seq: 6 } });
    expect(a).toEqual([{ seq: 6 }]);
    expect(b).toEqual([{ seq: 4 }, { seq: 6 }]);

    stopA();
    expect(posts).toHaveLength(2);
    stopB();
    expect(posts[2]).toEqual({ stream: "s1", change: { remove: ["chat:x"] } });
  });

  it("gone tells the followers and forgets the topic", () => {
    const { live, last, posts } = setup();
    live.start();
    last().hello("s1");
    const onGone = vi.fn();
    live.follow("chat:a", 0, () => {}, onGone);
    last().emit("gone", { topic: "chat:a" });
    expect(onGone).toHaveBeenCalledTimes(1);
    last().fail(CONNECTING);
    last().hello("s2");
    expect(posts).toHaveLength(1); // not re-added
  });

  it("ignores malformed frames", () => {
    const { live, last } = setup();
    const fn = vi.fn();
    live.onSignal("issues", fn);
    live.start();
    last().hello("s1");
    fn.mockClear();
    for (const fn of last().listeners.signal) fn({ data: "{not json" });
    expect(fn).not.toHaveBeenCalled();
  });

  it("retries with backoff after the browser gives up, and resets after hello", () => {
    const { live, sources, last, timers } = setup();
    live.start();
    last().fail(CLOSED);
    last().fail(CLOSED); // a stale source's error is ignored
    expect(timers.map((t) => t.ms)).toEqual([1000]);
    timers[0].fn();
    expect(sources).toHaveLength(2);
    last().fail(CLOSED);
    timers[1].fn();
    last().fail(CLOSED);
    expect(timers.map((t) => t.ms)).toEqual([1000, 2000, 4000]);
    timers[2].fn();
    last().hello("s1");
    last().fail(CLOSED);
    expect(timers[3].ms).toBe(1000);
  });

  it("caps the backoff at 30s", () => {
    const { live, last, timers } = setup();
    live.start();
    for (let i = 0; i < 8; i++) {
      last().fail(CLOSED);
      timers[timers.length - 1].fn();
    }
    expect(timers.map((t) => t.ms).slice(-3)).toEqual([30000, 30000, 30000]);
  });

  it("does not schedule a retry while the browser reconnects by itself", () => {
    const { live, last, timers } = setup();
    live.start();
    last().hello("s1");
    last().fail(CONNECTING);
    expect(timers).toHaveLength(0);
  });

  it("reconnects at once when the tab is visible again, cancelling a pending retry", () => {
    const { live, sources, last, cancel, setVisible } = setup();
    live.start();
    last().fail(CLOSED);
    cancel.mockClear();
    setVisible(true); // visible without having been hidden: nothing to do
    expect(sources).toHaveLength(1);

    setVisible(false);
    setVisible(true);
    expect(cancel).toHaveBeenCalledWith(1);
    expect(sources).toHaveLength(2);
    expect(sources[0].closed).toBe(true);
  });

  it("a 404 on a post reconnects: the server no longer knows the stream", async () => {
    const { live, sources, last } = setup({ post: () => Promise.reject(new PostError(404)) });
    live.start();
    last().hello("s1");
    live.follow("chat:a", 0, () => {});
    await flush();
    expect(sources).toHaveLength(2);
    expect(sources[0].closed).toBe(true);
  });

  it("other post errors don't reconnect", async () => {
    const { live, sources, last } = setup({ post: () => Promise.reject(new PostError(500)) });
    live.start();
    last().hello("s1");
    live.follow("chat:a", 0, () => {});
    await flush();
    expect(sources).toHaveLength(1);
  });

  it("stop closes the source, cancels a retry, and removes the visibility listener", () => {
    const { live, last, cancel, unwatch } = setup();
    live.start();
    last().hello("s1");
    live.stop();
    expect(last().closed).toBe(true);
    expect(unwatch).toHaveBeenCalled();

    live.start();
    last().fail(CLOSED);
    cancel.mockClear();
    live.stop();
    expect(cancel).toHaveBeenCalledWith(1);
  });
});
