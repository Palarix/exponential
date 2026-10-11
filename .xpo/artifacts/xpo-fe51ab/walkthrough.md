# Walkthrough: one scrollbar in the issue detail view

## The bug
On a device with classic (always-visible) scrollbars, the issue detail view showed three vertical bars side by side:

1. the **document**,
2. the **content pane** (`ViewContainer`'s scroller), and
3. the **PropertySidebar**.

Only #2 should exist. Each of the other two had its own cause.

## 1. The document scrolled: an absolute element escaped the clip
`Layout` is `h-screen overflow-hidden`, so the document "can't" scroll. But `overflow: hidden` only clips descendants whose containing block is inside it. An `absolute` element resolves its containing block against the nearest *positioned* ancestor, and there was none, so it resolved against the viewport.

`ActorName` (xpo-a9560c) renders an `sr-only` span, which is `position: absolute`. In the activity timeline, far down a long issue, that span sits at its static position well below the fold. Its containing block was the initial containing block, so it extended the document and gave it a scrollbar.

**Fix:** `Layout`'s root and `ViewContainer`'s scroller (`viewContentClass`) are now `relative`. Absolute descendants now lay out against the scroller they belong to and are clipped/scrolled with it. Doing this on the scroller, not just the root, keeps any future absolute element in a pane scrolling with its content.

## 2. The sidebar: wrong height, forced bar, cascade order
The sidebar is a sticky column inside the pane. It was `max-h-screen scroll-stable`, which caused two problems:

- **Wrong reference height.** `100vh` is taller than the pane (TopBar, banner and status bar take space). So the sidebar always overflowed itself, and it stretched the pane too. A percentage height doesn't work here, because the sidebar's parent is the pane's `flex min-h-full` inner column, not the scroller. Instead, IssueDetail marks the scroller as a **size container** (`contentClassName="[container-type:size]"`), and the sidebar uses `max-h-[100cqh]`. `cqh` resolves against the scroller's height, which is exactly the visible pane. Size containment is safe here: the scroller's size comes from the flex layout, never from its content.
- **Forced bar.** `scroll-stable` forces `overflow-y: scroll` (see its comment in `index.css`), which draws a track and arrows on classic-scrollbar platforms even when nothing overflows. The sidebar now uses `overflow-y-auto`.

### Hidden bar + fade (review revision)
Even when it correctly appeared only on overflow, a sidebar bar right next to the pane's bar looked wrong. So the sidebar hides its bar and shows a fade instead:

- `scrollbar-none` utility (`scrollbar-width: none` + `::-webkit-scrollbar { display: none }`). It still scrolls by wheel, trackpad, touch and keyboard.
- `fade-bottom` utility: a `mask-image` gradient. A mask applies to the element's box, not its scrolled content, so the fade stays pinned to the visible bottom edge.
- `useScrollFade` (`hooks/useScrollFade.ts`) returns a callback ref plus a `fade` flag. It recomputes on `scroll` and through a ResizeObserver on the scroller and its child, so expanding a sidebar section or resizing the window updates the cue. The comparison lives in a pure `hasMoreBelow` (`utils/scroll.ts`, unit tested). It allows 1px of slack for sub-pixel scroll positions.
- Sidebar popovers are portaled, so the mask can't clip them.

### The cascade gotcha
The first build still showed the bar. The global scrollbar styling in `index.css` (`*::-webkit-scrollbar*` and `* { scrollbar-width: thin }`) was **unlayered**, and unlayered rules beat anything in a cascade layer, including every Tailwind utility, whatever its specificity. Chrome also ignores `::-webkit-scrollbar` once `scrollbar-width` is non-auto, so the global `thin` won and the bar stayed visible.

These rules now live in `@layer base`. Their computed values are unchanged everywhere else, and utilities can override them. Keep this in mind for any future global rule that a utility should be able to override.

## Trade-offs
- The sidebar has no reserved gutter. If it starts overflowing while open, its content doesn't shift (the bar is hidden), so this is moot in practice.
- Only a bottom fade: the sidebar's `pt-12` already gives the top edge room.

## Files
- `web/src/components/Layout/Layout.tsx`: `relative` root
- `web/src/components/ui/view-container-utils.ts` (+ test): `relative` scroller
- `web/src/components/IssueDetail/IssueDetail.tsx`: size container on the pane
- `web/src/components/IssueDetail/PropertySidebar.tsx`: `100cqh` cap, hidden bar, fade
- `web/src/hooks/useScrollFade.ts`, `web/src/utils/scroll.ts` (+ test)
- `web/src/index.css`: `scrollbar-none`, `fade-bottom`, global scrollbar rules moved to `@layer base`