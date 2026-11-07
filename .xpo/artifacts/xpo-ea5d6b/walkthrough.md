# Walkthrough: thread doesn't go to the bottom after sending a message

## The bug
With Motion on or System (which resolves to full motion on the reporter's machine), a sent message sometimes left the thread where it was, and following stopped until the user scrolled to the end by hand. The failure was intermittent. With Motion off, the user could not trigger it on the latest build.

## Likely cause
This comes from reading the code and from a test that replays the sequence. It was never caught in a live failure, and what moves the position up is still unknown.

Sending calls `follow()`. Before this fix, `follow()` set `stuck` to true and called `pin()`. With motion allowed, `pin()` only schedules an animation frame, so nothing had moved when `follow()` returned, and `lastTop` kept its old value.

Browsers fire scroll events before animation frames. If an upward move had just happened, its event reached `onScroll` first. The position was still more than `STICK_GRACE` (40px) from the end and more than 1px above `lastTop`, so `nextStick` read it as the user scrolling up. `onScroll` then turned following off and cancelled the glide before its first frame.

With Motion off, `pin()` jumped in the same call. The waiting event then read a position at the end, where `nextStick` always keeps following. That explains why the setting mattered.

## The fix
`useStickToBottom` has a new `jump()` helper: it cancels any running glide, sets `scrollTop` to `scrollHeight`, and updates `lastTop`. `follow()` sets `stuck` to true and calls `jump()`. The no-glide path of `pin()` uses the same helper. `pin()` still glides.

`ChatThread` didn't change. It calls `follow()` in a layout effect, so the jump lands before paint and the user doesn't see the old position.

## What users notice
With Motion on, a sent message now jumps instead of gliding. So do a queued or steered message when its turn starts and a Work Mode continuation, because they share the `followsNewTurn` path. Output during the reply still glides. A turn the agent starts on its own still doesn't move a user who scrolled up.

## Tests
`useStickToBottom.dom.test.tsx` mocks `motionAllowed` to true and stubs animation frames, so each test controls when frames run. The regression test moves the position up without firing the event, calls `follow()`, then fires the event. It failed on the old hook and passes now.

## Known limit
A glide during a reply can still be stopped by an upward move the user didn't make. No such failure was reported in the tophat. xpo-7ffcf5 tracks hardening this, based on how Elysium releases following on user input.

## Acceptance criteria
- [x] With glide on and motion allowed, `follow()` sets `scrollTop` to the end in the same call. Evidence: "follow() goes to the end in the same call", failed before the fix.
- [x] `follow()` cancels a running glide. Evidence: "follow() cancels a running glide", failed before the fix.
- [x] With glide on and motion allowed, `pin()` still glides. Evidence: "pin() glides: it moves on later frames, not in the call".
- [x] A user who scrolled up and then sends goes to the end, and following resumes. Evidence: the existing `follow()` test, and the regression test. Covered in the manual check only as far as noted below.
- [x] With Motion on, a one-line send and a multi-line send both jump to the end and follow the reply, in the full view and the popup. Evidence: the user's tophat on a worktree build: "works with motion enabled, could not reproduce the reported issue with the new code". The user didn't report each case separately.
- [x] A turn the agent starts on its own does not move a user who scrolled up. Evidence: the existing `followsNewTurn` test in `chat-thread-utils.test.ts`.
- [x] `make test` passes. Evidence: exit 0, 111 files and 1640 frontend tests. Only `CHANGELOG.md` changed after that run.
