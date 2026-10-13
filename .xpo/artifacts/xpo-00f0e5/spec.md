# Agent chat: Chat Mode and Work Mode in one chat

## What

A chat has two modes. **Chat Mode** reads the repo and changes issues. **Work Mode** implements one issue inside that issue's worktree. A chat enters Work Mode in two ways: the Start Work button on an issue, or the user asks the agent in a chat. No second session and no handoff. The conversation stays the same across the switch.

## Why

Pillar 1 (chat) and Pillar 2 (Start Work → Drive) of xpo-3798b7 meet here.

- Today every chat turn gets the same tools. `internal/chat/args.go` passes `--allowedTools mcp__xpo__*`, so a Claude chat can call `merge`. `codex.go` approves every xpo tool, so a Codex chat can call `merge` too.
- A chat cannot implement anything. It has no Edit, Write or Bash.
- The first proposal for this issue opened a separate work session from the chat. One chat that changes mode is a better UX: the design discussion becomes the implementation context.

## Dependencies

- **xpo-7bc86e (blocks this issue):** approval cards for tool calls. This issue uses them for `start` in Chat Mode and `merge` in Work Mode.
- **xpo-a5b9bd (related):** which issues the agent may change outside its bound issue. Until it is decided, Work Mode has the same xpo write tools as Chat Mode, plus `merge` behind an approval card.

## Measurements that shape this design

These were measured with Claude Code 2.1.296 on 2026-10-13.

- `claude -p --resume <id>` from a different working directory keeps the conversation.
- A change of working directory, `--tools`, `--allowedTools` or `--permission-mode` between turns keeps the prompt cache. The turn after a switch wrote 39 to 2,120 cache tokens and read about 40k. The system prompt and the history stay cached.
- A change of model empties the cache. Caches belong to one model.
- In a Claude Code permission rule, `//` starts an absolute path: `Edit(//abs/wt/**)` allows edits in the worktree. A single `/` is relative and did not match. An `Edit(...)` rule also covers the Write tool; a `Write(...)` rule matched nothing.
- codex-cli 0.162.1 takes `mcp_servers.xpo.disabled_tools` and `mcp_servers.xpo.tools.<tool>.approval_mode = "prompt"`. Under the approval policy `never`, a `prompt` tool fails with "MCP tool call requires approval, but approval policy is never" instead of asking. `on-request` asks.

## Acceptance Criteria

- [ ] A new chat starts in Chat Mode.
- [ ] In Chat Mode, a Claude turn cannot call `mcp__xpo__merge`. The agent gets a denial.
- [ ] In Chat Mode, a Codex turn cannot call the xpo `merge` tool.
- [ ] In Chat Mode, a `start` call by the agent shows an approval card: "Enter Work Mode on xpo-…?".
- [ ] After the user allows it, the issue is DOING, its worktree exists, and the chat is bound to the issue.
- [ ] After the user declines it, the agent gets a denial and nothing changes.
- [ ] Start Work on issue details starts the issue and opens a new chat in Work Mode on it.
- [ ] That chat opens with a prefilled first message. Nothing runs until the user sends it.
- [ ] Start Work on an issue that already has a chat in Work Mode opens that chat. It creates no second chat.
- [ ] On a DOING issue with a local branch, issue details shows "Continue Work" in place of Start Work. It opens the issue's Work Mode chat, or creates one bound to the existing worktree. The issue does not change.
- [ ] A chat created by Start Work or Continue Work is titled "<issue ID> <issue title>", so it can be found in recent chats.
- [ ] The composer mode segment shows "Chat" or "Work · xpo-…" in place of the static "Read-only" label.
- [ ] The mode segment has no way to enter Work Mode. In Work Mode its menu has "Exit work mode".
- [ ] Mode changes happen only between turns. The mode segment is disabled while a turn runs.
- [ ] In Work Mode, the agent runs in the worktree. Read, Edit, Write and Bash run there without a card.
- [ ] In Work Mode, an Edit or Write outside the worktree shows an approval card.
- [ ] In Work Mode, a `merge` call shows an approval card. `merge` runs only after the user allows it.
- [ ] After a successful `merge`, the chat goes back to Chat Mode. The transcript shows a mode divider.
- [ ] The user can leave Work Mode without a merge. The issue stays DOING and the worktree stays.
- [ ] Two chats cannot be in Work Mode on the same issue. The second attempt shows an error.
- [ ] The chat keeps the same agent session ID across every mode change.
- [ ] Work Mode is not available in remote mode. Start Work stays hidden, and the chat profile denies `start`.
- [ ] Chat Meta records every issue the chat worked on in Work Mode.
- [ ] `make test` passes.

## Flow

### 1. Mode in the chat record

1. Add `Mode` (`chat` or `work`) to `chat.Meta`. Empty means `chat`.
2. Add `Work *WorkBinding` to `chat.Meta` with `IssueID` and `Worktree`.
3. Add `WorkedIssues []string` to `chat.Meta`. Append the issue ID on each entry to Work Mode.
4. Add `Profile` to `TurnInput`. The runner builds it from Meta before each turn.

### 2. Profiles

1. Add `internal/chat/profile.go` with `type Profile struct`, `ChatProfile()` and `WorkProfile(issue, worktree)`.
2. Make `claudeArgs` read tools, allowed tools, disallowed tools and working directory from the profile.
3. Chat profile (Claude): `--tools Read,Grep,Glob`. Allow `mcp__xpo__*`, with an ask rule for `mcp__xpo__start` in `--settings` (an ask rule wins over an allow rule). Add `--disallowedTools mcp__xpo__merge`. In remote mode, also deny `start`.
4. Work profile (Claude): `--tools Read,Grep,Glob,Edit,Write,Bash`. Set the working directory to the worktree.
5. Work profile (Claude): allow `Edit(//<worktree>/**)` (also for the symlink-resolved path) and `Bash`. Allow `mcp__xpo__*`, with an ask rule for `mcp__xpo__merge`.
6. Chat profile (Codex): sandbox `read-only`, working directory = project root, approval policy `on-request`. Disable `merge` (and `start` in remote mode) with `disabled_tools`. Set `start` to approval mode `prompt`.
7. Work profile (Codex): sandbox `workspace-write`, working directory = worktree, approval policy `on-request`. Set `merge` to approval mode `prompt`.
8. Keep the system prompt the same in both modes: `prompt.md` describes both modes per provider. Each message's `<xpo-context>` block carries the turn's mode (`mode: chat`, or `mode: work on xpo-…` and `worktree: …`). A system prompt that changed with the mode would invalidate the prompt cache.
9. Keep the model and effort the same across a mode change.

### 3. Enter Work Mode

Two entry points share one bind step.

1. Add `Runner.BindWork(chatID, issueID, worktree)`. It sets `Mode`, `Work` and `WorkedIssues` in Meta.
2. Refuse the bind while a turn runs, in remote mode, and when another chat holds the issue.
3. Settle a `mode_changed` event. The UI draws it as a divider in the transcript.

**From an issue (Start Work):**

4. Add `POST /api/local/issues/{id}/work-chat` (chat routes live under `/api/local/`). It returns the chat that holds the issue, if one exists.
5. Otherwise, for a DOING issue, it finds the issue's existing worktree and does not start the issue again. A DOING issue without a worktree here is an error. For any other status it runs the same code path as the MCP `start` tool.
6. Then it creates a chat titled "<issue ID> <issue title>", binds it, and returns the chat.
7. Change `handleStartWork` in `PropertySidebar.tsx` to call this endpoint. Show the button as "Start Work" on BACKLOG and PLANNED issues, and as "Continue Work" on DOING issues with a branch, when the agent is available.
8. Open the returned chat in the Agent popup. Prefill the composer with "Let's implement xpo-…." and focus it. When no agent CLI is available, Start Work only starts the issue, as before.

**From a chat (the agent calls `start`):**

9. When the user allows an agent `start` call in Chat Mode, let `start` run.
10. When `start` succeeds, bind after the turn ends. `RunnerConfig.ResolveWork` turns the `start` argument (a full ID or a suffix) into the full ID and the worktree path.
11. Add a line to the Chat Mode prompt: after `start` succeeds, end the turn. Work Mode starts on the next turn.

### 4. Leave Work Mode

1. Watch for a successful `merge` tool result on the bound issue. Then set the mode to `chat` and clear `Work`.
2. Add `POST /api/local/chats/{id}/leave-work`. It leaves Work Mode without a merge.
3. Settle a `mode_changed` event in both cases.

### 5. UI

1. Replace the static "Read-only" segment in `ComposerPickers.tsx` with a mode segment.
2. In Chat Mode, show "Chat" with no menu. The tooltip says: "Ask the agent to start an issue, or use Start Work on an issue."
3. In Work Mode, show "Work · xpo-…" with the issue title as a tooltip. Its menu has "Go to issue" and "Exit work mode".
4. Render `mode_changed` as a divider: "Work Mode on xpo-…" or "Back to Chat Mode".

## Decisions

- **One chat that changes mode, not a handoff to a second session.** The conversation carries the design context. The cache measurement shows a switch costs about 2k cache-write tokens, not the whole history.
- **Only the user changes the mode.** The agent can ask through `start`, but the approval card decides. The agent never gets more permissions by itself.
- **Work Mode starts from an intent, not from a picker.** On an issue, the intent is the Start Work button. In a chat, the intent is the user's message, and the `start` approval card confirms it. An issue picker in the composer would duplicate both.
- **Start Work opens a new chat, not the current one.** The issue page has no chat in focus. A chat on another topic would mix two contexts.
- **Start Work prefills the first message, it does not send it.** The user can add instructions before the agent runs.
- **Continue Work, not a deferred start.** Closing the Start Work chat without a message leaves the issue DOING with its worktree. Continue Work reopens or recreates the chat from the issue. It also covers issues started from the CLI. Row icons and a menu of all linked chats stay in xpo-219de9.
- **Modes change between turns, not inside a turn.** One turn runs under one tool set. This keeps every tool call auditable against one profile.
- **Bash is allowed without a card in Work Mode.** It runs in the worktree. Asking for every command would make Work Mode unusable. This accepts that Bash can reach outside the worktree until xpo-3d7608 adds a sandbox.
- **`merge` asks through Claude Code's own permission system.** An ask rule for `merge` wins over the xpo wildcard, so Claude Code asks before it runs. The call cannot run without an allow.
- **The mode lives in the message, not the system prompt.** Both modes are described in one system prompt. The `<xpo-context>` block says which mode the turn runs in. A mode change then keeps the cached system prompt and history.
- **Codex uses the approval policy `on-request` in both modes.** Under `never`, Codex fails `start` instead of asking. In Chat Mode the read-only sandbox still blocks writes; Codex may now ask to run a command outside the sandbox, and the user decides on a card.
- **Chat Mode denies `merge` with `--disallowedTools`.** Deny rules apply in every permission mode, also `bypassPermissions`.
- **Work Mode is bound to exactly one issue.** A second issue needs a second chat.
- **Chat-to-issue links live in chat Meta.** `.xpo/chats/` stays gitignored and user-local. This changes the epic statement "chat sessions have no issue association". The links stay personal.

## Edge Cases

**HIGH (developer decides)**
- Bash in Work Mode can write outside the worktree, for example with `cd ..`. Accepted until xpo-3d7608. Codex is safer here: its `workspace-write` sandbox blocks writes outside the worktree.
- A later "full access" permission picker must not lift the `merge` deny in Chat Mode. The Chat profile test asserts `--disallowedTools`.

**MEDIUM (agent proposes, developer confirms)**
- The user changes the model in Work Mode. The cache empties for one turn. The composer shows no warning.
- The bound worktree is removed outside the chat (CLI `merge` or manual delete). The next turn falls back to Chat Mode and settles a `mode_changed` event with the reason.
- The issue is already DOING with a worktree from the CLI. `start` without `force` fails, so the chat stays in Chat Mode. The agent can call `start` with `force` (behind the same card); a successful call binds the chat. On issue details, Continue Work binds a new chat to the existing worktree.
- Start Work succeeds, but the chat creation fails. The issue stays DOING with its worktree. The UI shows the error and the user can retry.
- The board restarts while a chat is in Work Mode. Meta keeps the binding, so the next turn runs in Work Mode.

**LOW (agent handles)**
- A chat saved before this change has no `Mode`. The runner reads it as Chat Mode.

## Assumptions

- Resume across working directories keeps working in later Claude Code versions.
- Claude Code keeps reading `//` as an absolute path in rules, and Codex keeps its per-tool approval config. `TestRealAgents_ChatProfile` (with `XPO_TEST_REAL_CLI=1`) pins the Chat Mode behaviour of both agents.

## Out of scope

- Approval cards themselves: xpo-7bc86e.
- Limits on xpo writes outside the bound issue: xpo-a5b9bd.
- Chat icon on backlog rows and a menu of every chat linked to an issue: xpo-219de9.
- Work Mode in remote mode. Remote mode hides git actions until Team Mode (xpo-a60065).
- Sandboxing for Bash (xpo-3d7608).
