/**
 * Whether a scroller has content below its visible area. The 1px slack
 * absorbs fractional scroll positions on zoomed or HiDPI displays.
 */
export function hasMoreBelow(scrollTop: number, clientHeight: number, scrollHeight: number): boolean {
  return scrollTop + clientHeight < scrollHeight - 1;
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
