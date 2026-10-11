/**
 * Whether a scroller has content below its visible area. The 1px slack
 * absorbs fractional scroll positions on zoomed or HiDPI displays.
 */
export function hasMoreBelow(scrollTop: number, clientHeight: number, scrollHeight: number): boolean {
  return scrollTop + clientHeight < scrollHeight - 1;
}
