# Agent chat and board: live updates stop after the tab is in the background

## What
After the tab has been in the background for a while:
- An open chat stops updating. When you come back, it shows the spinner and a running timer, but the turn has already finished on the server.
- The board misses issue events (created, status changed, ...). Issue lists stay stale.

Only a hard refresh shows the real state.

## Why (root cause)
Both streams hand events to each client through a buffered channel and **drop events silently** when it is full:

| Stream | Fan-out | Buffer |
|---|---|---|
| `GET /api/local/chats/{id}/stream` (`handleChatStream`) | `Runner.publish` | 256 |
| `GET /api/events` (`handleSSE`, `handleSSERelay`) | `SSEHub.Broadcast` | 16 |

```go
select {
case ch <- e:
default: // dropped
}
```

A background tab gets throttled or frozen and stops reading the socket. The handler's `Write` blocks, the channel fills, and later events are dropped. For a chat, that includes the settled `turn_done`. On the board stream, 16 slots fill within minutes from DB-watch `UPDATE`s and `chat_changed` (new since xpo-ae6385), and every issue event after that is lost. The connection stays open, so `EventSource` never reconnects.

The client has no safety net either:
- `useChatThread` has no `onerror` handling, and nothing resyncs when the tab is visible again.
- Issue queries use `staleTime: Infinity` and `refetchOnWindowFocus: false`. When the board stream reopens, `connectEvents` refetches only chats, not issues.

## How

### Server
1. `Runner.publish` (`internal/chat/runner.go`): when a subscriber's buffer is full, remove it and **close its channel**. Its unsubscribe func becomes idempotent.
2. `handleChatStream` returns when its channel is closed. The browser's `EventSource` reconnects with `Last-Event-ID`, and the server replays the missed settled events, without duplicates (existing behavior).
3. `SSEHub.Broadcast` (`internal/server/sse.go`): same as 1. `unsubscribe` closes the channel only if it is still subscribed, so it never closes twice. `handleSSE` and `handleSSERelay` already return on a closed channel. The browser then reconnects.

### Client
4. Move the chat stream handling (`useChatThread`) into `chat-stream-utils.ts` `followChatStream`. It opens with `?after=<last seq>`, tracks the last settled seq, and reopens from it:
   - on `error` when the source is `CLOSED` (the browser gave up), after 5s;
   - when the tab becomes visible again after it was hidden.
5. `connectEvents` (`hooks/sse-utils.ts`):
   - On `open`, it also calls `onEvent` (refetch issue data), not only `onChatEvent`. Any reconnect may have missed issue events.
   - When the tab becomes visible again after it was hidden, it reconnects right away (cancels a pending retry). That covers a stream that is stuck but still open, and it skips the 5s wait.

## Acceptance criteria
- [ ] Go test: a runner subscriber whose buffer is full gets its channel closed, and later publishes don't panic. Unsubscribing after that is a no-op.
- [ ] Go test: `handleChatStream` ends the response when the client falls behind.
- [ ] Go test: an `SSEHub` subscriber whose buffer is full gets its channel closed. Other subscribers still get events. Unsubscribing it after that doesn't panic.
- [ ] TS tests, `followChatStream`: it reopens from the last seq after a `CLOSED` error. It reopens on becoming visible after being hidden. Stopping it closes the source, cancels a pending retry, and removes the listener.
- [ ] TS tests, `connectEvents`: `open` calls both `onEvent` and `onChatEvent`. Becoming visible after being hidden reconnects and cancels a pending retry. Stopping it removes the visibility listener.
- [ ] Manual: open a chat, send a message, put the tab in the background until the turn ends, come back. The finished turn and its output show within a few seconds, with no refresh.
- [ ] Manual: with the tab in the background, create an issue or change a status through MCP. Come back: the board shows the change, with no refresh.
- [ ] `make test` passes.

## Out of scope
- xpo-9423f5's deep-link part: a link to an issue the board hasn't cached yet should fetch it before it says "not found".