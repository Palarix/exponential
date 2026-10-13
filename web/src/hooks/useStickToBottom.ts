import { useCallback, useLayoutEffect, useRef } from "react";
import { nextStick } from "../utils/scroll";

/**
 * Keeps a scroller at its end while the user is there, for transcripts that
 * grow. It follows when the scroller or its content resizes: cards that grow
 * after render, a taller composer, a resized popup. Scrolling up stops it,
 * scrolling back to the end resumes it. Attach the ref to the scroller, and
 * call pin() after rendering new content.
 */
export function useStickToBottom<T extends HTMLElement>() {
  const ref = useRef<T>(null);
  const stuck = useRef(true);
  const lastTop = useRef(0);

  const pin = useCallback(() => {
    const el = ref.current;
    if (!el || !stuck.current) return;
    el.scrollTop = el.scrollHeight;
    lastTop.current = el.scrollTop;
  }, []);

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
    };
    el.addEventListener("scroll", onScroll, { passive: true });
    const observer = new ResizeObserver(pin);
    observer.observe(el);
    for (const child of el.children) observer.observe(child);
    pin();
    return () => {
      el.removeEventListener("scroll", onScroll);
      observer.disconnect();
    };
  }, [pin]);

  return { ref, pin };
}
