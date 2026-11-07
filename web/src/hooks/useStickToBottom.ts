import { useCallback, useLayoutEffect, useRef, type RefObject } from "react";
import { motionAllowed } from "../preferences";
import { easeToward, nextStick, parseDuration, scrollsInside, STICK_GRACE, type ScrollReading } from "../utils/scroll";

let glideTau: number | null = null;

/** The glide's time constant: the `--duration-normal` token, read once. */
function glideMs(): number {
  glideTau ??= parseDuration(getComputedStyle(document.documentElement).getPropertyValue("--duration-normal"), 150);
  return glideTau;
}

function reading(el: HTMLElement): ScrollReading {
  return { top: el.scrollTop, clientHeight: el.clientHeight, scrollHeight: el.scrollHeight };
}

const UP_KEYS = new Set(["PageUp", "Home", "ArrowUp"]);
const EDITABLE = 'input, textarea, select, [contenteditable]:not([contenteditable="false"])';

/** The width at the scroller's right edge that a press on its scrollbar lands in. Overlay scrollbars report no width. */
const SCROLLBAR_ZONE = 16;

/**
 * Keeps a scroller at its end while the user is there, for transcripts that
 * grow. It follows when the scroller or its content resizes: cards that grow
 * after render, a taller composer, a resized popup. User input stops it: wheel
 * up, touch, scroll keys, a press on the scrollbar. So does an upward scroll
 * the scroller's sizes don't explain, as when drag-selecting text. Scrolling
 * back to the end resumes it. Attach the ref to the scroller, and
 * call pin() after rendering new content. follow() resumes following and
 * jumps to the end even when the user scrolled up, as after they send a
 * message. It never glides: a jump leaves no frames in which a scroll event
 * from an earlier upward move can read as the user scrolling up. release()
 * stops following, for code that moves the scroller away on the user's behalf.
 *
 * While the `glide` ref is true and movement is allowed, pin() eases to the end instead
 * of jumping: one frame loop that aims at the current end each frame, so new
 * output mid-glide only moves the target. Without `glide` it jumps, as when
 * a chat opens.
 */
export function useStickToBottom<T extends HTMLElement>(glide: RefObject<boolean>) {
  const ref = useRef<T>(null);
  const stuck = useRef(true);
  const baseline = useRef<ScrollReading>({ top: 0, clientHeight: 0, scrollHeight: 0 });
  const frame = useRef(0);

  const stop = useCallback(() => {
    cancelAnimationFrame(frame.current);
    frame.current = 0;
  }, []);

  const jump = useCallback(() => {
    const el = ref.current;
    if (!el) return;
    stop();
    el.scrollTop = el.scrollHeight;
    baseline.current = reading(el);
  }, [stop]);

  const pin = useCallback(() => {
    const el = ref.current;
    if (!el || !stuck.current) return;
    if (glide.current && motionAllowed()) {
      if (frame.current) return;
      let last = performance.now();
      const step = (now: number) => {
        frame.current = 0;
        // A scroll the user made since the last frame, before its scroll event.
        stuck.current = nextStick(stuck.current, baseline.current, reading(el));
        if (!stuck.current) return;
        const target = Math.max(0, el.scrollHeight - el.clientHeight);
        el.scrollTop = easeToward(el.scrollTop, target, now - last, glideMs());
        last = now;
        baseline.current = reading(el);
        // The browser clamps and rounds: stop once it reports the end.
        if (Math.abs(target - el.scrollTop) >= 1) frame.current = requestAnimationFrame(step);
      };
      frame.current = requestAnimationFrame(step);
      return;
    }
    jump();
  }, [glide, jump]);

  const follow = useCallback(() => {
    stuck.current = true;
    jump();
  }, [jump]);

  const release = useCallback(() => {
    stuck.current = false;
    stop();
  }, [stop]);

  useLayoutEffect(() => {
    const el = ref.current;
    if (!el) return;
    const onScroll = () => {
      const now = reading(el);
      stuck.current = nextStick(stuck.current, baseline.current, now);
      baseline.current = now;
      if (!stuck.current) stop();
    };
    const onWheel = (e: WheelEvent) => {
      if (e.deltaY < 0 && !e.ctrlKey && el.scrollTop > 0 && !scrollsInside(e.target, el, e.deltaY)) release();
    };
    // Touch has no direction to read. An upward swipe leaves the grace zone
    // within its first few events, and inside it the scroll event would resume.
    const onTouchMove = (e: TouchEvent) => {
      if (el.scrollHeight - el.scrollTop - el.clientHeight >= STICK_GRACE && !scrollsInside(e.target, el, -1)) release();
    };
    // On the document, so a key a card in the thread handled never gets here.
    // Only keys in the thread: a key on the body can belong to the view under
    // the popup, and the scroll it causes, if any, still stops following.
    const onKeyDown = (e: KeyboardEvent) => {
      if (!UP_KEYS.has(e.key) || e.defaultPrevented || e.isComposing || e.altKey || e.ctrlKey || e.metaKey || e.shiftKey) return;
      if (!(e.target instanceof Element) || !el.contains(e.target) || e.target.closest(EDITABLE)) return;
      if (el.scrollTop > 0 && !scrollsInside(e.target, el, -1)) release();
    };
    // The scroller is also the target of a press on its empty margins and padding.
    const onPointerDown = (e: PointerEvent) => {
      if (e.target === el && e.clientX >= el.getBoundingClientRect().right - SCROLLBAR_ZONE) release();
    };
    el.addEventListener("scroll", onScroll, { passive: true });
    el.addEventListener("wheel", onWheel, { passive: true });
    el.addEventListener("touchmove", onTouchMove, { passive: true });
    el.addEventListener("pointerdown", onPointerDown, { passive: true });
    document.addEventListener("keydown", onKeyDown);
    const observer = new ResizeObserver(pin);
    observer.observe(el);
    for (const child of el.children) observer.observe(child);
    pin();
    return () => {
      el.removeEventListener("scroll", onScroll);
      el.removeEventListener("wheel", onWheel);
      el.removeEventListener("touchmove", onTouchMove);
      el.removeEventListener("pointerdown", onPointerDown);
      document.removeEventListener("keydown", onKeyDown);
      observer.disconnect();
      stop();
    };
  }, [pin, release, stop]);

  return { ref, pin, follow, release };
}
