# Spec: Elysium-style composer pickers for model, effort and permissions

## What

The composer footer becomes one compact row of segments, like `✳ Claude Opus 5.5 ⌄ | Medium · 1M ⌄ | 🔒 Read-only`:

- **Model:** a provider icon plus the model's full name. One menu lists every provider's models, grouped by provider. It replaces the separate agent picker from xpo-5b3157.
- **Effort and context:** the effort level, plus the model's context window once it is known.
- **Permissions:** a read-only badge with no menu.

Full names and context windows are learned from turns the agent has already run. No probe and no hard-coded table.

## Why

Follow-up to xpo-56ae3a and xpo-5b3157, part of xpo-3798b7. Today the footer shows "Default model" and "Default effort", which tell the user nothing about what will run. Claude's init line already names the real model (`opus` → `claude-opus-5-5`, default → `claude-opus-5-5[1m]`), and its result reports the context window. Codex reports both too (`thread/start` → `model`, token usage → `modelContextWindow`).

## Acceptance Criteria

- [ ] After a turn, the chat meta stores the model the agent actually ran (`resolved_model`) and its context window in tokens (`context_window`). Claude: model from the `system/init` line, window from the result's `modelUsage[model].contextWindow`. Codex: model from the `thread/start`/`thread/resume` response, window from `modelContextWindow` in token usage.
- [ ] The catalog learns from every finished turn: for (provider, picked model ID), it stores the resolved model's full name and context window. At board start it learns from the saved chat metas, oldest first, so the newest turn wins. `GET /api/local/agent` returns them on each model as `name` and `context_window` (omitted when unknown).
- [ ] Full names: a Claude model ID is formatted as `claude-opus-5-5[1m]` → "Claude Opus 5.5". A Codex model uses its `model/list` display name, else the ID. Codex models, including the default (`isDefault`), are named from `model/list` before any turn; a turn's actual model overrides that, because a user's `~/.codex/config.toml` can pick a different default.
- [ ] The model segment shows the provider icon and the full name when known, else the alias label ("Opus", "Default model"). Its menu groups models under provider headings. The default option reads "Claude Opus 5.5 (Default)" once the agent has reported it, and is picked and checked for a new user. Each item shows the full name when known.
- [ ] Picking a model of another provider switches the provider and resets the effort, like `withProvider`. Once a chat has messages, the other providers' groups are disabled, with the existing tooltip.
- [ ] There is no "Default effort" option: the effort lists hold only real levels. The pre-selected effort is Medium, or the model's first level when it has no Medium. The picked effort is saved in localStorage with the model and remembered for new chats. When a saved effort is not offered by the picked model, it falls back the same way. The segment reads "Medium · 1M": the effort label, then the context window of the selected model when known (200K, 1M; rounded, `K`/`M`). Context is display only; there is no context choice. The API still accepts an empty effort, meaning the CLI's default.
- [ ] The permissions segment shows a lock icon and "Read-only". It has no menu. Its tooltip says chat can read code and change issues, and that code changes happen in work sessions (xpo-00f0e5).
- [ ] Segments are separated by thin vertical rules, in compact text. The row fits the 460px popup: only the model name truncates; effort and the badge keep their width. Menus open above the composer when there is no room below. The full-page view (xpo-c047ce) uses the same row.
- [ ] Opening an existing chat, in the popup or the full-page view, shows the agent, model and effort its last message used, from the chat's meta (`agent`, `model`, `effort`). A pick made in an open chat applies to that chat and is also remembered in localStorage for new chats. Opening a chat does not change the remembered prefs. A new chat, or one with no message yet, uses the remembered prefs.
- [ ] The chat popup is 460px wide (was 420px), still capped by the viewport width.
- [ ] Polish found in tophat: the full-page empty state spaces title, composer, examples and recent chats evenly (32px). The recent chats popover opens centered above its button (clamped to the viewport). A framed `Menu` with `maxHeight` measures its scrollbar gutter and pads to an even inset on every side, whatever scrollbar width the browser uses.
- [ ] The board refetches the agent status after a turn ends, so names learned from the first turn show without a reload.
- [ ] Claude's model list gains Fable (`fable`).
- [ ] `make test` passes. Go tests cover resolving and learning for both drivers, the name formatting, and learning from metas. Frontend tests cover the grouped menu, the labels, the context suffix and the badge.

## Flow

1. **Drivers report the model.** `TurnResult` gains `Model string` and `ContextWindow int`.
   - **Claude:** `ClaudeStreamParser` records the init `model`. The result's `modelUsage` is read for that model's `contextWindow`; if the key is missing, use the largest window among the entries.
   - **Codex:** `codexTurn` reads `result.model` from the thread response, and keeps the latest `modelContextWindow` from `thread/tokenUsage/updated`.
2. **Runner.** On `turn_done`, the runner saves `ResolvedModel` and `ContextWindow` to the meta. It calls `Learner.Learn(provider, meta.Model, model, window)`, which is optional (nil in tests).
3. **Catalog.**
   - `ModelOption` gains `Name string json:"name,omitempty"` and `ContextWindow int json:"context_window,omitempty"`.
   - `Learn(provider, picked, resolved, window)` records what `picked` ran as. `Status()` sets `Name` from `modelName(provider, resolved, models)`: Claude IDs are formatted, Codex IDs are looked up in the current list.
   - Learned values live in a separate map keyed by (provider, picked ID), applied in `Status()`. So `SetModels` never drops them, and a Codex name resolves against whichever model list is current.
   - The board calls `LearnFrom(store.List())` once after building the catalog.
4. **Frontend.**
   - `composerOptions(status, prefs, providerLocked)` returns `{groups, model, modelLabel, efforts, effortLabel}`. A group is `{provider, label, disabled, options}`. Menu values are `provider/model` (`modelKey`), and `pickModel` turns one back into prefs.
   - The new `ComposerPickers` component holds the three segments. Each segment opens the shared `Menu` (keyboard navigation, radio items, `MenuLabel` headings) in a `Popover`; the badge is static. With one provider the headings are hidden. When other providers are locked, a note at the end of the menu says why.
   - The agent context exposes `prefsFor(chatId)` and `setPrefsFor(chatId, prefs)` instead of one global `prefs`. `chatPrefs(meta, picked, saved)` picks: a pick made in this chat, else the meta, else the remembered prefs; `normalizePrefs` then checks it against the provider list.
   - `useChatThread` invalidates the agent status query on `turn_done`.
   - Provider icons (`ProviderIcon`): the Claude mark in #D97857 and the ChatGPT mark in white (the board is dark-only), inlined from the user's SVGs with `fill="currentColor"`; a generic lucide `Bot` for any other provider.

## Decisions

| Question | Choice | Rationale |
|---|---|---|
| Permissions | Read-only badge only (confirmed) | Unrestricted agents belong in worktree work sessions (xpo-00f0e5). Ask and Full access move there. |
| Full model names | Learned from turns (confirmed) | No extra API calls, and names stay current when a model ships. Until the first turn the alias shows. |
| Context window | Display only (confirmed) | Opus already reports 1M without `[1m]`, and Fable ignores the suffix. A choice would often be a no-op. |
| Provider picker | Folded into the model menu as groups | The issue asks for grouping by provider. One segment saves width in the 460px popup. |
| Effort default | Medium, no "Default" option (user) | "Default effort" did not say what runs. Medium is a named level every current model offers. |
| Unsupported effort | Already fixed in xpo-5b3157 | Efforts are per model. All current Claude models accept every effort (tested: haiku, sonnet, fable and opus with xhigh and max). |
| Where learned names live | In the catalog, rebuilt from chat metas | No new file. A deleted or pruned chat just loses its sample. |

## Edge Cases

- **MEDIUM — the default model changes** (the user changed Claude settings). The next turn re-learns it; until then the old name shows. Acceptable.
- **LOW — `chat.model` is set.** The picked ID is "" for the default, and the runner sends `chat.model`. The name learned for "" is then what `chat.model` resolves to, which is what the default really runs.
- **LOW — a failed or stopped turn** learns nothing; only `turn_done` does.
- **LOW — an unknown model ID format** (not `claude-…`) shows the ID as is.
