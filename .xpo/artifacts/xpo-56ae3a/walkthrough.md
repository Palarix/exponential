# Walkthrough: Agent chat in xpo board (engine + popup)

## What we built

`xpo board` now hosts chat sessions with the user's local Claude Code. The user clicks **Agent** in the status bar, or presses ⌘J. A popup opens over any view. The user asks about issues or the code behind them, and the reply streams in. The agent reads code with Read, Grep, and Glob. It changes issues through the xpo MCP tools. It cannot run commands or edit files.

This is the first slice of Pillar 1 of xpo-3798b7. The engine underneath is the one that the drive flow (Pillar 2) will reuse: spawn an agent, stream canonical events, persist them.

## How the pieces fit

```
Browser ── POST /api/local/chats/{id}/messages ──► server/chat.go ──► chat.Runner.Send
   ▲                                                                       │
   │                                                       claude -p (one process per turn)
   │                                                       stdin: <xpo-context> + text
   │                                                       --mcp-config → xpo mcp (stdio)
   │                                                                       │ stream-json
   │                                                       ClaudeStreamParser → chat.Event
   │                                                                       │
   └── GET /api/local/chats/{id}/stream (SSE) ◄── Runner.Subscribe ◄── settle (Store) / live
```

### Backend

- **`internal/exponential/claudestream.go`** holds the stream-json parser. It was extracted from `claudeExecutor`, so drive and chat share one implementation.
  - `Parse(line, cb)` sends `StreamEvent`s and returns a `ClaudeTurnResult` for the final `result` line.
  - New `StreamSession` kind: the runner learns the session ID from `system/init`, so a stopped turn can still resume.
  - `StreamToolDone` carries the tool name for tool blocks only. That is how the runner tells a finished tool from a finished text block.
- **`internal/chat/`** is the chat engine.
  - `event.go`: the canonical `Event` and `Meta`.
    - **Settled** events (`user_message`, `assistant_text`, `tool_call`, `turn_done`, `turn_stopped`, `error`) are persisted and numbered. Replaying them rebuilds a thread.
    - **Live** events (`text_delta`, `thinking`, `tool_start`) only go to subscribers.
  - `store.go`: `.xpo/chats/<uuid>.json` for meta and `<uuid>.jsonl` for events.
    - Files are 0600.
    - IDs must parse as UUIDs, so an ID cannot escape the directory.
    - `Append` assigns `seq`. The last `seq` is cached per chat and rebuilt from the log after a restart.
  - `args.go`: the `claude` command line, the MCP config, and the `<xpo-context>` prompt prefix. The system prompt is in the embedded `prompt.md`.
  - `runner.go`:
    - At most one turn per chat (`ErrBusy`).
    - Each turn runs in its own process group (`proc_unix.go`, `proc_windows.go`), so a cancel also stops the MCP server the turn spawned.
    - Text deltas are buffered. They become one `assistant_text` when a block ends or a tool starts.
  - `agent.go`: `CheckAgent` reads `chat.agent` and `chat.model` and checks that `claude` is on `PATH`.
- **`internal/server/chat.go`** has the routes under `/api/local/`. The board answers them in local and proxy mode alike, and headless `serve` never registers them.
  - The SSE handler subscribes **before** it reads the log, so no event can fall between the two.
  - Settled events carry `id: <seq>`. A reconnecting `EventSource` resumes from `Last-Event-ID` without duplicates.
- **`cmd/exponential/board.go`** contains `newChatService()`. It wires the store, the runner, and the status check.
  - The `.gitignore` entry is written at the hub root on the first chat.
  - Chats left `running` by a previous board become `interrupted`.
  - Shutdown stops all running turns.

### Frontend (`web/src/components/Agent/`)

- `chat-thread-utils.ts`: `chatThreadReducer` folds events into turns. It drops settled events at or below `lastSeq`, so the initial fetch and the stream can overlap safely.
  - A settled `assistant_text` replaces the live text it supersedes. That covers joining a stream mid-block after a reload.
  - `splitTurn` shows the text after the last tool as the answer. Everything before it goes in the collapsed "Worked for N seconds" row, as in Linear.
- `useChatThread.ts`: fetches the chat, then opens `EventSource(stream?after=<last seq>)`. If the server says no turn is running but the last turn has no end, the hook marks it interrupted.
- `AgentChatProvider.tsx`: popup state, pills, model and effort prefs, and the ⌘J shortcut.
  - Pills and prefs live in `localStorage`, through try/catch helpers in `agent-state-utils.ts`.
  - Pills whose chat no longer exists are filtered out while rendering, not in an effect.
- `AgentStatusBar.tsx` (pills and the Agent button), `ChatPopup.tsx`, `ChatThread.tsx`, and `ChatComposer.tsx` are the UI.
- `App.tsx` builds the context (view and open issue) with `chatContextFor` and passes it to the provider.

## Key decisions

- **The board hosts the agent.** It already runs on the user's machine and serves the UI from the same origin. So the browser needs no pairing, no CORS, and no Local Network Access prompt.
- **Local mode uses stdio `xpo mcp`, not an HTTP MCP route on the board.** This changed during implementation.
  - With a non-loopback `--host`, the board's `tokenGuard` requires a session cookie, which the `claude` process does not have. `hostGuard` also rejects some Host values.
  - The stdio server writes `issues.db`, and the existing `WatchDB` pushes the change to the UI.
  - Remote mode uses `<remote.url>/mcp` with the remote token. The MCP config goes in a 0600 temp file, so the token never shows in `ps`.
- **The prompt goes over stdin.** As a positional argument, a message that starts with `-` would parse as a flag.
- **One process per turn, with `--resume`.** No idle timeouts and no reconnect logic. A crash costs one turn. Startup adds about 1–2 s.
- **Read-only code tools plus xpo tools, with no permission UI.**
  - `--tools Read,Grep,Glob` limits the built-in set. User settings cannot widen it.
  - `--allowedTools mcp__xpo__*` pre-approves the xpo tools.
  - In `-p` mode every other request is denied. Reads outside the working directory need approval, so they are denied too.
- **User settings load** (no `--setting-sources`), so `CLAUDE.md` and project rules apply. The user asked for this because issues and code are closely tied.
- **Tool detail keys grew** to include `pattern`, `title`, `id`, `query`, and `match`. Grep, Glob, and xpo calls now show an argument, and drive's spinner shows it too.

## Acceptance criteria

- [x] The status bar shows an **Agent** button to the left of the version info. *Evidence: Playwright screenshot `1-statusbar`.*
- [x] A click on **Agent** opens the chat popup above the status bar, in any view. *Evidence: screenshot `2-popup-empty`, taken on the issue detail view.*
- [x] `mod+j` toggles the popup. *Evidence: Playwright pressed Meta+J with the popup open, and the dialog count went to 0.*
- [x] The empty popup shows a welcome line and three suggestion chips. A click on a chip sends its prompt. *Evidence: screenshot `2-popup-empty`. Chips call the same `send()` as the composer.*
- [x] With an issue open, a context chip shows the issue ID and title, and the user can remove it. *Evidence: the chip "upe-0000c7 Set up customer support ticketing integration" shows with an × button. The agent then called `xpo show upe-0000c7` without being told the ID.*
- [x] A sent message streams the assistant reply token by token. *Evidence: the SSE stream shows `text_delta` events. Screenshot `3-running`.*
- [x] Tool calls collapse into a "Worked for N seconds ▸" row, and expanding it lists each call with its argument. *Evidence: screenshot `4-done-expanded` shows "xpo show upe-0000c7" and "xpo list".*
- [x] "Create a bug about X" creates the issue, and the board shows it without a reload. *Evidence: the chat created upe-ef828b, and `GET /api/issues` returned it. `WatchDB` pushes the change.*
- [x] The agent can read and search code in the project. *Evidence: `--tools Read,Grep,Glob` (`TestClaudeArgs_Baseline`). The test chat called Read.*
- [x] The agent cannot run Bash, edit files, or write files. *Evidence: asked to run `whoami`, it answered "I can't run shell commands in this environment".*
- [x] The agent cannot read files outside the project root. *Evidence: the Read of `/etc/hosts` was denied, and the agent reported that. (`~/.ssh` was not tried, but the same rule covers it.)*
- [x] The composer has a model picker: Default, Haiku, Sonnet, Opus. *Evidence: screenshots. The server checks the value with `ValidModel`.*
- [x] The composer has an effort picker: Default, low, medium, high, xhigh, max. *Evidence: screenshots. The server checks the value with `ValidEffort`.*
- [x] The next turn uses the selected model and effort. The chat keeps the selection, and new chats start with the last selection. *Evidence: `TestRunner_TurnPersistsSettledEvents` checks `--model` and `--effort` and the meta. Prefs persist in `localStorage`, and `openChat` restores the chat's own values.*
- [x] A stop button during a turn ends the turn, and the thread shows "Stopped". *Evidence: `TestRunner_Cancel` (keeps partial text and the session ID). Stop button in screenshot `3-running`.*
- [x] Minimize turns the chat into a pill with the chat title, and a running chat shows a spinner on its pill. *Evidence: screenshots `3-running` (spinner pill) and `5-minimized`.*
- [x] A click on a pill reopens that chat. Close (×) removes the pill, and the chat stays on disk. *Evidence: screenshot `7-reopened`. `close()` only removes the pill ID.*
- [x] A page reload keeps the pills and the thread content, and a running turn continues to stream. *Evidence: "pills after reload: 1" and screenshot `6-after-reload`. The stream resumes with `?after=`, as `TestChat_StreamReplaysAfterLastEventID` shows.*
- [x] A follow-up message continues the same conversation. *Evidence: the second turn answered "Previous number: 3". `TestRunner_SecondTurnResumes` checks `--resume`.*
- [x] Chats persist in `.xpo/chats/`, and `xpo` adds `.xpo/chats/` to `.gitignore` on the first chat. *Evidence: `TestStore_CreateCallsOnCreateDirOnce`. The board wires the hook to `EnsureGitignoreEntryAt(<hub>/.gitignore)`.*
- [x] When `claude` is not on `PATH`, the Agent button is disabled, and its tooltip says how to install Claude Code. *Evidence: `TestCheckAgent_NotOnPath`. The button uses `disabled={!available}` with `status.reason` as its tooltip.*
- [x] Chat works when the board listens on a non-loopback `--host`, and the startup log says chat is on. *Evidence: the stdio MCP server avoids the guards, and `runBoard` logs the warning. Not run manually on a LAN address.*
- [x] Chat works in local mode and in remote mode. *Evidence: local mode tested end to end. `TestChat_RoutesServedLocallyInProxyMode` and `TestMCPConfig_Remote` cover remote mode. Not run against a live remote server.*
- [x] `make test` passes, with Go tests for the parser, store, runner, and routes, and frontend tests for the chat reducer. *Evidence: `make test` exits 0, with 675 frontend tests. New suites: `claudestream_test.go`, `internal/chat/*_test.go`, `server/chat_test.go`, `chat-thread-utils.test.ts`, `agent-state-utils.test.ts`.*

## Things to know

- A denied tool call still shows as a normal step, because the runner does not read `tool_result` blocks yet. Follow-up: xpo-8673f4.
- Only `claude` is supported. Codex and other agents, with a provider picker, come in xpo-5b3157. The canonical `chat.Event` type is the seam for that work.
- The runner tests use a fake `claude` shell script, so they skip on Windows. `GOOS=windows go vet` still covers the build.
