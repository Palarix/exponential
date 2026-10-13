# Walkthrough: maximized chat leaves the bottom toolbar; dock button returns it

## What changed
- **`AgentChat.maximize()`** (`agent-context.ts`, `AgentChatProvider.tsx`): closes the popup and drops the chat's pill. It shares `closeChat` with `close()`. `ChatPopup`'s maximize button now calls it instead of `minimize()`, so the pill no longer stays behind.
- **Dock button** (`AgentView.tsx`): the Agent view header shows a lucide `Dock` button (`aria-label="Dock chat"`) at the right for an existing chat. Clicking it calls `agent.openChat(chatId)`, which adds the pill back and opens the popup, then calls `leaveChatView()`.
- **`AppNav.leaveChatView()`** (`app/contexts.ts`, `App.tsx`): App keeps the last route outside the Agent view, and this function goes back to it. Without one, as on a direct `#/agent/...` load, it goes to the default route (dashboard). Otherwise the popup would open on top of the same chat in fullscreen.

## Decisions
- The new-chat home (`chatId` null) shows no dock button: there is no chat yet to dock.
- There is no tooltip on the dock button. Other chat header buttons (popup, title bar) have none either.

## Acceptance criteria
- [x] Maximizing a chat popup removes it from the bottom toolbar: `maximize()` removes the pill (`AgentChatProvider.tsx`).
- [x] Maximized chats show a lucide `dock` button at the right of the top bar. Covered by `AgentView.test.tsx` ("offers a dock button in the header for a chat").
- [x] Clicking dock puts the chat back on the bottom toolbar as a popup: `openChat(chatId)` adds the pill and opens the popup, and `leaveChatView()` returns to the previous view.

## Tests
- `web/src/components/Agent/AgentView.test.tsx` (new): the dock button is there for a chat and missing on the new-chat home.
- `make test`: green.
