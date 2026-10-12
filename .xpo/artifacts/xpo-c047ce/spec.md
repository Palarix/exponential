# Spec: Agent view with full-page chat and chat history

## What

The sidebar gets an **Agent** entry (`#/agent`, `g a`). It opens a full-page chat view modelled on Linear's Agent view:

- A header with a chat switcher ("New chat ⌄"). It lists recent chats.
- A large centered composer with suggestion cards when the chat is empty.
- The same thread rendering as the popup when the chat has messages.

The user can rename and delete chats from the view and from the popup. The popup gets its **expand** button, which opens the chat in the Agent view. A history icon right of the Agent button lists recent chats. The board prunes old chats when it starts.

## Why

This is the second slice of Pillar 1 of xpo-3798b7. The popup from xpo-56ae3a suits quick questions, but longer conversations need room, and old chats need a way back. The epic says chats are disposable, so `.xpo/chats/` must not grow forever.

## Acceptance Criteria

- [ ] The sidebar shows **Agent** (lucide `MousePointer2`) as the first entry of the personal section, above My Issues. A click opens `#/agent`.
- [ ] `g a` opens the Agent view.
- [ ] When chat is not available, the sidebar hides the Agent entry, and `#/agent` shows the unavailable reason.
- [ ] An empty Agent view shows a centered composer and three suggestion cards (icon, title, one-line description).
- [ ] A click on a card sends its prompt.
- [ ] Below the cards, the empty Agent view lists the 5 most recent chats (title, relative time, running spinner). A click opens the chat. The list is hidden when there are no chats.
- [ ] Sending the first message creates the chat, and the URL changes to `#/agent/<chat-id>`.
- [ ] `#/agent/<chat-id>` shows that chat's thread, with the composer at the bottom. A page reload keeps the chat.
- [ ] The header shows the full chat title (or "New chat") with a chevron. The title truncates only when the header has no room left. A click opens the chat switcher.
- [ ] The switcher lists "New chat" and the 20 most recent chats, each with its title and relative time. A running chat shows a spinner.
- [ ] Picking a chat in the switcher opens it in the Agent view.
- [ ] A "⋯" menu in the Agent view header and in the popup header has **Rename** and **Delete**.
- [ ] Rename edits the title inline. Enter saves. Escape cancels. An empty title is not saved.
- [ ] Delete asks for confirmation, then removes the chat files. Its pill disappears. A view or popup that showed it switches to a new chat.
- [ ] Delete of a running chat stops the turn first.
- [ ] The popup header shows an **expand** button. It opens the popup's chat in the Agent view and minimizes the popup. The pill stays.
- [ ] In the status bar, a history icon right of the Agent button opens a list of recent chats. Picking one opens it in the popup.
- [ ] On start, the board deletes idle chats not updated for 30 days. It keeps at most the 200 most recent chats.
- [ ] `make test` passes, with Go tests for delete, rename, and pruning, and frontend tests for the new route parsing and helpers.

## Flow

### Backend

1. Add `Store.Delete(id)`. It removes `<id>.json` and `<id>.jsonl` and forgets the cached `seq`. It returns `ErrNotFound` for an unknown ID.
2. Add `Store.Rename(id, title)`. It trims the title and cuts it to 60 characters with `TitleFrom`. An empty title returns `ErrEmptyTitle`.
3. Add `Store.Prune(maxAge, maxCount, now)`. It deletes `idle` and `interrupted` chats older than `maxAge`. Then it deletes the oldest such chats beyond `maxCount`. It never deletes a `running` chat. It returns the number deleted.
4. Add routes in `internal/server/chat.go`:
   - `DELETE /api/local/chats/{id}`: call `Runner.Cancel`, wait for the turn to end, then call `Store.Delete`. Return 204.
   - `PATCH /api/local/chats/{id}` with `{title}`: call `Store.Rename`. Return the meta. Return 400 for an empty title.
5. Add `Runner.CancelAndWait(id)`, which waits until the chat's turn has finished. The delete route uses it, so no turn writes to a deleted log.
6. In `newChatService()` (`cmd/exponential/board.go`), call `Store.Prune(30*24h, 200, time.Now())` after `MarkInterrupted`. Log the count when it is above zero.

### Frontend

7. Add `"agent"` to `ViewId` and `VIEWS`: route `agent`, title "Agent", nav label "Agent", go key `a`, section `personal` as its first entry (above My Issues), `requires: "agent"`.
8. Add `chatId` to `Route`. `parseRoute("#/agent/<id>")` returns `{view: "agent", chatId}`. `formatRoute` writes it back. Update `views.test.ts`.
9. In `Sidebar.tsx`, show `requires: "agent"` views only when the agent status is available. Read the status with `useAgentChat()`.
10. Add `openChatView(chatId | null)` to `AppNav`. It navigates to `#/agent` or `#/agent/<id>`.
11. Extract `useChatSession(chatId, onCreated)` from `ChatPopup`. It returns `thread`, `missing`, `running`, `send(text, context)`, and `stop()`. The popup and the Agent view both use it. `useChatThread` reports `missing` for a chat ID that returns 404.
12. Add `web/src/components/Agent/AgentView.tsx`.
    - Header: `ChatTitleBar` around `ChatSwitcher` (defined in `AgentView.tsx`).
    - Empty state: centered heading ("What do you want to know about <project>?"), the composer (wide variant), the suggestion cards, and a "Recent chats" list with the 5 most recent chats.
    - With messages: `ChatThread` in a centered column (max width 3xl) and the composer at the bottom.
13. Add `ChatSwitcher` and `recent-chats.tsx`: a popover with "New chat" and the 20 most recent chats (`chats` from the provider), with relative time from `updated_at`. `recentChatItems` builds the menu items for the switcher and the status-bar history.
    - In the shared `MenuItem`, the right-hand slot becomes `min-w-4 whitespace-nowrap`, so a suffix such as "31m ago" never overflows the row.
14. Add `ChatTitleBar.tsx`: the title plus a "⋯" `Menu` with Rename and Delete.
    - Rename turns the header title into an input.
    - Delete uses `ConfirmDialog`.
    - Both call the new API functions and invalidate `agentQueryKeys.chats`.
15. Add `deleteChat(id)` and `renameChat(id, title)` to `web/src/api/chat.ts`.
16. In `ChatPopup.tsx`:
    - Add the expand button (lucide `Maximize2`). It calls `openChatView(chatId)` and `minimize()`. For a new chat, it opens the empty Agent view.
    - Use `ChatTitleBar` in the header. The "⋯" menu shows when the chat exists.
17. In `AgentStatusBar.tsx`, add the history button (lucide `History`). It opens a popover with the 20 most recent chats. Picking one calls `openChat(id)`.
18. After a delete, call `chatDeleted(id)` on the provider. It drops the pill and resets the popup to a new chat when the popup showed it. The Agent view's `onDeleted` opens `#/agent`. The UI does not watch the chat list for disappearing chats, because a chat created a moment ago is not in the list until it refetches.
19. Register the view in `ViewRouter.tsx` (lazy import).

## Decisions

- **The view and the popup share one engine and one set of components.** `useChatSession`, `ChatThread`, and `ChatComposer` serve both. Only the layout differs.
- **The chat ID goes in the URL (`#/agent/<id>`), not in view state.** Reload, back/forward, and links work like issue routes.
- **The history UI is the switcher dropdown and the status-bar popover, not a separate list page.** Linear has no list page, and the switcher keeps the view focused on one chat.
- **Pruning happens on board start, not on a timer.** Board restarts are frequent enough, and pruning while a user browses history would surprise them.
- **Running chats are never pruned, and delete stops them first.** No turn writes to a log that no longer exists.
- **The Agent entry is hidden when chat is unavailable.** This matches how the sidebar handles Cycles and Drafts (`requires`). A disabled entry would add noise for users without Claude Code.

## Edge Cases

**HIGH (developer decides)**
- *Retention values.* **Decided:** 30 days and 200 chats, as constants in `internal/chat`.

**MEDIUM (agent proposes, developer confirms)**
- *The same chat is open in the popup and in the view.* Both follow the same SSE stream and stay in sync. Both composers can send, and the second send gets 409 while a turn runs. That shows as a toast, as today.
- *`#/agent/<id>` for a deleted or unknown chat.* The view shows "Chat not found" with a "New chat" button.
- *Delete from the status-bar history.* Out of scope. The history popover only opens chats.

**LOW (agent handles)**
- A rename to the same title does nothing.
- The switcher shows "Untitled chat" for a chat with an empty title.
- Pruning skips meta files it cannot parse.

## Assumptions

- `g a` is free. The current go keys are `o i b c t l d m n r`.
- The `Menu`, `Popover`, and `ConfirmDialog` primitives in `components/ui` cover the switcher, the "⋯" menu, and delete confirmation.
- Relative time uses the existing format helper in `utils/format.ts`.

## Resolved Questions

1. **Sidebar position:** the first entry of the personal section, above My Issues. (The first answer was after Notifications; the user moved it during review.)
2. **Retention:** constants for now (`PruneMaxAge = 30 days`, `PruneMaxCount = 200`).

## Not in scope

- @-mentions, skills picker, attachments.
- Making the Agent view the default landing view (Linear's "Agent is now your default view" banner).
- Search in the chat history.
