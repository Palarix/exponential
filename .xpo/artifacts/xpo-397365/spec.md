# Agent chat: with Motion on, the thread stops following after each sent message

## What
With Motion "On", the chat thread stops following new output after you send a message. You have to scroll to the end by hand to get following back. Fix the glide so it never turns following off by itself, and make a new user message always turn following back on.

## Why
`useStickToBottom` (xpo-63b755, glide added in xpo-50b9d9) turns following off when a scroll event reports a position above the last known one while more than `STICK_GRACE` (40px) from the end (`nextStick`).

The glide (`easeToward`) sets fractional `scrollTop` values and records them as `lastTop`. Chrome snaps the real position to device pixels, so the scroll event that follows can report a value a fraction below `lastTop`. Mid-glide, a new message puts the end more than 40px away, so `top >= prevTop` fails and following turns off. The jump path (Motion off) sets `scrollTop = scrollHeight`, which is already at the end, so it never hits this.

Separately, nothing turns following back on except scrolling to the end. Sending a message is a clear sign the user wants to see the reply.

## How
1. **Whole-pixel glide steps.** `easeToward` moves in whole pixels: it rounds each step and still moves at least 1px and lands on the target.
2. **Rounding slack in `nextStick`.** An upward move of less than 1px is not a user scroll: `stuck && top > prevTop - 1`. A real wheel or keyboard scroll moves far more than 1px.
3. **Follow on a new user turn.** The hook exposes `follow()`: it sets `stuck = true`, then calls `pin()`. `ChatThread` calls it in its layout effect when `followsNewTurn(prevCount, turns)` holds after the history has loaded: a turn was added, and it isn't one the agent started on its own (`byAgent`). User turns are sent, queued, or steered messages, and Work Mode continuations (`turn_continued`). The pin still glides when Motion is on.

## Edge cases
- First load of the history (`live` false): no forced follow. The initial pin already jumps to the end.
- A user turn added while the user reads further up: the view goes to the end. The user just sent a message, so that is intended.
- A turn the agent starts on its own (`agent_turn`, e.g. after a background task) does not force following, so it doesn't pull the user away from what they are reading. It still follows if they are at the end.
- Editing the last message doesn't add a turn and doesn't force following.

## Acceptance criteria
- [ ] `easeToward` returns whole pixels when given whole-pixel `top` and `target`, still moves at least 1px per step, and lands on the target (unit tests).
- [ ] `nextStick` keeps following on an upward move under 1px away from the end, and still stops following on a real upward scroll (unit tests).
- [ ] `follow()` resumes following and goes to the end after the user scrolled up (hook test). `followsNewTurn` is true only for an added turn the user started (unit tests).
- [ ] With Motion "On", sending several messages in a row in the popup and the full view follows each reply without manual scrolling (manual check by the user).
- [ ] `make test` passes.
