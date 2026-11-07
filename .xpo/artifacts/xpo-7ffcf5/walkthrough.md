# Walkthrough: stop following only on user input, not on any upward scroll

## The problem
`useStickToBottom` keeps a thread at its end while `stuck` is true. Before this change, `nextStick` turned `stuck` off on any upward move of 1px or more while over 40px from the end. It could not tell a browser-made move from a user's. xpo-ea5d6b removed this risk at send time, but a glide during a reply stayed exposed. No failure was reported after that fix, so this is hardening, not a fix for a confirmed bug.

## How it works now
Two paths stop following.

**1. User input.** The hook listens for input and calls the new `release()`, which sets `stuck` to false and cancels a running glide. This happens before the position moves, so the glide doesn't pull against the user.

| Input | Stops following when |
|---|---|
| `wheel` | Upward, not a pinch zoom (`ctrlKey`), `scrollTop > 0`, and no inner scroller takes the move |
| `touchmove` | `STICK_GRACE` or more from the end, and no inner scroller can scroll up |
| `keydown` | `PageUp`, `Home` or `ArrowUp` in the thread, with no modifier, not composing, not default-prevented, not in an editable target, `scrollTop > 0`, and no inner scroller takes the move |
| `pointerdown` | The target is the scroller, in the 16px at its right edge |

**2. The scroll fallback.** Some upward moves have no input we hear, such as drag-selecting text or a key with focus on the body. `nextStick` still stops following for those. It now compares against a baseline of position and sizes (`ScrollReading`), and it keeps following when `scrollHeight` shrank or `clientHeight` grew. The scroll handler and the glide's per-frame check share this rule.

The turn rail calls `release()` before `scrollTo`, because a bar click is the user choosing to leave the end. The thread scroller has `overflow-anchor: none`, which removes one possible source of unexplained upward moves.

## Decisions a future reader might question
- **Two size checks, not one on the maximum position.** A shrinking composer clamps `scrollTop` up. If a new turn lands before the scroll event, the maximum position is higher than before, so a net check misses the clamp. Elysium checks only `scrollHeight`, because its composer appears to overlay the list. Ours resizes the scroller.
- **The `keydown` listener is on `document`.** `QuestionCard` and `ApprovalCard` handle keys in React, which runs after a native listener on the scroller. On `document`, a key they stopped never arrives, and a default-prevented key is visible.
- **Keys count only with their target in the thread.** The popup floats over other views, so an `ArrowUp` on the body can belong to the view underneath. If such a key does scroll the thread, the fallback stops following.
- **The scrollbar check uses a 16px right-edge zone.** The thread's empty margins and padding also have the scroller as target, so Elysium's "target is the scroller" rule would stop following on a click there. A fixed zone also covers overlay scrollbars, which report no width.
- **The rail uses `release()`, not an "app scroll" hold.** A hold needs an end signal for a smooth scroll, and `scrollTo` gives none.

## Known limits
- Inside `STICK_GRACE`, a scroll event turns following back on. The user chose this over Elysium's "stay released until the user leaves and returns", so a tiny trackpad move at the end can't stop following. As a result, an input release there lasts only until that event, and a slow scroll up during a fast reply can be pulled back until it passes 40px.
- Without anchoring, content above the view that changes height shifts the view, for example rewrapping on a resize. Any fix gets a new issue.
- Touch direction is not read, so a downward swipe mid-glide also stops following.
- A shrink and a growth of `scrollHeight` inside one frame can net to growth, and the guard misses that.
- With an overlay scrollbar, a click in the 16px right padding stops following.

## Tests
`useStickToBottom.dom.test.tsx` gives the scroller a fake layout with changeable sizes and asserts on following through a helper: grow the content, call `pin()`, and see whether the position moves. `ChatThread.dom.test.tsx` is new and covers the rail click through the real component. jsdom has no layout or real scrolling, so these tests can't show how a browser moves the position. That part rests on the user's tophat.

## Acceptance criteria
- [x] Wheel up, in the thread and outside an inner scroller, stops following before the position moves. Evidence: "wheel up stops following before the position moves", failed before the change.
- [x] Wheel up over an inner scroller that can scroll up doesn't stop following. Evidence: "wheel up over an inner scroller that takes the move keeps following".
- [x] Wheel down and pinch zoom don't stop following. Evidence: "wheel down and pinch zoom keep following".
- [x] `touchmove` stops following only outside `STICK_GRACE`. Evidence: "touch stops following only outside the grace zone", failed before the change.
- [x] `PageUp`, `Home` and `ArrowUp` in the thread stop following. A modifier, an editable target or a key a card handled doesn't. Evidence: the key tests in `useStickToBottom.dom.test.tsx`.
- [x] A press in the scroller's right-edge zone stops following. A press in its margin doesn't. Evidence: the two press tests in the same file.
- [x] An upward scroll with no input and unchanged sizes stops following outside `STICK_GRACE`. Evidence: "unsticks when the user scrolls up out of the grace zone" in `scroll.test.ts`.
- [x] An upward scroll with a shrunk `scrollHeight` or a grown `clientHeight` doesn't stop following. Evidence: `scroll.test.ts`, and "the glide's frame check keeps going" tests, which failed before the change.
- [x] A turn rail click stops following, and the glide doesn't pull the view back. Evidence: `ChatThread.dom.test.tsx` covers the release, and the user's tophat.
- [x] The thread scroller has `overflow-anchor: none`. Evidence: "turns off the browser's scroll anchoring on the thread scroller".
- [x] `follow()` still jumps on send. Evidence: the tests from xpo-ea5d6b still pass.
- [x] Wheel, touch, keys, scrollbar drag, drag-select and the turn rail each behave during a streaming reply, in the full view and the popup, with Motion on. Evidence: the user's tophat on a worktree build: "lgtm". The user didn't report each case separately.
- [x] Toggling a tool card while reading above the end doesn't shift the view. Evidence: the user's tophat, as above, not reported separately.
- [x] `make test` passes. Evidence: exit 0, 112 files and 1673 frontend tests. No code changed after that run, only `CHANGELOG.md`.
