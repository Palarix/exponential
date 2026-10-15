# Walkthrough: generated chat titles

## What was built

A chat used to be named after the first line of its first message, up to 200 characters. A work chat was named `<id> <issue title>`. Now, after a chat's first finished turn, the board asks a small model for a 3–8 word title, in the background, and replaces the placeholder with it. The chat's own agent CLI writes the title, so Claude chats and Codex chats both get one. Work chats lose the ID prefix; an idle chat in Work Mode shows a hammer icon in the chat lists instead.

## How the pieces fit

### 1. The placeholder flag (`event.go`, `store.go`)

`Meta.AutoTitle` is true while the title is a placeholder. Two places set it:

- `Runner.Send` sets the first-message title, as before, and sets `AutoTitle`.
- `handleWorkChat` names a new work chat with `Store.SetAutoTitle(issue.Title)`.

`Store.Rename` clears the flag: a rename is final. The flag is positive on purpose. A chat saved before this change has no flag, so the board never touches it. A `title_source` field could not tell an old renamed title from an old first-message title.

The store has a `titleMu` mutex. `Rename`, `SetAutoTitle` and `ReplaceAutoTitle` take it. `ReplaceAutoTitle(id, placeholder, title)` saves only when the stored title still equals the placeholder, so a rename during the title call wins.

### 2. The titlers (`title.go`)

`Titler.Title(ctx, dir, model, prompt)` runs one CLI call with no tools and no saved session. The prompt goes on stdin.

| Titler | Command | Reply |
|---|---|---|
| `ClaudeTitler` | `claude -p --output-format json --no-session-persistence --tools "" --strict-mcp-config --setting-sources "" --disable-slash-commands --system-prompt … --model haiku` | `result` of the JSON |
| `CodexTitler` | `codex exec --ephemeral --skip-git-repo-check --ignore-rules -s read-only -m <model> -c model_reasoning_effort="low" -c model_instructions_file=<tmp> -c web_search="disabled" --json -` | the last `agent_message` item |

Codex takes its system prompt from a file, so `CodexTitler` writes a temp file and removes it after the call. Codex keeps the user's config: `--ignore-user-config` would break a custom model provider. `runTitleCLI` puts the CLI in its own process group, and a cancelled context kills the group.

### 3. Which model (`agent.go`, `codex.go`)

`ModelOption.Fast` marks a cheap model. It is `json:"-"`, so the API does not change. Claude's `haiku` is fast. A Codex model is fast when its `model/list` description says "fast" or "affordable". `Catalog.TitleModel(provider)` returns the first fast model. When it returns "", for example before Codex's `model/list` answers, the titler uses the chat's own model. Codex always runs at `low` effort. A hard-coded Codex ID was not an option: `gpt-5.4-mini`-style IDs fail with a ChatGPT account.

### 4. When the runner asks (`runner.go`, `title.go`)

In `endTurn`, under `r.mu`, a finished turn with `AutoTitle` set does three things:

1. It clears `AutoTitle` in the meta it saves. The board tries once per chat, so a failure that repeats does not spawn a CLI call on every turn.
2. It adds to `r.titles`, a wait group.
3. After the lock, it starts `generateTitle` in a goroutine.

A failed or stopped turn does not ask. The next finished turn does.

`generateTitle` builds the prompt from the settled events. `titleInput` takes the first user message and the agent text of the first finished turn. `titlePrompt` cuts each to 2,000 characters and asks for sentence case. In Work Mode it adds a hint, `issue xpo-abc "<issue title>"`. The model decides whether to name the issue. `RunnerConfig.IssueTitle` looks up the issue title. The call runs in the chat's directory (the root or the worktree), with a 60 s timeout. Its context ends on `Shutdown`.

`cleanTitle` keeps the first non-empty line and strips a `Title:` label, Markdown emphasis, quotes and a trailing period. It cuts the title to 80 characters. An empty result keeps the placeholder. After `ReplaceAutoTitle` saves, the runner publishes a live `title` event. A failure is logged, never added to the thread.

`Runner.Wait` now waits for title calls too, so tests and shutdown see a settled title.

### 5. Wiring (`cmd/exponential/board.go`)

The board passes both titlers, `catalog.TitleModel` and `issueTitle`. `NewRunner` has no default titlers, so a test that builds a runner never calls a real CLI.

### 6. Web app

- `changesChatMeta` (`chat-thread-utils.ts`) replaces the `META_EVENTS` set in `useChatThread.ts`, and adds `title`. The chat list refetches, and the open chat's header reads its title from the list. The list stops polling when a turn ends, so it needs this event.
- `idleChatIcon(chat, size, chatIcon)` (`recent-chats.tsx`) returns a `Hammer` in Work Mode, else `chatIcon`. Recent chats, the Agent view list and the All chats rows use it. The All chats page keeps its muted ring for Chat Mode. The issue chat icon's `work` state is a `Hammer` too. The icon follows the current mode, like the composer: a chat that left Work Mode shows its Chat Mode icon again.

## Acceptance criteria

- [x] After the first finished turn of a new chat, the title changes to a generated 3–8 word title. Evidence: `TestRunner_GeneratesTitleAfterFirstTurn`.
- [x] A Codex chat gets its title from Codex; a Claude chat gets its title from Claude (Haiku). Evidence: `TestRealAgents_Title` with `XPO_TEST_REAL_CLI=1`. claude/haiku returned "Chat chip approval shield icon"; codex/gpt-6-luna returned "Show a Shield While Approval Waits".
- [x] A new work chat is named after its issue title until its first finished turn. Evidence: `TestChat_WorkChatStartsIssueAndBindsNewChat` checks `Fix login` with `auto_title`.
- [x] A work chat's title has no fixed prefix; the model gets the issue as a hint. Evidence: `TestRunner_WorkChatTitleHint`, `TestTitlePrompt`.
- [x] In the chat lists, an idle chat in Work Mode shows a `Hammer`; Chat Mode keeps its icon. Evidence: the `idleChatIcon` tests, and the `IssueChatIcon` work chat test checks `lucide-hammer`.
- [x] The running, waiting, interrupted and error icons stay as they are. Evidence: `chatStateIcon` runs first, and its tests are unchanged and pass.
- [x] A chat the user renamed keeps its title, also during the call. Evidence: `TestRunner_RenameDuringTitleWins`, `TestStore_RenameMakesTheTitleFinal`, `TestStore_ReplaceAutoTitle`.
- [x] The turn ends without waiting for the title. Evidence: `endTurn` starts `generateTitle` after it saves the idle state. `TestRunner_RenameDuringTitleWins` renames while the title call is held.
- [x] The header and lists show the new title without a reload. Evidence: the `title` live event in `TestRunner_GeneratesTitleAfterFirstTurn`, and `changesChatMeta("title")`.
- [x] A failed or empty title call keeps the placeholder and adds nothing to the thread. Evidence: `TestRunner_TitleErrorKeepsPlaceholder`, `TestRunner_EmptyTitleKeepsPlaceholder`.
- [x] Chats saved before this change keep their titles. Evidence: `TestRunner_OldChatKeepsItsTitle`.
- [x] `make test` passes. Evidence: exit 0, including `go vet` for Windows, eslint and 1085 vitest tests. `make build` passes too.

## Things to know

- Run `XPO_TEST_REAL_CLI=1 go test -run TestRealAgents_Title ./internal/chat/` after a Claude Code or Codex upgrade. It pins the title calls' flags.
- Cost per title: about $0.00015 on Haiku, about 9.5k input tokens (half cached) on Codex.
- Codex sometimes capitalizes like a headline, even when the prompt asks for sentence case.
