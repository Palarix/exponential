// @vitest-environment jsdom
import { afterEach, beforeAll, describe, expect, it, vi } from "vitest";
import { act, useLayoutEffect } from "react";
import { mountDom, setupDom } from "../test-dom";
import { useStickToBottom } from "./useStickToBottom";

// Motion on. The tests choose gliding or jumping with the `glide` ref.
vi.mock("../preferences", () => ({ motionAllowed: () => true }));

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
  vi.unstubAllGlobals();
});

/** Animation frames that run only when the test says so. */
function stubFrames() {
  const frames = new Map<number, FrameRequestCallback>();
  let next = 1;
  vi.stubGlobal("requestAnimationFrame", (cb: FrameRequestCallback) => {
    frames.set(next, cb);
    return next++;
  });
  vi.stubGlobal("cancelAnimationFrame", (id: number) => {
    frames.delete(id);
  });
  return {
    pending: () => frames.size,
    runOne: (now: number) => {
      const due = [...frames.values()];
      frames.clear();
      for (const cb of due) cb(now);
    },
  };
}

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
  return { el, glide, stick: () => api! };
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

  describe("with glide on and motion allowed", () => {
    it("pin() glides: it moves on later frames, not in the call", () => {
      const frames = stubFrames();
      const { el, glide, stick } = mount();
      glide.current = true;
      act(() => stick().pin());
      expect(el.scrollTop).toBe(0);
      expect(frames.pending()).toBe(1);
      frames.runOne(performance.now() + 16);
      expect(el.scrollTop).toBeGreaterThan(0);
      expect(el.scrollTop).toBeLessThan(500);
    });

    it("follow() goes to the end in the same call", () => {
      const frames = stubFrames();
      const { el, glide, stick } = mount();
      scrollTo(el, 100);
      glide.current = true;
      act(() => stick().follow());
      expect(el.scrollTop).toBe(500);
      expect(frames.pending()).toBe(0);
    });

    it("follow() cancels a running glide", () => {
      const frames = stubFrames();
      const { el, glide, stick } = mount();
      glide.current = true;
      act(() => stick().pin());
      expect(frames.pending()).toBe(1);
      act(() => stick().follow());
      expect(frames.pending()).toBe(0);
      expect(el.scrollTop).toBe(500);
    });

    it("keeps following when a scroll event from an earlier upward move fires after follow()", () => {
      const frames = stubFrames();
      const { el, glide, stick } = mount();
      scrollTo(el, 100);
      glide.current = true;
      // The browser moved the position up, and its scroll event is still pending.
      el.scrollTop = 90;
      act(() => stick().follow());
      act(() => {
        el.dispatchEvent(new Event("scroll"));
      });
      expect(el.scrollTop).toBe(500);
      act(() => stick().pin());
      expect(frames.pending()).toBe(1);
    });
  });
});
