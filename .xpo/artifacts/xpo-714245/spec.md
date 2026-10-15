# Agent chat: generate short, descriptive chat titles

## What

After a chat's first finished turn, the board asks a small, cheap model for a 3–8 word title and replaces the placeholder title with it. The chat's own agent CLI generates the title, so the feature works for Claude chats and for Codex chats. A title the user renamed is never replaced. Chat lists show a hammer icon for a chat in Work Mode, so a title needs no issue prefix.

## Why

Today a chat title is the first line of the first message, up to 200 characters (`TitleFrom`). A work chat is named `<id> <issue title>`. Both read like the prompt, not like a name. The All chats page (xpo-1446f2) is hard to scan.

## Findings

- **Claude.** Headless sessions (`claude -p`) write no `ai-title` record. A one-shot call works: `claude -p --model haiku --tools "" --strict-mcp-config --setting-sources "" --no-session-persistence --system-prompt …`. Measured: about 2 s, $0.00015, about 700 input tokens.
- **Codex.** app-server has `thread/name/set` and `thread/name/updated`, but it never names a thread on its own. A one-shot call works: `codex exec --ephemeral --skip-git-repo-check --ignore-rules -s read-only -m <model> -c model_reasoning_effort="low" -c model_instructions_file=… --json -`. Measured on `gpt-6-luna`: about 4 s, about 9.5k input tokens (half cached), 11 output tokens. `gpt-5.4-mini`-style IDs fail with a ChatGPT account, so the model must come from Codex's own `model/list`.
- **Icons.** The composer already shows `MessageSquare` in Chat Mode and `Hammer` in Work Mode (`ComposerPickers.tsx`). The chat lists show `MessageSquare` for every idle chat. The All chats page shows a muted ring instead.

## Acceptance Criteria

- [ ] After the first finished turn of a new chat, the title changes to a generated 3–8 word title.
- [ ] A Codex chat gets its title from Codex; a Claude chat gets its title from Claude (Haiku).
- [ ] A new work chat is named after its issue title until its first finished turn.
- [ ] A work chat's title has no fixed prefix. The model gets the issue ID and title as a hint, and decides the title.
- [ ] In the chat lists, an idle chat in Work Mode shows a `Hammer` icon. An idle chat in Chat Mode keeps its current icon: `MessageSquare`, or the muted ring on the All chats page.
- [ ] The running, waiting, interrupted and error icons stay as they are, in both modes.
- [ ] A chat the user renamed keeps its title. This includes a rename while the title is being generated.
- [ ] The turn ends, and the chat goes idle, without waiting for the title.
- [ ] The open chat header and the chat lists show the new title without a reload.
- [ ] A failed or empty title call keeps the placeholder title and logs nothing to the chat thread.
- [ ] Chats saved before this change keep their titles.
- [ ] `make test` passes.

## Flow

### Backend

1. Add `AutoTitle bool` (`json:"auto_title,omitempty"`) to `Meta` in `internal/chat/event.go`. True means the title is a placeholder that the board replaces once.
2. In `Runner.Send`, set `AutoTitle = true` when the runner sets the title from the first message.
3. Add `Store.SetAutoTitle(id, title)`. It sets a placeholder title with `AutoTitle`.
4. In `handleWorkChat` (`internal/server/chat.go`), name a new work chat with `SetAutoTitle(issue title)`.
5. Make `Store.Rename` clear `AutoTitle`. A user rename is final.
6. Add `Store.ReplaceAutoTitle(id, placeholder, title)`. It saves the title only when the stored title still equals the placeholder.
7. Order `Rename`, `SetAutoTitle` and `ReplaceAutoTitle` with one store mutex (`titleMu`).
8. Add `internal/chat/title.go` with a `Titler` interface: `Title(ctx, dir, model, prompt string) (string, error)`.
9. Add `ClaudeTitler` and `CodexTitler`. Each runs its CLI once with the flags in Findings, with the prompt on stdin.
10. Make `CodexTitler` write its short instructions to a temp file, and remove the file after the call.
11. Mark Claude's `haiku` model as fast in `claudeModels`.
12. Mark a Codex model as fast in `codexModelOptions` when its `model/list` description contains "fast" or "affordable".
13. Add `Catalog.TitleModel(provider)`. It returns the first fast model of the provider, else "".
14. Add `Titlers`, `TitleModel` and `IssueTitle` to `RunnerConfig`. A provider with no titler keeps the placeholder.
15. Wire the Claude and Codex titlers, `catalog.TitleModel` and an issue title lookup in `cmd/exponential/board.go`.
16. Add `titlePrompt(issueID, issueTitle, firstUser, firstReply)`. It cuts the first message and the first reply to 2,000 characters each.
17. Ask for sentence case in the prompt, so Claude and Codex titles look the same.
18. In Work Mode, add a hint to the prompt: the chat works on issue `<id>` "`<issue title>`". The model decides whether the title names the issue.
19. Add `cleanTitle(s)`. It keeps the first line, and strips a "Title:" label, Markdown emphasis, quotes and a trailing period.
20. Make `cleanTitle` cut the title to 80 characters.
21. In `Runner.endTurn`, after a finished turn with `AutoTitle` set, clear `AutoTitle` and save the meta.
22. Then start `r.generateTitle` in a goroutine. The board tries once per chat.
23. Make `generateTitle` use `TitleModel`, else the chat's own model. It runs in the chat's directory: the project root, or the worktree.
24. Make `generateTitle` save the title with `ReplaceAutoTitle`. A rename in between wins.
25. Publish a live `title` event (`Text` = the new title) on the chat stream after the save.
26. Give `generateTitle` a 60 s timeout. Cancel it on `Runner.Shutdown`.
27. Make `Runner.Wait` wait for title calls too.

### Web app

28. Add `title` to the events that refresh the chat list (`changesChatMeta` in `chat-thread-utils.ts`).
29. Add `idleChatIcon(chat, size, chatIcon)` in `recent-chats.tsx`. It returns `Hammer` in Work Mode, else `chatIcon`.
30. Use `idleChatIcon` in `recent-chats.tsx`, `AgentView.tsx` and the All chats rows in `ChatHistory.tsx`.
31. Make the `work` state icon in `IssueChats.tsx` a `Hammer`.

### Tests

32. Add Go tests: placeholders set `AutoTitle`; rename clears it; `ReplaceAutoTitle` refuses a stale placeholder; a fake titler retitles after the first finished turn only; a rename during the call wins; a titler error or empty reply keeps the placeholder; a failed first turn waits for the next finished turn; an old chat keeps its title; the Work Mode hint and directory; Shutdown cancels the call.
33. Add Go tests for the CLI args, the reply parsers, the CLIs with fake binaries, `cleanTitle`, `titlePrompt`, `TitleModel` and fast Codex models.
34. Add `TestRealAgents_Title`. It runs the real CLIs with `XPO_TEST_REAL_CLI=1`.
35. Add web tests: `idleChatIcon`, the `Hammer` on a work chat's issue icon, and `changesChatMeta("title")`.

## Decisions

- **One-shot call to a small model, not an xpo tool for the agent.** A tool spends main-model tokens on every chat and depends on the agent remembering to call it. A one-shot call is cheap and reliable.
- **The chat's own provider generates its title.** A Codex-only user needs no Claude install, and the reverse.
- **Not Claude's `ai-title` record.** Headless sessions do not write it.
- **Not Codex `thread/name/set`.** xpo keeps the title in its own meta; the Codex thread name is not shown anywhere in xpo.
- **Codex keeps the user's config.** `--ignore-user-config` would break a custom model provider. The fast model also comes from `model/list`, which reads the same config.
- **The board wires the titlers; `NewRunner` has no default.** Tests that build a runner never call a real CLI.
- **No issue prefix in the title; an icon marks Work Mode.** A prefix spends title space on what the icon and the issue chip already show. The model gets the issue as a hint, so it can still name it.
- **The icon follows the current mode, like the composer.** A chat that left Work Mode after a merge shows its Chat Mode icon again.
- **The All chats page keeps its muted ring for Chat Mode.** Only Work Mode chats get the hammer there, so the page stays quiet.
- **`AutoTitle` is a positive flag, not a `title_source` field.** A chat saved before this change cannot tell a renamed title from a first-message title. With a positive flag, old chats are never touched.
- **No backfill.** Backfill would spawn one CLI call per old chat, and could overwrite titles the user renamed. Old chats keep their titles; the user can rename them.
- **One attempt per chat.** A failure that repeats (no Haiku access, offline) would otherwise spawn a CLI call on every turn. The board logs the failure.
- **A live `title` event, not polling.** The chat list stops polling when the turn ends, before the title arrives.

## Edge Cases

- **HIGH — user rename races the title call.** `ReplaceAutoTitle` compares the stored title with the placeholder under `titleMu`. The rename wins.
- **MEDIUM — the turn that put the chat in Work Mode is the first turn.** The prompt gets the Work Mode hint, because the chat is in Work Mode when the title call starts.
- **MEDIUM — the first turn fails or is stopped.** No title is generated. The next finished turn tries, with the first user message and the reply of the first finished turn.
- **MEDIUM — Codex `model/list` has not answered yet, or lists no fast model.** The Codex titler uses the chat's own model at `low` effort.
- **LOW — the model returns an empty or long reply.** `cleanTitle` cuts it; an empty title keeps the placeholder.
- **LOW — the chat is deleted or archived during the call.** The save finds no meta, or saves only the title; nothing else changes.
- **LOW — the board shuts down during the call.** Shutdown cancels the call; the placeholder stays.
- **LOW — old work chats named `<id> <issue title>`.** They keep their titles and get the `Hammer` icon while in Work Mode.

## Assumptions

- A few seconds of delay before the title appears is fine.
- Codex quota for one small call per chat is negligible for the user.
- The title call runs in the chat's directory with no tools, so it reads and writes nothing.

## Resolved Questions

1. **Codex model.** The fast model from `model/list` (today `gpt-6-luna`), else the chat's own model at `low` effort.
2. **Work chat titles.** No prefix. The model gets the issue ID and title as a hint. Chat lists mark Work Mode with a `Hammer` icon.
