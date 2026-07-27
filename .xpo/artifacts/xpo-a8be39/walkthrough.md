# Walkthrough: Move issue detail scrollbar to right edge with sticky sidebar

## What changed

Two files changed: `IssueDetail.tsx` (layout restructure) and `PropertySidebar.tsx` (sticky positioning).

## IssueDetail.tsx

The body section (everything below the breadcrumb bar) was restructured:

**Before:**
```
div.flex-1.flex.overflow-hidden          ← flex row, no scroll
  div.flex-1.overflow-y-auto.min-w-0    ← LEFT: scrolls independently
    div.max-w-4xl.mx-auto.px-8.py-12   ← content wrapper
  PropertySidebar                        ← RIGHT: scrolls independently
```

**After:**
```
div.flex-1.overflow-y-auto               ← SINGLE scroll container
  div.flex.min-h-full                    ← flex row inside scroll
    div.flex-1.min-w-0                   ← LEFT: no own scroll
      div.max-w-4xl.mx-auto.px-8.py-12  ← content wrapper
    PropertySidebar                      ← RIGHT: sticky inside scroll
```

The key insight: `overflow-y-auto` moved up one level (from the content div to the body div), making both content and sidebar children of the same scroll container. The inner flex row uses `min-h-full` so short content still fills the viewport height.

## PropertySidebar.tsx

The root div changed from:
```
w-80 overflow-y-auto shrink-0
```
to:
```
w-80 shrink-0 sticky top-0 self-start max-h-screen overflow-y-auto
```

- `sticky top-0` — stays pinned at the top of the scroll container's viewport
- `self-start` — prevents the sidebar from stretching to the full content height (which would break sticky behavior)
- `max-h-screen overflow-y-auto` — on short screens where the sidebar content is taller than the viewport, it gets its own scroll

## Why `self-start` matters

Without `self-start`, the sidebar would be `align-self: stretch` (flex default), making it as tall as the content column. A sticky element that's as tall as its scroll container has nowhere to "stick" — it just scrolls normally. `self-start` makes the sidebar only as tall as its content, enabling the sticky behavior.