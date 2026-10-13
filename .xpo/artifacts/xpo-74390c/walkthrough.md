# Walkthrough: Elysium-style composer pickers for model, effort and permissions

## What changed

The chat composer's footer used to show "Default model" and "Default effort", which told the user nothing about what would run. Now it is one compact row:

`✳ Claude Opus 5.5 ⌄ | Medium · 1M ⌄ | 🔒 Read-only`

- **Model:** the agent's mark plus the model's full name. One menu lists every installed agent's models under provider headings, which replaces the separate agent picker from xpo-5b3157.
- **Effort and context:** a real effort level (Medium by default) and the model's context window once it is known.
- **Read-only badge:** no menu. Chat stays read-only; code changes belong to work sessions (xpo-00f0e5).

Each chat also reopens with the agent, model and effort it last used. Tophat added some UI polish, covered at the end.

## How the pieces fit

### Learning real model names (backend)

There is no API to ask Claude Code which model "default" or "opus" means. The agents do report it while they run, though:

| | Model | Context window |
|---|---|---|
| Claude | `system/init` line → `model` (`claude-opus-5-5[1m]`) | result → `modelUsage[model].contextWindow` |
| Codex | `thread/start` / `thread/resume` response → top-level `model` | `thread/tokenUsage/updated` → `modelContextWindow` |

1. **Parsers.** `ClaudeStreamParser` (`internal/exponential/claudestream.go`) keeps the init model and reads the context window from the result. If the init line names an alias that has no `modelUsage` key, it falls back to the largest window. `codexTurn` (`internal/chat/codex.go`) keeps the thread response's model and the latest `modelContextWindow`. Both end up in `TurnResult.Model` and `TurnResult.ContextWindow`.
2. **Runner.** On `turn_done`, and only then, `runner.run` saves `resolved_model` and `context_window` to the chat meta and calls `Learner.Learn(provider, meta.Model, model, window)`. A failed or stopped turn teaches nothing (tested).
3. **Catalog.** `Catalog.learned` maps (provider, picked model ID) to the resolved model and its window. `Status()` applies it on every read: `Name` comes from `modelName`, which formats Claude IDs (`claudeModelName`: `claude-opus-5-5[1m]` → "Claude Opus 5.5", trailing release dates dropped, unknown formats returned as is) and looks Codex IDs up in the current model list. Because learned values sit beside the model list instead of inside it, `SetModels` can replace Codex's list at any time without losing them. A Codex name learned before `model/list` answers resolves once the list arrives.
4. **Startup.** `board.go` calls `catalog.LearnFrom(store.List())`. Metas are sorted oldest first, so the newest turn for each picked model wins.
5. **Codex default before any turn.** `codexModelOptions` names every model, including the default, from `model/list` (`isDefault`). A real turn overrides that. This matters in practice: on the dev machine `model/list` called GPT-6.1-Sol the default, but turns ran GPT-5.6-Sol because the user's `~/.codex/config.toml` picks it.

The frontend sees all this as `name` and `context_window` on each `ModelOption`. `useChatThread` invalidates the agent status query on `turn_done`, so a name learned in the first turn shows without a reload.

### Effort: no "Default"

The effort lists in `agent.go` and `codexModelOptions` no longer contain `""`. `Catalog.Resolve` still accepts an empty effort, which means the CLI's default, so API callers that send none keep working. In the frontend, `pickEffort` keeps the wanted level if the model offers it, else Medium, else the model's first level. `normalizePrefs` and `withModel` both use it. Old chats, saved with an empty effort, open on Medium.

### Prefs per chat (frontend)

The agent context used to hold one global `prefs`, which caused two bugs:

- Opening a chat in the popup copied its settings into the remembered new-chat prefs.
- The full-page view never restored a chat's settings at all, because it opens chats through URL navigation, not `openChat`.

Now the context exposes `prefsFor(chatId)` and `setPrefsFor(chatId, prefs)`. `chatPrefs(meta, picked, saved)` decides, in this order:

1. A pick made in this chat during the session.
2. What the chat's last message used, from its meta (`agent`, `model`, `effort`).
3. The remembered prefs from localStorage, for a new chat or one with no message yet.

`normalizePrefs` then checks the result against the provider list. `setPrefsFor` records the pick for that chat and also saves it to localStorage, so the last thing the user picked becomes the default for new chats. Opening a chat no longer writes to localStorage. `useChatSession` sends `prefsFor(chatId)`, so a message uses what the composer shows.

### The picker row

- `composerOptions(status, prefs, providerLocked)` in `agent-state-utils.ts` builds everything the row needs:
  - `groups`: one per available provider, disabled for other providers once the chat has started.
  - `model` and `modelLabel`: the menu value and the trigger text.
  - `efforts` and `effortLabel`: the effort list, and "Medium · 1M" for the trigger.
- Menu values are `provider/model` (`modelKey`), so one menu can hold every provider's models. `pickModel` turns a value back into prefs: switching provider resets the model and keeps a supported effort.
- The default entry reads "Claude Opus 5.5 (Default)" once the name is known, else "Default model". It is the pre-selected value, so it shows checked from the start.
- `ComposerPickers.tsx` renders the row. Each segment opens the shared `Menu` (keyboard navigation, radio items, `MenuLabel` headings) inside a `Popover`, which flips above the composer by itself. Headings are hidden when only one provider is installed. When other providers are locked, a note at the end of the menu explains why.
- `ProviderIcon.tsx` inlines the user's Claude and ChatGPT SVG paths with `fill="currentColor"`. Claude is drawn in #D97857 and Codex in white, which is safe because the board is dark-only. Other agents get a lucide `Bot`.
- Layout: only the model segment may shrink and truncate; effort and the badge keep their width. Without that, the first version truncated both labels in the popup.

### Tophat polish

- **Popup width:** the chat popup is 460px wide (was 420px), still capped by the viewport.
- **Empty-state spacing:** the full-page empty state's examples row has `-mt-6`, which cancels the composer's `pb-6`. The composer keeps that padding for when it sits at the bottom of a running chat. Title, composer, examples and recent chats are now 32px apart (measured).
- **Even menu insets:** a framed `Menu` with `maxHeight` uses `scroll-stable`, whose side inset is the scrollbar gutter, and the browser picks its width: 11px in Chrome with thin scrollbars, 0 with macOS overlay scrollbars. `Menu` now measures the gutter in a layout effect and applies `scrollMenuPadding(gutter)`: block padding `max(8, gutter)` and inline padding `max(0, 8 - gutter)`. Every side ends up with the same inset (12px measured in Chrome, where it was 9 at the top and bottom against 12 at the sides). This also fixes rows that sat flush against the border with overlay scrollbars.
- **Recent chats popover:** it uses `bottom-center` placement, which flips above the status bar. Because the button is near the right edge, the 384px popover is clamped to the viewport and does not fully center on it. The user accepted that over making the popover narrower.

## Key decisions

- **Permissions** (user): the read-only badge only. Ask and Full access move to worktree work sessions (xpo-00f0e5).
- **Names learned from turns** (user): no extra API calls, and names stay current when models ship. The cost is that a model shows its alias until it has run once.
- **Context window display only** (user): a 200K/1M choice would often do nothing. Opus 5.5 already reports 1M without `[1m]`, and Fable ignores the suffix.
- **Effort defaults to Medium** (user, in tophat): "Default effort" did not say what would run.
- **Fable added** to Claude's model list.
- **Unsupported efforts:** there is no special handling. All current Claude models accept every level (tested: haiku, sonnet, fable and opus with xhigh and max).

## Acceptance criteria

- [x] After a turn the meta stores `resolved_model` and `context_window`, from the Claude init line and result and from the Codex thread response and token usage. Evidence: `TestClaudeStreamParser_ModelAndContextWindow`, `_ContextWindowFallsBackToLargest`, `TestRunner_RecordsResolvedModel`, and the assertions in `TestCodexDriver_Turn`. Live: claude-haiku-5-5 and claude-opus-5-5[1m] (1M), and gpt-5.6-sol (258400).
- [x] The catalog learns from finished turns and from saved metas, newest first, and `GET /api/local/agent` returns `name` and `context_window`. Evidence: `TestCatalog_Learn`, `TestCatalog_LearnFromMetas`, `TestRunner_FailedTurnLearnsNothing`, `TestChat_AgentStatusListsProviders`.
- [x] Full names: Claude IDs are formatted, and Codex uses its display names, including the default before any turn, with a real turn overriding it. Evidence: `TestModelName`, `TestCatalog_LearnCodexUsesDisplayName`, `TestCodexModels`.
- [x] The model segment shows the agent's mark and the full name, the menu is grouped by provider, and the default reads "Name (Default)" and is checked from the start. Evidence: `composerOptions` tests and `ChatComposer.test.tsx`. Browser screenshot of the menu with both groups.
- [x] Picking another provider's model switches the provider. Once the chat has started, the other groups are disabled. Evidence: the `pickModel` tests and "disables the other providers once the chat has started".
- [x] No "Default effort": Medium is pre-selected (or the first level), picks are saved in localStorage, and the trigger reads "Medium · 1M". Evidence: the `normalizePrefs`, `withModel` and `composerOptions` tests, `TestCatalog_Resolve` (an empty effort is still accepted), and `formatContextWindow`.
- [x] The read-only badge has a lock and a tooltip. Evidence: `ChatComposer.test.tsx`, "shows a read-only badge…".
- [x] The row fits the 460px popup, only the model name truncates, and the full-page view uses the same row. Evidence: browser screenshots of both views.
- [x] Opening an existing chat restores its agent, model and effort in both views, without changing the remembered prefs. Evidence: the `chatPrefs` tests. Browser check against six saved chats: Codex chats restored as GPT-5.6-Sol at Low, the haiku chat as Claude Haiku 5.5, and a new chat stayed on Claude Opus 5.5 at Medium.
- [x] The popup is 460px wide. Evidence: measured at 460px in the browser.
- [x] Polish: even 32px spacing in the empty state, the recent chats popover centered above its button (clamped), and even menu insets. Evidence: measured gaps of 32/32/32, menu insets of 12/12/12/12, and the `scrollMenuPadding` test.
- [x] The agent status is refetched after `turn_done`. Evidence: `useChatThread.ts`. Live: names appeared after the first turn without a reload.
- [x] Fable is in Claude's list. Evidence: `TestCatalog_FableModel`.
- [x] `make test` passes, with Go tests for resolving, learning, naming and metas, and frontend tests for the menu, labels, context, badge and per-chat prefs.
