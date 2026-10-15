# Agent chat: real-time chat events

## What

Push chat-list changes to every open board over `/api/events`, and drop chat-list polling.

## Why

The chat list only polls while the browser already knows a chat is running. A second browser (e.g. All Chats on another machine) that loaded with every chat idle never refetches, so chats created, started, renamed or archived elsewhere stay stale until reload.

## How

### Server

1. `chat.Store` gets an `OnChange func(id string)` hook. It is called after:
   - every successful `writeMeta` (create, rename, auto title, archive/unarchive, auto-archive, turn state, usage/cost, grants, interrupted/orphaned, mode changes), and
   - a successful `Delete`.
2. **Waiting state.** `waiting` in the list is live runner state (open approval/question cards) and never touches meta. `Store.Append` also calls `OnChange` when it appends an `approval_requested` or `approval_resolved` event. Both are settled after the card map changes, so a refetch sees the new `waiting` value.
3. `Server.ChatChanged(id)` broadcasts `SSEEvent{Type: "chat_changed", IssueID: id}`, throttled per chat: the first change sends at once, further changes within 500ms collapse into one trailing send at the end of the window. The last state is always delivered.
4. `handleSSE` sends it as SSE event `chat_changed` (the ID rides in the existing `issue_id` field, like `draft_changed`).
5. `board.go` wires `srv.Chat.Store.OnChange = srv.ChatChanged`.
6. **Remote mode.** The board proxies `/api/*` to the remote server, but chats live on the local board. In proxy mode the board answers `GET /api/events` itself (`handleSSERelay`): it opens the remote `/api/events` with the bearer token, and writes both the remote stream's events and its own hub's events into one response.
   - The remote request is bound to the browser request; a browser disconnect closes it.
   - If the remote can't be reached or answers non-200, respond 502 (the browser retries in 5s).
   - If the remote stream ends, end the response, so the browser reconnects and refetches.
   - Remote events are forwarded verbatim, block by block, so a local event never lands inside a remote one.

### Frontend

7. Pull the EventSource logic out of `useSSE` into a pure, testable `connectEvents` (takes an EventSource factory and a timer), keeping the 5s reconnect.
8. `useSSE` gets `onChatEvent` and calls it on `chat_changed` **and on every `open`** (first connect and reconnects), so changes missed while disconnected are caught. `chat_changed` does **not** fire the issue invalidation (`onEvent`).
9. `App.tsx` passes `onChatEvent` = invalidate `agentQueryKeys.chats`.
10. Remove `chatListPollMs` (and its test), `refetchInterval` and `refetchIntervalInBackground` from the chat list query. Hidden tabs keep their SSE connection, so title/favicon still update.

## Acceptance criteria

- [ ] Creating, renaming, archiving, unarchiving, deleting a chat, and a turn starting/ending, update the chat list in a second browser without reload.
- [ ] An approval or question card opening/closing updates `waiting` (pill/title/favicon) in a second browser.
- [ ] A browser that loses and regains the SSE connection refetches the chat list.
- [ ] Bursts of changes to one chat send at most one `chat_changed` per 500ms, and the last change is always sent.
- [ ] `chat_changed` does not refetch issue queries.
- [ ] No chat-list polling remains.
- [ ] Remote mode: the browser gets both remote issue events and local `chat_changed` on its one `/api/events` stream.
- [ ] Go tests: store calls `OnChange` on meta writes, delete, approval events (not on other appends); throttle; SSE streams `chat_changed`; relay forwards remote events with the token, adds local events, 502s on an unreachable remote, ends when the remote stream ends.
- [ ] Frontend tests: `connectEvents` routes `chat_changed` to the chat handler only, calls it on open and reconnect.
- [ ] `make test` passes.

## Decisions

- **Throttle, not pure debounce.** A trailing-only debounce could starve during a long stream of writes; leading + trailing gives instant updates and a bounded rate.
- **Approval events via `Append`** rather than explicit calls in the runner: one place, and the ordering (card map changed before settle) is already right.
- **Invalidate on first open too**, not only reconnects: costs one extra list fetch on load, closes the gap between the first list fetch and the stream opening.
- **Merge streams in remote mode** (user's choice) rather than a second `/api/local/events` EventSource (one more HTTP/1.1 connection per tab) or keeping polling in remote mode.

## Out of scope

- Chats run by a separate `xpo` server process on the same repo (polling didn't cover that either).
- Refetching issue queries on SSE reconnect.
