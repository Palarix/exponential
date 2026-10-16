# Walkthrough: one resilient, extensible event stream per board

## What was built
Each board tab now holds **one** SSE connection, `GET /api/events`. Every live update arrives over it: issues, drafts, the chat list, and every chat the tab has open. Before, each open chat had its own stream, and two separate pub/sub implementations each dropped events in their own way.

```
 producers                         internal/live                      browser (web/src/live)
 ─────────                         ─────────────                      ──────────────────────
 issue writes, DB/git watch ─┐
 draft writes ───────────────┼─ Notify(topic) ─▶ Hub ── Sub per stream ── GET /api/events ──▶ createLive
 ChatChanged (500ms) ────────┘                    │      (dirty flags,                          ├ onSignal → React Query
 chat.Runner ── OnEvent ── Append("chat", id) ───┘       per-topic queues)  POST …/topics ◀── └ follow("chat:<id>") → useChatThread
 remote /api/events ── Mirror ── Notify(issues|drafts)
```

## The two kinds of topic
The central idea: **a slow reader must never miss an update without knowing it**. Each kind of data gets the rule that fits it:

| Kind | Example | What it carries | Slow-reader rule |
|---|---|---|---|
| Signal | `issues`, `drafts`, `chats` | "Something changed, refetch." | Repeats merge into one dirty flag per stream, so it can't overflow. |
| Log | `chat:<id>` | Ordered events, settled ones with a `seq` | A queue of 256 frames per stream and topic. On overflow, the queue is dropped and **that one topic** replays from the store. |

Signals have no payload worth keeping: the browser invalidates queries, so ten "issues changed" events mean the same as one. Logs have a store, `chat.Store`, that already holds every settled event with its seq, so a reader that fell behind can always be made whole from there.

## Server: `internal/live`
- **`hub.go`**: `Hub` holds the registered topics and the open subscribers (`Sub`, one per stream).
  - `Notify` sets a flag; `Append` queues a frame. Neither ever blocks: both only take short mutexes and do a non-blocking send on a `wake` channel of capacity 1.
  - `Sub.Take` is the writer's side. It returns the dirty signals first, then, for each log topic, its replay (when flagged) and then its queue. Settled frames with `seq <= last` are skipped, so a frame that sits in the queue *and* came back in the replay goes out once.
  - The store is read **without** the subscriber's lock. Frames that settle during the read queue up as usual, and the seq check removes the overlap. `TestAdd_NoEventFallsBetweenReplayAndLive` covers exactly this race.
  - **Subscribing to a log topic is itself a replay.** `Sub.Add(topic, after)` sets `last = after` and the replay flag. So the first subscribe, a resubscribe after a reconnect, and an overflow all run the same code.
  - If a replay returns any error, the topic is dropped and a `gone` frame is sent. `ErrGone` is what a deleted chat returns. Retrying a broken log could spin, and the browser can reload the chat instead.
- **`http.go`**: `Stream.ServeEvents` and `ServeTopics`.
  - The first frame is `hello` with the stream id. Every write sets `http.ResponseController.SetWriteDeadline(now+10s)` first, so a tab that stops reading frees its goroutine. `TestStream_WriteDeadlineEndsAFrozenClient` fails if the deadline is an hour.
  - `ServeTopics` checks every `add` before it applies anything: one unknown topic means 400 and no change.
- **`mirror.go`**: in proxy mode, one connection per board to the remote's stream. It turns remote `issues` and `drafts` signals into local ones and ignores anything else. Every remote `hello` notifies both topics, because changes may have been missed while it was away. It retries with backoff from 1s, doubling up to 30s.
- **`sse.go`**: `ReadEvents`, a small SSE parser that the mirror and the tests use.

## Server wiring (`internal/server/live.go`)
- `Server.Live` replaces `SSEHub`. `NewLiveHub` registers the three signal topics. `registerLiveRoutes` registers the `chat` log topic, with `replayChat` reading `Store.Get`, and the two routes. In proxy mode it mounts them directly, so they win over the `/api/{path...}` catch-all.
- `broadcastEvent(type, id)` became `notify(topic)`. The event type and issue ID were never used by the browser; it always refetched everything.
- **`chat.Runner` no longer fans out.** It calls `RunnerConfig.OnEvent` under its lock, which keeps one chat's events in publish order across the goroutines that publish them. `ChatEventsTo(hub)` adapts that to `hub.Append`. The runner tests record events with the in-package `tap` helper (`tap_test.go`), which panics if a test stops reading, so a dropped event can't pass silently.
- `statusRecorder` (the `serve` request logger) gained `Unwrap`. Without it, `ResponseController` can't reach the connection to set deadlines.
- `serve` calls `hub.Close` from `RegisterOnShutdown`, because `http.Server.Shutdown` waits for open requests and a stream never ends by itself. `board` calls it before exiting.

## Browser (`web/src/live/`)
- **`createLive`** owns the one `EventSource`.
  - On `hello` it calls every signal handler and POSTs every followed log topic with its last seq. That one rule covers the first connect, a browser auto-reconnect, a board restart, and a stream the server forgot (a 404 on POST reconnects).
  - It retries with backoff only once the source is `CLOSED`. While `CONNECTING`, the browser retries on its own.
  - It reconnects right away when the tab is visible again, which covers a stream stuck behind a frozen tab.
- **Followers share topics.** The same chat can be open in a popup and in the Agent view. Each follower keeps its own `last` and drops duplicates. The topic is added from the lowest seq, and `remove` is posted only when the last follower stops.
- `live` (`index.ts`) is the tab's one client. `useLive` starts it in `App.tsx` and maps `issues` → `invalidateOnServerEvent` (now without drafts), `drafts` → the drafts query, and `chats` → the chat list.
- `useChatThread` fetches the chat, then follows `chat:<id>` from its last seq. On `gone` it bumps a counter, which reruns the effect: either the chat loads again, or a 404 shows it as missing.

## Key decisions
- **Topic changes go through a POST, not a reconnect.** A reconnect per opened chat would refetch every issue query and leave a short gap with no stream.
- **The browser holds the positions, not `Last-Event-ID`.** One SSE id can't hold a seq per topic. `hello` → POST with `after` does the same job with the same server code as overflow.
- **Overflow replays one topic; it doesn't close the stream.** That's cheaper than xpo-7e9c4f's "close and reconnect", which refetched everything.
- **`drafts` is its own topic**: a draft save no longer refetches issues, and in multi-user `serve` one user's draft doesn't make everyone refetch.
- **No compatibility with older remotes**: the protocol changed in place, as you asked.
- **Live deltas (`seq == 0`) can be lost on overflow.** The settled `assistant_text` that follows replaces them.

## Gotchas for the next reader
- `httptest.Server.Close` waits for open requests. In tests, close the server with `t.Cleanup` registered **before** the client's cancel, not with `defer`. Otherwise `Close` waits forever on the open stream.
- A single chat event over 4 MiB can't be read back by `Store.Get`, so its replay goes `gone`, and the chat won't open either. This was already true before; it's filed as xpo-f5df49.

## Acceptance criteria
- [x] A board tab holds exactly one SSE connection, whatever the number of open chats. *You opened 3 chats; DevTools showed one `events` row and `topics` POSTs.*
- [x] Issue, draft, chat-list and chat-thread updates all arrive over it. `/api/local/chats/{id}/stream` is gone. *You saw an issue move live while the chats ran. `TestEvents_ChatStreamEndpointIsGone`; the route is removed. A GET now gets only the SPA's HTML fallback.*
- [x] Go, hub: `Notify` merges repeats. An `Append` overflow on one topic replays only that topic, with no duplicates and no gaps, and doesn't touch other topics or subscribers. Neither call blocks. *`TestNotify_MergesRepeatsIntoOneSignal`, `TestAppend_OverflowReplaysThatTopicOnly`, `TestAppend_LiveFramesAreDroppedOnOverflow`.*
- [x] Go, hub: subscribing with `after` replays, then goes live, and no event falls between the two. A replay error writes `gone`. *`TestAdd_ReplaysFromTheStoreThenGoesLive`, `TestAdd_NoEventFallsBetweenReplayAndLive`, `TestAdd_SkipsQueuedFramesTheReplayAlreadySent`, `TestReplay_GoneDropsTheTopic`, `TestReplay_ErrorDropsTheTopic`.*
- [x] Go, endpoint: `hello` first; the topics POST adds and removes; 404 for an unknown stream, 400 for an unknown topic; the write deadline ends a frozen client; `hub.Close` ends streams. *`http_test.go`: 6 tests. The deadline test was checked to fail with a 1h deadline.*
- [x] Go, producers: drafts notify `drafts` only, issue writes and watches notify `issues`, and chat events reach `chat:<id>` through `OnEvent`. *`TestIssueDrafts_NotifyDraftsOnly`, `TestIssueWrites_NotifyIssuesOnly`, `TestWatchGitRefs_SSEBroadcast`, `TestEvents_ChatTopicReplaysThenStreamsRunnerEvents`.*
- [x] Go, end to end: a client that stops reading during a long turn gets every settled event once and in order. *`TestChat_StreamCatchesUpAfterTheClientFellBehind`: 3000 deltas. It asserts that deltas were actually dropped, so it can't pass without an overflow.*
- [x] Go, proxy: remote signals become local, a reconnect notifies both, and a down remote doesn't end the browser's stream. *`TestMirror_*`, `TestStartMirror_NotifiesRemoteTopics`, `TestEvents_ProxyModeStreamWorksWhileTheRemoteIsDown`.*
- [x] TS, `createLive`: resync and re-add on hello, routing, backoff, visibility reconnect, add/remove POSTs, shared topics, 404 reconnect. *`live-client.test.ts`: 17 tests.*
- [x] TS: `drafts` invalidates only drafts, and `issues` no longer does. *`query-keys.test.ts`, updated.*
- [x] Manual: several chats plus a change made through MCP, all live. *You checked it in a demo board.*
- [x] Manual: board restart while a chat is open catches up with no reload. *I restarted the board (down about 5s); you confirmed that the board and the chat recovered.*
- [x] `docs/design.md` documents topics and how to add one. *The "Event stream" section, plus the proxy-mode paragraph.*
- [x] `make test` passes. *Lint, 1203 frontend tests, and the Go suite; `go test -race ./...` passes too.*
