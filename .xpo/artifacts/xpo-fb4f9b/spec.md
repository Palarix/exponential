# Agent chat: background processes keep the turn running and block replies until Stop

## What

Each chat keeps one live agent process across turns. A turn ends when the agent replies. Background tasks that the agent started keep running after the turn ends. The composer shows a "Monitoring N background tasks" banner with a Stop button. When a background task ends, the agent can start a turn by itself. That turn shows in the chat as an agent turn. This applies to Claude Code and Codex.

## Why

Today a turn is one agent process (xpo-7bc86e: "One process per turn stays"). A measurement with Claude Code shows why this breaks with background tasks:

- `claude -p --input-format stream-json` does not exit while it has background tasks, even after stdin closes.
- When a task ends, Claude Code emits `task_notification` and starts a new turn by itself (`system init`, assistant text, a second `result`).
- The driver reads stdout until EOF (`internal/chat/claude.go:85-101`). A server that never exits keeps the turn `running` forever. The composer shows Stop and cannot send.
- When a task ends, the agent's follow-up turn streams into the same xpo turn. Its `result` replaces the first one.

Codex has the opposite bug: `codexProc.stop()` kills the process group 3 s after `turn/completed`. A server that the agent started dies with it.

A second measurement shows the fix works. With stdin kept open, a second user message sent while a background task runs gets its own `result`. The task's follow-up turn comes later with a third `result`.

Elysium (`apps/server/src/provider/Layers/ClaudeAdapter.ts`, `orchestration/ThreadBackgroundLiveness.ts`, `provider/Layers/ProviderSessionReaper.ts`) uses the same model: one live session per thread, synthetic turns for agent output between user turns, an in-memory background task registry, a "Monitoring" banner with Stop, and an idle reaper that skips threads with live tasks.

## Acceptance Criteria

- [ ] After the agent starts a background server and replies, the turn shows as done and the composer can send.
- [ ] The background server keeps running across later turns until the user presses Stop, the chat is deleted, or the board stops.
- [ ] While background tasks run, the composer shows a banner with a Stop button: "N agents working" when a subagent runs, else "Monitoring N tasks". A card on the title lists the task descriptions.
- [ ] Stop in the banner kills the agent process group, including the background tasks. The next message resumes the session.
- [ ] When a background task ends and the agent replies by itself, the chat shows that reply as its own turn, marked as started by the agent.
- [ ] A user message sent during an agent turn waits until that turn ends. The UI does not show the message as lost.
- [ ] A reply after `xpo board` restarts resumes the session from disk, as today.
- [ ] After a crash of `xpo board`, the next board start kills agent process groups that the crash left behind.
- [ ] A model or effort change between turns applies to the next turn without a restart. Background tasks keep running.
- [ ] The agent stays locked per chat, as today (`ErrProviderLocked`, `providerLocked` in the composer).
- [ ] Codex: the same criteria hold for the Codex driver, except agent turns. Codex does not start turns by itself.
- [ ] Regression tests (fake agent binaries) cover:
  - a turn whose background child outlives `result`
  - a second message on the live process
  - an agent turn after a task ends
  - restart of the process on a launch config change
  - a model and effort change that sends control requests, not a restart
  - the startup sweep

## Flow

### 1. Codex measurement (done)

Measured with codex-cli 0.162.1, `codex app-server`, `sandbox: workspace-write`:

- A background command is a `commandExecution` item that stays `inProgress` after `turn/completed`.
- The item gets `item/commandExecution/outputDelta` and `item/completed` later, outside any turn.
- Codex does not start a turn by itself when the command ends.
- `process/exited` and `process/outputDelta` do not fire. They belong to the `command/exec` API, not to agent commands.
- Codex starts each command in its own process group, not in the app-server's group.
- SIGTERM to `codex app-server` stops its running commands. The app-server exits with code 0 within 4 s.
- The agent sometimes uses `nohup … &` instead. That item completes at once, so Codex does not track the process.

### 2. Driver interface (`internal/chat/driver.go`)

1. Replace `Driver.Turn` with `Driver.Open(SessionInput, SessionHandler) (Session, error)`.
2. Add `SessionInput`: session ID, model, effort, dir, MCP target, profile, approver, grants.
3. Add `Session` with:
   - `Send(prompt, model, effort string) error`
   - `Interrupt()`
   - `Close()`
   - `PID() int`
   - `Send` returns `errAgentExited` when the agent already exited. The runner then starts the agent once more and resends.
4. Add `SessionHandler` with:
   - `AgentTurn(reason string)`: a turn the agent started on its own. `reason` is the summary of the task that ended, when known.
   - `Stream(exponential.StreamEvent)`
   - `TurnEnded(*TurnResult, error)`
   - `Tasks([]BackgroundTask)`
   - `Exited(error)`
5. Add `BackgroundTask{ID, Kind, Description}`. `Kind` is `agent` (subagents, workflows) or `monitor` (background shells, monitors). The driver maps the agent's task type to `Kind`.
6. Add `LaunchKey(SessionInput) string`. It covers provider, dir and profile tools. It does not cover model, effort or grants.

### 3. Claude driver (`internal/chat/claude.go`)

1. Start `claude` once in `Open`. Keep stdin open.
2. Read stdout in one goroutine for the life of the process.
3. In `Send`, compare model and effort with the session's current values.
   - If the model differs, send a `set_model` control request (`{"subtype":"set_model","model":…}`).
   - If the effort differs, send an `apply_flag_settings` control request (`{"subtype":"apply_flag_settings","settings":{"effortLevel":…}}`). An empty effort sends `null`.
   - Wait for each `control_response`. On an error response, return the error and do not send the prompt.
   - Write a `claudeUserMessage` line. Mark the next turn as started by the user.
4. On turn content (`system init`, `assistant`, `stream_event`, `user`) with no turn running, call `AgentTurn` with the last ended task's summary.
5. On `result`, call `TurnEnded`. Do not close stdin.
6. On `task_started`, `task_updated`, `task_notification` and `background_tasks_changed`, update the task list. Then call `Tasks`. `background_tasks_changed` is Claude Code's own roster and replaces the list. Task types `monitor`, `monitor_mcp`, `local_bash` and `shell` are `monitor`; `plan` and `dream` are skipped; all others are `agent` (Elysium's lists).
7. Keep answering `control_request` lines for the life of the process. Bind each approval to the turn that is running.
8. In `Interrupt`, send the `interrupt` control request.
9. In `Close`, send SIGTERM to the `claude` PID. Claude Code then stops its background shells (measured). When the process exited, or after 3 s, send SIGKILL to the process group.
10. On stdout EOF, call `Exited` with the stderr tail.
11. Read stdout from an `os.Pipe` of xpo's own. After the process exits, stop reading within 2 s, even when a detached child keeps the write end open.
12. Before `TurnEnded`, end the turn's open approval requests and wait for them, so `approval_resolved` settles before the turn's end.

### 4. Codex driver (`internal/chat/codex.go`)

1. Start `codex app-server` once in `Open`. Run `initialize`, then `thread/start` or `thread/resume`.
2. In `Send`, call `turn/start` with `model` and `effort`. Codex takes both per turn, so a change needs no extra call.
3. Codex does not start turns by itself (step 1). On `turn/started` for a turn that xpo did not request, call `TurnStarted(true)` anyway.
4. On `turn/completed`, call `TurnEnded`. Do not stop the process.
5. Track `commandExecution` items that are still `inProgress` at `turn/completed`. Report them to `Tasks` with `Kind` `monitor`.
6. On `item/completed` for a tracked item, remove it from the list. Call `Tasks`.
7. In `Interrupt`, call `turn/interrupt`.
8. In `Close`, send SIGTERM to the app-server PID. After 3 s, send SIGKILL to the process group.

### 5. Runner (`internal/chat/runner.go`)

1. Add `sessions map[string]*liveSession`. A `liveSession` holds the `Session`, its launch key, the task list and the last activity time.
2. In `Send`:
   - If a turn runs (user or agent), return `ErrBusy`. Keep today's behavior.
   - Compute the launch key from the meta and profile.
   - If a live session has a different key, close it.
   - If no live session exists, call `Driver.Open` with the stored session ID.
   - Call `Session.Send`.
3. Move the per-turn bookkeeping out of `run` into the handler: text buffer, pending tool calls, settle, `TurnResult` handling, mode apply and hand-over.
4. On `TurnStarted(true)`, register a turn for the chat. Settle `EventAgentTurn`. Set `meta.State` to `running`.
5. On `TurnEnded`, settle `turn_done` or `error` as `run` does today. Set `meta.State` to `idle`.
6. On `Tasks`, store the list. Publish `EventBackgroundTasks` live (not settled).
7. On `Exited`, end a running turn with `error`, clear the tasks, and remove the live session.
8. Change `Cancel` to `Session.Interrupt` while a turn runs. It does not kill background tasks. If the turn did not end after `InterruptGrace` (3 s), close the session. Open approval cards resolve as `cancelled` at once.
9. Add `StopBackground(id)`. It closes the live session and clears the tasks.
10. On chat delete, close the live session.
11. In `Shutdown`, close every live session and wait for them to exit.
12. Add an idle reaper. Every 5 minutes, close live sessions that have no running turn, no tasks, and no activity for 30 minutes. The timeout is a constant, not a setting.

### 6. Crash cleanup (`internal/chat/store.go`, `proc_unix.go`)

1. Store the agent PID and its start time in the chat meta (`Meta.AgentPID`, `Meta.AgentStarted`) when the session opens.
2. Clear both fields when the session exits.
3. On board start, check each chat with a non-zero `Meta.AgentPID`. If a process with that PID and start time (within 3 s, from `LC_ALL=C ps -o lstart=`) exists, send SIGTERM to its process group. Then clear the fields.
4. Run this sweep before the board accepts requests.

### 7. HTTP API (`internal/server`)

1. Add `POST /api/local/chats/{id}/background/stop`. It calls `Runner.StopBackground`.
2. Add `background` (the task list) to the chat response.
3. Send `background_tasks` events on the chat SSE stream.

### 8. Frontend (`web/src/components/Agent`)

1. In `chat-thread-utils.ts`, open a turn on `agent_turn` events. Mark it `byAgent`.
2. Render an agent turn without a user bubble. Show a small line with a bot icon and the reason ("The agent continued on its own" when empty).
3. Track the background task list from the chat response and `background_tasks` events.
4. In `ChatComposer.tsx`, show a banner above the input while the list is not empty and no turn runs.
   - If any task is a subagent, the banner reads "1 agent working" or "N agents working".
   - Otherwise the banner reads "Monitoring 1 task" or "Monitoring N tasks". The tasks are shells and monitors only.
   - Hover, focus or click on the title opens a card that lists each task's description.
5. Add a Stop button to the banner. It calls the new endpoint and shows "Stopping…" until the list is empty.
6. Keep the composer's Stop button for the running turn only.

## Decisions

- **One live agent process per chat, not one per turn.** Background tasks live inside the agent process. Ending the process ends them. This reverses "One process per turn stays" from xpo-7bc86e.
- **No new MCP tools for background processes.** The agent uses its own background tasks. xpo shows and stops them.
- **Task state stays in memory, not in the event log.** After a restart no task is live. Elysium uses the same rule.
- **Stop in the banner ends the whole session.** Claude Code's `interrupt` can return while background tasks keep the process alive. Closing the session is the only reliable stop. The next message resumes from disk.
- **The composer's Stop interrupts the turn only.** It does not kill background tasks. Today it kills the process group.
- **A message during an agent turn returns `ErrBusy`.** Elysium sends it into the running turn instead. That is a separate feature.
- **Grants are not part of the launch key.** The runner's approver already allows granted rules (`Runner.granted`). A new grant does not restart the session or kill tasks.
- **A launch key change restarts the session.** This is a mode or worktree change. Background tasks die.
- **Model and effort switch in place.** Claude Code takes `set_model` and `apply_flag_settings` control requests in streaming input mode. Codex takes `model` and `effort` on each `turn/start`. The live process and its background tasks survive the switch. Measured with Claude Code 2.1.299: haiku/low → sonnet/high between turns, the next `system init` reports the new model, `CLAUDE_EFFORT` reads `high`, and a background `sleep` keeps running.
- **The agent stays locked per chat.** A Claude session cannot move to Codex. The lock exists today and does not change.
- **Keep the agent in its own process group.** Ctrl-C in the terminal then reaches only xpo, and `Shutdown` stops the agents in order. A group kill also stops the MCP servers that the agent started. The group does not reach Claude Code's Bash commands: Claude Code starts each one in its own process group (measured). Claude Code stops them itself on SIGTERM (measured), so xpo always sends SIGTERM to the agent PID first.
- **SIGTERM on the recorded agent PID at startup.** A crash or `kill -9` of xpo leaves the agents running, with or without a process group. macOS has no `Pdeathsig`. The sweep checks the PID and the process start time, so it does not kill a process that reused the PID. Elysium has no such sweep. It only marks orphaned sessions as failed.

## Edge Cases

**HIGH (developer decides)**

- A mode change (Chat → Work, Work → Chat) restarts the agent. The user sees no extra line for the restart. The `mode_changed` line stays the only line.
  - Chat Mode has no Bash, so Chat → Work never has background tasks.
  - If Work → Chat kills live tasks, the runner adds "Background tasks stopped." to the `mode_changed` event's `Message`. The chat shows one line, not two.
- The model switch changes the prompt cache. The first turn on the new model reads the full context at the uncached price. xpo does not warn about this.

**MEDIUM (agent proposes, developer confirms)**

- The agent process crashes between turns. `Exited` clears the tasks. The next message opens a new session with `--resume`.
- An agent turn asks for approval. The approval card shows as today. A user message is `ErrBusy` until the turn ends.
- The Work Mode hand-over (xpo-ad6069) changes the dir. The runner closes the session and opens a new one in the worktree.
- A background task outlives an agent turn that the agent started by itself. The banner stays. The idle reaper skips the chat.

- The agent detaches a process with `nohup … &` or `cmd &`. Neither agent tracks it, so xpo cannot show or stop it. The process outlives the chat. xpo does not handle this case.

**LOW (agent handles)**

- `background_tasks_changed` comes before `task_started`. The task list uses the latest snapshot.
- A `result` arrives with no turn registered. The runner ignores it.
- The chat SSE stream reconnects. The chat response carries the current task list.

## Assumptions

- Claude Code keeps the session file on disk up to date during a live process. `--resume` after a kill loses no settled turn.
- One live process per chat is cheap enough. The idle reaper bounds the count.
- Claude Code's `interrupt` control request stops the running turn and keeps the process alive.
- Claude Code 2.1.299 emits `system init` at the start of every turn, including turns it starts by itself (measured).

## Open Questions

None.
