## What
The issue detail view shows one scrollbar (the content pane) instead of three.

## Why
On devices with classic scrollbars, the view shows a document scrollbar, a sidebar scrollbar, and the pane scrollbar side by side.

## How
1. **Document scroll.** Add `relative` to the `Layout` root and to `ViewContainer`'s scroller. Absolute descendants (`ActorName`'s `sr-only` span) then resolve against a clipping/scrolling ancestor instead of the viewport, so they can't extend the document.
2. **Sidebar height.** `PropertySidebar` uses `max-h-[100cqh] overflow-y-auto` in place of `max-h-screen scroll-stable`. IssueDetail's pane is a size container (`[container-type:size]`), so `100cqh` is the pane's visible height.
3. **Sidebar scrollbar hidden** (revision after review). A scrollbar next to the pane's scrollbar looks wrong. The sidebar stays scrollable by wheel, trackpad, touch and keyboard, but its scrollbar isn't drawn (new `scrollbar-none` utility in `index.css`).
   - The global scrollbar styles in `index.css` (`*::-webkit-scrollbar*` and `* { scrollbar-width: thin }`) move into `@layer base`. Unlayered, they beat every Tailwind utility, so `scrollbar-none` was ignored. Chrome also ignores `::-webkit-scrollbar` once `scrollbar-width` is non-auto.
4. **Overflow cue.** While the sidebar has content below the visible area, its bottom edge fades out (`mask-image` gradient, `fade-bottom` utility). A `useScrollFade` hook sets the flag. It recomputes on scroll and on resize of the scroller or its content (ResizeObserver), so expanding a section updates the cue. Popovers are portaled, so the mask never clips them.

## Decisions
- Hide the bar instead of making the sidebar scroll with the page: the sidebar stays reachable while you read long content.
- The fade only shows at the bottom. Top fade not needed: the sidebar's `pt-12` gives the top edge breathing room.

## Acceptance criteria
- The document never scrolls.
- The issue detail view only ever shows the pane scrollbar.
- When the sidebar is taller than the pane, it scrolls by wheel/trackpad, and its bottom edge fades until scrolled to the end.
- The sidebar stays sticky while the main content scrolls.
- `make test` passes.