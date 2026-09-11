# Walkthrough: Extract `Popover` primitive with anchor-ref positioning and dismissal

## Summary

Refactored the Popover component from a marker-span hack into a proper anchor-ref-based positioning primitive. Added `forwardRef` to Button, migrated all 17 Popover consumer instances, and rewrote InlineDropdown on top of Popover. The codebase now has one canonical way to position floating elements with built-in viewport clamping and dismissal.

## What was the marker-span pattern?

The old Popover rendered a hidden `<span ref={markerRef} className="hidden" />` alongside its portal. It found its anchor by reading `marker.parentElement` — whatever DOM element happened to wrap the Popover. This was fragile (positioning depended on where you placed the component in the tree) and couldn't accept configuration (placement, offset, anchor element).

## New Popover API

```tsx
<Popover
  anchorRef={ref}          // RefObject<HTMLElement | null> — positions relative to this
  onClose={fn}             // called on click-outside or Escape
  placement="bottom-start" // bottom-start | bottom-center | bottom-end
  offset={4}               // gap in px between anchor and popover
  className=""             // merged via cn() with base styles
>
```

Component renders when mounted — parent controls visibility via conditional rendering. No `open` prop, no internal state.

## New files

### `web/src/components/ui/popover-utils.ts`

Pure function `computePopoverPosition(anchorRect, popoverDims, placement, offset, viewport)` that returns `{ top, left }`. Handles:

1. Initial placement based on variant (start/center/end alignment)
2. Bottom overflow flip — moves above the anchor when it would overflow the bottom
3. 4-edge clamping with 8px padding — matches ContextMenu, the most thorough existing implementation

Extracted as a pure function so positioning logic is testable without a DOM environment.

### `web/src/components/ui/Popover.test.ts`

10 vitest cases covering: all 3 placements, bottom overflow flip, left/right/top edge clamping, combined flip+clamp, and bottom-end with left clamp.

## Changed files

### `web/src/components/ui/Button.tsx`

Converted from a plain function component to `forwardRef<HTMLButtonElement, ButtonProps>`. No API change — existing consumers are unaffected; consumers that need a ref (for popover anchoring) can now pass one.

### `web/src/components/ui/Popover.tsx`

- Removed marker span and `markerRef`
- Accepts `anchorRef`, `placement`, `offset`, `className`
- Uses `computePopoverPosition()` in `useLayoutEffect`
- Click-outside now excludes the anchor element (old version only checked the popover)
- Class merging via `cn()` — base styles are defaults, `className` can override
- `PopoverHeader` export preserved unchanged

### `web/src/components/ui/InlineDropdown.tsx`

Rewritten on Popover. Removed:
- `createPortal` import and usage
- `useEffect` for click-outside
- `pos` state and `handleOpen` positioning logic
- `menuRef`

The menu is now `<Popover anchorRef={btnRef} onClose={...} className="min-w-40">`. This adds viewport clamping and Escape handling that InlineDropdown previously lacked.

### Consumer migrations

All consumers use a single shared `useRef` per component, conditionally assigned to the wrapping div that's active:

```tsx
<div ref={openPopover === "status" ? popoverAnchorRef : undefined}>
  <Button .../>
  {openPopover === "status" && (
    <Popover anchorRef={popoverAnchorRef} ...>
  )}
</div>
```

React sets refs before layout effects, so the ref is populated by the time Popover's `useLayoutEffect` reads it.

| File | Instances | Anchor strategy |
|------|-----------|-----------------|
| `Board.tsx` | 3 | Single ref on board root div (keyboard-triggered popovers, no click target) |
| `PropertySidebar.tsx` | 7 | Shared ref, conditionally set per property row |
| `NewIssueModal.tsx` | 3 | Shared ref, conditionally set per popover type |
| `BacklogIssueRow.tsx` | 4 | Shared ref, conditionally set per popover type |

## Key decisions

- **No `open` prop** — parent controls visibility via conditional rendering, matching the existing pattern everywhere. Simpler than internal state management.

- **Wrapping-div anchor for existing consumers** — gives identical positioning to the marker-span approach. Now that Button supports forwardRef, a future follow-up could point the ref directly at the trigger button for more precise positioning.

- **Board anchors to root div** — Board's popovers are triggered by keyboard shortcuts (s/l/e), not clicks. There's no natural click target. The old marker span happened to anchor to the board root div; we preserve that behavior.

- **BacklogIssueRow was an uncounted consumer** — the original issue and initial survey listed Board, PropertySidebar, and NewIssueModal. BacklogIssueRow has 4 Popover instances that weren't in the initial count. Discovered during the "no old-style usage remaining" check and migrated.

- **ContextMenu, FilterMenu, FormattingBar, ViewMenus not migrated** — these have specialized sub-menu positioning, arrow-key navigation, or domain-specific logic. They can adopt the Popover primitive incrementally in future issues.

## Acceptance Criteria

- [x] `Button` component supports `forwardRef`
- [x] `anchorRef` positioning with `placement` options (bottom-start, bottom-center, bottom-end)
- [x] Full 4-edge viewport clamping (8px pad) with bottom overflow flip
- [x] Click-outside dismissal (excludes anchor) + Escape dismissal built in
- [x] `computePopoverPosition()` extracted and tested in `popover-utils.ts` — 10 tests
- [x] All 17 existing Popover consumer instances migrated (Board 3, PropertySidebar 7, NewIssueModal 3, BacklogIssueRow 4)
- [x] `InlineDropdown` rewritten on Popover primitive
- [x] `PopoverHeader` export preserved
- [x] `make test` passes (lint + 304 frontend tests + Go tests)
- [x] No visual regression — confirmed by user
