# ContextMenu sub-menu viewport clamping

## What

Sub-menus clip outside the viewport when the context menu is near the right or bottom edge.

## Why

The main menu clamps to viewport bounds (lines 128-141), but the sub-menu is a flex sibling with `ml-1` and a `marginTop` offset — no overflow detection.

## How

Add a `subMenuRef` and a `useLayoutEffect` that runs when `subMenu` or `subMenuOffset` changes:

1. Measure the sub-menu's bounding rect after render
2. **Horizontal flip:** if the sub-menu's right edge exceeds `window.innerWidth - 8`, apply `order: -1` (move before main menu in flex) and swap `ml-1` to `mr-1`
3. **Vertical clamp:** if the sub-menu's bottom edge exceeds `window.innerHeight - 8`, reduce the `marginTop` to fit

Use direct style/class manipulation on the ref to avoid re-render loops.

## Acceptance Criteria

- Sub-menu flips left when near the right viewport edge
- Sub-menu doesn't extend below the viewport
- No visual flicker during repositioning
