// @vitest-environment jsdom
import { afterEach, beforeAll, describe, expect, it } from "vitest";
import { act, useLayoutEffect } from "react";
import { mountDom, setupDom } from "../test-dom";
import { useStickToBottom } from "./useStickToBottom";

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

/** jsdom has no layout: give the scroller a 500px viewport over 1000px of content. */
function fakeLayout(el: HTMLElement) {
  let top = 0;
  Object.defineProperty(el, "clientHeight", { configurable: true, get: () => 500 });
  Object.defineProperty(el, "scrollHeight", { configurable: true, get: () => 1000 });
  Object.defineProperty(el, "scrollTop", {
    configurable: true,
    get: () => top,
    set: (v: number) => {
      top = Math.max(0, Math.min(500, v));
    },
  });
}

function mount() {
  type Stick = { pin: () => void; follow: () => void };
  let api: Stick | undefined;
  const glide = { current: false };
  function Scroller({ onStick }: { onStick: (s: Stick) => void }) {
    const { ref, pin, follow } = useStickToBottom<HTMLDivElement>(glide);
    useLayoutEffect(() => {
      onStick({ pin, follow });
    });
    return <div ref={ref} data-testid="scroller" />;
  }
  const m = mountDom(<Scroller onStick={(s) => {
    api = s;
  }} />);
  cleanup = m.cleanup;
  const el = m.host.querySelector("[data-testid=scroller]") as HTMLDivElement;
  fakeLayout(el);
  return { el, stick: () => api! };
}

function scrollTo(el: HTMLElement, top: number) {
  act(() => {
    el.scrollTop = top;
    el.dispatchEvent(new Event("scroll"));
  });
}

describe("useStickToBottom", () => {
  it("does not pin after the user scrolled up", () => {
    const { el, stick } = mount();
    act(() => stick().pin());
    expect(el.scrollTop).toBe(500);
    scrollTo(el, 100);
    act(() => stick().pin());
    expect(el.scrollTop).toBe(100);
  });

  it("follow() resumes following and goes to the end, even after the user scrolled up", () => {
    const { el, stick } = mount();
    act(() => stick().pin());
    scrollTo(el, 100);
    act(() => stick().follow());
    expect(el.scrollTop).toBe(500);
    // Following again: the next pin keeps it at the end.
    act(() => stick().pin());
    expect(el.scrollTop).toBe(500);
  });
});
