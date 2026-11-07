# Agent chat: thread doesn't go to the bottom or keep following after sending a message

## What
Sometimes, after the user sends a message, the thread does not go to the bottom and following stops. The failure is intermittent. It occurs in regular chat sessions on all boards, on the latest binary.

This change makes a sent message jump to the end at once, also with Motion on. New output during the reply still glides.

## Why
The user must scroll by hand to see the reply.

The root cause is not confirmed. The evidence points to the glide. With Motion "Off", the user has not been able to trigger the failure on the latest build. That is suggestive, not proof, because the failure is intermittent.

A jump lands at the end at once, within `STICK_GRACE`, where `nextStick` always keeps following. A glide spends several frames more than 40px from the end. During those frames, `nextStick` stops following after any upward move of 1px or more, including a move the user did not make. A jump on send removes this window at the moment the failure occurs.

## How
Change `follow()` in `web/src/hooks/useStickToBottom.ts`:
1. Set `stuck` to true.
2. Call `stop()` to cancel a running glide.
3. Set `scrollTop` to `scrollHeight`.
4. Set `lastTop` to the new `scrollTop`.

`follow()` no longer calls `pin()`. `pin()` does not change, so new output and a resized scroller still glide.

`ChatThread` does not change. It calls `follow()` when `followsNewTurn` is true and the thread is `live`. The layout effect runs before paint, so the user does not see the old position.

Update the doc comment of `useStickToBottom` to say that `follow()` jumps.

## Decisions
- Jump on send, not a glide. The user asked for this. A send is a change of context, so the animation adds nothing.
- Every turn that `followsNewTurn` accepts jumps: a sent message, a queued or steered message when its turn starts, and a Work Mode continuation (`turn_continued`). They share one path, and all of them already force following.
- Do not change `nextStick` in this issue. Input-based detection of user scrolls is a larger change, and the cause is not confirmed.
- No logging build. If the failure returns, add the logging then.

## Known limit
The glide window remains for new output during a reply. A chunk that puts the end more than 40px away, plus an upward move the user did not make, can still stop following. If this failure occurs after this fix, the next step is input-based detection in `nextStick`. Record the logging and that change in a new issue.

## Acceptance criteria
- [ ] With glide on and motion allowed, `follow()` sets `scrollTop` to the end in the same call. Evidence: a new test in `useStickToBottom.dom.test.tsx`, written first.
- [ ] `follow()` cancels a running glide. Evidence: the same test file.
- [ ] With glide on and motion allowed, `pin()` still glides. Evidence: the same test file.
- [ ] A user who scrolled up and then sends goes to the end, and following resumes. Evidence: the existing `follow()` test, plus a manual check.
- [ ] With Motion on, a one-line send and a multi-line send both jump to the end and follow the reply, in the full view and the popup. Evidence: the user checks this in normal use.
- [ ] A turn the agent starts on its own does not move a user who scrolled up. Evidence: the existing `followsNewTurn` test.
- [ ] `make test` passes.

## Assumptions
- The user keeps Motion on after the fix lands, so that normal use tests it.
