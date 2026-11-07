// @vitest-environment jsdom
import { afterEach, beforeAll, describe, expect, it, vi } from "vitest";
import { act, useLayoutEffect } from "react";
import { mountDom, pressKey, setupDom } from "../test-dom";
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

/** jsdom has no layout: by default a 500px viewport over 1000px of content, 800px wide. */
function fakeLayout(el: HTMLElement) {
  const layout = { clientHeight: 500, scrollHeight: 1000 };
  let top = 0;
  Object.defineProperty(el, "clientHeight", { configurable: true, get: () => layout.clientHeight });
  Object.defineProperty(el, "scrollHeight", { configurable: true, get: () => layout.scrollHeight });
  Object.defineProperty(el, "scrollTop", {
    configurable: true,
    get: () => top,
    set: (v: number) => {
      top = Math.max(0, Math.min(layout.scrollHeight - layout.clientHeight, v));
    },
  });
  el.getBoundingClientRect = () => ({ left: 0, right: 800, top: 0, bottom: 500, width: 800, height: 500, x: 0, y: 0, toJSON: () => ({}) });
  return layout;
}

function mount() {
  type Stick = { pin: () => void; follow: () => void; release: () => void };
  let api: Stick | undefined;
  const glide = { current: false };
  function Scroller({ onStick }: { onStick: (s: Stick) => void }) {
    const { ref, pin, follow, release } = useStickToBottom<HTMLDivElement>(glide);
    useLayoutEffect(() => {
      onStick({ pin, follow, release });
    });
    return (
      <div ref={ref} data-testid="scroller">
        <p data-testid="row">text</p>
        <div data-testid="inner" style={{ overflowY: "auto" }}>
          <p data-testid="inner-row">text</p>
        </div>
        <input data-testid="field" />
        <div data-testid="contenteditable" contentEditable suppressContentEditableWarning />
      </div>
    );
  }
  const m = mountDom(<Scroller onStick={(s) => {
    api = s;
  }} />);
  cleanup = m.cleanup;
  const el = m.host.querySelector("[data-testid=scroller]") as HTMLDivElement;
  const layout = fakeLayout(el);
  const part = (id: string) => m.host.querySelector(`[data-testid=${id}]`) as HTMLElement;
  return { el, glide, layout, part, stick: () => api! };
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

  describe("user input stops following", () => {
    /** At the end, following. */
    function atEnd() {
      const m = mount();
      act(() => m.stick().pin());
      expect(m.el.scrollTop).toBe(500);
      return m;
    }
    type Mounted = ReturnType<typeof atEnd>;
    /** Output grows the content: a thread that follows goes to the new end. */
    function following({ el, layout, stick }: Mounted): boolean {
      const top = el.scrollTop;
      layout.scrollHeight += 300;
      act(() => stick().pin());
      return el.scrollTop > top;
    }
    function wheel(target: HTMLElement, deltaY: number, init: WheelEventInit = {}) {
      act(() => {
        target.dispatchEvent(new WheelEvent("wheel", { deltaY, bubbles: true, ...init }));
      });
    }
    function touchmove(target: HTMLElement) {
      act(() => {
        target.dispatchEvent(new Event("touchmove", { bubbles: true }));
      });
    }
    function press(target: HTMLElement, clientX: number) {
      act(() => {
        target.dispatchEvent(new MouseEvent("pointerdown", { clientX, bubbles: true }));
      });
    }

    it("wheel up stops following before the position moves", () => {
      const m = atEnd();
      wheel(m.part("row"), -10);
      expect(m.el.scrollTop).toBe(500);
      expect(following(m)).toBe(false);
    });

    it("wheel up cancels a running glide", () => {
      const frames = stubFrames();
      const m = atEnd();
      m.glide.current = true;
      m.layout.scrollHeight += 300;
      act(() => m.stick().pin());
      expect(frames.pending()).toBe(1);
      wheel(m.part("row"), -10);
      expect(frames.pending()).toBe(0);
    });

    it("wheel down and pinch zoom keep following", () => {
      const m = atEnd();
      wheel(m.part("row"), 10);
      wheel(m.part("row"), -10, { ctrlKey: true });
      expect(following(m)).toBe(true);
    });

    it("wheel up with nowhere to go keeps following", () => {
      const m = mount();
      m.layout.scrollHeight = 500;
      wheel(m.part("row"), -10);
      expect(following(m)).toBe(true);
    });

    it("wheel up over an inner scroller that takes the move keeps following", () => {
      const m = atEnd();
      m.part("inner").scrollTop = 30;
      wheel(m.part("inner-row"), -10);
      expect(following(m)).toBe(true);
    });

    it("wheel up over an inner scroller already at its top stops following", () => {
      const m = atEnd();
      wheel(m.part("inner-row"), -10);
      expect(following(m)).toBe(false);
    });

    it("touch stops following only outside the grace zone", () => {
      const near = atEnd();
      touchmove(near.part("row"));
      expect(following(near)).toBe(true);
      cleanup?.();

      const far = atEnd();
      far.layout.scrollHeight += 300;
      touchmove(far.part("row"));
      expect(following(far)).toBe(false);
    });

    it("touch in an inner scroller that can scroll up keeps following", () => {
      const m = atEnd();
      m.layout.scrollHeight += 300;
      m.part("inner").scrollTop = 30;
      touchmove(m.part("inner-row"));
      expect(following(m)).toBe(true);
    });

    it.each(["PageUp", "Home", "ArrowUp"])("%s on the thread stops following", (key) => {
      const m = atEnd();
      pressKey(m.part("row"), key);
      expect(following(m)).toBe(false);
    });

    it("downward keys and other keys keep following", () => {
      const m = atEnd();
      for (const key of ["PageDown", "End", "ArrowDown", "a"]) pressKey(m.part("row"), key);
      expect(following(m)).toBe(true);
    });

    it("a scroll key with a modifier or while composing keeps following", () => {
      const m = atEnd();
      for (const opts of [{ altKey: true }, { ctrlKey: true }, { metaKey: true }, { shiftKey: true }, { isComposing: true }]) {
        pressKey(m.part("row"), "ArrowUp", opts);
      }
      expect(following(m)).toBe(true);
    });

    it("a scroll key in an editable field keeps following", () => {
      const m = atEnd();
      pressKey(m.part("field"), "ArrowUp");
      pressKey(m.part("contenteditable"), "ArrowUp");
      expect(following(m)).toBe(true);
    });

    it("a scroll key outside the thread keeps following", () => {
      const m = atEnd();
      pressKey(document.body, "ArrowUp");
      expect(following(m)).toBe(true);
    });

    it("a key that a card in the thread handled keeps following", () => {
      const m = atEnd();
      const row = m.part("row");
      const handle = (e: Event) => e.preventDefault();
      row.addEventListener("keydown", handle);
      pressKey(row, "ArrowUp");
      row.removeEventListener("keydown", handle);
      expect(following(m)).toBe(true);
    });

    it("a press at the scroller's right edge stops following", () => {
      const m = atEnd();
      press(m.el, 790);
      expect(following(m)).toBe(false);
    });

    it("a press on the scroller's margin keeps following", () => {
      const m = atEnd();
      press(m.el, 100);
      expect(following(m)).toBe(true);
    });

    it("a press on content at the right edge keeps following", () => {
      const m = atEnd();
      press(m.part("row"), 790);
      expect(following(m)).toBe(true);
    });

    it("release() stops following and cancels a running glide", () => {
      const frames = stubFrames();
      const m = atEnd();
      m.glide.current = true;
      m.layout.scrollHeight += 300;
      act(() => m.stick().pin());
      act(() => m.stick().release());
      expect(frames.pending()).toBe(0);
      m.glide.current = false;
      expect(following(m)).toBe(false);
    });

    it("follow() resumes following after input stopped it", () => {
      const m = atEnd();
      wheel(m.part("row"), -10);
      act(() => m.stick().follow());
      expect(following(m)).toBe(true);
    });

    it("a scroll event inside the grace zone resumes following after input stopped it", () => {
      const m = atEnd();
      wheel(m.part("row"), -10);
      scrollTo(m.el, 490);
      expect(following(m)).toBe(true);
    });
  });

  describe("an upward move with no input", () => {
    /** Far from the end with a glide running, as during a reply. */
    function gliding() {
      const frames = stubFrames();
      const m = mount();
      act(() => m.stick().pin());
      m.glide.current = true;
      m.layout.scrollHeight += 1000;
      act(() => m.stick().pin());
      frames.runOne(performance.now() + 16);
      expect(frames.pending()).toBe(1);
      return { ...m, frames };
    }

    it("the scroll handler stops the glide when the sizes don't explain it", () => {
      const { el, frames } = gliding();
      el.scrollTop -= 100;
      act(() => {
        el.dispatchEvent(new Event("scroll"));
      });
      expect(frames.pending()).toBe(0);
    });

    it("the glide's frame check stops when the sizes don't explain it", () => {
      const { el, frames } = gliding();
      el.scrollTop -= 100;
      const top = el.scrollTop;
      frames.runOne(performance.now() + 32);
      expect(el.scrollTop).toBe(top);
      expect(frames.pending()).toBe(0);
    });

    it("the scroll handler keeps the glide when the viewport grew", () => {
      const { el, layout, frames } = gliding();
      layout.clientHeight += 100;
      el.scrollTop -= 100;
      act(() => {
        el.dispatchEvent(new Event("scroll"));
      });
      expect(frames.pending()).toBe(1);
    });

    it("the glide's frame check keeps going when the viewport grew", () => {
      const { el, layout, frames } = gliding();
      layout.clientHeight += 100;
      el.scrollTop -= 100;
      const top = el.scrollTop;
      frames.runOne(performance.now() + 32);
      expect(el.scrollTop).toBeGreaterThan(top);
    });

    it("the glide's frame check keeps going when the content shrank", () => {
      const { el, layout, frames } = gliding();
      layout.scrollHeight -= 100;
      el.scrollTop -= 100;
      const top = el.scrollTop;
      frames.runOne(performance.now() + 32);
      expect(el.scrollTop).toBeGreaterThan(top);
    });
  });
});
