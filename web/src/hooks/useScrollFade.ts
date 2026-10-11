import { useEffect, useState } from "react";
import { hasMoreBelow } from "../utils/scroll";

/**
 * Tracks whether a scroller has content below its visible area, for an
 * overflow cue on panes whose scrollbar is hidden. Recomputes on scroll and
 * when the scroller or its content resizes. Attach the returned ref to the
 * scroller.
 */
export function useScrollFade<T extends HTMLElement>(): [(el: T | null) => void, boolean] {
  const [el, setEl] = useState<T | null>(null);
  const [fade, setFade] = useState(false);

  useEffect(() => {
    if (!el) return;
    const update = () => setFade(hasMoreBelow(el.scrollTop, el.clientHeight, el.scrollHeight));
    update();
    el.addEventListener("scroll", update, { passive: true });
    const observer = new ResizeObserver(update);
    observer.observe(el);
    for (const child of el.children) observer.observe(child);
    return () => {
      el.removeEventListener("scroll", update);
      observer.disconnect();
    };
  }, [el]);

  return [setEl, fade];
}
