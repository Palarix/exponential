# Walkthrough: widen Claude's built-in tools in agent chat

## What was built

Chat turns now get more of Claude Code's built-in tools. Both modes get WebSearch, WebFetch and LSP. Work Mode also gets the task (to-do) tools and subagents (`Agent`). WebFetch asks the user every time. The other new tools run without a card. Codex turns get live web search in both modes. The transcript shows a verb and an icon for each new tool.

## Why

The old lists (`Read,Grep,Glob,Skill` and `Read,Grep,Glob,Edit,Write,Bash,Skill`) left the other tools out by omission, not by design. No earlier spec gave a reason. The agent could not look up docs, navigate symbols, keep a to-do list, or split up large work.

## How the pieces fit

### Profile (`internal/chat/profile.go`)

Two package variables hold the new tools:

- `claudeLookupTools`: `WebSearch`, `WebFetch`, `LSP`. Both modes get them.
- `claudeWorkTools`: `TaskCreate`, `TaskUpdate`, `TaskList`, `TaskGet`, `Agent`. Only Work Mode gets them.

`claudeTools()` builds the `--tools` string from these lists. The existing order stays the same, so the old prefix of the string does not change.

Permissions come from three places:

- `claudeSettings()` adds `WebFetch` to the ask rules in every mode. The model picks the URL. A fetch can leak data in the URL or bring back prompt injection.
- `claudeAllowed()` adds `WebSearch` and `LSP` in both modes, and `claudeWorkTools` in Work Mode. WebSearch only sends a query. LSP only reads code.
- `WebFetch` is never in the allow list. `TestProfile_WebFetchNeverAllowed` guards this.

### Subagents

Claude Code 2.1.300 was probed before the change:

- A subagent gets only the parent turn's `--tools`. With `Read,Grep,Glob,Task`, the subagent had no Bash and no Write.
- A subagent's permission ask goes to the parent's `--permission-prompt-tool stdio`. xpo's approval cards therefore cover subagent calls.
- Subagent `stream_event` lines do not reach the parent stream. Only the parent's `Agent` call becomes a step. `ClaudeStreamParser` already skips subagent tool results by `parent_tool_use_id`.
- An `Agent` call can run in the background (`task_type: local_agent`). The xpo-fb4f9b banner already counts it as an agent. No new code was needed.

`Agent` is the name in `--tools` and in transcripts. claude also accepts `Task` as an alias. `TodoWrite` no longer exists. The `Task*` tools replace it.

### System prompt (`internal/chat/args.go`)

The Claude `codeAccess` lines name the new tools. Both lines say that WebFetch asks first, so the agent expects the card.

### Step details (`internal/exponential/claudestream.go`)

`toolDetailKeys` picks the value shown next to a step's verb. Three keys are new:

- `filePath`: LSP's file.
- `subject`: TaskCreate's title. It comes before `description`, because TaskCreate also sends a long description.
- `taskId`: TaskUpdate and TaskGet.

### Transcript (`web/src/components/Agent/`)

`TOOL_VERBS` in `chat-thread-utils.ts` has entries for the eight new tools. `Task` maps to the same verbs as `Agent`, for chats from older claude versions. A new `agent` icon (lucide `Bot`) marks subagent calls in `ToolIcon.tsx`. `toolDetail` shortens an LSP path to its file name, the same as Read, Edit and Write.

The unknown-tool fallback tests used `WebFetch`. They now use `NotebookEdit`, which stays excluded.

### Codex (`internal/chat/codex.go`)

`codexArgs` passes `-c web_search="live"` in both modes. This is the same setting as the `--search` flag. The search runs as OpenAI's hosted tool, so the read-only sandbox does not block it. A probe with `codex exec -s read-only` on codex 0.162.1 returned `web_search` items. `codex.go` already maps `webSearch` items to `WebSearch` steps.

## Decisions

- **All four `Task*` tools.** TaskUpdate needs a task ID, and TaskList finds it. The user confirmed this.
- **Task tools and subagents in Work Mode only.** Chat Mode turns are short Q&A. The user confirmed this.
- **Codex web search with no card.** This matches Claude's WebSearch. The user confirmed this.
- **Kept excluded:** Edit, Write and Bash in Chat Mode (xpo-00f0e5). Also EnterPlanMode/ExitPlanMode, ToolSearch, NotebookEdit and AskUserQuestion.

## Acceptance Criteria

- [x] Chat Mode `--tools` is `Read,Grep,Glob,Skill,WebSearch,WebFetch,LSP`. Evidence: `TestChatProfile_Claude`, `TestClaudeArgs_Baseline`, and the `mode_test.go` runner tests (`chatTools`).
- [x] Work Mode `--tools` is `Read,Grep,Glob,Edit,Write,Bash,Skill,WebSearch,WebFetch,LSP,TaskCreate,TaskUpdate,TaskList,TaskGet,Agent`. Evidence: `TestWorkProfile_Claude` and `TestRunner_StartEntersWorkModeAfterTheTurn` (`workTools`).
- [x] `--settings` has `WebFetch` as an ask rule in both modes. Evidence: `TestChatProfile_Claude`, `TestWorkProfile_Claude`, `TestProfile_WebFetchNeverAllowed`.
- [x] `--allowedTools` has `WebSearch` and `LSP` in both modes. Work Mode also has the four `Task*` tools and `Agent`. Evidence: `TestChatProfile_Claude`, `TestWorkProfile_Claude`, `TestClaudeArgs_GrantsFollowTheAllowList`.
- [x] `--allowedTools` never has `WebFetch`. Evidence: `TestProfile_WebFetchNeverAllowed` (chat, remote chat, work).
- [x] Chat Mode still has no Edit, Write or Bash, and no subagents or task tools. Evidence: `TestChatProfile_NoWorkTools`.
- [x] The Claude system prompt access lines name the web tools. Evidence: `TestSystemPromptFor`.
- [x] The transcript shows a verb and icon for each new tool, with no unknown-tool fallback. Evidence: the `chat-thread-utils.test.ts` cases "describes the lookup, task and subagent tools in words" and the `stepIcon` icon table.
- [x] Step details show the TaskCreate subject, the TaskUpdate task ID, the LSP file name and the Agent description. Evidence: `TestToolDetail` cases and the LSP `toolSummary` case (`profile.go`).
- [x] Codex turns get live web search in both modes. Evidence: `TestProfile_Codex` (all three profiles), and the codex 0.162.1 read-only probe.
- [x] `make test` passes. Evidence: lint, 971 frontend tests, and all Go packages pass.
