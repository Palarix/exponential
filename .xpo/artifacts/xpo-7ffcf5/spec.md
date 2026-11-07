# Agent chat: stop following only on user input, not on any upward scroll

## What
`useStickToBottom` stops following when it hears user input: wheel up, touch, scroll keys or a scrollbar press. An upward scroll with no such input still stops following, unless the scroller's sizes explain the move. The turn rail stops following explicitly. The thread scroller gets `overflow-anchor: none`.

## Why
`nextStick` stops following on any upward move of 1px or more while over 40px from the end. It can't tell a browser-made move from a user's. xpo-ea5d6b removed this risk at send time, but a glide during a reply can still be stopped. No failure has been reported since that fix, so this is hardening.

Elysium solved this (`../elysium/apps/web/src/components/chat/timelineViewport.ts`, `timelineScrollTarget.ts`).

## Flow
1. Wait for the user to merge xpo-ea5d6b.
2. Rebase this branch on main.
3. In `web/src/utils/scroll.ts`, change `nextStick` to take the previous position and sizes as `{ top, clientHeight, scrollHeight }`.
4. In `nextStick`, keep following when `scrollHeight` shrank or `clientHeight` grew since the previous reading.
5. In `web/src/utils/scroll.ts`, add `scrollsInside(target, scroller, deltaY)`. It returns true when an inner scroller takes the move. Port it from `isTimelineScrollTarget`.
6. In `useStickToBottom`, replace `lastTop` with a baseline of position and sizes. Update the baseline at every place that updates `lastTop` today.
7. In `useStickToBottom`, add `release()`. It sets `stuck` to false and cancels a running glide.
8. In `useStickToBottom`, add passive listeners that call `release()` for the inputs in the table below. The `keydown` listener goes on `document`, the others on the scroller.
9. In `useStickToBottom`, return `release` next to `pin` and `follow`.
10. In `ChatThread.tsx`, pass `release` to `TurnRail`. A bar click calls `release()` before `scrollTo`.
11. In `ChatThread.tsx`, add `[overflow-anchor:none]` to the thread scroller's class.

| Input | Releases when |
|---|---|
| `wheel` | `deltaY < 0`, no `ctrlKey`, `scrollTop > 0`, and no inner scroller takes the move |
| `touchmove` | The position is `STICK_GRACE` or more from the end, and no inner scroller can scroll up |
| `keydown` with its target in the scroller | The key is `PageUp`, `Home` or `ArrowUp`, with no modifier, not composing, not default-prevented, `scrollTop > 0`, the target is not editable, and no inner scroller takes the move |
| `pointerdown` | The target is the scroller, and the pointer is in the 16px at its right edge |

## Decisions
- **Merge order: the user merges xpo-ea5d6b first, and this branch then rebases on main.** This branch needs its jump on send, and both issues change `useStickToBottom.ts`.
- **Keep today's `STICK_GRACE` zone, not Elysium's "stay released until the user leaves and returns".** The user chose this. A scroll event inside 40px of the end turns following back on, so a tiny trackpad move at the end can't stop following. The cost: a slow scroll up during a fast reply can still be pulled back until it passes 40px, as it can today. Revisit if the tophat feels sticky.
- **Guard on `clientHeight` as well as `scrollHeight`, not on the maximum scroll position.** A shrinking composer clamps `scrollTop` up. If a new turn lands before the scroll event, the maximum position has grown again, so a net check misses the clamp. Separate checks catch it.
- **Rail stops following explicitly, not with an "app scroll" hold.** A bar click is the user's choice to leave the end. A hold needs an end signal for a smooth scroll, and `scrollTo` gives none.
- **Keys only when the target is in the scroller, not on the body as in Elysium.** The popup floats over other views, so a body-level `ArrowUp` can belong to that view and would stop following by mistake. A body-target key scroll still stops following through the scroll fallback.
- **The `keydown` listener sits on `document`, not on the scroller.** Added during implementation. `QuestionCard` and `ApprovalCard` in the thread handle keys in React, which runs after a native listener on the scroller. On `document`, a key they stopped never arrives, and a default-prevented key is visible.
- **Scrollbar press needs the 16px right-edge check, unlike Elysium.** Elysium treats any pointer press on the scroller as the scrollbar. Here the empty margins and padding of the scroller also have the scroller as target, so a click there would stop following. The zone also covers overlay scrollbars, which report no width.
- **Touch direction stays unread, as in Elysium.** A downward swipe mid-glide also stops following. This is rare, and it is still user input.
- **No sub-pixel accumulation.** Wheel and touch input cover slow scrolling, and this keeps the xpo-397365 tolerance as it is.

## Edge cases
- HIGH: `overflow-anchor: none` removes the browser's position hold while the user reads above the end. A `ToolGroup` toggles from its own header, which is in view, so only content below it moves. Content above the view that changes height shifts the view: rewrapping on a window or popup resize, or a late-loading image. Accept this unless the tophat shows it is disturbing. A fix is out of scope, and any fix gets a new issue.
- MEDIUM: Shrink and growth of `scrollHeight` inside one frame can net to growth, so the guard misses it. Removing anchoring removes the likely source. Accept this.
- MEDIUM: A size change with no scroll event leaves the baseline stale. The next real upward scroll then reads as explained, and following stops one event later. Wheel, touch and keys are not affected.
- LOW: A press on the scrollbar with no drag stops following. The next scroll event inside `STICK_GRACE` resumes it.
- LOW: A click in the 16px right padding with an overlay scrollbar stops following. The zone must cover the overlay thumb.
- LOW: A pinch zoom sends `wheel` with `ctrlKey`. It doesn't stop following.

## Acceptance criteria
- [ ] Wheel up, in the thread and outside an inner scroller, stops following before the position moves. Evidence: `useStickToBottom.dom.test.tsx`, written first.
- [ ] Wheel up over an inner scroller that can scroll up doesn't stop following. Evidence: same file.
- [ ] Wheel down and pinch zoom don't stop following. Evidence: same file.
- [ ] `touchmove` stops following only outside `STICK_GRACE`. Evidence: same file.
- [ ] `PageUp`, `Home` and `ArrowUp` in the thread stop following. A modifier, an editable target or a key a card handled doesn't. Evidence: same file.
- [ ] A press in the scroller's right-edge zone stops following. A press in its margin doesn't. Evidence: same file.
- [ ] An upward scroll with no input and unchanged sizes stops following outside `STICK_GRACE`. Evidence: `scroll.test.ts`.
- [ ] An upward scroll with a shrunk `scrollHeight` or a grown `clientHeight` doesn't stop following. Evidence: `scroll.test.ts` and the glide's per-frame check in the DOM test.
- [ ] A turn rail click stops following, and the glide doesn't pull the view back. Evidence: `ChatThread.dom.test.tsx`, and the tophat.
- [ ] The thread scroller has `overflow-anchor: none`. Evidence: `ChatThread.test.tsx`.
- [ ] `follow()` still jumps on send. Evidence: the tests from xpo-ea5d6b still pass.
- [ ] Wheel, touch, keys, scrollbar drag, drag-select and the turn rail each behave during a streaming reply, in the full view and the popup, with Motion on. The rail applies to the full view only. Evidence: the user's tophat.
- [ ] Toggling a tool card while reading above the end doesn't shift the view. Evidence: the user's tophat.
- [ ] `make test` passes.

## Assumptions
- xpo-ea5d6b lands with its `jump()` helper as committed on its branch (b431ca6).
- `ApprovalCard`'s `max-h-40 overflow-auto` block is the only inner vertical scroller in the thread today. `scrollsInside` checks computed style, so later ones are covered.
