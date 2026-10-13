# Walkthrough: Chat Mode and Work Mode in one chat

## What was built

A chat now has two modes.

- **Chat Mode:** the agent reads the repo and changes issues.
- **Work Mode:** the agent implements one issue inside that issue's worktree, with Edit, Write and Bash.

A chat enters Work Mode in one of two ways: the Start Work (or Continue Work) button on an issue, or an approved `start` call by the agent. It leaves Work Mode in one of two ways: after a successful `merge`, or through "Exit work mode". The conversation and the agent session stay the same across every switch.

Before this change, every chat turn ran with the same tools. A Claude chat could call `merge` through the `mcp__xpo__*` allow rule, and a Codex chat approved every xpo tool. A chat could not write code at all.

## How the pieces fit

### 1. The mode lives in chat Meta (`internal/chat/event.go`)

`Meta` gained three fields:

- `Mode`: `chat` or `work`. Empty means `chat`, so chats saved before this change read as Chat Mode.
- `Work *WorkBinding`: `{issue_id, worktree}`.
- `WorkedIssues`: every issue the chat has entered Work Mode on, without duplicates.

A new `mode_changed` event carries `mode` and `issue_id`. The UI draws it as a divider in the transcript.

### 2. Profiles turn the mode into agent flags (`internal/chat/profile.go`)

`Profile` is built from Meta before every turn. `ChatProfile(root, canWork)` and `WorkProfile(issue, worktree)` hold everything that differs between modes. Both drivers read the same struct.

| | Chat Mode | Work Mode |
|---|---|---|
| Working dir | project root | worktree |
| Claude `--tools` | `Read,Grep,Glob` | `Read,Grep,Glob,Edit,Write,Bash` |
| Claude allow | `mcp__xpo__*` | `mcp__xpo__*`, `Bash`, `Edit(//<wt>/**)` and the symlink-resolved variant |
| Claude ask (`--settings`) | `mcp__xpo__start` | `mcp__xpo__merge` |
| Claude deny (`--disallowedTools`) | `merge` (also `start` without a local board) | none |
| Codex sandbox | `read-only` | `workspace-write` |
| Codex `disabled_tools` | `merge` (also `start` without a local board) | none |
| Codex `approval_mode="prompt"` | `start` | `merge` |

Three non-obvious rules come from measurements on 2026-10-13:

- **`Edit(//abs/**)`:** in a Claude Code rule, a double slash starts an absolute path. A single slash is relative and did not match. An `Edit` rule also covers Write. A `Write(...)` rule matched nothing, so there isn't one.
- **An ask rule wins over an allow rule.** `start` and `merge` stay inside the `mcp__xpo__*` wildcard, and the ask rule still makes them go through an approval card.
- **Codex approval policy is `on-request` in both modes.** Under `never`, a tool with `approval_mode = "prompt"` fails with "requires approval, but approval policy is never" instead of asking.

**Chat Mode denies `merge` with a deny rule, not just a missing allow.** Deny rules hold in every permission mode, so a later "full access" picker cannot lift them.

### 3. The mode goes in the message, not the system prompt (`args.go`, `prompt.md`)

`prompt.md` describes both modes once. Each turn's `<xpo-context>` block now always carries the mode:

- Chat Mode: `mode: chat`.
- Work Mode: `mode: work on xpo-…` and `worktree: …`.

A system prompt that changed with the mode would invalidate the prompt cache on every switch. The measurement showed that a switch done this way costs about 2k cache-write tokens, not the whole history. The prompt tells the agent to:

- call `start` to begin work, then end the turn;
- call `merge` only when the user asks;
- tell the user that Work Mode needs a local board when `start` is unavailable.

### 4. Mode changes happen between turns (`runner.go`, `mode.go`)

One turn runs under one profile, so every tool call can be audited against one tool set.

- **Agent-initiated:** `modeTools.observe` watches the tool results of a running turn.
  - In Chat Mode, a successful `mcp__xpo__start` is recorded.
  - In Work Mode, a successful `mcp__xpo__merge` on the bound issue is recorded. The argument can be a suffix, so it is matched by suffix.

  Nothing changes mid-turn. The run's deferred function takes `r.mu`, then `apply` either binds or leaves Work Mode. To bind, it resolves the `start` argument through `RunnerConfig.ResolveWork`, which gives the full ID and worktree path. It then saves Meta and settles a `mode_changed` event. If the bind fails, for example because another chat holds the issue, the chat stays in Chat Mode, and the divider reads "Could not enter Work Mode on …".
- **User-initiated:** `Runner.BindWork` and `Runner.LeaveWork` refuse while a turn runs (`ErrChatBusy`). `BindWork` also refuses without a local board (`ErrWorkUnavailable`) and when another chat holds the issue (`ErrIssueHeld`). `WorkChat(issueID)` finds the holder.
- **Race guard:** `Send` re-reads Meta after it registers the turn. A `BindWork` that lands between the HTTP request and the turn start is never lost.
- **Gone worktree:** `checkWorktree` runs before each turn. If the bound worktree was removed outside the chat, the chat falls back to Chat Mode with the reason in the divider.

### 5. Server endpoints (`internal/server/chat.go`)

**`POST /api/local/issues/{id}/work-chat`** backs both Start Work and Continue Work:

1. A chat already in Work Mode on the issue: return it (`created: false`). Nothing is started.
2. No local board (`CanWork` false): 501.
3. Unknown issue: 404.
4. DOING issue: look up its existing worktree with `FindWorktree`. If none exists on this machine: 409 "… has no worktree here". The issue is not started again.
5. Otherwise: run the same start path as the MCP `start` tool. `startIssue` was extracted from `handleStartWork` for this.
6. Create a chat titled `<issue ID> <issue title>` and bind it. If the bind fails, delete the new chat so no empty chat is left behind.

**`POST /api/local/chats/{id}/leave-work`** is "Exit work mode". It only changes the chat. The issue stays DOING and the worktree, branch and changes stay.

`cmd/exponential/board.go` wires `CanWork` and `ResolveWork` for the local board. Remote mode leaves `CanWork` false.

### 6. UI (`web/src/components/Agent`, `IssueDetail/PropertySidebar.tsx`)

- **Mode segment** (`ComposerPickers.tsx`), which replaces the static "Read-only" badge:
  - Chat Mode: "Chat" with no menu. Its tooltip says how to enter Work Mode.
  - Work Mode: "Work · xpo-…" with the issue title as a tooltip, and a menu with "Go to issue" and "Exit work mode".
  - The segment is disabled while a turn runs.
- **Dividers:** the thread reducer turns `mode_changed` into dividers, which `ChatThread` places between turns by time.
- **`useWorkMode.ts`:**
  - `useStartWorkChat` calls the work-chat endpoint and opens the chat. The composer is prefilled with "Let's implement xpo-…." and focused. Nothing is sent, so the user can add instructions first. Without an available agent, Start Work only starts the issue, as before.
  - `useWorkMode` provides the segment's data and actions.
- **Sidebar buttons:** Start Work shows for BACKLOG and PLANNED issues. Continue Work shows for DOING issues with a branch when the agent is available.

## Decisions made during review

- **Menu labels:** "Go to issue" and "Exit work mode", as requested. We tried "Pause work", "Back to Chat Mode" and "Detach from issue". We kept "Exit work mode" until a better one comes up.
- **Continue Work:** in the first version, closing a fresh Start Work chat without sending stranded the issue. It was DOING, so Start Work was hidden, and the chat was hard to find. Continue Work fixes that.
- **Chat titles:** chats are titled `<id> <title>` so they show up recognizably in Recent chats.
- **Continue Work reuses only a chat that is still in Work Mode.** A chat that used "Exit work mode" is not reopened; Continue Work creates a new chat on the same worktree.

## Known gaps and follow-ups

- Bash in Work Mode can write outside the worktree, for example with `cd ..`. This is accepted until xpo-3d7608 adds a sandbox. Codex's `workspace-write` sandbox already blocks it.
- Codex Work Mode was not verified end to end. Commits may fail under `workspace-write`, because a worktree's git dir lives in the hub's `.git`.
- Bugs found while testing, filed separately:
  - xpo-926f0b: Review Changes stays hidden when a worktree has only untracked files.
  - xpo-c94d85: a removed issue context chip comes back on reopen, and is redundant in Work Mode.
  - xpo-b45049: opening a chat from Recent chats shows no status-bar pill.
- xpo-219de9 still covers the chat icon on backlog rows and a menu for an issue with several chats. Continue Work covers the single-chat case of its "Open chat" button.

## Acceptance Criteria

- [x] A new chat starts in Chat Mode. *`TestRunner_NewChatRunsInChatMode`; `TestRunner_OldChatWithoutModeIsChatMode`.*
- [x] In Chat Mode, a Claude turn cannot call `mcp__xpo__merge`. The agent gets a denial. *`TestChatProfile_Claude` asserts `--disallowedTools mcp__xpo__merge`. `TestRealAgents_ChatProfile` (real CLI) saw the denial.*
- [x] In Chat Mode, a Codex turn cannot call the xpo `merge` tool. *`TestProfile_Codex/chat` (`disabled_tools=["merge"]`); `TestRealAgents_ChatProfile`.*
- [x] In Chat Mode, a `start` call by the agent shows an approval card: "Enter Work Mode on xpo-…?". *Ask rule in `TestChatProfile_Claude`; card wording in `chat-thread-utils.test.ts`. Seen live with Haiku on the demo board.*
- [x] After the user allows it, the issue is DOING, its worktree exists, and the chat is bound to the issue. *`TestRunner_StartEntersWorkModeAfterTheTurn`. Live: after Allow, the next turn ran in Work Mode.*
- [x] After the user declines it, the agent gets a denial and nothing changes. *`TestRunner_FailedStartKeepsChatMode`; `TestRealAgents_ChatProfile` denies every card and checks that nothing changed.*
- [x] Start Work on issue details starts the issue and opens a new chat in Work Mode on it. *`TestChat_WorkChatStartsIssueAndBindsNewChat`; verified live.*
- [x] That chat opens with a prefilled first message. Nothing runs until the user sends it. *`ChatComposer.test.tsx` "prefills the draft"; the server test checks that the chat has no user message and isn't running.*
- [x] Start Work on an issue that already has a chat in Work Mode opens that chat. It creates no second chat. *Second call in `TestChat_WorkChatStartsIssueAndBindsNewChat` returns 200 with the same chat and starts nothing.*
- [x] On a DOING issue with a local branch, issue details shows "Continue Work" in place of Start Work. It opens the issue's Work Mode chat, or creates one bound to the existing worktree. The issue does not change. *`TestChat_WorkChatContinuesDoingIssue`; `TestChat_WorkChatDoingIssueWithoutWorktree`; `PropertySidebar.test.tsx` Continue Work tests. Live: closing the chat without sending, then clicking Continue Work, reopened the same chat.*
- [x] A chat created by Start Work or Continue Work is titled "<issue ID> <issue title>", so it can be found in recent chats. *Title assertion in `TestChat_WorkChatStartsIssueAndBindsNewChat`; live: "upe-000050 Build mood detection API endpoint".*
- [x] The composer mode segment shows "Chat" or "Work · xpo-…" in place of the static "Read-only" label. *`ChatComposer.test.tsx` mode segment tests.*
- [x] The mode segment has no way to enter Work Mode. In Work Mode its menu has "Exit work mode". *`ChatComposer.test.tsx`: Chat Mode has no menu; the Work Mode menu has "Go to issue" and "Exit work mode".*
- [x] Mode changes happen only between turns. The mode segment is disabled while a turn runs. *`TestRunner_BindWorkRefusedWhileRunning`; `TestChat_LeaveWorkWhileRunningIsConflict`; "disables the mode segment while a turn runs".*
- [x] In Work Mode, the agent runs in the worktree. Read, Edit, Write and Bash run there without a card. *`TestWorkProfile_Claude`; `TestCodexDriver_WorkProfile`; the runner test's fake claude records the working dir. Live: Bash and Write in the worktree ran with no card.*
- [x] In Work Mode, an Edit or Write outside the worktree shows an approval card. *Live: a Write outside the worktree showed a card. I denied it and no file was written.*
- [x] In Work Mode, a `merge` call shows an approval card. `merge` runs only after the user allows it. *Ask rule in `TestWorkProfile_Claude`; `approval_mode` in `TestProfile_Codex/work`. Live: the merge card appeared and the issue was DONE only after Allow.*
- [x] After a successful `merge`, the chat goes back to Chat Mode. The transcript shows a mode divider. *`TestRunner_MergeLeavesWorkMode`; `TestRunner_FailedMergeStaysInWorkMode`; divider test in `ChatThread.test.tsx`. Live: "Back to Chat Mode" divider.*
- [x] The user can leave Work Mode without a merge. The issue stays DOING and the worktree stays. *`TestRunner_LeaveWork`; `TestChat_LeaveWork`.*
- [x] Two chats cannot be in Work Mode on the same issue. The second attempt shows an error. *`TestRunner_BindWork` (`ErrIssueHeld`). An agent `start` on a held issue shows a "Could not enter Work Mode" divider.*
- [x] The chat keeps the same agent session ID across every mode change. *`TestRunner_StartEntersWorkModeAfterTheTurn` checks the next turn resumes the same session. Profiles change only tools and the working dir, never the session.*
- [x] Work Mode is not available in remote mode. Start Work stays hidden, and the chat profile denies `start`. *`TestChatProfile_ClaudeWithoutWork`; `TestProfile_Codex/chat remote`; `TestChat_WorkChatUnavailableWithoutWork` (501); `TestRunner_BindWorkRefusedWithoutWork`; PropertySidebar remote-mode test.*
- [x] Chat Meta records every issue the chat worked on in Work Mode. *`TestRunner_BindWork` (deduplicated `WorkedIssues`). Live: kept after merge.*
- [x] `make test` passes. *Run before commit.*
