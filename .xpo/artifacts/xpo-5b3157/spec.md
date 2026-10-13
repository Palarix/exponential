# Spec: agent chat with a provider picker (Claude, Codex)

## What

Agent chat can run Codex as well as Claude. A driver interface sits behind `chat.Runner`. The board detects which agent CLIs are installed. `GET /api/local/agent` lists the providers with their models and effort levels. The composer gets a provider picker, and its model and effort lists depend on the provider. Every chat remembers its provider and resumes that provider's own session.

ACP (Gemini, Cursor, Copilot) is out of scope. It is xpo-dc43e8.

## Why

Follow-up to xpo-56ae3a, part of xpo-3798b7. Chat only supports `claude` today, and `CheckAgent` rejects any other agent. Many users have Codex, not Claude Code. The research comment on xpo-3798b7 planned a canonical event stream so other agents could plug in. This issue proves that design with a second agent.

## Acceptance Criteria

- [ ] `chat.Runner` runs turns through a `Driver` interface. The Claude driver keeps today's behavior: same command line, same events, same tool restriction. The existing runner tests pass unchanged or with mechanical changes only.
- [ ] With `codex` on `PATH`, a chat can use Codex. The answer text streams live, and tool calls show up in the work list with status `done`, `error` or `denied`, like Claude's.
- [ ] Codex chats can use the xpo MCP tools without approval prompts, in local mode (stdio `xpo mcp`) and in remote mode (the remote `/mcp` with the bearer token). The token never appears on a command line.
- [ ] Codex chats are read-only: `read-only` sandbox, approval policy `never`. A write attempt fails. Codex sends no item for a command its sandbox blocks, so the work list shows the failure only when Codex reports the step; the agent's answer reports it either way.
- [ ] A follow-up message in a Codex chat continues the same Codex thread.
- [ ] Stop cancels a running Codex turn. The chat records `turn_stopped`.
- [ ] `GET /api/local/agent` returns every known provider with `available`, a `reason` when it is not available, its models (ID and label), and the efforts each model supports. It also returns the default provider (`chat.agent`, else the first available one). `chat.model` applies only to the default provider.
- [ ] Codex models and efforts come from `codex app-server`'s `model/list`, fetched once when the board starts. If the call fails, Codex offers only "Default model" and the standard efforts.
- [ ] The composer shows a provider picker when more than one provider is available. Changing the provider resets the model and effort to "Default". The picker is locked once a chat has messages, because a session belongs to one provider.
- [ ] Chat meta stores `agent` and a generic `session_id`. Chats saved with `claude_session_id` still resume.
- [ ] The Agent button is enabled when at least one provider is available. Its tooltip says how to install Claude Code or Codex when none is.
- [ ] `chat.agent: codex` in the config is accepted. An unknown or uninstalled `chat.agent` logs a startup warning, and chat defaults to the first installed provider.
- [ ] `make test` passes. Go tests cover the Codex event mapping with recorded app-server output, the provider listing and the meta migration. Frontend tests cover the provider picker.

## Flow

1. **Driver interface** (`internal/chat/driver.go`):
   ```go
   type Driver interface {
       Turn(ctx context.Context, in TurnInput, emit exponential.StreamCallback) (*TurnResult, error)
   }
   ```
   - `TurnInput` holds the prompt, the session ID, model, effort, working directory and the MCP target.
   - Drivers emit the `exponential.StreamEvent` kinds the runner already handles: text delta, thinking, tool start, tool input done, tool result, session ID.
   - `TurnResult` holds the session ID, duration, tokens, cost (0 when unknown), and an error text. A nil result with an error means the agent ended without one.
   - The runner keeps everything that is not agent-specific: turn bookkeeping, settling and pending tool calls, persistence, and subscribers. Cancel cancels `ctx`, and the driver stops its process group.
2. **Claude driver:** move today's `claudeArgs`, MCP config file and stream parsing out of `runner.run` into `ClaudeDriver`. No behavior change.
3. **Codex driver:** one `codex app-server` process per turn over stdio JSON-RPC.
   - Start the process with `-c` overrides. `mcp_servers.xpo` is the stdio `xpo mcp` or the remote URL. In remote mode the token goes in `bearer_token_env_var`, an environment variable set on the child process. Also set `mcp_servers.xpo.default_tools_approval_mode="approve"`.
   - Send `initialize` and `initialized`.
   - Send `thread/start`, or `thread/resume` with the stored thread ID. Pass the working directory, sandbox `read-only`, approval policy `never` and the system prompt as developer instructions.
   - Send `turn/start` with the prompt, model and effort.
   - Map notifications until `turn/completed`:
     - `item/agentMessage/delta` → text delta.
     - `item/started` → tool start, for `commandExecution`, `mcpToolCall`, `fileChange` and `webSearch` items.
     - `item/completed` → tool result, with the status from the item.
     - reasoning items → thinking.
     - token usage → `TurnResult`.
   - Any server request (approval) is answered with a JSON-RPC error, so a turn can never hang.
   - Cancel sends `turn/interrupt`, then kills the process.
4. **Tool names in the work list:**
   - Codex shell commands map to `Bash` with the command as the argument, without the `<shell> -lc '…'` wrapper. The UI already has verbs for these.
   - MCP calls map to `mcp__xpo__<tool>`, so the existing xpo verbs and issue chips apply.
5. **Providers** (`internal/chat/agent.go`): `CheckAgent` becomes a `Catalog` (`NewCatalog(chat.agent, lookPath)`), with `Status()`, `SetModels()` and `Resolve(provider, model, effort)`. The board fetches the Codex models in the background at startup (`CodexModels`, 5s timeout) and calls `SetModels`.
6. **API:** `GET /api/local/agent` returns `{available, default_provider, providers: [...]}`. `POST /messages` takes `provider`. The model and effort are checked against that provider's lists, and a chat whose stored agent differs is rejected with 409.
7. **Meta:** `Agent` is set on the first message; a new chat has none. Add `SessionID string json:"session_id"`. When reading, a non-empty `claude_session_id` moves into `session_id`.
8. **Frontend:**
   - `CHAT_MODELS` and `CHAT_EFFORTS` are no longer constants: the lists come from the provider list.
   - `ChatPrefs` gains `provider`, and the remembered prefs are checked against the current provider list.
   - The composer shows the provider picker first, and only with 2+ providers. It is disabled once the chat has events.
9. **Docs:** update the README chat section and the `chat.agent` comment.

## Decisions

| Question | Recommendation | Rationale |
|---|---|---|
| Codex transport | `app-server`, one process per turn (confirmed) | Live text deltas and `model/list`. `exec --json` only sends whole messages. One process per turn keeps the runner's cancel and restart model. |
| ACP (Gemini, Cursor, Copilot) | Follow-up xpo-dc43e8 (confirmed) | Halves the scope. The driver interface is designed so ACP slots in. |
| Codex tool restriction | `read-only` sandbox plus approval `never`; xpo MCP tools pre-approved (confirmed) | Codex has no tool allow-list like `--tools`. The sandbox is its read-only equivalent. Shell reads (`cat`, `rg`) stay allowed, as Claude's Read and Grep are. |
| Other MCP servers from `~/.codex/config.toml` | Left enabled (confirmed) | Codex has no `--strict-mcp-config`. Disabling them would mean overriding each user server by name. Revisit if a user server misbehaves in chat. |
| Write access in chat | None, for every provider (confirmed) | Chat runs headless in the hub checkout with no approval UI, and reads text others wrote. Unrestricted agents belong in work sessions inside the issue's worktree (xpo-00f0e5). Like Claude's `mcp__xpo__*` today, the Codex chat gets every xpo MCP tool; xpo-00f0e5 narrows that for both. |
| Provider change mid-chat | Locked after the first message | A session or thread belongs to one agent; there is no transcript to hand over. |
| Cost for Codex | Tokens only, `cost_usd` 0 | Codex does not report a price. The UI already hides a zero cost. |
| Model list source | `model/list` at board start | Codex model names change often, and a hard-coded list would rot. |
| System prompt | `{{code_access}}` placeholder in `prompt.md` | The shared prompt told Codex it has Read, Grep and Glob. Each provider now gets its own sentence about how it reads code. |
| Token usage | Sum of `last` usage per turn ID | `thread/resume` replays the previous turn's last usage update before the new turn starts. |

## Edge Cases

- **HIGH — app-server protocol drift.** The protocol is marked experimental. Mitigations: unknown notifications are ignored, the tests use recorded output, and a JSON-RPC error on `initialize` or `thread/start` becomes a chat `error` event with the message.
- **HIGH — a turn hangs on an approval request.** Every server request is declined at once. `approvalPolicy: never` should mean none arrive.
- **MEDIUM — a Codex thread is gone** (for example, its session file was deleted). `thread/resume` fails. The turn ends with an error that says to start a new chat. No silent new thread.
- **MEDIUM — the model list fetch is slow or Codex is not logged in.** It times out after 5s. Codex stays available with only the default model, and a turn shows the login error from Codex.
- **LOW — `chat.agent` names a provider that is not installed.** It is listed as unavailable, and the default falls back to the first available one.
- **LOW — chats saved before this change** have `agent: "claude"` and `claude_session_id`. They resume as before.
