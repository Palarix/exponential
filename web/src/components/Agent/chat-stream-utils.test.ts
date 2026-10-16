import { describe, expect, it, vi } from "vitest";
import type { ChatEvent } from "../../api/chat";
import { followChatStream, type ChatSourceLike } from "./chat-stream-utils";

class FakeSource implements ChatSourceLike {
  readyState = 1;
  listeners: Record<string, ((msg: { data?: unknown }) => void)[]> = {};
  readonly after: number;
  constructor(after: number) {
    this.after = after;
  }
  addEventListener(type: string, fn: (msg: { data?: unknown }) => void) {
    (this.listeners[type] ??= []).push(fn);
  }
  close() {
    this.readyState = 2;
  }
  emit(type: string, data?: unknown) {
    for (const fn of this.listeners[type] ?? []) fn({ data });
  }
  send(event: Partial<ChatEvent>) {
    this.emit("message", JSON.stringify(event));
  }
}

function setup(after = 4) {
  const sources: FakeSource[] = [];
  const timers: (() => void)[] = [];
  const cancel = vi.fn();
  const onEvent = vi.fn();
  let setVisible: (visible: boolean) => void = () => {};
  const unwatch = vi.fn();
  const stop = followChatStream({
    after,
    create: (from) => {
      const s = new FakeSource(from);
      sources.push(s);
      return s;
    },
    onEvent,
    schedule: (fn) => {
      timers.push(fn);
      return timers.length;
    },
    cancel,
    watchVisibility: (fn) => {
      setVisible = fn;
      return unwatch;
    },
  });
  return { sources, timers, cancel, onEvent, unwatch, stop, setVisible: (v: boolean) => setVisible(v) };
}

describe("followChatStream", () => {
  it("opens from the given seq and passes events on", () => {
    const { sources, onEvent } = setup(4);
    expect(sources[0].after).toBe(4);
    sources[0].send({ type: "text_delta", text: "hi" });
    expect(onEvent).toHaveBeenCalledWith(expect.objectContaining({ type: "text_delta", text: "hi" }));
  });

  it("ignores a malformed frame", () => {
    const { sources, onEvent } = setup();
    sources[0].emit("message", "{nope");
    expect(onEvent).not.toHaveBeenCalled();
  });

  it("lets the browser retry an error while the source is still connecting", () => {
    const { sources, timers } = setup();
    sources[0].readyState = 0;
    sources[0].emit("error");
    expect(timers).toHaveLength(0);
    expect(sources[0].readyState).toBe(0);
  });

  it("reopens from the last settled seq after the browser gives up", () => {
    const { sources, timers } = setup(4);
    sources[0].send({ type: "assistant_text", seq: 5 });
    sources[0].send({ type: "text_delta" });
    sources[0].readyState = 2;
    sources[0].emit("error");
    expect(sources).toHaveLength(1);
    timers[0]();
    expect(sources).toHaveLength(2);
    expect(sources[1].after).toBe(5);
  });

  it("reopens from the last settled seq when the tab is visible again", () => {
    const { sources, setVisible } = setup(4);
    sources[0].send({ type: "turn_done", seq: 7 });
    setVisible(true);
    expect(sources).toHaveLength(1);
    setVisible(false);
    setVisible(true);
    expect(sources[0].readyState).toBe(2);
    expect(sources).toHaveLength(2);
    expect(sources[1].after).toBe(7);
  });

  it("reopening on visible cancels a pending retry", () => {
    const { sources, cancel, setVisible } = setup();
    sources[0].readyState = 2;
    sources[0].emit("error");
    setVisible(false);
    setVisible(true);
    expect(cancel).toHaveBeenCalledWith(1);
    expect(sources).toHaveLength(2);
  });

  it("stops: closes the source, cancels a retry, stops watching visibility", () => {
    const { sources, cancel, unwatch, stop } = setup();
    sources[0].readyState = 2;
    sources[0].emit("error");
    stop();
    expect(cancel).toHaveBeenCalledWith(1);
    expect(unwatch).toHaveBeenCalled();

    const live = setup();
    live.stop();
    expect(live.sources[0].readyState).toBe(2);
  });
});
