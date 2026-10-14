# Walkthrough: background processes no longer keep the turn running

## What was built and why

Before this change, a chat turn was one agent process. The turn ended when the process exited. Claude Code does not exit while it has background tasks, even after stdin closes. A dev server that the agent started kept the turn `running` forever, and the composer showed Stop instead of Send.

Now each chat keeps one live agent process across turns. A turn ends when the agent replies (`result` for Claude Code, `turn/completed` for Codex). The process stays, and its background tasks keep running. The design follows Elysium: one live session per thread, turns the agent starts on its own, an in-memory task list, a banner with Stop, and an idle reaper.

## How the pieces fit

### 1. Driver interface (`internal/chat/driver.go`)

`Driver.Turn` is replaced by `Driver.Open(SessionInput, SessionHandler) (Session, error)`.

- `Session`: `Send(prompt, model, effort)`, `Interrupt()`, `Close()`, `PID()`.
- `SessionHandler` gets the agent's output from one reader goroutine, in order:
  - `AgentTurn(reason)`
  - `Stream(ev)`
  - `TurnEnded(res, err)`
  - `Tasks(list)`
  - `Exited(err)`
- `launchKey(provider, profile)` covers what a live process cannot change: the agent, its directory and its tools. Model, effort and grants are not part of it. They change in place.

### 2. The agent process (`internal/chat/proc.go`, `proc_unix.go`)

`startAgent` starts the CLI in its own process group, so a Ctrl-C in the terminal reaches only xpo.

- **stdout:** stdout is an `os.Pipe` that xpo owns. After the process exits, the reader stops within 2 s, even when a detached child keeps the write end open. This was the second half of the original hang.
- **`stop(grace)`:**
  1. It closes stdin and sends SIGTERM to the agent PID. Claude Code and Codex then stop their background commands (measured). They run them in process groups of their own, so a group kill alone would miss them.
  2. When the agent has exited, or after grace, it SIGKILLs what is left of the group, such as MCP servers.

### 3. Claude driver (`internal/chat/claude.go`)

- **One process per chat:** each user message is one more stream-json line on stdin.
- **Model and effort:** `Send` switches a different model or effort first, with `set_model` and `apply_flag_settings` (`effortLevel`) control requests. It waits for their `control_response`.
- **Background tasks:** `background_tasks_changed` (Claude Code's own roster) and the `task_*` lines track them.
  - `local_bash`, `shell`, `monitor` and `monitor_mcp` are `monitor` tasks.
  - `plan` and `dream` are skipped.
  - Everything else is an `agent` task.
- **Agent turns:** turn content while no turn runs is a turn the agent started on its own. The driver calls `AgentTurn` with the summary of the last `task_notification`.
- **Approvals:** before `TurnEnded`, the driver ends the turn's approval requests and waits for them. Their cards therefore settle before the turn's end event.

### 4. Codex driver (`internal/chat/codex.go`)

- **One app-server and one thread per chat:** a `Send` before the thread is ready starts its turn once the thread is ready.
- **Model and effort:** both go on each `turn/start`, so a switch needs no extra call.
- **Background commands:** a `commandExecution` item still `inProgress` at `turn/completed` becomes a background task. Its later `item/completed` removes it. Codex never starts a turn on its own (measured with codex-cli 0.162.1).

### 5. Runner (`internal/chat/runner.go`, `session.go`)

- **`Send`:** registers the turn and settles the user message. `start` then sends the prompt to the chat's live session. It opens a session when there is none or the launch key changed. If the agent already exited, `Send` returns `errAgentExited`, and `start` starts the agent once more.
- **Per-turn state:** the text buffer and pending tool calls moved from the old `run` closure onto `turn`. `stream` and `endTurn` work on it. `endTurn` runs once per turn and keeps the old meta updates, mode apply and Work Mode hand-over.
- **`Cancel`:** sends `Interrupt` and leaves background tasks alone. If the turn has not ended after `InterruptGrace` (3 s), it closes the agent. The approver also watches the turn's context, so open cards resolve as `cancelled` at once.
- **`StopBackground`:** closes the agent with its tasks (the banner's Stop, and chat delete).
- **Mode changes** (`detachLocked`): `BindWork`, `LeaveWork`, merge, an approved start and a missing worktree take the live session out and close it. When it had tasks, the `mode_changed` message becomes one line: "Entered Chat Mode. Background tasks stopped."
- **Idle reaper:** every 5 minutes, it closes agents that have no turn, no task and no activity for 30 minutes.
- **Shutdown:** closes every agent. The closers are kept in a list under the mutex, because a `WaitGroup` raced with the reaper.

### 6. Crash cleanup (`Store.StopOrphans`, `cmd/exponential/board.go`)

The live agent's PID and start time are stored in the chat meta (`agent_pid`, `agent_started`). They are cleared on exit. At board start, `StopOrphans` sends SIGTERM to each recorded group, but only when a process with that PID started at that time (±3 s, from `LC_ALL=C ps -o lstart=`). A reused PID is left alone. `LC_ALL=C` matters: macOS prints the date in the locale's order.

### 7. Server and UI

- **Server:** `POST /api/local/chats/{id}/background/stop`, a `background` field in `GET /chats/{id}`, and live `background_tasks` events on the SSE stream. Deleting a chat also closes its agent.
- **Reducer:** `ChatThread.background` comes from the chat response and the `background_tasks` events. `agent_turn` opens a turn with `byAgent`.
- **Banner:** `ChatComposer` shows it between turns.
  - `backgroundBanner` gives the title: "Monitoring N tasks", or "N agents working".
  - A Radix tooltip card on the title lists the task descriptions.
  - Stop shows "Stopping…" until the list is empty.
- **Agent turns:** `ChatThread` renders an agent turn with a bot icon and the reason, without a user bubble.

## Key decisions

- **One live process per chat, not one per turn.** This reverses "One process per turn stays" from xpo-7bc86e. Background tasks live inside the agent process, so only a live process can keep them. Resume from disk still works whenever no process is alive.
- **No new MCP tools.** The agent uses its own background tasks. xpo shows them and stops them.
- **Task state stays in memory.** After a restart no task is live, so nothing needs persisting.
- **A message during an agent turn returns `ErrBusy`.** Sending into a running turn (Elysium's steering) is a separate feature.
- **Fakes live across turns.** The fake `claude` and `codex` scripts in the tests now loop over stdin like the real CLIs. Tests that need a new process call `StopBackground` first.

## Acceptance criteria

- [x] After the agent starts a background server and replies, the turn shows as done and the composer can send.
  - Tests: `TestRunner_BackgroundChildDoesNotHoldTheTurn` (a child holding stdout), `TestRealClaude_BackgroundTask` (real `sleep 20`, the turn ends at the reply).
  - Tophat: the Vite dev server in the worktree, with the turn done in 10 s and Send enabled.
- [x] The background server keeps running across later turns until the user presses Stop, the chat is deleted, or the board stops.
  - The real Claude test runs a second turn with the task still live.
  - `TestRunner_StopBackgroundEndsTheAgent` and `TestRunner_ShutdownStopsAgents` cover Stop and shutdown; `handleDeleteChat` calls `StopBackground`.
  - Tophat: Stop took the Vite server down; stopping `xpo board` left no agent or child process.
- [x] While background tasks run, the composer shows a banner with a Stop button: "N agents working" when a subagent runs, else "Monitoring N tasks". A card on the title lists the task descriptions.
  - Tests: `backgroundBanner` and `ChatComposer background banner`, `BackgroundTaskList`. Tophat confirmed.
- [x] Stop in the banner kills the agent process group, including the background tasks. The next message resumes the session.
  - Tests: `TestRunner_StopBackgroundEndsTheAgent`, `TestRunner_ResumesAfterAgentEnds`, `TestCodexDriver_ResumesThreadAfterAgentEnds`.
- [x] When a background task ends and the agent replies by itself, the chat shows that reply as its own turn, marked as started by the agent.
  - Tests: `TestRunner_BackgroundTasksAndAgentTurn`, `TestRealClaude_BackgroundTask`, and the `ChatThread agent turn` render test.
- [x] A user message sent during an agent turn waits until that turn ends. The UI does not show the message as lost.
  - `Send` returns `ErrBusy` (HTTP 409) while any turn runs. The composer shows Stop during an agent turn, as for a user turn.
- [x] A reply after `xpo board` restarts resumes the session from disk, as today.
  - With no live session, `start` opens the agent with `--resume` or `thread/resume` (`TestRunner_ResumesAfterAgentEnds`).
- [x] After a crash of `xpo board`, the next board start kills agent process groups that the crash left behind.
  - Test: `TestStore_StopOrphans`, which also covers a reused PID. Wired in `newChatService`.
- [x] A model or effort change between turns applies to the next turn without a restart. Background tasks keep running.
  - Tests: `TestRunner_ModelAndEffortSwitchInPlace` (control requests, one start) and `TestRealClaude_BackgroundTask` (haiku → sonnet on the same PID with a live task). For Codex: `TestCodexDriver_SecondTurnUsesLiveSession`.
- [x] The agent stays locked per chat, as today.
  - `ErrProviderLocked` is unchanged (`TestRunner_ProviderLockedAfterSession`).
- [x] Codex: the same criteria hold for the Codex driver, except agent turns.
  - Tests: `TestCodexDriver_BackgroundCommand` and `TestRealCodex_BackgroundCommand` (real codex-cli 0.162.1), plus the live-session and resume tests.
- [x] Regression tests (fake agent binaries) cover:
  - a child that outlives `result`
  - a second message on the live process
  - an agent turn
  - a restart on a launch config change (`TestRunner_ModeChangeRestartsTheAgentInTheWorktree`)
  - in-place model and effort switches
  - the startup sweep

`make test` passes. The chat and server packages pass `-race` five times in a row. The real-CLI tests (`XPO_TEST_REAL_CLI=1`) pass for Claude Code 2.1.299 and codex-cli 0.162.1.

## Known gaps

- **Detached processes:** a process the agent detaches with `nohup … &` or `cmd &` is not tracked by either agent. xpo cannot show it or stop it.
- **Cache cost of a model switch:** the first turn on the new model re-reads the context at the uncached price. xpo does not warn about it.
