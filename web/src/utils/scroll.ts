/**
 * Whether a scroller has content below its visible area. The 1px slack
 * absorbs fractional scroll positions on zoomed or HiDPI displays.
 */
export function hasMoreBelow(scrollTop: number, clientHeight: number, scrollHeight: number): boolean {
  return scrollTop + clientHeight < scrollHeight - 1;
}

/**
 * One frame of a glide toward target: it covers the share of the distance
 * that an exponential approach with time constant tauMs covers in dtMs, so
 * the glide takes as long at any frame rate. Steps are whole pixels, at
 * least 1px: browsers round scrollTop, and a fraction rounded back toward
 * the start reads as the user scrolling. It lands within 1px.
 */
export function easeToward(top: number, target: number, dtMs: number, tauMs: number): number {
  const distance = target - top;
  if (tauMs <= 0 || Math.abs(distance) <= 1) return target;
  const step = distance * (1 - Math.exp(-dtMs / tauMs));
  const next = top + Math.sign(distance) * Math.max(1, Math.round(Math.abs(step)));
  return Math.abs(target - next) < 1 ? target : next;
}

/** A CSS duration ("150ms", "0.2s") in milliseconds, or fallback. */
export function parseDuration(value: string, fallback: number): number {
  const m = /^\s*([\d.]+)(ms|s)\s*$/.exec(value);
  if (!m) return fallback;
  const n = Number(m[1]);
  if (!Number.isFinite(n)) return fallback;
  return m[2] === "s" ? n * 1000 : n;
}

/** How close to the end still counts as "at the bottom", in pixels. */
export const STICK_GRACE = 40;

/** A scroller's position and sizes at one moment. */
export type ScrollReading = { top: number; clientHeight: number; scrollHeight: number };

/**
 * Whether a follow-the-output scroller keeps following after a scroll event.
 * Within STICK_GRACE of the end it follows. Outside it, an upward move stops
 * following, unless something other than the user explains it: content growth
 * and clamping (the content shrank or the viewport grew) and the browser
 * rounding a fractional position (under 1px) never do. The sizes are checked
 * apart, not as the maximum position: a clamp followed by new content before
 * the scroll event leaves that maximum higher than before.
 */
export function nextStick(stuck: boolean, prev: ScrollReading, { top, clientHeight, scrollHeight }: ScrollReading): boolean {
  if (scrollHeight - top - clientHeight < STICK_GRACE) return true;
  return stuck && (top > prev.top - 1 || scrollHeight < prev.scrollHeight || clientHeight > prev.clientHeight);
}

/**
 * Whether an inner scroller between target and scroller takes a move in the
 * direction of deltaY: it can still scroll that way, or it doesn't pass the
 * move on to the scroller.
 */
export function scrollsInside(target: EventTarget | null, scroller: HTMLElement, deltaY: number): boolean {
  if (!(target instanceof Element)) return false;
  for (let el: Element | null = target; el && el !== scroller; el = el.parentElement) {
    const style = getComputedStyle(el);
    if (style.overflowY !== "auto" && style.overflowY !== "scroll") continue;
    const canScroll = deltaY < 0 ? el.scrollTop > 0 : el.scrollTop < el.scrollHeight - el.clientHeight;
    if (canScroll || style.overscrollBehaviorY === "contain" || style.overscrollBehaviorY === "none") return true;
  }
  return false;
}
