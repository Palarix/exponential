# Walkthrough: the chat thread stays at the bottom

## What was wrong

An open chat sometimes stopped following new output, and the latest lines hid below the fold. `ChatThread` re-pinned to the bottom only when its `thread` prop changed, in a `useEffect`. Four things slipped past that:

1. **Content that grows after render.** Approval cards, code blocks, images and Markdown can get taller after the effect has run. The thread doesn't change, so nothing re-pins.
2. **A shrinking viewport.** A taller composer (multi-line draft, context chip, Work Mode bar) or a resized popup lowers the scroller's `clientHeight`. No scroll event fires, and the last lines slide out of view.
3. **Stray scroll events.** `onScroll` recomputed "at the bottom" from distance alone. When content grew by more than 40px and a scroll event landed before the re-pin (scroll anchoring, a clamp), following switched off although the user never scrolled.
4. **Pinning after paint.** `useEffect` runs after paint, so new output could flash below the fold first.

## The changes

**`web/src/utils/scroll.ts`: `nextStick(stuck, prevTop, {top, clientHeight, scrollHeight})`**
- Decides, after a scroll event, whether the scroller keeps following.
- Within `STICK_GRACE` (40px, as before) of the end it always follows, so scrolling back down resumes.
- Outside the grace zone, it stops only when the scroll moved **up** (`top < prevTop`). Content growth and programmatic scrolls never move the position up, so they never switch following off.
- It is a pure function, so it is unit-tested without a DOM.

**`web/src/hooks/useStickToBottom.ts`**

Returns `{ref, pin}`.
- `pin()` scrolls to the end while following and records the position, so the next scroll event compares against it.
- A `ResizeObserver` watches the scroller and its children and calls `pin` on any size change. This covers cases 1 and 2.
- The scroll listener feeds `nextStick`. This covers case 3.

**`web/src/components/Agent/ChatThread.tsx`**
- Uses the hook, and calls `pin` in a `useLayoutEffect` on `[thread, cold]`, so new output and the cold-cache marker are pinned before paint (case 4).

## Limitation

The observer watches the children that exist when the scroller mounts. `ChatThread` renders a single stable wrapper div, which grows with the content, so that's enough here. A scroller whose direct children come and go would need a `MutationObserver` too.

## Acceptance Criteria

- [x] At the bottom, new streamed output keeps the view at the bottom. Evidence: `pin` in `useLayoutEffect` on `thread`. The user tested it.
- [x] At the bottom, a card or block that grows after render keeps the view at the bottom. Evidence: the ResizeObserver on the content wrapper.
- [x] At the bottom, growing the composer or resizing the popup keeps the last message visible. Evidence: the ResizeObserver on the scroller.
- [x] Scrolled up past the grace zone, new output does not move the view. Evidence: `nextStick` tests, upward scroll outside 40px.
- [x] Scrolling back into the 40px zone resumes following. Evidence: `nextStick` tests.
- [x] Opening or reopening a chat shows the bottom. Evidence: `pin()` on mount, with following on by default.
- [x] Unit tests cover `nextStick`, and `make test` passes.