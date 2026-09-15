# Spec: Popover visibility broken in production builds

## What
The `Popover` component's visibility toggle fails in production builds because it uses imperative DOM mutation (`pop.style.visibility = "visible"`) on a React-controlled inline style prop.

## Why
React reconciliation overwrites the imperative change on any parent re-render, resetting `visibility` back to `"hidden"`. The `useLayoutEffect` that sets it to `"visible"` has stable deps and doesn't re-fire. Dev mode masks this via strict mode double-invocation.

## How

### Popover.tsx
Replace the imperative `pop.style.top/left/visibility` mutations with React state:

1. Add `useState` for the computed position style
2. Initial state: `{ position: "fixed", visibility: "hidden" as const }`
3. In `useLayoutEffect`, measure the element, compute position, then call `setPopStyle(...)` with the final position + `visibility: "visible"`
4. `setPopStyle` inside `useLayoutEffect` triggers a synchronous re-render before browser paint — same flash-prevention, but React-managed

### ContextMenu.tsx (submenu flyout)
Same pattern fix for the submenu `<div>` at line 460 that starts with `visibility: "hidden"` and gets set to `"visible"` at line 166.

1. Add `useState` for submenu style
2. `useLayoutEffect` computes position and calls `setSubStyle(...)` with visibility visible
3. Reset state when `subMenu` changes (already triggered by deps)

## Acceptance Criteria
- [ ] Popover renders correctly in production builds (visibility transitions from hidden to visible)
- [ ] No flash of mispositioned content (popover doesn't appear at 0,0 before repositioning)
- [ ] ContextMenu submenu flyout also works correctly in production
- [ ] All existing tests pass (`make test`)
- [ ] Popover positioning logic unchanged (same viewport clamping/flipping)
