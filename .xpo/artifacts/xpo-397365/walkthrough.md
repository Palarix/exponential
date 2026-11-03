# Walkthrough: thread stops following after each sent message (Motion on)

## The bug
`useStickToBottom` keeps a `stuck` flag. While it is true, new content scrolls the thread to the end. `nextStick` turns it off when a scroll event reports a position above the last one while more than `STICK_GRACE` (40px) from the end, and only scrolling back to the end turns it on again.

With Motion "On", `pin()` glides: each animation frame sets `scrollTop` to `easeToward(...)`, a fractional value, and stores it as `lastTop`. Chrome snaps the real position to device pixels, so the next scroll event can report a value a fraction *below* `lastTop`. The old check was `top >= prevTop` with no slack. When a new message arrived mid-glide, the end was more than 40px away, so that fraction read as an upward user scroll and following stopped for good. With Motion off, `pin()` jumps straight to `scrollHeight`, which is already at the end, so it never hit this. That's why only the tabs with Motion "On" showed it.

## The fix, in three layers
1. **Whole-pixel steps** (`utils/scroll.ts`, `easeToward`). Each step is `sign(distance) * max(1, round(|step|))`. The browser has nothing left to round, and the "at least 1px" rule that keeps the glide from stalling is kept.
2. **Rounding slack** (`nextStick`). An upward move only counts when it is 1px or more: `stuck && top > prevTop - 1`. This covers fractional positions that don't come from the glide, such as zoom or a fractional start position. A real wheel or key scroll moves far more than 1px.
3. **A user turn brings following back** (`useStickToBottom.follow()`, `ChatThread`). `follow()` sets `stuck = true` and calls `pin()`. `ChatThread` remembers the previous turn count in a ref. In its layout effect it calls `follow()` instead of `pin()` when `followsNewTurn(prevCount, turns)` holds and the thread is `live`. This is a safety net: whatever turned following off, sending a message turns it back on.

`followsNewTurn` (`chat-thread-utils.ts`) is a pure helper, so it can be tested without a DOM. It is true when turns were added and the last one isn't `byAgent`. User messages, queued and steered messages, and Work Mode continuations (`turn_continued`) all count. Agent-started turns (`agent_turn`, e.g. after a background task) don't, so they don't pull the user away from what they are reading. They still follow when the user is already at the end.

## Why `live` gates it
`live` turns true in an effect after the commit that brought the history in. That commit's layout effect still sees `false`, so opening a chat jumps to the end through the normal `pin()`, and only later turns can force following. Switching chats goes through an empty thread first: the turn count drops, so there is no false "new turn".

## Tests
- `scroll.test.ts`: glide steps are integers; the frame-rate test now allows the 1px rounding; `nextStick` keeps following on a 0.6px upward move and stops on a 2px one.
- `chat-thread-utils.test.ts`: `followsNewTurn` for an added user turn, no change or fewer turns, and an agent-started turn.
- `hooks/useStickToBottom.dom.test.tsx` (new, jsdom): fakes `clientHeight`, `scrollHeight` and a clamped `scrollTop`, and stubs `ResizeObserver`. Checks that `pin()` respects a user scroll-up and that `follow()` overrides it and stays stuck.

## Acceptance criteria
- [x] `easeToward` returns whole pixels, moves at least 1px, lands on the target. Evidence: `scroll.test.ts` "returns whole pixels…", "moves at least 1px…", "snaps to the target…".
- [x] `nextStick` tolerates sub-pixel rounding and still stops on a real upward scroll. Evidence: `scroll.test.ts` "stays stuck when the browser rounds…" and "unsticks when the user scrolls up…".
- [x] `follow()` resumes following after a scroll-up; `followsNewTurn` is true only for an added user turn. Evidence: `useStickToBottom.dom.test.tsx` and the `followsNewTurn` tests.
- [x] With Motion "On", sending messages follows each reply in the popup and full view. Evidence: the user checked it ("lgtm").
- [x] `make test` passes. Evidence: exit 0, 106 files and 1580 frontend tests, Go tests green.
