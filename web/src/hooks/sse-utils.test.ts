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
    cancel: () => {},
  });
  return { sources, timers, onEvent, onChatEvent, stop };
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

  it("calls onChatEvent when the stream opens, to catch missed changes", () => {
    const { sources, onChatEvent } = setup();
    sources[0].emit("open");
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

  it("closes the stream and cancels a pending reconnect when stopped", () => {
    const cancel = vi.fn();
    const sources: FakeSource[] = [];
    const stop = connectEvents({
      create: () => {
        const s = new FakeSource();
        sources.push(s);
        return s;
      },
      onEvent: () => {},
      onChatEvent: () => {},
      schedule: () => 7,
      cancel,
    });
    sources[0].emit("error");
    stop();
    expect(cancel).toHaveBeenCalledWith(7);
    const live = connectEvents({ create: () => { const s = new FakeSource(); sources.push(s); return s; }, onEvent: () => {}, onChatEvent: () => {} });
    live();
    expect(sources[1].closed).toBe(true);
  });
});
