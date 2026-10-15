# Walkthrough: browser notifications when an agent waits on you

## What was built
An opt-in desktop notification for one event only: an agent chat starts waiting on the user (an approval card or a question card) while the xpo tab is hidden. It is for people who don't run Nocchi. Notifications are noisy, so the design keeps them rare:
- off by default;
- only for waits;
- one per chat;
- closed as soon as they no longer apply.

## How the pieces fit
1. **The server already signals the event.** The chat store's `OnChange` fires when an approval or question card opens or closes. The board turns that into a `chat_changed` SSE event, and the browser refetches the chat list. `ChatMeta.waiting` covers both card kinds. A hidden tab therefore sees `waiting` flip with no polling.
2. **Pure rules** (`agent-state-utils.ts`):
   - `notificationPlan(prev, next, hidden)` diffs two chat lists:
     - `show`: chats that wait now but didn't before, and only while hidden;
     - `close`: chats that stopped waiting or are gone.
   - `notificationFor(chat)` builds the content. The tag is `xpo-chat-<id>`, so the browser replaces rather than stacks, across xpo tabs on the same origin too.
   - `notifyState(on, permission)` folds the user's setting and the browser's permission into `on | off | blocked | unsupported`.
   - `loadNotify` / `saveNotify` keep the setting in localStorage (`xpo.agent.notify`).
3. **`useApprovalNotifications`** applies the plan:
   - It keeps the last chat list in a ref, and the open `Notification` objects by chat ID so it can close them.
   - On `visibilitychange` to visible, it closes them all.
   - A click focuses the window and calls `openChat(id)`, which opens the popup.
4. **`AgentChatProvider`** owns the setting and the permission state, exposes `notify` / `toggleNotify` on the agent context, and calls the hook.
5. **`NotifyBell`** is a presentational button in the Agent view header:
   - on: `Bell`, `aria-pressed`
   - off: `BellOff`
   - blocked: disabled, with a tooltip pointing to the browser's site settings
   - unsupported: not rendered

## Decisions worth knowing
- **The hook lives in the provider, not `TabStatus`.** The spec first put it in TabStatus. Only the provider knows when the chat list query has loaded. The hook gets `null` until then, so the first real list seeds `prev`. Without that, the empty list before loading would make every existing wait look new and fire on page load.
- **Permission is requested only from the bell's click.** Browsers reject `requestPermission` without a user gesture, and asking on load is what makes notifications feel hostile. If the browser denies, the setting stays off and the bell shows blocked.
- **The setting and the permission are separate.** A user can turn the bell off while permission stays granted, and turn it back on without being asked again.
- **No custom styling or action buttons.** The OS draws notifications. Buttons such as "Approve" need a service worker and work in Chrome only, so they are out of scope.
- **Finished turns don't notify.** The favicon dot from xpo-bbcab4 covers them. Only a blocked agent is worth interrupting someone for.
- **Testing:** the frontend tests render static markup and have no DOM. All decisions therefore sit in pure functions, and the hook only applies them. A real notification was checked by hand in the browser.

## Acceptance criteria
- [x] Nothing asks for permission until the user turns the toggle on. `requestPermission` is called only in `toggleNotify`, from the bell's click.
- [x] With the toggle on and the tab hidden, a chat that starts waiting shows one notification with the chat title. Covered by the `notificationPlan` "shows chats that start waiting while the tab is hidden" test and the `notificationFor` content test. Tried by the user.
- [x] A second wait in the same chat replaces the notification instead of adding one: the per-chat tag `xpo-chat-<id>`, asserted in the `notificationFor` test.
- [x] No notification while the tab is visible, or for waits that existed before. Covered by the "skips new waits while the tab is visible" and "skips waits that already existed" tests. Page-load waits are skipped because the hook gets `null` until the list loads.
- [x] Clicking it focuses the tab and opens that chat: `onclick` calls `window.focus()` and `openChat(id)`.
- [x] It closes when the chat stops waiting or the tab becomes visible. The "closes chats that stop waiting or are gone" test covers the first case; the hook's `visibilitychange` listener covers the second.
- [x] Denied or unsupported permission is handled without errors. Covered by the `notifyState` tests (blocked, unsupported) and the `NotifyBell` markup tests (disabled with a tooltip, not rendered). The hook checks for `Notification` before using it.
- [x] Unit tests cover `notificationPlan`, `notificationFor`, `notifyState`, the notify setting, and the bell's markup: `agent-state-utils.test.ts` and `NotifyBell.test.tsx`.
- [x] `make test` passes: 1154 frontend tests, lint, and the Go tests.
