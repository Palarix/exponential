# Agent chat: queue, steer, and edit the last message

## What

The chat composer keeps working while a turn runs. A message sent during a turn waits in a queue on the board, shown above the composer in every client. The board sends it when the turn ends. **Steer** sends a queued message into the running turn. When the chat is idle, the user can edit the last user message and send it again. The board then drops the old turn from the transcript and from the agent session.

## Why

Before this change, the composer blocked Send while a turn ran (`ChatComposer.tsx` `canSend`), and the runner refused a second message with `ErrBusy`. The user had to wait, copy the thought somewhere, or press Stop. Modern agent composers (Claude Code, Codex, Cursor) queue, steer, and edit. Both agents support steering and rewinding in their current protocols.

## Research: what the agents support

| | Claude Code 2.1.302 (stream-json) | Codex 0.162.1 (app-server) |
|---|---|---|
| Steer | Write a `user` line to stdin during a turn. Claude reads it before its next model call. | `turn/steer {threadId, expectedTurnId, input}`. Codex refuses it when the turn has ended. |
| Rewind | Start with `--resume <sid> --resume-session-at <uuid>`. Claude keeps the messages up to and including that assistant message. | `thread/revert {threadId, beforeTurnId}` drops that turn and all later turns. |
| Message IDs | Every stream-json line carries a `uuid`. Claude has no request that lists a session's messages. | `turn/start` and `turn/started` give the turn ID. |

Neither rewind reverts file changes on disk. The old turn's edits stay in the worktree.

## Acceptance Criteria

- [ ] Sending while a turn runs queues the message on the board. The board sends it when the turn ends with `turn_done`, with or without a browser open.
- [ ] Queued messages go out as one combined message.
- [ ] Every client of the chat shows the same queue, whichever client queued the message. The queue lives in the board's memory: a board restart drops it.
- [ ] A queued message can be edited, removed, or sent now via **Steer** (`CornerDownRight` icon).
- [ ] Stop keeps the queue and does not send it. An error or an interrupted turn also keeps it.
- [ ] A steered message shows in the running turn as a user message. The agent changes course without a restart. This works for Claude and Codex.
- [ ] Each turn-ending event records the turn's anchor (Claude: last assistant `uuid`; Codex: turn ID).
- [ ] When the chat is idle and not archived, and the anchor it needs exists, the last user message has an **Edit** action. Its text and attachments go back into the composer.
- [ ] Chats without anchors (from before this change) show no Edit action. The server refuses such an edit with 409.
- [ ] ↑ in an empty composer loads the last user message for editing, under the same rules.
- [ ] Sending an edit replaces that turn. The transcript and the agent session no longer contain the old turn.
- [ ] ↑ with queued messages pulls the last queued message back into the composer.
- [ ] Esc or the **Cancel** button ends edit mode and brings back the draft that was in the composer.
- [ ] `make test` passes. Go tests cover the queue, steer, anchors, and rewind for both fake agents. Vitest covers the reducer and the composer. Opt-in tests (`XPO_TEST_REAL_CLI=1`) check steer and rewind against the real CLIs.

## Flow

### Queue (board)

1. Add `QueuedMessage` in `internal/chat/queue.go`: `id`, `text`, `context`, `attachments` (saved), `model`, `effort`, `time`. The runner keeps a list per chat in memory, under `r.mu`.
2. Add `Runner.SendOrQueue(id, req)`. It sends as `Send` does. When a turn runs, it saves the attachments and appends the message to the queue instead. An edit is never queued: it gets `ErrBusy`.
3. Publish the queue after each change as a live `queue` event with the whole list (`Event.Queue`), like `background_tasks`. `GET /chats/{id}` returns it as `queue`.
4. In `Runner.endTurn`, drain the queue when the turn finished as `turn_done`, was not stopped, and the board is not closing. Register the next turn under `r.mu` (`handOverLocked`), so no message gets in between. A Work Mode hand-over goes first; the queue drains after that turn.
5. The drained turn sends one message: the texts joined with a blank line, every attachment, the latest context, and the latest model and effort.
6. Stop, an error, or an interrupted turn keeps the queue. The next finished turn drains it.
7. Add `DELETE /chats/{id}/queue/{qid}`. It removes the message and returns it; 404 when it is gone. Archive and delete clear the chat's queue.
8. `POST /chats/{id}/messages` answers `{ok, queued}`.

### Queue (web)

9. The thread state holds the queue, from `fetchChat` and from the live `queue` event.
10. While a turn runs and the input has content, the composer shows a **Queue message** button next to Stop. Enter also queues.
11. Render the queue as chips above the composer. Each chip shows the text (one line, truncated), an **Edit** button, a **Steer** button, and a remove (×) button.
12. Chip **Edit** removes the message on the board. It puts the text and attachments (fetched back as files) in this client's composer. It is disabled while the input has content.

### Steer

13. Add `POST /chats/{id}/queue/{qid}/steer`. It returns 409 when no turn takes the message, and 404 when the message is gone.
14. Add `Runner.SteerQueued(id, qid)` in `queue.go`. It calls `Session.Steer` with the message and its saved attachments. On success it removes the message and settles an `EventSteer` (`steer`) event.
15. Add `Steer(Prompt) error` to the `Session` interface in `driver.go`.
16. In `claude.go`, implement `Steer` as a stdin write of the `claudeUserMessage` line, only while a turn runs.
17. In `codex.go`, implement `Steer` as a `turn/steer` call with the current `turnID`. It waits for the answer; a refusal wraps `ErrNotRunning`.
18. In `chat-thread-utils.ts`, apply `steer` to the running turn as a `SteerStep`. `ChatThread.tsx` renders it as a user bubble with a `CornerDownRight` icon inside the turn.
19. On error, the chip's Steer button shows a toast and the chip stays. Steer is disabled while no turn runs, while a card waits for the user, and while a steer is on its way.

### Anchors

20. Add `Anchor string` (`anchor,omitempty`) to `Event` and to `TurnResult`.
21. In `claude.go`, keep the `uuid` of the latest top-level `assistant` line of the turn. A subagent's lines (`parent_tool_use_id` set) do not count.
22. In `codex.go`, put the turn ID of `turn/completed` in `TurnResult.Anchor`.
23. In `Runner.endTurn`, copy the anchor to the event it settles: `turn_done`, `turn_stopped`, or `error`. A turn that ends without a result has no anchor.

### Edit the last message

24. Add `Edit bool` to `SendRequest` and `edit` to the `/messages` body.
25. Add `editPoint(events, provider)` in `internal/chat/rewind.go`. It ignores events an earlier rewind dropped, and finds the last `user_message`:
    - Claude needs the anchor of the turn before. The first message needs none: the edit starts a new session.
    - Codex needs the anchor of the edited turn itself.
26. In `Runner.Send`, when `Edit` is set, call `editPoint`. Return `ErrCannotEdit` (409) when there is no message, an anchor is missing, or the chat belongs to another agent.
27. Settle an `EventRewound` (`rewound`) event with `FromSeq` set to the edited message's seq, before any other event of the turn. Then run the turn as usual.
28. In `Runner.start`, close the chat's live agent for an edit. Open a new one with `SessionInput.Rewind` set to the anchor.
29. Claude: start with `--resume <sid> --resume-session-at <anchor>`. A first-message edit starts a fresh session without `--resume`.
30. Codex: after `thread/resume`, call `thread/revert {threadId, beforeTurnId: <anchor>}`. The turn starts once the revert succeeds. A refused revert ends the turn with an error.
31. In `chatThreadReducer`, on `rewound`, drop every turn whose start seq is at or after `from_seq`, and the dividers after them. Store each turn's anchor from its ending event.
32. Add `editTarget(thread, agent)` in `chat-thread-utils.ts`. It mirrors `editPoint`: idle, and the needed anchor exists.
33. In `ChatThread.tsx`, show an **Edit** button (Pencil icon) under the editable user message. Its tooltip says "Edit and send again. Files the agent changed stay changed."
34. Edit loads the message's text and attachments (fetched back as files) with `drafts.startEdit`. The draft that was in the composer waits. The composer shows "Editing your last message" with a **Cancel** button.
35. In edit mode, Send calls `sendChatMessage` with `edit: true`. On success the waiting draft comes back.
36. In the composer `onKeyDown`, `composerArrowUp` handles ↑ when the input is empty:
    - With queued items, pull the last queued item into the composer.
    - Else, when Edit is possible, load the last user message in edit mode.
    - Else, do nothing.

## Decisions

- **The queue lives on the board, not in the browser.** Drafts are private to one client, but a queue is chat state: every client shows it, and it drains with no browser open. A first version kept it in the browser; review moved it.
- **The queue lives in the board's memory, not in the chat's meta.** It matches the running turn and the background tasks. A board restart interrupts the turn, so a saved queue would not drain on its own either.
- **The board decides between send and queue.** A client cannot know for sure that a turn runs. `SendOrQueue` checks under the runner's lock.
- **Join queued messages into one send, not one turn per message.** Claude Code does the same. One turn is cheaper and the agent sees the whole follow-up at once.
- **↑ acts only in an empty composer, not "caret at the start".** With text in the composer, ↑ at the start would overwrite the user's text. ↑ in a non-empty composer moves the caret as usual.
- **Record anchors as turns end, not read Claude's session file.** The session file is Claude's private format. Anchors come from the stream the driver already parses. Edit works only on chats recorded after this change.
- **Anchors live on the turn-ending event, not in `Meta`.** The event log is the per-turn record already (cost, tokens). A `rewound` event drops the old turns, so later edits find the right anchors.
- **Rewind is an appended `rewound` event, not a rewrite of the `.jsonl` log.** The store stays append-only. Cost totals keep the old turn: its tokens were spent.
- **An edit starts a new agent process for both agents.** Claude can only rewind at start. Codex could revert in place, but one path for both is simpler. The cost: background tasks of the chat stop on an edit.
- **Codex rewinds with `thread/revert`, not `thread/fork`.** Revert keeps one thread ID per chat. Fork would change `Meta.SessionID` on every edit.
- **Claude rewinds in place with `--resume-session-at`, without `--fork-session`.** The chat keeps its session ID.
- **Steer settles its event after the agent took it.** A refused steer leaves nothing in the log, and the chip stays.

## Edge Cases

- HIGH — **File changes stay.** Edit drops the old turn from the conversation only. The agent's file edits stay in the worktree. The Edit tooltip says so.
- MEDIUM — **Claude reads a steer only after the turn ended.** Claude then runs it as its own turn. The driver reports it as an agent-started turn; the steer bubble stays in the turn before.
- MEDIUM — **The turn ends while the user clicks Steer.** The server returns 409. The chip stays in the queue and drains with the turn end.
- MEDIUM — **Two clients pull the same message into their composers.** The second gets 404 and a toast.
- MEDIUM — **Edit after an agent-started turn.** The edit drops the last user turn and every turn after it, including agent-started turns.
- MEDIUM — **The turn before ended without an anchor** (Claude), or the edited turn did (Codex). Edit is hidden, and the server refuses it.
- MEDIUM — **A turn starts while the user edits** (an agent-started turn). Sending the edit gets 409 and a toast; the edit stays in the composer.
- LOW — Archive and delete clear the chat's queue.
- LOW — Steer with attachments: images go in as image input, like a normal send.

## Assumptions

- `--resume-session-at` accepts the `uuid` of an `assistant` stream-json line, and `thread/revert` accepts a turn ID. `TestRealClaude_EditRewinds` and `TestRealCodex_EditRewinds` check this.
- Edit applies to the last user message only. Editing older messages is out of scope.
- The board does not restore files on edit.
