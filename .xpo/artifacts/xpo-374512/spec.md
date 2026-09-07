# Center popovers on trigger button

## What

FilterMenu and ViewOptionsMenu popovers right-align to the trigger button, making them hard to reach with the mouse.

## Why

The popover extends far to the left of the button. Centering makes the menu feel anchored to the button.

## How

### FilterMenu (`FilterMenu.tsx`, line 255)

Current: `left = aRect.right - mRect.width` (right-aligned).

Change to: `left = aRect.left + aRect.width / 2 - mRect.width / 2` (centered on trigger). Then clamp to viewport bounds (min 8px from left edge, max `window.innerWidth - mRect.width - 8` from right).

### ViewOptionsMenu (`Backlog.tsx`, line 1429)

Current: `absolute right-0 top-full mt-1` (CSS right-aligned).

Change to: `absolute top-full mt-1 left-1/2 -translate-x-1/2` (CSS centered on parent). No viewport clamping needed — the menu is small (min-w-44 = 176px) and the trigger is well inside the viewport.

## Acceptance Criteria

- Both menus center horizontally on their trigger button
- FilterMenu falls back to edge-aligned if centering would overflow viewport
- No clipping or off-screen positioning
