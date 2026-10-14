# Walkthrough: status indicator dots in toolbar chat chips

## What was built
Each chat chip in the status bar shows the chat's state as a small dot. Every recent-chats list uses the same indicator: the status-bar history menu, the Agent view chat switcher, and the Agent view home "Recent chats".

| State | Indicator |
|---|---|
| waiting (an approval card waits) | yellow dot, `--color-warning` |
| running | the existing border spinner |
| interrupted | red dot, `--color-error` |
| unseen (a reply finished that the viewer did not watch) | blue dot, `--color-info` |
| idle | nothing on chips; the menus keep their message-square icon |

The yellow `ShieldQuestion` icon is gone. It read as a security warning.

## How the pieces fit together

### Indicator logic: `agent-state-utils.ts`
- `chatIndicator(meta, seen)` returns one state, most urgent first: waiting, running, interrupted, unseen, idle.
- A chat is unseen when its `updated_at` is later than the seen record for it.
- A chat without a seen record counts as seen. Without this rule, every old chat lights up on first load.
- The function compares times with `Date.parse`, not as text. Go writes fractional seconds of varying length, so `.5Z` sorts after `.25Z` as text but is earlier in time. A unit test covers this case.
- `loadSeen`/`saveSeen` keep the records in localStorage under `xpo.agent.seen`. Bad or blocked storage gives `{}`.
- `markSeen` and `pruneSeen` return the same object when nothing changes. React state updates then do not cause extra renders.

### Rendering: `recent-chats.tsx`
`chatStateIcon(c, seen, size, idle)` maps the state to the spinner, a dot, or the `idle` fallback. The dot is 8px and sits centered in the `size` box, so chips and menu rows keep their alignment. Each dot has a `title` and an `aria-label`: "Waiting for your approval", "Interrupted", "New reply". `recentChatItems` takes `seen` and passes it through.

### State: `AgentChatProvider` and `agent-context.ts`
The provider holds `seen` and exposes `seen` and `markSeen(id, updatedAt)` on the context. After the chat list loads, it prunes records of chats that no longer exist, then saves.

### Watching: `useWindowAttention` and `useMarkSeen`
- `hooks/useWindowAttention.ts` is true while the tab is visible and the window has focus. It uses `useSyncExternalStore` over the window `focus`/`blur` events and `visibilitychange`. A focused terminal in front of the browser counts as no attention.
- `useMarkSeen(meta)` runs in `ChatPopup` and `AgentView`, the two places that show a chat. When the chat becomes watched, a 1.5s timer starts (`SEEN_GRACE_MS`). The chat becomes watched when it opens, or when the window gets attention back. When the timer ends, the chat is "settled", and each `updated_at` change marks it seen at once. A loss of attention or a chat switch cancels the timer and clears the settled state.

### Why `updated_at` works as the signal
The server bumps `updated_at` at turn start, turn end and mode changes (`internal/chat/runner.go`, `mode.go`). The chat list polls every 2s while any chat runs. The poll that sees the turn end therefore also gets the new `updated_at`. If nobody watches the chat then, it stays unseen.

## Decisions
- **Dots, not circle icons.** The user picked plain dots over issue-status-style circles. The spinner stays for running.
- **Grace delay of 1.5s.** The dot stays on screen for a moment after you come back, so you can see which chat finished. The user chose this as a starting point for UX evaluation. To tune it, change `SEEN_GRACE_MS`.
- **Seen is per browser.** localStorage is enough for a viewer convenience. It does not sync across devices.
- **No error state.** `ChatMeta` has no last-turn-error field. The follow-up is xpo-c56e70.

## Limits
- The code cannot know that you looked at the screen. A visible, focused window counts as watched.
- Pills from before this change have no seen record. They show no blue dot until the chat is watched once.

## Acceptance criteria
- [x] A chip shows a warning dot while an approval waits, with no shield icon anywhere. Evidence: `recent-chats.test.tsx` "shows a warning dot while an approval waits" checks for `--color-warning` and that no lucide icon renders. `ShieldQuestion` has no remaining imports.
- [x] A chip shows the spinner while running. Evidence: `recent-chats.test.tsx` "keeps the spinner while running".
- [x] A chip shows a red dot for an interrupted chat. Evidence: `recent-chats.test.tsx` "shows an error dot for an interrupted chat".
- [x] A reply that finishes while the chat is minimized shows a blue dot, and opening the chat clears it after about 1.5s. Evidence: `chatIndicator` unit tests; `useMarkSeen` grace timer; approved by the user after review.
- [x] A reply that finishes while the chat is open and the window is focused shows no dot. Evidence: `useMarkSeen` marks each `updated_at` at once after the chat settles; approved by the user after review.
- [x] A reply that finishes while the window is unfocused or hidden shows a blue dot, which clears about 1.5s after focus returns. Evidence: `useWindowAttention` gates `useMarkSeen`; approved by the user after review.
- [x] Idle, seen chats show no dot on chips. Evidence: `recent-chats.test.tsx` "shows the idle fallback otherwise"; the chips pass `null` as the fallback.
- [x] The recent-chats menus and the Agent view list use the same indicators. Evidence: `AgentStatusBar` and `AgentView` (switcher and `RecentChatsList`) all call `chatStateIcon` and `recentChatItems` with `seen`.
- [x] `make test` passes. Evidence: 67 test files / 911 frontend tests plus Go tests, exit 0.
