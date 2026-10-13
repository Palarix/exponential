# Agent chat: opening a chat shows its status-bar pill

## What

Every way of opening a chat in the popup adds its pill to the status bar, if it isn't there already.

## Why

`openChat` in `web/src/components/Agent/AgentChatProvider.tsx` sets the chat and opens the popup, but never adds a pill. Only `chatCreated` does. So pills exist only for chats created in this browser session. A chat opened any other way has no pill, and after you minimize the popup the only way back is a menu again. That affects:

- Recent chats in the status bar;
- Start Work or Continue Work reopening an existing chat (xpo-00f0e5);
- the chat pill on issue rows, the issue chats list and "Open work chat" (xpo-219de9).

## Fix

1. `openChat` calls `addPill`, as `chatCreated` does. `addPill` already ignores a chat that has a pill.
2. The popup opens centered above its anchor instead of the bottom-right corner: the open chat's pill, or the Agent button for a new chat. It stays inside the window with a 12px margin, and follows the anchor when the window resizes or the pills change.
3. The popup is resizable. Its bottom stays on the anchor and it stays centered, so it grows upward and to both sides. Handles: the top edge (height), the left and right edges (width), the top corners (both). Size limits: 360×400 to 600×900, and never larger than the window. The size is remembered in browser storage and applies to every popup.

## Acceptance Criteria

- [ ] Opening a chat from Recent chats shows its pill in the status bar.
- [ ] Start Work or Continue Work that reopens an existing chat shows its pill.
- [ ] Opening a chat from an issue's chat pill, the chats list or "Open work chat" shows its pill.
- [ ] Opening a chat that already has a pill doesn't add a second one.
- [ ] Closing the popup still removes the open chat's pill; minimizing keeps it.
- [ ] The popup opens centered above the open chat's pill, or above the Agent button for a new chat.
- [ ] Near a window edge, the popup shifts to stay 12px inside the window.
- [ ] Dragging the top edge, a side edge or a top corner resizes the popup, from 360×400 up to 600×900 (width×height), never beyond the window.
- [ ] The popup keeps its bottom on the anchor and stays centered while resizing.
- [ ] The size is remembered across popups and reloads; it opens at 460×600 by default.
- [ ] `make test` passes.

## Decisions

- **Anchoring the popup to its pill.** With several pills, a popup in the corner doesn't say which chat it belongs to. Centering it over its pill does, like a menu over its button.
- **Resize grows from the anchor.** The popup is anchored at the bottom and centered, so a side drag grows it on both sides (twice the drag distance), keeping the dragged edge under the pointer.
- **Size in browser storage.** It is a per-viewer convenience, like the pills, not chat data.
- **Tested in the browser, plus unit tests for the pure parts.** `addPill` is already tested for deduplication; the popup's horizontal position, the size clamp and the stored size are pure functions with their own tests. The web tests render static markup and have no DOM environment, so the provider's actions and the measuring can't be driven from a test.
