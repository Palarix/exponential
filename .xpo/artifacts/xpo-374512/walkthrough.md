# Center popovers on trigger button

## What changed

FilterMenu and ViewOptionsMenu popovers now center horizontally on their trigger buttons instead of right-aligning.

## FilterMenu (`FilterMenu.tsx`)

Changed the `left` calculation from `aRect.right - mRect.width` (right-aligned) to `aRect.left + aRect.width / 2 - mRect.width / 2` (centered). Added right-edge clamping (`window.innerWidth - mRect.width - 8`) alongside the existing left-edge clamp.

## ViewOptionsMenu (`Backlog.tsx`)

Converted from CSS absolute positioning (`right-0`) to the same JS portal approach used by FilterMenu:
- Portaled to `document.body` via `createPortal`
- `position: fixed` with `visibility: hidden` until positioned
- Centered on trigger with the same viewport clamping on all edges
- Removed the `relative` wrapper div since positioning is no longer CSS-based

Both menus now use identical positioning logic: center on trigger, clamp to 8px from all viewport edges.
