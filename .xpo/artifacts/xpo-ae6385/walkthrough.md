# Walkthrough: Agent chat: real-time chat events

## What was built and why

The chat list used to poll every 2s, but only while the browser already knew a chat was running. A browser that loaded with every chat idle, for example All Chats open on another machine, never refetched. Chats created, started, renamed or archived elsewhere stayed stale until a reload.

Now the board pushes a `chat_changed` event on `/api/events` whenever a chat's list entry may have changed. Every browser refetches the chat list when it gets one. Chat-list polling is gone.

## How the pieces fit

```
chat.Store ──OnChange(id)──▶ Server.ChatChanged ──throttle──▶ SSEHub ──▶ /api/events
                                                                         │ (remote mode:
                                                                         │  handleSSERelay merges
                                                                         │  the remote stream in)
                                                                         ▼
                                     connectEvents ──chat_changed / open──▶ invalidate agentQueryKeys.chats
```

### Server

- **`chat.Store.OnChange`** (`internal/chat/store.go`) is the one place that notices changes.
  - `writeMeta` calls it once the meta file is in place (after the atomic `os.Rename` of the temp file). Every meta change goes through `writeMeta`: create, rename, title, archive, turn state, usage, grants, mode, interrupted and orphaned. New meta fields are covered for free.
  - `Delete` calls it after the meta file is removed.
  - `Append` calls it when it stores an `approval_requested` or `approval_resolved` event. The list's `waiting` flag is live runner state (`Runner.Waiting`) and is never written to meta, so watching `writeMeta` alone would miss it. The runner settles both events after it updates its card map, so a refetch triggered by the event already sees the new `waiting` value.
- **`throttle`** (`internal/server/throttle.go`) limits each chat to one event per 500ms.
  - The first trigger fires at once. More triggers inside the window set a flag, and the window's end fires once and opens a new window.
  - A trailing-only debounce could hold events back for as long as a turn keeps writing. Leading plus trailing gives an instant update, a bounded rate, and a guaranteed final event.
- **`Server.ChatChanged`** (`server.go`) creates the throttle lazily with `sync.Once`. Test servers that aren't built by `NewServer` still work. `board.go` wires it with `srv.Chat.Store.OnChange = srv.ChatChanged`.
- **`writeSSEEvent` / `setSSEHeaders`** (`sse.go`) were pulled out of `handleSSE` so the relay can share them. `chat_changed` uses the existing `issue_id` field for the chat ID, the same way `draft_changed` does.

### Remote mode relay

In remote mode the board proxies `/api/*` to the remote server, but chats live on the local board. A local `chat_changed` would never reach the browser. `handleSSERelay` (`sse_relay.go`) is registered as `GET /api/events` only in proxy mode, and only when the board isn't headless. It is more specific than the proxy's `/api/{path...}` catch-all, so it takes the request.

- It opens the remote `/api/events` with the bearer token, bound to the browser's request context. When the browser disconnects, the remote request closes.
- If the remote can't be reached or answers non-200, it responds 502. The browser retries after 5s.
- It subscribes to the local hub before the first flush. A local event sent after the browser sees the stream open is never lost.
- `readSSEBlocks` splits the remote stream on blank lines and forwards each block whole. A local event is never written into the middle of a remote one.
- When the remote stream ends, the response ends too. The browser reconnects, and because it refetches on `open`, it catches anything it missed.

### Frontend

- **`connectEvents`** (`web/src/hooks/sse-utils.ts`) holds the EventSource logic that used to live inside `useSSE`. It takes an EventSource factory and a timer, so it can be unit-tested without a DOM.
  - `ISSUE_EVENTS` go to `onEvent`, as before.
  - `chat_changed` and every `open` go to `onChatEvent`. Calling it on `open` covers changes missed while the stream was disconnected. It also covers the gap between the first list fetch and the stream opening, at the cost of one extra list fetch per page load.
  - A chat event does not refetch issue queries.
- **`useSSE`** now takes `onChatEvent`. `App.tsx` passes a callback that invalidates `agentQueryKeys.chats`.
- **Polling removed:** `chatListPollMs`, its test, and `refetchInterval` / `refetchIntervalInBackground` on the chat list query. A hidden tab keeps its SSE connection, so tab titles and favicons still update.

## Things a future reader should know

- **One server process only.** Events come from the process that writes the change. A chat run by a separate `xpo` server on the same repo isn't seen live. Polling didn't cover that either.
- **Reading old chats can emit events.** `Store.Meta` backfills usage for chats saved before usage totals existed, and that write goes through `writeMeta`. Each old chat triggers one event, once. It's harmless.
- **`OnChange` runs under the store mutex** (from `writeMeta` and `Append`). The board's hook only queues into a non-blocking broadcast. A hook must never call back into the store.
- Adding a new list-visible field that isn't stored in meta, like `waiting`? It needs its own `OnChange` trigger.

## Acceptance criteria

- [x] Creating, renaming, archiving, unarchiving, deleting a chat, and a turn starting or ending, update the chat list in a second browser without reload. *All of these write meta or delete, which calls `OnChange` (`TestStore_OnChangeOnMetaWritesAndDelete`). `TestChatChanged_StreamsChatChanged` shows the event reaching `/api/events`. The user tried it and approved ("lgtm").*
- [x] An approval or question card opening or closing updates `waiting` in a second browser. *`TestStore_OnChangeOnApprovalEventsOnly`.*
- [x] A browser that loses and regains the SSE connection refetches the chat list. *`connectEvents` test "reconnects after an error and calls onChatEvent when it reopens".*
- [x] Bursts of changes to one chat send at most one `chat_changed` per 500ms, and the last change always arrives. *`TestThrottle_BurstCollapsesIntoOneTrailingFire`, `TestThrottle_FirstFiresAtOnce`, `TestThrottle_NoTrailingFireWithoutChange`, `TestThrottle_KeysAreIndependent`.*
- [x] `chat_changed` does not refetch issue queries. *`connectEvents` test "routes chat_changed to onChatEvent only".*
- [x] No chat-list polling remains. *`chatListPollMs` and the chat query's `refetchInterval` are deleted. A grep finds no remaining use.*
- [x] Remote mode: the browser gets both remote issue events and local `chat_changed` on its one `/api/events` stream. *`TestSSERelay_MergesRemoteAndLocalEvents`. It also checks that the bearer token is forwarded.*
- [x] Go tests cover the store hook, the throttle, SSE `chat_changed`, and the relay (merge, token, 502 on an unreachable remote, 502 on a remote refusal, ending when the remote ends). *`store_test.go`, `throttle_test.go`, `sse_test.go`, `sse_relay_test.go`. Each was run red before its implementation.*
- [x] Frontend tests: `connectEvents` routes `chat_changed` to the chat handler only, and calls it on open and on reconnect. *`sse-utils.test.ts`, 5 tests, run red first.*
- [x] `make test` passes. *Exit 0: lint, 76 frontend test files (1124 tests), all Go packages.*
