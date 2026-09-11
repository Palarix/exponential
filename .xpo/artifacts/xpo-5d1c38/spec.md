# Spec: Extract `Popover` primitive with anchor-ref positioning and dismissal

## What

Refactor `web/src/components/ui/Popover.tsx` from its current marker-span pattern into a proper anchor-ref-based positioning primitive. Add placement options, full viewport clamping, and built-in dismissal. Add `forwardRef` to `Button` so consumers can anchor popovers directly to trigger buttons. Migrate all existing consumers and rewrite `InlineDropdown` on top of it.

## Why

The codebase has 12 independent floating-element positioning implementations across 8 files, each rolling its own `getBoundingClientRect` + viewport clamping logic. Viewport clamping ranges from none (InlineDropdown, Board ViewMenu) to all 4 edges (ContextMenu only). Escape handling is missing in 5 of 12 implementations. A proper Popover primitive eliminates this duplication and inconsistency for all future popover-like UI.

## Current State

The existing `Popover.tsx` is minimal:
- Uses a hidden `<span ref={markerRef}>` to find `marker.parentElement` as anchor
- Portals to `document.body` with `position: fixed`
- Clamps right + bottom only (not left/top)
- Click-outside: checks popover only (not anchor)
- Escape: via `useKeyboardShortcuts`
- Hardcoded styling (min-w-50, surface-3, radius-lg, shadow-popover)
- Used by 3 consumers: Board (3 instances), PropertySidebar (6 instances), NewIssueModal (3 instances)

`Button.tsx` doesn't use `forwardRef`, so consumers can't attach refs for popover anchoring.

`InlineDropdown.tsx` has its own bare positioning with no viewport clamping and no Escape handling.

## New API

```tsx
interface PopoverProps {
  anchorRef: RefObject<HTMLElement | null>;
  onClose: () => void;
  placement?: 'bottom-start' | 'bottom-center' | 'bottom-end';
  offset?: number;
  className?: string;
  children: ReactNode;
}
```

- `anchorRef` — ref to the element the popover positions relative to (replaces marker-span)
- `onClose` — called on click-outside or Escape
- `placement` — where to place relative to anchor (default: `'bottom-start'`)
- `offset` — gap in px between anchor and popover (default: `4`)
- `className` — merged via `cn()` with base popover styles
- Component renders when mounted — parent controls visibility via conditional rendering (`{open && <Popover>}`)
- `PopoverHeader` export preserved

## Positioning Logic

Extract `computePopoverPosition()` into `popover-utils.ts` as a pure, testable function:

```
Input: anchorRect, popoverDims, placement, offset, viewport
Output: { top, left }
```

1. Calculate initial position based on `placement`:
   - `bottom-start`: top = anchor.bottom + offset, left = anchor.left
   - `bottom-center`: top = anchor.bottom + offset, left = anchor.centerX - popover.width/2
   - `bottom-end`: top = anchor.bottom + offset, left = anchor.right - popover.width
2. **Flip above** if overflows bottom: top = anchor.top - popover.height - offset
3. **Clamp all 4 edges** with 8px pad (matching ContextMenu, the most thorough existing implementation)

## Dismissal

- **Click-outside**: `mousedown` listener, exclude both popover and anchor element
- **Escape**: `useKeyboardShortcuts` with scope `"popover"`, priority `"overlay"` (existing behavior)

## Flow

### 0. Add `forwardRef` to `Button`

Convert `Button.tsx` from a plain function component to `forwardRef<HTMLButtonElement, ButtonProps>`. No API change — existing consumers unaffected; consumers that need a ref can now pass one.

### 1. Tests first — `popover-utils.ts` and `Popover.test.ts`

Test `computePopoverPosition()`:
- Each placement calculates correct initial position
- Flips above when overflowing bottom
- Clamps all 4 edges
- Combined: flip + clamp
- Edge case: popover larger than viewport

### 2. Extract `popover-utils.ts`

Pure function `computePopoverPosition()`.

### 3. Refactor `Popover.tsx`

- Remove marker span
- Accept new props
- Use `computePopoverPosition()` in `useLayoutEffect`
- Click-outside excludes `anchorRef` element
- Merge `className` via `cn()` with base styles
- Keep `PopoverHeader` export

### 4. Migrate existing consumers (12 instances across 3 files)

With `forwardRef` on Button, consumers can attach refs directly to trigger buttons for precise positioning.

**Board.tsx** (3 instances): status, estimate, labels popovers
**PropertySidebar.tsx** (6 instances): status, estimate, priority, parent, assignee, cycle popovers
**NewIssueModal.tsx** (3 instances): parent, assignee, labels popovers

### 5. Rewrite `InlineDropdown` on Popover

Replace internal positioning logic with `<Popover anchorRef={btnRef} onClose={...} placement="bottom-start">`. This also adds viewport clamping and Escape handling that InlineDropdown currently lacks.

### 6. `make test`

## Decisions

- **No `open` prop** — component renders when mounted. Parent controls visibility via conditional rendering, which is the existing pattern everywhere in this codebase. Simpler, no internal state.
- **Button gets `forwardRef`** — a basic primitive should support refs. This lets consumers anchor popovers to the actual trigger button rather than a wrapping div, giving more precise positioning.
- **Base styles remain as defaults** — `min-w-50 bg-surface-3 border radius-lg shadow-popover py-1`. `className` can override via `cn()`. This matches the existing Popover look.
- **ContextMenu, FilterMenu, FormattingBar, ViewMenus not in scope** — these have specialized sub-menu, arrow-key nav, or domain-specific logic. They can adopt the Popover primitive incrementally in future issues.

## Acceptance Criteria

- [ ] `Button` component supports `forwardRef`
- [ ] `anchorRef` positioning with `placement` options (bottom-start, bottom-center, bottom-end)
- [ ] Full 4-edge viewport clamping (8px pad) with bottom overflow flip
- [ ] Click-outside dismissal (excludes anchor) + Escape dismissal built in
- [ ] `computePopoverPosition()` extracted and tested in `popover-utils.ts`
- [ ] All 12 existing Popover consumer instances migrated (Board 3, PropertySidebar 6, NewIssueModal 3)
- [ ] `InlineDropdown` rewritten on Popover primitive
- [ ] `PopoverHeader` export preserved
- [ ] `make test` passes
- [ ] No visual regression
