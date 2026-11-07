// @vitest-environment jsdom
import { afterEach, beforeAll, describe, expect, it, vi } from "vitest";
import { act, type ReactNode } from "react";
import * as TooltipPrimitive from "@radix-ui/react-tooltip";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ChatEvent } from "../../api/chat";
import { mountDom, setupDom } from "../../test-dom";
import ChatThread from "./ChatThread";
import { chatThreadReducer, emptyThread } from "./chat-thread-utils";

beforeAll(() => {
  setupDom();
  globalThis.ResizeObserver ??= class {
    observe() {}
    unobserve() {}
    disconnect() {}
  } as unknown as typeof ResizeObserver;
});

let cleanup: (() => void) | undefined;
afterEach(() => {
  cleanup?.();
  cleanup = undefined;
});

const T0 = "2026-10-13T10:00:00Z";
function thread(events: Partial<ChatEvent>[]) {
  return events.reduce((s, e) => chatThreadReducer(s, { type: "event", event: { time: T0, ...e } as ChatEvent }), emptyThread);
}

const client = new QueryClient();
function wrap(node: ReactNode) {
  return (
    <QueryClientProvider client={client}>
      <TooltipPrimitive.Provider>{node}</TooltipPrimitive.Provider>
    </QueryClientProvider>
  );
}

describe("ChatThread turn rail", () => {
  const events: Partial<ChatEvent>[] = [
    { type: "user_message", seq: 1, text: "first ask" },
    { type: "turn_done", seq: 2, duration_ms: 1_000 },
    { type: "user_message", seq: 3, text: "second ask" },
    { type: "assistant_text", seq: 4, text: "Working." },
  ];

  it("a bar click stops following, so new output leaves the view where it went", () => {
    const m = mountDom(wrap(<ChatThread thread={thread(events)} wide />));
    cleanup = m.cleanup;
    const scroller = m.host.querySelector("[data-thread-scroller]") as HTMLElement;
    const layout = { scrollHeight: 1000 };
    let top = 0;
    Object.defineProperty(scroller, "clientHeight", { configurable: true, get: () => 500 });
    Object.defineProperty(scroller, "scrollHeight", { configurable: true, get: () => layout.scrollHeight });
    Object.defineProperty(scroller, "scrollTop", {
      configurable: true,
      get: () => top,
      set: (v: number) => {
        top = Math.max(0, Math.min(layout.scrollHeight - 500, v));
      },
    });
    const scrollTo = vi.fn();
    scroller.scrollTo = scrollTo;

    // New output follows to the end.
    m.rerender(wrap(<ChatThread thread={thread([...events, { type: "assistant_text", seq: 5, text: "More." }])} wide />));
    expect(top).toBe(500);

    act(() => {
      (m.host.querySelector('[data-turn-bar="0"]') as HTMLElement).click();
    });
    expect(scrollTo).toHaveBeenCalledOnce();

    layout.scrollHeight = 1300;
    m.rerender(wrap(<ChatThread thread={thread([...events, { type: "assistant_text", seq: 5, text: "More." }, { type: "assistant_text", seq: 6, text: "Again." }])} wide />));
    expect(top).toBe(500);
  });
});
