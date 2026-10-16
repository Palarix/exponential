import { describe, expect, it, vi } from "vitest";
import { connectEvents, ISSUE_EVENTS, type EventSourceLike } from "./sse-utils";

class FakeSource implements EventSourceLike {
  listeners: Record<string, (() => void)[]> = {};
  closed = false;
  addEventListener(type: string, fn: () => void) {
    (this.listeners[type] ??= []).push(fn);
  }
  close() {
    this.closed = true;
  }
  emit(type: string) {
    for (const fn of this.listeners[type] ?? []) fn();
  }
}

function setup() {
  const sources: FakeSource[] = [];
  const timers: (() => void)[] = [];
  const onEvent = vi.fn();
  const onChatEvent = vi.fn();
  const cancel = vi.fn();
  const unwatch = vi.fn();
  let setVisible: (visible: boolean) => void = () => {};
  const stop = connectEvents({
    create: () => {
      const s = new FakeSource();
      sources.push(s);
      return s;
    },
    onEvent,
    onChatEvent,
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
  return { sources, timers, onEvent, onChatEvent, cancel, unwatch, stop, setVisible: (v: boolean) => setVisible(v) };
}

describe("connectEvents", () => {
  it("routes issue events to onEvent only", () => {
    const { sources, onEvent, onChatEvent } = setup();
    for (const type of ISSUE_EVENTS) sources[0].emit(type);
    expect(onEvent).toHaveBeenCalledTimes(ISSUE_EVENTS.length);
    expect(onChatEvent).not.toHaveBeenCalled();
  });

  it("routes chat_changed to onChatEvent only", () => {
    const { sources, onEvent, onChatEvent } = setup();
    sources[0].emit("chat_changed");
    expect(onChatEvent).toHaveBeenCalledTimes(1);
    expect(onEvent).not.toHaveBeenCalled();
  });

  it("calls onEvent and onChatEvent when the stream opens, to catch missed changes", () => {
    const { sources, onEvent, onChatEvent } = setup();
    sources[0].emit("open");
    expect(onEvent).toHaveBeenCalledTimes(1);
    expect(onChatEvent).toHaveBeenCalledTimes(1);
  });

  it("reconnects after an error and calls onChatEvent when it reopens", () => {
    const { sources, timers, onChatEvent } = setup();
    sources[0].emit("open");
    sources[0].emit("error");
    expect(sources[0].closed).toBe(true);
    expect(sources).toHaveLength(1);
    timers[0]();
    expect(sources).toHaveLength(2);
    sources[1].emit("open");
    expect(onChatEvent).toHaveBeenCalledTimes(2);
  });

  it("reconnects at once when the tab is visible again", () => {
    const { sources, setVisible } = setup();
    setVisible(true);
    expect(sources).toHaveLength(1);
    setVisible(false);
    setVisible(true);
    expect(sources[0].closed).toBe(true);
    expect(sources).toHaveLength(2);
  });

  it("reconnecting on visible cancels a pending retry", () => {
    const { sources, cancel, setVisible } = setup();
    sources[0].emit("error");
    setVisible(false);
    setVisible(true);
    expect(cancel).toHaveBeenCalledWith(1);
    expect(sources).toHaveLength(2);
  });

  it("closes the stream, cancels a pending reconnect and stops watching visibility when stopped", () => {
    const { sources, cancel, unwatch, stop } = setup();
    sources[0].emit("error");
    stop();
    expect(cancel).toHaveBeenCalledWith(1);
    expect(unwatch).toHaveBeenCalled();

    const live = setup();
    live.stop();
    expect(live.sources[0].closed).toBe(true);
  });
});
