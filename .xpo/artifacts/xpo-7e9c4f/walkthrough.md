# Walkthrough: live updates stop after the tab is in the background

## The bug
After the board tab was in the background for a while, two things went stale until a hard reload:
- An open chat showed a running turn (spinner, timer counting up) that had already finished on the server.
- The board missed issue events: new issues, status changes.

## Root cause
The server has two places that fan events out to SSE clients, and both had the same flaw:

| Stream | Fan-out | Buffer per client |
|---|---|---|
| `/api/local/chats/{id}/stream` | `chat.Runner.publish` | 256 |
| `/api/events` (direct and relay) | `SSEHub.Broadcast` | 16 |

Both used a non-blocking send with `default:`. When a client's buffer was full, the event was **dropped and the connection kept open**.

A background tab gets throttled or frozen and stops reading its socket. The handler's `Write` blocks, the channel fills, and every later event is lost. For a chat, that includes `turn_done`. For the board, 16 slots fill within minutes from DB-watch `UPDATE`s and `chat_changed`. Because the connection never broke, `EventSource` never reconnected, so it could never catch up.

The browser had no safety net either:
- `useChatThread` ignored stream errors.
- Issue queries use `staleTime: Infinity` and `refetchOnWindowFocus: false`.
- `connectEvents` refetched only chats on reconnect.

## The fix: close on overflow, then catch up on reconnect
The key idea: **a stream that has lost events must end**, because SSE recovers through reconnects, not through gaps in an open connection.

### Server
- **`Runner.publish`** (`internal/chat/runner.go`): when a subscriber's buffer is full, `dropSub` removes it and its channel is closed. The unsubscribe func goes through `dropSub` too, which is a no-op for a subscriber that is already gone, so it's safe after an overflow. The unsubscribe func never closes the channel; only `publish` does, under `r.mu`.
- **`handleChatStream`** (`internal/server/chat.go`) returns when its channel is closed. The browser's `EventSource` reconnects with `Last-Event-ID`, and the handler replays the settled events after that id from the store. That replay already existed, so nothing is duplicated.
- **`SSEHub.Broadcast`** (`internal/server/sse.go`) does the same: it deletes and closes a full client. `unsubscribe` now closes only if the client is still registered, so the handler's deferred unsubscribe can't double-close. `handleSSE` and `handleSSERelay` already returned on a closed channel.

### Browser
- **`followChatStream`** (`web/src/components/Agent/chat-stream-utils.ts`, used by `useChatThread`) owns the chat's `EventSource`. It tracks the last settled `seq` it saw and reopens with `?after=<seq>`:
  - 5s after an `error` that leaves the source `CLOSED`, when the browser has given up. While it's `CONNECTING`, the browser retries on its own with `Last-Event-ID`.
  - Right away when the tab becomes visible after being hidden. This covers a stream that is stuck but still open, and it's cheap: only missed settled events come back.
- **`connectEvents`** (`web/src/hooks/sse-utils.ts`):
  - On every `open` it calls `onEvent` (invalidate issue data) as well as `onChatEvent`, because any reconnect may have missed both.
  - It reconnects right away when the tab is visible again, and cancels a pending retry.
- `watchDocumentVisibility` and `onVisibleAgain` live in `sse-utils.ts`, and both helpers use them. Both helpers take `schedule`, `cancel` and `watchVisibility` as parameters, so the tests run without timers or a DOM.

## Why not a bigger buffer, or blocking sends?
- A bigger buffer only delays the overflow. A tab frozen for an hour fills any buffer.
- Blocking sends would let one frozen tab stall a running turn, or every board client.

Closing the slow client costs nothing for the others, and SSE's reconnect plus our replay or refetch makes the client whole again.

## Out of scope / follow-ups
- xpo-6d1102: one resilient, extensible SSE stream per board. This fix showed the broader smell: two separate pub/sub implementations, two protocols with no name, one connection per open chat, and no write deadlines.
- xpo-9423f5 keeps its deep-link part: fetch an issue that isn't cached yet before saying "not found".
- In proxy mode, the remote server must also run this fix. An older remote still drops events while the stream stays open.

## Acceptance criteria
- [x] Go test: a runner subscriber whose buffer is full gets its channel closed, later publishes don't panic, and unsubscribing after that is a no-op. *`TestRunner_ClosesASubscriberThatFallsBehind`. It failed before the fix ("subscriber still open after 256 events").*
- [x] Go test: `handleChatStream` ends the response when the client falls behind. *`TestChat_StreamEndsWhenClientFallsBehind`: 2000 deltas of 16 KiB while the client doesn't read. Before the fix it reproduced the bug (the stream stayed open until the 10s deadline).*
- [x] Go test: an `SSEHub` subscriber whose buffer is full gets its channel closed, other subscribers still get events, and unsubscribing it after that doesn't panic. *`TestSSEHub_BroadcastClosesSlowClient`, which replaces `TestSSEHub_BroadcastDropsSlowClient` (that test locked in the drop behavior). It failed before the fix.*
- [x] TS tests, `followChatStream`: it reopens from the last seq after a `CLOSED` error and on becoming visible after being hidden. Stopping it closes the source, cancels a pending retry, and removes the listener. *`chat-stream-utils.test.ts`, 7 cases. The module didn't exist when they first ran.*
- [x] TS tests, `connectEvents`: `open` calls both `onEvent` and `onChatEvent`. Becoming visible after being hidden reconnects and cancels a pending retry. Stopping it removes the visibility listener. *`sse-utils.test.ts`: 4 cases failed before the change and pass now.*
- [x] Manual: chat in a background tab catches up without a refresh. *The user approved the change for merge. I didn't run this check in a browser myself.*
- [x] Manual: the board catches up on issue changes made while the tab was in the background. *The user approved the change for merge. I didn't run this check in a browser myself.*
- [x] `make test` passes. *1180 frontend tests plus the Go suite. `go test -race ./internal/chat ./internal/server` passes too.*