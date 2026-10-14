# Agent chat: status indicator dots in toolbar chat chips

## What
Each chat chip in the status bar shows the chat's state at a glance. The same indicator is used in every recent-chats list (status-bar history menu, Agent view chat switcher, Agent view home "Recent chats").

## Why
The chips today show a spinner while running, a yellow `ShieldQuestion` when an approval waits, and nothing otherwise. The shield reads as a security warning, interrupted chats look idle, and there is no way to tell that a chat finished a reply you have not read.

## Behavior

States, most urgent first (a chat shows only the first that applies):

| State | Condition | Indicator |
|---|---|---|
| waiting | `meta.waiting` | warning dot (`--color-warning`) |
| running | `meta.state === "running"` | the current border spinner (unchanged) |
| interrupted | `meta.state === "interrupted"` | error dot (`--color-error`) |
| unseen | idle and `meta.updated_at` is newer than when the viewer last saw the chat | blue dot (`--color-info`) |
| idle | otherwise | nothing on chips; the menus keep their message-square icon |

Each dot has a `title` / `aria-label` ("Waiting for your approval", "Interrupted", "New reply").

### Seen tracking
- The provider keeps `seen: Record<chatId, updated_at>` in localStorage (`xpo.agent.seen`). It is a per-browser convenience.
- A chat is **watched** while all of these hold:
  - it is shown, either open in the popup or the chat in the Agent view
  - the tab is visible (`document.visibilityState === "visible"`)
  - the window has focus (`document.hasFocus()`, plus the `focus`/`blur` events)
- While a chat is watched, its seen value follows its `updated_at`. A reply that finishes while you watch never shows a dot.
- A reply that finishes while the window is unfocused or hidden shows the blue dot, even on the open chat.
- **Grace delay:** when a chat becomes watched (focus returns, the tab becomes visible, or the chat is opened), it is marked seen after 1.5s, not right away. That way you see which chat finished. Losing the watched condition before the delay ends cancels it. This is a starting point for evaluating the UX.
- A chat without a seen record counts as seen. Otherwise every old chat would light up on first load.
- Records for chats that no longer exist are pruned once the chat list loads.
- The server bumps `updated_at` at turn start, turn end and mode changes. A turn that ends while the chat is not watched therefore marks it unseen.

## How
- `agent-state-utils.ts`: `loadSeen` / `saveSeen`, `markSeen(seen, id, updatedAt)`, `pruneSeen(seen, ids)`, and `chatIndicator(meta, seen): "waiting" | "running" | "interrupted" | "unseen" | "idle"`, all pure and unit-tested.
- `useWindowAttention()` hook: true while the tab is visible and the window is focused.
- `AgentChatProvider`: holds `seen` and exposes `seen` and `markSeen(id, updatedAt)` on the context.
- `useMarkSeen(meta)` hook, used by `ChatPopup` and `AgentView`: while the window has attention and the chat is shown, it marks the chat seen 1.5s after it first became watched. Once watched, a later `updated_at` change marks it seen right away.
- `recent-chats.tsx`: `chatStateIcon` uses `chatIndicator` and renders dots in place of the shield.

## Out of scope
- An "error" state. `ChatMeta` has no last-turn-error field; that needs a follow-up issue.
- The issue-row chat icons (`IssueChatStateIcon`) are unchanged.

## Acceptance criteria
- [ ] A chip shows a warning dot while an approval waits, with no shield icon anywhere.
- [ ] A chip shows the spinner while running.
- [ ] A chip shows a red dot for an interrupted chat.
- [ ] A reply that finishes while the chat is minimized shows a blue dot, and opening the chat clears it after about 1.5s.
- [ ] A reply that finishes while the chat is open and the window is focused shows no dot.
- [ ] A reply that finishes while the window is unfocused or hidden shows a blue dot, which clears about 1.5s after focus returns.
- [ ] Idle, seen chats show no dot on chips.
- [ ] The recent-chats menus and the Agent view list use the same indicators.
- [ ] `make test` passes.
