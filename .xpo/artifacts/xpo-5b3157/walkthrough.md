# Walkthrough: agent chat with a provider picker (Claude, Codex)

## What changed

Agent chat used to be hard-wired to the `claude` CLI. Now the runner talks to a `Driver`, and there are two drivers: Claude and Codex. The board detects which CLIs are installed, the API lists them with their models and efforts, and the composer has an agent picker. Each chat remembers which agent it belongs to and resumes that agent's own session or thread.

## How the pieces fit

### Driver interface (`internal/chat/driver.go`)

```go
type Driver interface {
    Turn(ctx context.Context, in TurnInput, emit exponential.StreamCallback) (*TurnResult, error)
}
```

- A driver runs one turn and emits the same `exponential.StreamEvent` kinds the Claude stream parser already produced: text deltas, thinking, tool start, tool input done, tool result.
- The runner keeps everything that is not agent-specific: turn bookkeeping, pending tool calls, persistence and subscribers.
- Cancellation is `ctx`. Stop cancels the context, and each driver is responsible for killing its own process group.

The spec sketched `ID()`/`Status()`/`RunTurn` methods. In the end provider status lives in the `Catalog`, so the driver only needs `Turn`.

### Claude driver (`internal/chat/claude.go`)

This is the old `runner.run` body, moved: same `claudeArgs`, same MCP config file, same `ClaudeStreamParser`. `exec.CommandContext` with `cmd.Cancel = killProcessGroup` and a 5s `WaitDelay` replaces the hand-rolled kill. If Claude exits without a result event, the error includes the tail of stderr.

### Codex driver (`internal/chat/codex.go`)

One `codex app-server` process per turn, JSON-RPC over stdio:

1. `initialize`, then the `initialized` notification.
2. `thread/start`, or `thread/resume` (with `excludeTurns: true`) for a stored thread ID. Both pass `cwd`, sandbox `read-only`, approval policy `never`, the model, and the system prompt as `developerInstructions`.
3. `turn/start` with the prompt and effort.
4. Map notifications until `turn/completed`.

**MCP config.** Codex has no `--mcp-config` file flag, so the xpo server is injected with `-c mcp_servers.xpo.<key>=<value>` overrides. Values are TOML, and a JSON-encoded string or array is valid TOML, so `tomlValue` is just `json.Marshal`. Local mode uses `command`/`args` (`xpo mcp`). Remote mode uses `url` plus `bearer_token_env_var = "XPO_MCP_BEARER_TOKEN"`, and the token is set only in the child's environment. It never appears in `ps`. `default_tools_approval_mode="approve"` means the xpo tools run without prompting.

**Event mapping** (`codexTurn.notification`, `itemStarted`, `itemCompleted`):

| Codex | Work list |
|---|---|
| `item/agentMessage/delta` | text delta |
| `reasoning` item | thinking |
| `commandExecution` | `Bash` with the command. `unwrapShell` strips the `/bin/zsh -lc '…'` wrapper Codex adds; a single parsed `commandActions` entry is preferred |
| `mcpToolCall` | `mcp__<server>__<tool>`, so the existing xpo verbs and issue chips apply |
| `fileChange` | `Edit` with the first path |
| `webSearch` | `WebSearch` |

- Codex sends whole tool items, not streamed input, so `item/started` emits tool start and tool done together.
- On `item/completed`, status `declined` → denied, `failed`/non-zero exit/`isError` → error. The error text comes from `error.message`, the MCP result content, or the command output, via the now-exported `exponential.ToolResultText`.
- An `agentMessage` that arrives complete without deltas is emitted as text. A ToolDone flush follows each message so commentary between tools is split the way Claude's is.

**Token usage.** `thread/tokenUsage/updated` is summed per turn ID, and only the current turn's sum counts. This is needed because `thread/resume` replays the previous turn's last usage update before the new turn starts. Counting it would double the totals. Codex reports no price, so `CostUSD` stays 0 and the UI hides it.

**Failure handling.**
- Any server *request* (for example an approval) gets a JSON-RPC error back right away, so a turn can't hang waiting on a UI that doesn't exist.
- A failed `thread/resume` ends the turn with "Codex could not resume this chat (…). Start a new chat." It does not quietly start a new thread.
- `turn/completed` with status `failed` becomes an error result with Codex's message (for example a login error).
- If the process exits early, the error includes stderr.

**Cancel.** When ctx is done the driver sends `turn/interrupt`. It waits up to `InterruptGrace` (3s) for `turn/completed`, then kills the process group. If no turn ID exists yet, it returns at once. In the live test Stop ended a turn in about 1s.

### Providers (`internal/chat/agent.go`)

`CheckAgent` became a `Catalog`:

- `NewCatalog(chat.agent, lookPath)` marks each known provider available or not, with an install hint as the reason. The default is `chat.agent` if it is installed, otherwise the first installed provider.
- `SetModels(provider, models)` replaces a provider's model list. The board calls it after `CodexModels` (a short-lived app-server `model/list`, 5s timeout) returns, in a goroutine, so startup isn't blocked. Until then, or if it fails, Codex offers "Default model" with the standard efforts. Hidden models are skipped. The default model's efforts become the efforts of "Default model".
- `Resolve(provider, model, effort)` checks the triple against the lists. It is the only gate before values reach a command line, so injection-shaped models like `--dangerously-skip-permissions` are rejected (tested).
- `Status()` returns a deep copy, because `SetModels` can run concurrently with requests.

Claude's model and effort lists moved here from `args.go`.

### Runner (`internal/chat/runner.go`)

- `SendRequest` gains `Provider` (empty → the default).
- **Provider lock:** a chat that has a session ID and a different stored `Agent` returns `ErrProviderLocked` (409 from the API). A chat with no session yet, for example one whose first turn failed, can still switch, and switching clears the session ID.
- `chat.model` (`DefaultModel`) applies only when the chat's agent is the default provider. A Claude model name would be invalid for Codex.

### Meta and migration (`event.go`, `store.go`)

`Meta` gains a generic `session_id`. `Store.Meta()` moves a legacy `claude_session_id` into it on read, so older chats resume. A new chat's `Agent` is empty until the first message sets it.

### System prompt (`prompt.md`, `args.go`)

The shared prompt said the agent reads code with Read, Grep and Glob. That's false for Codex. The line is now `{{code_access}}`, filled per provider by `systemPromptFor`. Codex is told it reads code with read-only shell commands.

### API (`internal/server/chat.go`)

- `GET /api/local/agent` returns `Catalog.Status()`: `{available, default_provider, reason, providers: [{id, name, available, reason, models: [{id, label, efforts}]}]}`.
- `POST /messages` takes `provider`. No provider available → 503; a bad provider, model or effort → 400; a locked chat → 409.

### Frontend

- `api/chat.ts`: the `CHAT_MODELS`/`CHAT_EFFORTS` constants are gone. The lists come from the agent status.
- `agent-state-utils.ts`: `ChatPrefs` gains `provider`. `normalizePrefs` checks remembered prefs against the current status and falls back to defaults. Legacy prefs without a provider load as Claude. `withProvider` resets model and effort; `withModel` resets an effort the new model doesn't support. Effort labels are capitalized for values without a fixed label (Codex's `ultra`).
- `ChatComposer`: the agent dropdown shows first, only when 2+ providers are available. `InlineDropdown` gained `disabled` and `title`. When the chat has messages, the dropdown is disabled with a tooltip that explains how to switch.
- `AgentChatProvider` opens a chat with its stored agent selected.

## Key decisions

- **app-server, not `exec --json`** (user): live text deltas and `model/list`. One process per turn keeps the runner's existing cancel and restart model.
- **Read-only for every provider** (user): chat runs headless in the hub checkout, has no approval UI, and reads text other people wrote (prompt-injection risk). Unrestricted agents belong to worktree work sessions, xpo-00f0e5.
- **The user's other Codex MCP servers stay loaded** (user): Codex has no strict-config flag. Disabling them would mean overriding each one by name.
- **ACP agents** (Gemini, Cursor, Copilot) are deferred to xpo-dc43e8. The Driver interface is shaped so an ACP driver slots in.

## Non-obvious things

- **Blocked writes leave no work-list step.** When the read-only sandbox blocks a command, Codex emits no item for it. The write still fails, and the agent says so in its answer, but no red × appears. The acceptance criterion was reworded to match.
- **Fake app-server tests.** `codex_test.go` drives a shell-script fake app-server built from recorded output. It runs in phases (init/thread/turn) because the fake must answer each request before the driver sends the next. Two lessons from writing it: a fake that reads more lines than the driver has sent deadlocks, and a cancel test must keep reading stdin (`cat`), or the interrupt is never recorded.
- **Process cleanup.** `codexProc.stop` closes stdin, waits briefly, then kills the process group. The live test left no Codex processes behind.

## Acceptance criteria

- [x] `chat.Runner` runs turns through a `Driver` interface; the Claude driver keeps today's behavior. Evidence: `claude.go` reuses `claudeArgs`/`ClaudeStreamParser`; the runner tests needed only mechanical changes (`Drivers`, `SessionID`); `TestClaudeArgs_*` pass.
- [x] With `codex` on PATH a chat can use Codex, with live text and tool calls showing done/error/denied. Evidence: `TestCodexDriver_Turn`, `_LiveEvents`; live run on the demo board showed a commentary message, an xpo `list` call, `ls -a` and the answer.
- [x] Codex gets the xpo MCP tools without prompts, local and remote; the token is never on a command line. Evidence: `codexArgs` sets `default_tools_approval_mode="approve"`; `TestCodexDriver_RemoteTokenOnlyInEnv` checks the token is in the env and absent from args.
- [x] Codex chats are read-only (sandbox `read-only`, approval `never`); a write attempt fails. Evidence: params asserted in `TestCodexDriver_Turn`; live run: `touch` was blocked and no file was created (no work-list step, as noted above).
- [x] A follow-up continues the same Codex thread. Evidence: `TestCodexDriver_SecondTurnResumesThread`; live run: same thread ID, and Codex remembered its earlier answer.
- [x] Stop cancels a running Codex turn and records `turn_stopped`. Evidence: `TestCodexDriver_Cancel` (sends `turn/interrupt`); live run: stopped in ~1s with partial text kept.
- [x] `GET /api/local/agent` returns every provider with availability, reason, models and efforts, and the default; `chat.model` applies to the default provider only. Evidence: `TestCatalog_*`, `TestAgentStatusListsProviders`, `TestRunner_DefaultModelOnlyForDefaultProvider`.
- [x] Codex models and efforts come from `model/list` at board start, with a fallback. Evidence: `TestCodexModels`, `TestCodexModels_Timeout`; live: 7 models loaded.
- [x] The composer shows a provider picker with 2+ providers, resets model and effort on change, and locks once the chat has messages. Evidence: `ChatComposer.test.tsx`, `agent-state-utils.test.ts` (`withProvider`).
- [x] Meta stores `agent` and `session_id`; `claude_session_id` chats still resume. Evidence: `TestStore_MetaMigratesClaudeSessionID`.
- [x] The Agent button is enabled when any provider is available; the tooltip gives install hints for both when none is. Evidence: `TestCatalog_NoneInstalled`; `AgentStatus.available`/`reason` drive the button.
- [x] `chat.agent: codex` is accepted; an unknown or uninstalled agent warns and falls back. Evidence: `TestCatalog_ConfiguredDefault`, `TestCatalog_DefaultFallsBackToInstalled`; warning in `board.go`.
- [x] `make test` passes; Go tests cover the Codex mapping with recorded output, the provider listing and the migration; frontend tests cover the picker. Evidence: `make test` green.
