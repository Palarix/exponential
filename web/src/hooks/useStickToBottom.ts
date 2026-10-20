import { useCallback, useLayoutEffect, useRef, type RefObject } from "react";
import { motionAllowed } from "../preferences";
import { easeToward, nextStick, parseDuration } from "../utils/scroll";

let glideTau: number | null = null;

/** The glide's time constant: the `--duration-normal` token, read once. */
function glideMs(): number {
  glideTau ??= parseDuration(getComputedStyle(document.documentElement).getPropertyValue("--duration-normal"), 150);
  return glideTau;
}

/**
 * Keeps a scroller at its end while the user is there, for transcripts that
 * grow. It follows when the scroller or its content resizes: cards that grow
 * after render, a taller composer, a resized popup. Scrolling up stops it,
 * scrolling back to the end resumes it. Attach the ref to the scroller, and
 * call pin() after rendering new content.
 *
 * While the `glide` ref is true and movement is allowed, it eases to the end instead
 * of jumping: one frame loop that aims at the current end each frame, so new
 * output mid-glide only moves the target. Without `glide` it jumps, as when
 * a chat opens.
 */
export function useStickToBottom<T extends HTMLElement>(glide: RefObject<boolean>) {
  const ref = useRef<T>(null);
  const stuck = useRef(true);
  const lastTop = useRef(0);
  const frame = useRef(0);

  const stop = useCallback(() => {
    cancelAnimationFrame(frame.current);
    frame.current = 0;
  }, []);

  const pin = useCallback(() => {
    const el = ref.current;
    if (!el || !stuck.current) return;
    if (glide.current && motionAllowed()) {
      if (frame.current) return;
      let last = performance.now();
      const step = (now: number) => {
        frame.current = 0;
        // A scroll the user made since the last frame, before its scroll event.
        stuck.current = nextStick(stuck.current, lastTop.current, { top: el.scrollTop, clientHeight: el.clientHeight, scrollHeight: el.scrollHeight });
        if (!stuck.current) return;
        const target = Math.max(0, el.scrollHeight - el.clientHeight);
        el.scrollTop = easeToward(el.scrollTop, target, now - last, glideMs());
        last = now;
        lastTop.current = el.scrollTop;
        // The browser clamps and rounds: stop once it reports the end.
        if (Math.abs(target - el.scrollTop) >= 1) frame.current = requestAnimationFrame(step);
      };
      frame.current = requestAnimationFrame(step);
      return;
    }
    stop();
    el.scrollTop = el.scrollHeight;
    lastTop.current = el.scrollTop;
  }, [glide, stop]);

  useLayoutEffect(() => {
    const el = ref.current;
    if (!el) return;
    const onScroll = () => {
      stuck.current = nextStick(stuck.current, lastTop.current, {
        top: el.scrollTop,
        clientHeight: el.clientHeight,
        scrollHeight: el.scrollHeight,
      });
      lastTop.current = el.scrollTop;
      if (!stuck.current) stop();
    };
    el.addEventListener("scroll", onScroll, { passive: true });
    const observer = new ResizeObserver(pin);
    observer.observe(el);
    for (const child of el.children) observer.observe(child);
    pin();
    return () => {
      el.removeEventListener("scroll", onScroll);
      observer.disconnect();
      stop();
    };
  }, [pin, stop]);

  return { ref, pin };
}
