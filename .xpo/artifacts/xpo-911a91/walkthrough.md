# ContextMenu sub-menu viewport clamping

## What changed

ContextMenu sub-menus no longer clip outside the viewport when opened near window edges.

## Approach

Pulled the sub-menu out of the main menu's flex container and rendered it as a separate `position: fixed` div in the same portal. A `useLayoutEffect` runs whenever `subMenu` or `subMenuOffset` changes and calculates the sub-menu's position:

1. Default: place to the right of the main menu with a 4px gap
2. If that would overflow the right edge: flip to the left of the main menu
3. Clamp vertically so the sub-menu stays within 8px of viewport edges on all sides
4. Set `visibility: visible` after positioning to prevent flash

Also updated the click-outside handler to check `subMenuRef` — since the sub-menu is no longer a DOM child of `menuRef`, clicks inside it would have incorrectly dismissed the menu.

## Key decision

Used independent fixed positioning (JS measurement) rather than the flex `order` trick, which caused the sub-menu to overlap the main menu because reordering within a fixed-position flex container shifts both children rather than just repositioning one.
