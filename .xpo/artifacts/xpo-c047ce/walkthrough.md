# Walkthrough: Agent view with full-page chat and chat history

## What we built

Agent chat now has a home in the sidebar. **Agent** is the first entry of the personal section, and `g a` opens it. The view has three states:

- **Empty chat:** a heading, a wide composer, three suggestion cards, and the 5 most recent chats.
- **Chat with messages:** the thread in a centered column, with the composer at the bottom.
- **Unknown chat ID:** "Chat not found", with a button to start a new chat.

The chat ID is part of the URL (`#/agent/<id>`). The header's switcher ("Title ⌄") lists recent chats. A "⋯" menu renames or deletes a chat, in the view and in the popup. The popup's expand button opens its chat in the view. A history icon in the status bar reopens recent chats in the popup. The board deletes old chats when it starts.

This is the second slice of Pillar 1 of xpo-3798b7. It is built on the engine from xpo-56ae3a.

## How the pieces fit

### Backend

- `internal/chat/store.go`:
  - `Delete` removes both chat files and forgets the cached `seq`.
  - `Rename` cleans the title with `TitleFrom` (first line, at most 60 characters). A blank title returns `ErrEmptyTitle`.
  - `Prune(maxAge, maxCount, now)` walks the chats newest first. It keeps running chats always. It deletes a chat when the chat is older than `maxAge`, or when `maxCount` chats are already kept. The constants are `PruneMaxAge` (30 days) and `PruneMaxCount` (200).
- `internal/chat/runner.go`: every turn now has a `done` channel, which `finish` closes. `CancelAndWait` cancels the turn and waits on that channel. When it returns, the turn has written its last event and saved its meta.
- `internal/server/chat.go`: two new routes.
  - `PATCH /api/local/chats/{id}` renames a chat.
  - `DELETE /api/local/chats/{id}` calls `CancelAndWait` before `Store.Delete`. Without that wait, a finishing turn could recreate the files that the delete just removed.
- `cmd/exponential/board.go`: `newChatService` prunes after `MarkInterrupted` and logs the count.

### Frontend

- **Routing:** `views.ts` has the `agent` view (`requires: "agent"`) and `Route.chatId`. `parseRoute` and `formatRoute` handle `#/agent/<id>`. `AppNav.openChatView(id | null)` navigates there. `Sidebar.tsx` hides the entry unless `useAgentChat().status.available` is true.
- **`useChatSession`:** returns the thread, the `missing` flag, `running`, `send(text, context)`, and `stop()`. It is extracted from the popup, so the popup and the view share it. `useChatThread` now sets `missing` when the server answers 404.
- **`AgentView.tsx`:** contains the `ChatSwitcher`, the `RecentChatsList` (the 5 newest chats on the empty state), and the empty, thread, and not-found layouts. The composer and the thread have a `wide` variant.
- **`ChatTitleBar.tsx`:** the title plus the "⋯" menu. Rename swaps the title for an input: Enter saves, and Escape or blur cancels. Delete asks in a `ConfirmDialog`, then calls `chatDeleted(id)` on the provider. That drops the pill and resets the popup if it showed the chat. The view's `onDeleted` opens `#/agent`.
- **`recent-chats.tsx`:** `recentChatItems` builds the menu items for the switcher and the status-bar history. Each item shows a spinner for a running chat and a relative time.
- **`ChatPopup.tsx`:** uses `useChatSession` and `ChatTitleBar`. The new expand button calls `openChatView(chatId)`, then `minimize()`. The pill stays.
- **`AgentStatusBar.tsx`:** the history button and its popover.
- **`ui/Menu.tsx`:** `MenuItem`'s right-hand slot changed from `w-4` to `min-w-4 whitespace-nowrap`. Suffixes such as "31m ago", filter counts, and "⌘⌫" used to overflow the 16px slot, which widened the row past the menu.

## Key decisions

- **The chat ID goes in the URL.** Reload, back/forward, and links behave like issue routes.
- **One session hook and one set of components serve the popup and the view.** Only the layout differs.
- **Deletes notify the provider directly.** The UI does not watch the chat list for chats that disappear. A chat created a moment ago is not in the list until the list refetches, so watching the list would reset it.
- **Pruning runs on board start, not on a timer.** It never runs while someone browses their history.
- **The Agent entry is hidden without chat**, like Cycles and Drafts.

## Bugs fixed during review

- **The header title was cut to "New …".** The switcher button had `max-w-full` inside a flex item that sizes itself to its content. The browser resolved that cyclic percentage too small. `min-w-0` replaces it.
- **The switcher overflowed sideways.** This was the `MenuItem` slot described above.
- **Tool calls looked broken:** cut-off arguments, raw commands, and misaligned bullets. These were filed as xpo-6afc58 and fixed on this branch; see that walkthrough.

## Acceptance criteria

- [x] The sidebar shows **Agent** as the first entry of the personal section, above My Issues, and a click opens `#/agent`. *Evidence: screenshot `8-home`. `views.test.ts` checks the sidebar order.*
- [x] `g a` opens the Agent view. *Evidence: Playwright pressed `g` then `a` on the board and landed on `#/agent`. `views.test.ts` checks the go keys.*
- [x] When chat is unavailable, the sidebar hides Agent, and `#/agent` shows the reason. *Evidence: the `visible()` switch in `Sidebar.tsx` and the unavailable branch in `AgentView`. The `requires` mapping is tested.*
- [x] An empty Agent view shows a centered composer and three suggestion cards. *Evidence: screenshot `1-empty`.*
- [x] A click on a card sends its prompt. *Evidence: the cards call `sendText(s.prompt)`, the same path as the composer.*
- [x] Below the cards, the 5 most recent chats are listed, a click opens one, and the list is hidden without chats. *Evidence: screenshot `8-home`. Playwright opened a chat from the list.*
- [x] The first message creates the chat, and the URL becomes `#/agent/<chat-id>`. *Evidence: Playwright waited for the `#/agent/<uuid>` hash after sending.*
- [x] `#/agent/<chat-id>` shows the thread, and a reload keeps it. *Evidence: Playwright reported "reload keeps chat: true".*
- [x] The header shows the full title, truncates only when it is out of room, and opens the switcher. *Evidence: screenshots `8-home` ("New chat") and `9-long-title`.*
- [x] The switcher lists "New chat" plus the 20 most recent chats, with relative times and running spinners. *Evidence: screenshot `10-switcher`. `RECENT_CHAT_LIMIT` is 20.*
- [x] Picking a chat in the switcher opens it in the view. *Evidence: Playwright picked "Summarize this issue…" and the URL changed.*
- [x] The "⋯" menu in the view and in the popup has Rename and Delete. *Evidence: both headers use `ChatTitleBar`. Screenshot `7-popup-title`.*
- [x] Rename works inline: Enter saves, Escape cancels, and an empty title is not saved. *Evidence: Playwright renamed a chat to "Renamed chat". `TestStore_Rename` and `TestChat_Rename` cover the empty case.*
- [x] Delete confirms, removes the files, drops the pill, and leaves the deleted chat. *Evidence: Playwright deleted a chat, landed on `#/agent`, and found 0 pills. `TestStore_Delete` covers the files.*
- [x] Deleting a running chat stops the turn first. *Evidence: `TestChat_DeleteStopsRunningTurn`, which also checks that the finished turn does not recreate the chat.*
- [x] The popup's expand button opens the chat in the view and minimizes the popup, and the pill stays. *Evidence: Playwright logged "expand -> #/agent/<uuid>, popup: 0".*
- [x] The status-bar history lists recent chats and opens one in the popup. *Evidence: screenshot `4-history`. Playwright logged "popup open: 1".*
- [x] On start, the board deletes idle chats older than 30 days and keeps at most 200. *Evidence: `TestStore_Prune` and `TestStore_PruneKeepsMaxCount`, plus the call in `newChatService`.*
- [x] `make test` passes, with Go tests for delete, rename, and pruning, and frontend tests for routing and helpers. *Evidence: `make test` exits 0. The new frontend tests are in `views.test.ts` and `chat-thread-utils.test.ts`.*

## Things to know

- When the Agent view opens a chat, it does not restore that chat's saved model and effort, as the popup does. The pickers keep the last selection.
- The status-bar history only opens chats. Delete lives in the "⋯" menus.
