# Spec: widen Claude's built-in tools in agent chat

## What

Chat turns get more of Claude Code's built-in tools. Both modes get WebSearch, WebFetch and LSP. Work Mode also gets the task (to-do) tools and subagents. WebFetch asks the user before it runs. The others run without a card. The transcript shows a readable verb and icon for each new tool.

## Why

`claudeTools()` names only `Read,Grep,Glob,Skill` (Chat) and `Read,Grep,Glob,Edit,Write,Bash,Skill` (Work). No spec gives a reason to leave the other tools out (rationale search: xpo-00f0e5, xpo-a058b4, xpo-a7f115). The agent cannot look up docs, navigate symbols, keep a to-do list, or split up large work.

## Verified against claude 2.1.300 (2026-10-14)

- `--tools` accepts `WebSearch`, `WebFetch`, `LSP`, `TaskCreate`, `TaskUpdate`, `TaskList`, `TaskGet`, `Agent`.
- `Agent` is an alias. The init line reports it as `Task`. Transcript `tool_use` blocks name it `Agent`.
- `TodoWrite` no longer exists. `TaskCreate` and the other `Task*` tools replace it.
- A subagent gets only the parent's `--tools` list. A probe with `Read,Grep,Glob,Task` gave the subagent `Agent,Glob,Grep,Read`, and no Bash and no Write.
- A subagent's permission ask goes to the parent's `--permission-prompt-tool stdio`. xpo's approval card therefore covers subagent calls.
- Subagent `stream_event` lines do not reach the parent stream. Only the parent's `Agent` call becomes a step. Subagent tool results carry `parent_tool_use_id`, and `ClaudeStreamParser` already skips them.
- An `Agent` call can run in the background. It sends `background_tasks_changed` with `task_type: local_agent`. The xpo-fb4f9b banner shows it as "1 agent working".
- Inputs: TaskCreate `{subject, description}`. TaskUpdate `{taskId, status}`. LSP `{operation, filePath, line, character}`. Agent `{description, subagent_type, prompt}`. WebSearch `{query}`. WebFetch `{url, prompt}`.

## Acceptance Criteria

- [ ] Chat Mode `--tools` is `Read,Grep,Glob,Skill,WebSearch,WebFetch,LSP`.
- [ ] Work Mode `--tools` is `Read,Grep,Glob,Edit,Write,Bash,Skill,WebSearch,WebFetch,LSP,TaskCreate,TaskUpdate,TaskList,TaskGet,Agent`.
- [ ] `--settings` has `WebFetch` as an ask rule in both modes.
- [ ] `--allowedTools` has `WebSearch` and `LSP` in both modes. Work Mode also has the four `Task*` tools and `Agent`.
- [ ] `--allowedTools` never has `WebFetch`.
- [ ] Chat Mode still has no Edit, Write or Bash. Chat Mode has no subagents and no task tools.
- [ ] The Claude system prompt access lines name the web tools.
- [ ] The transcript shows a verb and icon for WebSearch, WebFetch, LSP, TaskCreate, TaskUpdate, TaskList, TaskGet and Agent. None of them falls back to the unknown-tool icon.
- [ ] Step details: the TaskCreate subject, the TaskUpdate task ID, the LSP file name, the Agent description.
- [ ] Codex turns get live web search in both modes.
- [ ] `make test` passes.

## Flow

### 1. Profile (`internal/chat/profile.go`)

1. Add the shared tools `WebSearch,WebFetch,LSP` to `claudeTools()` in both modes.
2. Add `TaskCreate,TaskUpdate,TaskList,TaskGet,Agent` to `claudeTools()` in Work Mode only.
3. Add `WebFetch` to the ask list in `claudeSettings()`. Keep the `mcp__xpo__*` ask entries.
4. Add `WebSearch` and `LSP` to `claudeAllowed()` in both modes.
5. Add the task tools and `Agent` to `claudeAllowed()` in Work Mode.
6. Update the comments that explain each list.

### 2. System prompt (`internal/chat/args.go`)

1. Change the Claude Chat Mode line to: "You can read and search the code with Read, Grep, Glob, and LSP, and search the web with WebSearch. WebFetch asks the user first. You cannot edit files or run commands."
2. Change the Claude Work Mode line to also name WebSearch, WebFetch (asks first), the task tools, and subagents.

### 3. Step details (`internal/exponential/claudestream.go`)

1. Add `subject` to `toolDetailKeys` before `description`. A TaskCreate step then shows its subject, not its long description.
2. Add `taskId` and `filePath` to `toolDetailKeys`.

### 4. Transcript verbs (`web/src/components/Agent/chat-thread-utils.ts`)

1. Add these entries to `TOOL_VERBS`:

| Tool | Icon | verb / bare | action / bareAction |
|---|---|---|---|
| WebSearch | search | Searched the web for / Searched the web | Search the web for / Search the web |
| WebFetch | read | Fetched / Fetched a page | Fetch / Fetch a page |
| LSP | search | Looked up symbols in / Looked up symbols | Look up symbols in / Look up symbols |
| TaskCreate | add | Added the to-do / Added a to-do | Add the to-do / Add a to-do |
| TaskUpdate | edit | Updated to-do / Updated a to-do | Update to-do / Update a to-do |
| TaskList | read | Listed the to-dos | List the to-dos |
| TaskGet | read | Read to-do / Read a to-do | Read to-do / Read a to-do |
| Agent | agent | Delegated / Delegated to a subagent | Delegate / Delegate to a subagent |

2. Add the `agent` icon to `ToolIconId` and to `ToolIcon`.
3. Shorten the LSP detail to the file name, the same as Read, Edit and Write.
4. Add `Task` as a second key for the Agent verbs. Older claude versions name the call `Task`.

### 5. Codex (`internal/chat/codex.go`)

1. Pass `-c web_search="live"` in `codexArgs` in both modes.
2. Probe `codex app-server` 0.162.1 with that flag, and confirm a `webSearch` item.

### 6. Tests

1. Update the `--tools` assertions in `profile_test.go`, `mode_test.go` and `args_test.go`.
2. Add profile tests for the ask and allow lists of each mode.
3. Add `claudestream_test.go` cases for the TaskCreate, TaskUpdate and LSP details.
4. Add `chat-thread-utils.test.ts` cases for each new verb, and for the `stepIcon` icons.
5. Update the WebFetch tests that expect the unknown-tool fallback.
6. Add a `codexArgs` test for the web search flag.

## Decisions

- **`Agent`, not `Task`, in `--tools`.** `Agent` is the current name, and transcripts use it. claude 2.1.300 accepts both.
- **TaskList and TaskGet too.** The issue names TaskCreate and TaskUpdate. TaskUpdate needs a task ID, and TaskList finds the ID. Both are read-only bookkeeping.
- **No subagents or task tools in Chat Mode.** Chat Mode turns are short Q&A. A subagent there gets only the read-only tools, so it is safe, but it costs tokens for little gain.
- **WebFetch asks every time.** The model picks the URL. A fetch can leak data in the URL or carry prompt injection back. "Allow for this chat" uses the existing grant flow, and Claude Code then proposes a `WebFetch(domain:…)` rule.
- **WebSearch runs without a card.** The search runs on Anthropic's side, and the model only sends a query.
- **No new background handling.** xpo-fb4f9b already shows `local_agent` tasks as agents.

## Edge Cases

- HIGH: A subagent in Work Mode runs Bash or edits files with the parent's allow rules. This is the same power the parent has. Asks reach the user as cards. No change needed.
- MEDIUM: A background subagent outlives the turn. The xpo-fb4f9b rules apply: the turn keeps running, and the user cannot reply until Stop.
- MEDIUM: An older claude without `LSP` or the `Task*` tools. claude drops unknown names from `--tools` without error. The turn runs with fewer tools.
- LOW: LSP fails when no language server exists for the file. The step shows as failed, the same as other tool errors.

## Assumptions

- The user's claude version is 2.1.300 or close to it.
- Codex hosted web search does not need sandbox network access, so Chat Mode's read-only sandbox does not block it.

## Resolved Questions (2026-10-14)

1. All four `Task*` tools: yes.
2. Codex live web search in both modes: yes.
3. Subagents and task tools in Work Mode only: yes.
