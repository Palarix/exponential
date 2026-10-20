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
 * the glide takes as long at any frame rate. It moves at least 1px, as
 * browsers may round scrollTop, and lands within 1px.
 */
export function easeToward(top: number, target: number, dtMs: number, tauMs: number): number {
  const distance = target - top;
  if (tauMs <= 0 || Math.abs(distance) <= 1) return target;
  const step = distance * (1 - Math.exp(-dtMs / tauMs));
  const next = top + (Math.abs(step) < 1 ? Math.sign(distance) : step);
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

/**
 * Whether a follow-the-output scroller keeps following after a scroll event.
 * Within STICK_GRACE of the end it follows. Outside it, only an upward
 * scroll by the user stops following: content growth and clamping never do.
 */
export function nextStick(
  stuck: boolean,
  prevTop: number,
  { top, clientHeight, scrollHeight }: { top: number; clientHeight: number; scrollHeight: number },
): boolean {
  if (scrollHeight - top - clientHeight < STICK_GRACE) return true;
  return stuck && top >= prevTop;
}
