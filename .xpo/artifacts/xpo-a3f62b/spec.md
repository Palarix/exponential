# Agent chat: browser notifications when an agent waits on approval

## What
An opt-in desktop notification when an agent chat starts waiting on you (an approval or a question) while the xpo tab is hidden. One notification per chat, replaced rather than stacked, and closed once it no longer applies.

## Why
The favicon dot from xpo-bbcab4 only helps when you can see the tab strip. When you work in another app, a blocked agent sits idle until you happen to look. Nocchi covers this for people who run it; this is for everyone else. Notifications are noisy, so this one is off by default and fires only when the agent is stuck on you.

## Behavior
- **Off by default.** The board never asks for notification permission on page load.
- **Opt in** with the bell in the Agent view header. Turning it on calls `Notification.requestPermission()` from that click. If the browser denies, the toggle stays off and says notifications are blocked in the browser's settings.
- **Fires when** a chat goes from not waiting to `waiting` (approval card or question card), and the tab is hidden. Edge-triggered: a chat that is already waiting when the page loads, or when you turn notifications on, does not fire.
- **Doesn't fire** for turns that finish, fail, or are interrupted. The favicon dot covers those.
- **One per chat:** `tag: "xpo-chat-<id>"`. A new approval in the same chat replaces the old notification. Several xpo tabs on the same origin share the tag, so they don't stack.
- **Content:**
  - title: `Agent needs your input` (the `waiting` flag covers approvals and questions alike)
  - body: the chat title
  - icon: the favicon
  - no custom styling (the OS draws it)
- **Click:** focuses the tab, opens the chat in the popup (`openChat(id)`), and closes the notification.
- **Closes itself** when the chat stops waiting (approved elsewhere) or the tab becomes visible.
- **Unsupported browsers** (no `Notification` in `window`): the toggle is hidden.

## How
- `agent-state-utils.ts` (pure, unit-tested):
  - `notificationPlan(prev, next, hidden)` returns `{ show, close }`:
    - `show`: chats that wait in `next` but did not in `prev`, but only while the tab is hidden.
    - `close`: IDs that waited in `prev` and no longer wait, or are gone.
  - `notificationFor(chat)` returns the title and options (body, tag, icon).
  - `notifyState(on, permission)` returns `on | off | blocked | unsupported`.
  - `loadNotify` / `saveNotify` store a boolean under `xpo.agent.notify`.
- `useApprovalNotifications(chats | null, enabled, openChat)`:
  - Keeps the previous chat list in a ref and applies the plan on each change.
  - Notifies only when enabled and permission is `granted`.
  - Keeps the open `Notification`s by chat ID, and closes them all on `visibilitychange` to visible.
- `AgentChatProvider` owns the setting and the permission state, and exposes `notify` / `toggleNotify` on the agent context. It calls the hook with `null` until the chat list query has loaded, so waits that exist at page load don't fire.
- `NotifyBell` (presentational, markup-tested) sits in the Agent view header next to "All chats":
  - on: `Bell`, pressed
  - off: `BellOff`
  - blocked: disabled, with a tooltip
  - unsupported: not rendered
- A hidden tab sees `waiting` change because the store's `OnChange` fires when an approval card opens or closes. That sends `chat_changed` over SSE, which refetches the chat list.

## Edge cases
- First load: `prev` starts as the first loaded chat list, so existing waits don't fire.
- Permission revoked in the browser later: the hook checks `Notification.permission` on every event and does nothing unless it's `granted`. The toggle shows it as blocked.
- `requestPermission` with no user gesture is rejected by browsers. We only call it from the toggle's click.

## Decisions
- **Toggle:** a bell button in the Agent view header, with three states: on, off, and blocked (permission denied in the browser). Hidden when the browser has no `Notification` API.

## Out of scope
- Notifications for finished turns.
- Action buttons ("Approve" in the notification). They need a service worker and work in Chrome only.
- Per-project settings. The toggle is per browser (localStorage).

## Acceptance criteria
- [ ] Nothing asks for permission until the user turns the toggle on.
- [ ] With the toggle on and the tab hidden, a chat that starts waiting shows one notification with the chat title.
- [ ] A second wait in the same chat replaces the notification instead of adding one.
- [ ] No notification while the tab is visible, or for waits that existed before.
- [ ] Clicking it focuses the tab and opens that chat.
- [ ] It closes when the chat stops waiting or the tab becomes visible.
- [ ] Denied or unsupported permission is handled without errors.
- [ ] Unit tests cover `notificationPlan`, `notificationFor`, `notifyState`, the notify setting, and the bell's markup.
- [ ] `make test` passes.
