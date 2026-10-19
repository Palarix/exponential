# Walkthrough: queue, steer, and edit the last message

## What changed

Before, the composer blocked Send while a turn ran, and the runner refused a second message with `ErrBusy`. Now:

- A message sent during a turn waits in a **queue on the board**. Every client shows it as chips above the composer. The board sends the queue as one message when the turn is done.
- A queued message can **steer** the running turn: the agent gets it mid-turn and changes course without a stop.
- The last user message can be **edited**: the board drops the old turn from the transcript and from the agent's own session, then runs the edited message.

## How the pieces fit

```
composer ──POST /messages──► SendOrQueue ──busy?──► queue (runner memory) ──live "queue" event──► every client
                                   │                       │
                                   ▼ idle                  ├─ endTurn(turn_done) ─► one combined turn
                                 begin()                   ├─ /queue/{qid}/steer ─► Session.Steer
                                                           └─ DELETE /queue/{qid} ─► back into one composer
```

### 1. The queue lives in the runner (`internal/chat/queue.go`)

`Runner.queues` maps a chat ID to its `QueuedMessage` list, under `r.mu`, next to `turns` and `sessions`.

- **Send or queue:** `SendOrQueue` runs the old `Send` checks, then asks under the lock whether a turn runs. If one does, `enqueue` saves the attachments and appends the message. If the turn ended while the attachments were being saved, `enqueue` sends the message after all, so a message cannot get stuck in an idle chat's queue. `Send` itself keeps returning `ErrBusy`; edits always go through that path.
- **Sharing:** every change publishes a live `queue` event with the whole list, the same pattern as `background_tasks`. `GET /chats/{id}` returns `queue` for a client that opens the chat later.
- **Draining:** `endTurn` drains the queue only after a turn that finished with `turn_done`, was not stopped, and while the board is not shutting down. It takes the list and registers the next turn with `handOverLocked` in the same critical section, so no other message can start a turn in between. `sendQueued` then joins the texts with a blank line, keeps every attachment, takes the latest context, model and effort, and calls `begin`.
- **Work Mode first:** an approved Work Mode hand-over goes before the queue. The queue drains when that continued turn ends.

`Send` was split so the drained turn can reuse it. `send` validates and registers the turn, and `begin` does the rest: meta, attachments, events, prompt, and `go r.start`. `begin` takes already-saved attachments, because queued messages saved theirs when they were queued.

### 2. Steer (`Session.Steer`, `Runner.SteerQueued`)

- **Claude:** `claudeSession.Steer` writes a stream-json user line while `inTurn`. A real run showed that Claude reads it before its next model call, in the same turn, with one `result`.
- **Codex:** `codexSession.Steer` sends `turn/steer` with `expectedTurnId` and waits for the answer. A refusal wraps `ErrNotRunning`. The `steers` map routes the response; the request is registered under `s.mu`, so the reader cannot answer before it is registered.
- **Runner:** `SteerQueued` takes the message out of the queue first, so two clients cannot steer it twice. If the agent refuses, the message goes back in its old place. The `steer` event settles only after the agent took the message, so a refused steer leaves nothing in the log.

### 3. Anchors

To rewind a session, the board needs to know where each turn ended in the agent's own terms. Neither agent can list a session's messages, so the drivers record that point as turns end:

- **Claude:** the `uuid` of the turn's last top-level `assistant` line. Lines with `parent_tool_use_id` belong to a subagent's side chain and are skipped.
- **Codex:** the turn ID from `turn/completed`.

`endTurn` writes the anchor on whichever event ends the turn: `turn_done`, `turn_stopped` or `error`. It sits next to cost and tokens, so nothing new goes into `Meta`.

### 4. Edit (`internal/chat/rewind.go`)

- **Finding the cut:** `editPoint(events, provider)` first applies earlier rewinds (`activeEvents`), then finds the last `user_message`.
  - Claude resumes *at* the anchor of the turn before. A first-message edit starts a fresh session.
  - Codex reverts *before* the edited turn's own ID.
  - If the turn before has a start event but no end event (the board died mid-turn), the edit is refused rather than guessing.
- **Running the edit:** `begin` settles `rewound{from_seq}` before any other event of the turn. `start` then closes the live agent and opens a new one with `SessionInput.Rewind`.
  - Claude gets `--resume <sid> --resume-session-at <uuid>`.
  - Codex sends `thread/revert` after `thread/resume`, and only marks the thread ready once the revert succeeds.

### 5. Web

- **Thread state:** `chatThreadReducer` keeps `queue` from the live event and the fetch. It turns `steer` into a `SteerStep` inside the running turn, drops turns at or after `from_seq` on `rewound`, and stores each turn's `anchor`. `editTarget` mirrors `editPoint`, so the Edit button only shows when the board would accept the edit.
- **Composer:**
  - queue chips with Edit, Steer and remove;
  - a "Queue message" button next to Stop;
  - an edit bar with Cancel;
  - `composerArrowUp` for ↑, which acts only in an empty input.
- **Drafts:** `startEdit` and `cancelEdit` park the draft that was in the composer, and `sent` brings it back after an edit goes out.
- **Wiring:** `useChatSession` connects all of this to `steerQueued`, `removeQueued` and `sendChatMessage({edit})`. Attachments are fetched back as `File`s when a message returns to the composer.

## Key decisions

- **The queue lives on the board, not in the browser.** The first version kept it in the browser like drafts. Review pointed out that a queue is chat state: every client should show it, whichever client queued the message. Drafts stay per client.
- **It lives in memory, not in the chat's meta.** That matches the running turn and the background tasks. A restart interrupts the turn, so a saved queue would not drain on its own anyway.
- **One combined message, not one turn per queued item.** It is cheaper, and the agent sees the whole follow-up at once.
- **An edit restarts the agent for both providers.** Claude can only rewind at start. Codex could revert in place, but one path is simpler. The cost: background tasks of that chat stop on an edit.
- **Anchors are recorded as turns end, not read from Claude's session files.** Those files are Claude's private format. The trade-off: chats from before this change cannot be edited.
- **Rewind appends an event and never rewrites the log.** The store stays append-only. Cost totals keep the dropped turn, because its tokens were spent.

## Non-obvious details

- If Claude reads a steer only after the turn's `result`, it runs the steer as its own turn. That turn shows as agent-started.
- Codex steering needs the turn ID. A steer sent before `turn/start` answers gets 409, and the chip stays.
- In the real-CLI edit tests, the edited turn's reply ignores the dropped turn's "PEAR" instruction. That is the actual check that the session, and not only the transcript, was rewound.

## Acceptance criteria

- [x] Sending while a turn runs queues the message on the board, and it is sent when the turn ends with `turn_done`, with or without a browser open. Evidence: `TestRunner_QueueDrainsAsOneMessage`, `TestChat_SendWhileRunningQueues`, and an HTTP check on the demo board: the second POST answered `queued:true` and drained as its own turn.
- [x] Queued messages go out as one combined message. Evidence: `TestRunner_QueueDrainsAsOneMessage`, where "two" and "three" arrive as `two\n\nthree` with the attachment and the latest context.
- [x] Every client shows the same queue; it lives in the board's memory. Evidence: the live `queue` event sizes `[1 2 0]` in `TestRunner_QueueDrainsAsOneMessage`, and `GET` returning `queue` in `TestChat_SendWhileRunningQueues`.
- [x] A queued message can be edited, removed, or steered. Evidence: `TestRunner_RemoveQueued`, `TestRunner_SteerGoesIntoTheRunningTurn`, the "ChatComposer queue" tests, and the `DELETE /queue/{qid}` checks.
- [x] Stop keeps the queue and does not send it; errors and interrupts keep it too. Evidence: `TestRunner_StopKeepsTheQueue` and the `drain` condition in `endTurn`.
- [x] A steered message shows in the running turn, and the agent changes course without a restart, for Claude and Codex. Evidence: `TestRealClaude_Steer` and `TestRealCodex_Steer` (the reply ends with BANANA), `TestCodexDriver_Steer`, and the "ChatThread steer" test.
- [x] Each turn-ending event records its anchor. Evidence: `TestRunner_TurnEndRecordsAnchor` and `TestCodexDriver_TurnEndRecordsAnchor`.
- [x] Edit shows on the last user message when idle, not archived, and anchored; text and attachments return to the composer. Evidence: `editTarget` tests, "ChatThread edit action", and the "composer drafts: edit mode" tests.
- [x] Chats without anchors show no Edit, and the server refuses it with 409. Evidence: `TestRunner_EditNeedsAnAnchor` and the `editTarget` "needs Claude's anchor" test.
- [x] ↑ in an empty composer loads the last message for editing. Evidence: the `composerArrowUp` tests.
- [x] Sending an edit replaces the turn in the transcript and the agent session. Evidence: `TestRunner_EditRewindsClaude`, `TestCodexDriver_EditRevertsTheTurn`, `TestRealClaude_EditRewinds`, `TestRealCodex_EditRewinds`, and the "drops the turns a rewind dropped" reducer test.
- [x] ↑ with queued messages pulls the last one back. Evidence: the `composerArrowUp` "pulls the last queued message first" test.
- [x] Esc or Cancel ends edit mode and restores the earlier draft. Evidence: the "brings the old draft back on cancel" and "ChatComposer edit mode" tests.
- [x] `make test` passes, and the opt-in real-CLI tests pass against Claude Code 2.1.302 and codex-cli 0.162.1.
