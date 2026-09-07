# Scrollbar layout shift fix

## What changed

Added `scrollbar-gutter: stable` to prevent the horizontal layout jump that occurred across all views when vertical scrollbars appeared or disappeared.

## Approach

A single CSS rule in `web/src/index.css` targets the Tailwind overflow utility classes (`.overflow-y-auto`, `.overflow-auto`, `.overflow-y-scroll`) — these are the actual scroll containers. This reserves gutter space even when no scrollbar is visible, eliminating the shift.

Also removed a one-off inline `scrollbarGutter: "stable"` style from `IssueDetail.tsx` since the global rule covers it.

## Key decision

Initially tried `* { scrollbar-gutter: stable }` but it reserved gutter space on non-scrolling containers too (Layout wrapper, main element), creating an ugly gap between content and browser edge. Scoping to Tailwind's overflow classes targets only elements that actually scroll.
