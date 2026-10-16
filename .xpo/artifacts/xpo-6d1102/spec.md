# One resilient, extensible SSE stream per board

## What
Each board tab opens **one** `EventSource` on `GET /api/events`. It carries every live event: issue changes, draft changes, chat-list changes, and the events of every chat the tab has open. A tab that falls behind, sleeps, or loses the network never misses an update without knowing it, and it catches up on its own. A new event source is one registration on the server and one handler in the browser.

## Why
See the issue description. In short:
- There are two pub/sub implementations (`SSEHub` and `chat.Runner.Subscribe`) and two unnamed protocols, each with its own browser helper.
- Each open chat costs one connection, and HTTP/1.1 allows 6 per host across all tabs.
- The only way to recover from an overflow is to drop the connection, which refetches everything.
- No write has a deadline.

## Concepts

| Term | Meaning |
|---|---|
| **Hub** | The one broadcaster, in `internal/live`. It replaces `SSEHub` and `Runner.subs`. |
| **Topic** | A named event source: a *signal* or a *log*. |
| **Signal topic** | "Something changed, refetch." Repeats merge into one *dirty* flag per subscriber, so it can't overflow. Topics: `issues` (issues, git refs, the DB watch), `drafts` (issue drafts), `chats` (the chat list). |
| **Log topic** | An ordered event log with `seq` ids and a store to replay from. A key picks the instance: `chat:<id>`. |
| **Stream** | One SSE connection. It has a random id, and its subscriber holds its set of topics. |

## Design

### Server: `internal/live`
```go
hub := live.NewHub()
hub.Signal("issues")
hub.Signal("drafts")
hub.Signal("chats")
hub.Log("chat", replay) // replay(key string, after int) ([]live.Frame, error)

hub.Notify("issues")                                    // signal: never blocks
hub.Append("chat", id, live.Frame{Seq: e.Seq, Data: e}) // log: never blocks
```
- **Subscriber state**, under the subscriber's own mutex:
  - a set of dirty signal topics;
  - for each log topic: `last` (the last seq it delivered), a queue of pending frames (cap 256, `Hub.QueueCap`), and a replay flag.
  - A `wake` channel (cap 1) tells the writer that there is work.
- **`Notify`** sets the dirty flag on each subscriber and wakes it.
- **`Append`** queues the frame for each subscriber of `chat:<id>`. **Overflow:** if the queue is full, it clears the queue and sets the replay flag. The connection stays open, and no other topic is affected.
- **The writer** (`Sub.Take`, called by the SSE handler) takes the pending state and returns, in order:
  1. one `signal` per dirty topic;
  2. for each log topic with the replay flag: `replay(key, last)`, read without the lock, with only the frames with `seq > last`;
  3. the queued frames. It skips settled frames with `seq <= last`. Live frames (`seq == 0`) always go out.
- Subscribing to a log topic (`Sub.Add`) **is** a replay: it registers the topic with `last = after` and the replay flag set. The topic is registered before the store is read, so no event falls between the two.
- If a replay returns any error (`ErrGone` for a deleted chat, or a read error), the writer drops the topic and writes a `gone` frame. The browser then loads the topic again.

### Server: the endpoint (`live.Stream`)
- `GET /api/events` (`ServeEvents`) opens a stream that is subscribed to every signal topic. The first frame is `hello` with the stream id.
- `POST /api/events/{stream}/topics` (`ServeTopics`) changes the stream's topics. Body: `{"add":[{"topic":"chat:<id>","after":12}], "remove":["chat:<id>"]}`.
  - An unknown stream id gets 404. An unknown topic gets 400, and nothing in the body is applied.
  - The local board answers the route. It is more specific than the proxy catch-all, so it wins in proxy mode too.
- **Wire format:** named SSE events with JSON data. There are no SSE `id:` lines, because the browser holds the positions (see Decisions).
  ```
  event: hello   data: {"stream":"<id>"}
  event: signal  data: {"topic":"issues"}
  event: log     data: {"topic":"chat:<id>","data":<chat.Event>}
  event: gone    data: {"topic":"chat:<id>"}
  ```
- **Hygiene:**
  - Before each write: `http.ResponseController.SetWriteDeadline(now + 10s)`. A write error ends the stream. `statusRecorder` (the `serve` request logger) gets an `Unwrap`, so the controller reaches the connection.
  - A keepalive comment every 25s.
  - `hub.Close()` ends every stream. `serve` registers it with `httpServer.RegisterOnShutdown`, and `board` calls it before it exits.

### Producers (`internal/server/live.go`)
- `Server.Live *live.Hub` replaces `SSEHub`. `NewLiveHub()` registers the three signal topics. `registerLiveRoutes` registers the `chat` log topic when the board has chats.
- `Server.broadcastEvent(type, id)` becomes `Server.notify(topic)`. Draft writes use `drafts`, and everything else uses `issues`. The exported `BroadcastEvent` becomes `Notify(topic)`. `mcpserver` keeps its `onEvent(type, id)` callback, and `serve` passes one that notifies `issues`.
- `ChatChanged` (still throttled to 500ms) → `notify("chats")`.
- `chat.Runner` stops being a pub/sub. `RunnerConfig.OnEvent func(chatID string, e Event)` gets every published event, settled and live, in order, under the runner's lock. `ChatEventsTo(hub)` wires it to `hub.Append("chat", …)`. `Subscribe`, `subs` and `dropSub` are removed. The runner tests use an in-package `tap(r, id) <-chan Event` helper (`tap_test.go`) that records what `OnEvent` gets.
- `GET /api/local/chats/{id}/stream` and `handleChatStream` are **removed**, and so are `chatStreamURL` and `followChatStream`.

### Proxy mode (`live.Mirror`)
- The board keeps **one** upstream connection to the remote's `GET /api/events` for all its tabs, and it speaks the same protocol.
  - A remote `signal` for `issues` or `drafts` → local `Notify` on the same topic. Other topics are ignored.
  - A remote `hello` (any reconnect) → `Notify("issues")` and `Notify("drafts")`, because changes may have been missed.
  - It reconnects with backoff, starting at 1s and doubling up to 30s.
- The browser's stream is always the local board's. Chats keep working while the remote is down.
- `handleSSERelay` and `readSSEBlocks` are removed. `live.ReadEvents` is the SSE parser.

### Browser: `web/src/live/`
- **`createLive({ create, post, schedule, cancel, watchVisibility })`** (`live-client.ts`) owns the `EventSource`:
  - On `hello`: it stores the stream id, calls every signal handler (resync), and POSTs every followed log topic with its last seq (replay).
  - On `signal`: it calls that topic's handlers.
  - On `log`: each follower of the topic drops settled events it already has (`seq <= last`), updates `last`, and gets the event.
  - On `gone`: it forgets the topic and calls each follower's `onGone`.
  - On an `error` that leaves the source `CLOSED`: it retries with backoff, starting at 1s and doubling up to 30s, and resets after a `hello`. While the source is `CONNECTING`, the browser retries on its own, and the next `hello` resyncs.
  - When the tab is visible again after being hidden: it reconnects right away and cancels a pending retry.
- **Handlers:**
  - `live.onSignal(topic, fn)` returns an unsubscribe.
  - `live.follow(topic, after, onEvent, onGone)` returns a stop function. It POSTs `add` if the stream is open; otherwise the next `hello` does it. Followers of one topic share it: the topic is added from the lowest seq, and `remove` is POSTed when the last follower stops. A POST that fails with 404 (the stream is gone) reconnects.
- `live` (`index.ts`) is the tab's one client. **`useLive()`** in `App.tsx` replaces `useSSE`, starts it, and maps the signals:
  - `issues` → `invalidateOnServerEvent`, now without `issueDrafts`;
  - `drafts` → invalidate `queryKeys.issueDrafts`;
  - `chats` → invalidate the chat list.
- `useChatThread` uses `live.follow("chat:<id>", …)`. On `gone` it loads the chat again; a 404 then shows the chat as missing.
- `hooks/sse-utils.ts` and `hooks/useSSE.ts` are removed. `watchDocumentVisibility` and `onVisibleAgain` move to `live/visibility.ts`.

### Adding a topic (documented in `docs/design.md`)
1. Server: `hub.Signal("name")` or `hub.Log("name", replay)`, then call `Notify` or `Append` where the data changes.
2. Browser: `live.onSignal("name", fn)` in `useLive`, or `live.follow("name:<key>", after, fn)`.

## Decisions
- **No compatibility with older remotes or boards.** The protocol changes in place. A board and its remote must run the same version.
- **Topic changes go through a POST, not a reconnect.** A reconnect each time a chat opens or closes would resync every signal topic, which refetches all issue queries, and it would leave a short gap with no stream. The cost is a stream registry on the server and a 404 path.
- **The browser holds the log positions, not `Last-Event-ID`.** One SSE id can't hold a position for each topic without encoding them all. After any reconnect, `hello` → POST with `after` does the same job, and the server code that replays is the same code that handles overflow.
- **Overflow replays one topic instead of closing the stream.** A frozen tab with an open chat comes back to one store read for that chat, not to a full refetch of everything.
- **Any replay error drops the topic with `gone`.** Retrying could spin on a broken log. Reloading the chat in the browser either works or shows the error.
- **`drafts` is its own topic.** A draft save then refetches only the drafts list, not the 6 issue query groups. Drafts are per user, so in multi-user `serve` one person's draft doesn't make every tab refetch issues.
- **`chat.Runner` only emits; it doesn't fan out.** There is one pub/sub implementation in production, and the `chat` package knows nothing about SSE. The runner's overflow test moved to the hub's tests.
- **Live frames (`seq == 0`, the streaming deltas) can be lost on overflow.** The settled events that follow replace them, as they do on a reconnect today.

## Out of scope
- xpo-9423f5: a deep link to an uncached issue says "not found".
- HTTP/2 for the board.
- Drive trail events (xpo-7a4b19). That issue registers its own topic on top of this one.
- The chat store's 4 MiB line limit: a single event larger than that can't be read back, by a replay or by `GET /api/local/chats/{id}`.

## Acceptance criteria
- [ ] A board tab holds exactly one SSE connection, whatever the number of open chats (popups and the Agent view).
- [ ] Issue, draft, chat-list and chat-thread updates all arrive over it. `/api/local/chats/{id}/stream` is gone.
- [ ] Go, hub: `Notify` merges repeats into one frame for a subscriber that isn't reading. An `Append` overflow on one topic replays that topic from the store, with no duplicates and no gaps in the settled seqs, and it doesn't touch other topics or subscribers. Neither call ever blocks.
- [ ] Go, hub: subscribing to a log topic with `after` replays from the store, then goes live, and no event falls between the two. A replay error writes `gone`.
- [ ] Go, endpoint: `hello` comes first. The topics POST adds and removes topics. An unknown stream id gets 404, and an unknown topic gets 400. A write that hits its deadline ends the stream. `hub.Close` ends open streams.
- [ ] Go, producers: draft writes notify `drafts` only. Issue writes, the DB watch and the git-ref watch notify `issues`. Chat events reach `chat:<id>` through `RunnerConfig.OnEvent`.
- [ ] Go, end to end: a client that stops reading during a long turn gets every settled event once and in order when it reads again.
- [ ] Go, proxy: remote `issues` and `drafts` signals become local signals. An upstream reconnect notifies both. A remote that is down doesn't end the browser's stream.
- [ ] TS, `createLive`: `hello` resyncs signals and re-POSTs followed log topics with their last seq. It routes `signal`, `log` and `gone` to their handlers. It retries with backoff after a `CLOSED` error and reconnects when the tab is visible again. `follow` and its stop function POST `add` and `remove`, and shared topics are added once. A 404 on a POST reconnects.
- [ ] TS: `drafts` invalidates only the drafts query, and `issues` no longer invalidates drafts.
- [ ] Manual: open two chats in popups and one in the Agent view. DevTools shows one `/api/events`. Put the tab in the background while a turn runs, and change an issue through MCP. Come back: the chats and the board both catch up with no refresh.
- [ ] Manual: restart `xpo board` while a chat is open. The board and the chat catch up with no reload.
- [ ] `docs/design.md` documents topics and how to add one.
- [ ] `make test` passes.
