# Spec: Agent chat in xpo board (engine + popup)

## What

`xpo board` hosts chat sessions with the user's local `claude` CLI. Each user message starts one `claude -p` turn in the project root. The turn resumes the chat's Claude session. It can use the xpo MCP tools and the read-only code tools (Read, Grep, Glob), so a request like "find the root cause of this bug and file an issue" works. The board streams the reply to the browser over SSE. In the UI, an **Agent** button in the status bar opens a chat popup from any view. The popup sends the open issue as context. A minimized chat becomes a pill in the status bar.

## Why

This is Pillar 1 of xpo-3798b7, and it is the first part of the agent engine that the drive flow (Pillar 2) will reuse.

The agent must run on the user's machine, so it can use their Claude subscription and, later, their worktrees. `xpo board` already runs there and serves the UI from the same origin. Because of that, the browser needs no pairing, no CORS, and no Local Network Access prompt.

## Acceptance Criteria

- [ ] The status bar shows an **Agent** button to the left of the version info.
- [ ] A click on **Agent** opens the chat popup above the status bar, in any view.
- [ ] `mod+j` toggles the popup.
- [ ] The empty popup shows a welcome line and three suggestion chips. A click on a chip sends its prompt.
- [ ] With an issue open, a context chip above the composer shows the issue ID and title. The user can remove the chip.
- [ ] A sent message streams the assistant reply token by token.
- [ ] Tool calls collapse into a "Worked for N seconds ▸" row. Expanding the row lists each tool call with its argument.
- [ ] "Create a bug about X" creates the issue. The board and backlog show the new issue without a reload.
- [ ] The agent can read and search code in the project ("find where X is handled" returns file paths and lines).
- [ ] The agent cannot run Bash, edit files, or write files. A request to do so gets a refusal, not an action.
- [ ] The agent cannot read files outside the project root. A request to read `~/.ssh/id_rsa` gets denied.
- [ ] The composer has a model picker: Default, Haiku, Sonnet, Opus.
- [ ] The composer has an effort picker: Default, low, medium, high, xhigh, max.
- [ ] The next turn uses the selected model and effort. The chat keeps the selection. New chats start with the last selection.
- [ ] A stop button during a turn ends the turn. The thread shows "Stopped".
- [ ] Minimize turns the chat into a pill in the status bar, with the chat title. A running chat shows a spinner on its pill.
- [ ] A click on a pill reopens that chat. Close (×) removes the pill. The chat stays on disk.
- [ ] A page reload keeps the pills and the thread content. A running turn continues to stream after the reload.
- [ ] A follow-up message continues the same conversation (Claude `--resume`).
- [ ] Chats persist in `.xpo/chats/`. `xpo` adds `.xpo/chats/` to `.gitignore` on the first chat.
- [ ] When `claude` is not on `PATH`, the Agent button is disabled. Its tooltip says how to install Claude Code.
- [ ] Chat works when the board listens on a non-loopback `--host`. The startup log says that chat is on.
- [ ] Chat works in local mode and in remote mode (`remote.url` set).
- [ ] `make test` passes, with Go tests for the parser, store, runner, and routes, and frontend tests for the chat reducer.

## Flow

### Backend

1. Extract the stream-json line parser from `claudeExecutor` into `ClaudeStreamParser` in `internal/exponential/claudestream.go`.
   - Drive and chat both call the new parser.
   - `Parse` returns a `ClaudeTurnResult` (session ID, cost, tokens, duration, `IsError`) for the final `result` line.
   - Add the `StreamSession` kind, sent for the `system/init` line, so a stopped turn keeps its session ID.
   - `StreamToolDone` carries the tool name and argument for tool blocks. It has no name for text blocks.
   - Drive behaviour must stay the same. New parser tests cover the shared code.
2. Create package `internal/chat`.
3. Add `chat.Store`, which reads and writes `.xpo/chats/<id>.json` (meta) and `.xpo/chats/<id>.jsonl` (events).
   - Meta fields: `id`, `title`, `created_at`, `updated_at`, `agent`, `model`, `effort`, `claude_session_id`, `state` (`idle` | `running` | `interrupted`).
   - Use UUIDv4 for `id`.
4. Define the canonical `chat.Event` JSON type. Every event has `seq` and `type`. The settled types are:
   - `user_message` (`text`, `context`)
   - `assistant_text` (`text`)
   - `tool_call` (`name`, `arg`, `status`)
   - `turn_done` (`duration_ms`, `cost_usd`, `tokens_in`, `tokens_out`)
   - `turn_stopped`
   - `error` (`message`)
5. Add live-only event types: `text_delta`, `thinking`, `tool_start`, `tool_done`. The store does not persist these.
6. Add `chat.Runner`.
   - `Send(chatID, text, ctx)` spawns one `claude` process per turn, in its own process group.
   - The runner turns the `claude` output into `chat.Event` values, appends the settled events to the store, and publishes every event to a per-chat broadcaster.
7. Build the `claude` arguments in `chat.claudeArgs`:
   - `-p --output-format stream-json --verbose --include-partial-messages`
   - Send the prompt over stdin, not as an argument. A message that starts with `-` must not parse as a flag.
   - `--tools "Read,Grep,Glob"`, which makes only these built-in tools available. User settings cannot add tools to this set.
   - `--allowedTools "mcp__xpo__*"`
   - No `--permission-mode` and no `--setting-sources`. User and project settings, `CLAUDE.md`, hooks, and plugins load as in a normal session.
   - Claude Code allows reads in the working directory without a prompt. A read outside it needs approval, and `-p` denies it.
   - `--mcp-config <tmpfile>` `--strict-mcp-config`
   - `--append-system-prompt <chat prompt>`
   - `--resume <claude_session_id>` when the meta has one.
   - `--model <m>` from the chat meta. When the meta has no model, use `chat.model`. When both are empty, pass no flag.
   - `--effort <level>` from the chat meta. When the meta has no effort, pass no flag.
   - Run the process with the project root as the working directory.
8. Write the MCP config to a 0600 temp file. Delete the file when the turn ends. The remote token must not appear in `ps` output.
   - Local mode: `{"mcpServers":{"xpo":{"command":"<os.Executable()>","args":["mcp"]}}}`. The stdio server writes to `issues.db`. The existing `WatchDB` sends the SSE refresh to the UI.
   - Remote mode: `{"mcpServers":{"xpo":{"type":"http","url":"<remote.url>/mcp","headers":{"Authorization":"Bearer <remote.token>"}}}}`.
9. Do not set `XPO_AGENT_IDENTITY`. The MCP `clientInfo` (`claude-code/<version> <agent@mcp>`) attributes agent writes to Claude Code.
10. Build the prompt text as `<xpo-context>…</xpo-context>` followed by the user text, when the message has context.
    - Context fields: `view`, `issue_id`, `issue_title`.
11. Put the chat system prompt in `internal/chat/prompt.md`, embedded with `go:embed`.
    - It says: you are the xpo assistant in the web board; use the `mcp__xpo__*` tools; write bare issue IDs; use the context block for "this issue".
12. Set the chat title from the first user message: first line, at most 60 characters.
13. Register the routes in `SetupRoutes` under `!s.Headless`, beside the existing local route:
    - `GET /api/local/agent` returns `{available, agent, model, reason}`.
    - `GET /api/local/chats` returns the meta list, newest first.
    - `POST /api/local/chats` creates a chat and returns its meta.
    - `GET /api/local/chats/{id}` returns the meta and the settled events.
    - `POST /api/local/chats/{id}/messages` takes `{text, context, model, effort}` and returns 202. It returns 409 when a turn is running.
    - The handler validates `model` against `""`, `haiku`, `sonnet`, `opus`. It validates `effort` against `""`, `low`, `medium`, `high`, `xhigh`, `max`. Other values return 400.
    - The handler saves `model` and `effort` in the chat meta before it starts the turn.
    - `POST /api/local/chats/{id}/cancel` kills the process group of the running turn.
    - `GET /api/local/chats/{id}/stream` is SSE. It replays settled events after `Last-Event-ID`, then sends live events.
14. Read `chat.agent` (default `claude`) and `chat.model` (default empty) from `.xpo/config.yaml`.
    - With any agent other than `claude`, `/api/local/agent` returns `available: false` with a reason.
15. On board start, set `state: interrupted` on every chat whose meta says `running`.
16. On board shutdown, kill every running turn's process group.
17. Call `EnsureGitignoreEntry(".xpo/chats/")` when the store creates its directory.

### Frontend

18. Add `web/src/api/chat.ts` with typed calls. Add the `useChatThread(chatId)` hook: it loads the settled events, then follows the SSE stream with `?after=<last seq>`. A reconnect sends `Last-Event-ID`.
19. Add `web/src/components/Agent/chat-thread-utils.ts` (`chatThreadReducer`). It folds settled and live events into turns: user message, work steps, answer, and status. Write tests for it first.
20. Add `AgentChatProvider` and `agent-context.ts` in `web/src/components/Agent/`. They hold the open chat ID, the popup state, and the pill IDs. `agent-state-utils.ts` holds the storage and context helpers, with tests.
    - Store the pinned pill IDs in `localStorage`. Wrap each `localStorage` call in try/catch.
21. Add `web/src/components/Agent/ChatPopup.tsx`.
    - Size: 420×600 px, anchored bottom-right above the status bar.
    - Header: title, minimize, close. Expand stays hidden until xpo-c047ce.
22. Add `ChatThread.tsx`.
    - User messages render as right-aligned bubbles.
    - Assistant text renders through `react-markdown`. Bare issue IDs become links to `#/issues/<id>`.
    - Tool groups render as a collapsible "Worked for N seconds" row.
23. Add `ChatComposer.tsx`.
    - Enter sends. Shift+Enter adds a new line.
    - The stop button replaces send during a turn.
    - The context chip shows above the input.
    - The footer has the model picker and the effort picker, where Linear has "Skills".
    - The pickers show the chat's saved values. For a new chat they show the last selection, which `localStorage` keeps.
24. Derive the context from the current `Route`: `issueId` and the view name. Read the title from the react-query issue cache.
25. In `Layout.tsx`, render `<AgentStatusBar/>` between the status slot and the version info. It holds the pills and the **Agent** button (lucide `MousePointer2` icon).
26. Register `mod+j` in the keyboard registry to toggle the popup. Add it to KeyboardHelp.

## Decisions

- **The board hosts the agent, not `serve`.** The agent needs the user's machine and subscription. Same origin removes all browser cross-origin work. `serve` stays agent-free.
- **One `claude -p` process per turn, with `--resume`, not one long-lived stream-json process.** Per turn, the board needs no idle timeouts or reconnect logic, and a crash affects one turn only. The cost is about 1–2 s of startup per turn. We can switch to a long-lived process later behind `chat.Runner`.
- **xpo MCP tools plus read-only code tools, not a permission UI.** Issues and code are closely tied, so the agent must read code to find root causes. Read, Grep, and Glob change nothing. Every write the agent can do is one the board UI already allows. So v1 needs no approval flow and no `--dangerously-skip-permissions`. Edits and Bash with approvals come in a later issue (`--permission-prompt-tool`).
- **Project root with the user's settings, not an isolated session.** `CLAUDE.md` and project rules give the agent the project's conventions, such as the xpo workflow.
- **Model and effort are per chat, chosen in the composer.** Aliases (`haiku`, `sonnet`, `opus`) always point to the latest version of each family. "Default" defers to `chat.model`, then to the CLI default.
- **A canonical `chat.Event` type, not raw Claude messages in the UI.** ACP or Codex drivers can come later without UI changes. Elysium uses the same pattern (`ProviderRuntimeEvent`).
- **SSE plus POST, not WebSocket.** xpo already has SSE, and chat does not need two-way streaming.
- **Local mode uses the stdio `xpo mcp` server (same binary). Remote mode uses the remote `/mcp`.** An HTTP MCP route on the board would sit behind `tokenGuard` and `hostGuard`. With a non-loopback `--host`, those guards block the `claude` process, which has no session cookie. The stdio server needs no token and no route. In remote mode the issues live on the server, so the agent must use the remote `/mcp`.
- **The title comes from the first message, not from an LLM.** It is cheap and predictable. An AI title can follow.
- **The remote token goes in a 0600 temp file, not in an argv flag.** Other local users can read argv through `ps`.

## Edge Cases

**HIGH (developer decides)**
- *The board listens on a non-loopback `--host`.* **Decided: chat stays on**, so the user can chat from another device. The existing token guard protects the chat routes. Anyone with the access token can use the host user's Claude subscription and read project code through the agent. The startup warning names this risk.
- *Reads outside the project root.* Claude Code confines reads without a prompt to the working directory, and `-p` denies the prompt. A manual check confirms this (see AC). If the check fails, add `--disallowedTools` rules for paths outside the project.

**MEDIUM (agent proposes, developer confirms)**
- *The board restarts during a turn.* The child process dies with the board. On start, the chat shows "Interrupted", and the next message resumes the Claude session.
- *The remote token has expired.* The MCP tool call fails. The thread shows the tool error text. No special handling.
- *Two tabs send to the same chat.* The second POST gets 409. The UI shows "A reply is already running".
- *`claude` is not logged in.* The turn fails with the CLI error text. The thread shows it as an `error` event.
- *A model does not support the chosen effort.* The CLI error shows as an `error` event. The picker does not filter effort levels by model.
- *A user hook or plugin blocks or slows a turn.* The thread shows the CLI output. No special handling.

**LOW (agent handles)**
- A chat ID that does not exist returns 404.
- An empty message is not sent.
- A pill for a deleted chat file is dropped from `localStorage` when it is loaded.
- An SSE reconnect uses `Last-Event-ID` and does not duplicate events.

## Assumptions

- In remote mode, `board` runs from a project directory that has `.xpo/`, so `.xpo/chats/` can live there.
- `claude` 2.1.x supports `--tools`, `--effort`, `--strict-mcp-config`, and `--allowedTools` wildcards for MCP tools. Installed version: 2.1.296. A runner test checks the argument list. A manual test checks the behaviour.
- Agent attribution: `XPO_AGENT_IDENTITY` or MCP `clientInfo` becomes the agent identity, and `OnBehalfOf` is the configured user. This matches the current MCP behaviour.
- `mod+j` is free. The registry has `c`, `j`, `k`, `x`, `Escape`, and the command palette.

## Resolved Questions

1. **Chat model default:** the CLI default. The user picks Haiku, Sonnet, or Opus and an effort level in the composer.
2. **Project context:** the project root with the user's settings. Issues and code are closely tied.
3. **Non-loopback board:** chat stays on, so the user can chat from another device.
4. **Suggestion chips:** "File a bug", "What's blocking <epic>?", "Summarize my open issues".

## Not in scope

- Sidebar Agent view, chat history list, expand, delete, pruning: xpo-c047ce.
- @-mentions, skills picker, attachments.
- Permission approvals and write tools (Edit, Write, Bash).
- ACP or Codex drivers.
- A Drive / Start Work flow on the chat engine.