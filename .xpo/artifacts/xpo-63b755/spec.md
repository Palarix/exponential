## What
Keep the chat thread pinned to the bottom while the user is at the bottom, whatever makes the content or the viewport change size.

## Why
`ChatThread` already treats "within 40px of the end" as the bottom (the Elysium-style grace zone). It still loses the bottom because it only re-pins when the `thread` prop changes:

1. **Content grows after render.** Approval cards, code blocks, images, and Markdown can change height after the effect ran. No `thread` change means no re-pin.
2. **Viewport shrinks.** The composer grows (multi-line draft, context chip, Work Mode bar), or the popup is resized. The scroller's `clientHeight` drops, no scroll event fires, and the last lines slide below the fold.
3. **Stray scroll events unstick.** `onScroll` recomputes `stick` from distance alone. If content grew by more than 40px and any scroll event lands before the re-pin (scroll anchoring, a size clamp), `stick` turns false although the user never scrolled up.
4. The re-pin runs in `useEffect`, after paint, so new output can flash below the fold first.

## How
- Move the logic into a small hook, `useStickToBottom()`, in `web/src/hooks/`. It returns a ref for the scroller and one for the content.
- A `ResizeObserver` watches both the scroller and its content. On any size change, while stuck, set `scrollTop = scrollHeight`.
- Also re-pin in a `useLayoutEffect` on `[thread, cold]` (before paint).
- Unstick only when the user scrolls **up** (scrollTop decreased) and ends outside the 40px grace zone. Re-stick whenever the position is within 40px of the end. Content growth and programmatic scrolls never unstick.
- Put the decision in a pure function in `web/src/utils/scroll.ts` (`nextStick(prev, prevTop, top, clientHeight, scrollHeight)`) with unit tests. Same 40px grace as today.

## Acceptance criteria
- [ ] At the bottom, new streamed output keeps the view at the bottom.
- [ ] At the bottom, a card or block that grows after render keeps the view at the bottom.
- [ ] At the bottom, growing the composer or resizing the popup keeps the last message visible.
- [ ] Scrolled up past the grace zone, new output does not move the view.
- [ ] Scrolling back into the 40px zone resumes following.
- [ ] Opening or reopening a chat shows the bottom.
- [ ] Unit tests cover `nextStick`; `make test` passes.
