# Walkthrough: Codex reports Luna after GPT-6.1-Sol is selected

## What was wrong
Codex ran the picked model, but xpo reported the wrong one. The session log in `~/.codex/sessions` for the reported chat shows `turn_context.model = gpt-6.1-sol`. The chat meta said `resolved_model = gpt-6-luna`.

The Codex driver set `codexSession.model` once, from the `thread/start` / `thread/resume` response. That value is the thread's default from `~/.codex/config.toml`, not the model a turn runs. Every `turn/completed` reported it as `TurnResult.Model`. The runner stored it as `resolved_model`, and `Catalog.Learn` then mapped the pick `gpt-6.1-sol` to `gpt-6-luna`, so the composer named the Sol option "GPT-6-Luna".

The test fixture hid the bug: its thread default was the same model the test picked.

## What changed (`internal/chat/codex.go`)
- **Two fields instead of one.** `defaultModel` is the thread's default, taken from the thread response. `model` is the model the thread runs now.
- **`startTurn`** sets `model` from the turn's pick. In app-server, `turn/start`'s `model` overrides "this turn and subsequent turns", so the override sticks to the thread. A turn that picks Default (`""`) after an override therefore sends `defaultModel` explicitly. Without that, Default would quietly keep running the earlier pick. When no override happened, a Default turn still sends no model.
- **`model/rerouted`** (app-server notifies it when Codex moves a turn to another model) sets `model` to `toModel` for the current turn, so the turn reports the model that answered.
- `turn/completed` still reports `s.model`. It is now correct.

## Not changed
- Effort has the same sticky-override behaviour for Default, but nothing reports it wrong. Left alone.
- Chats saved with the wrong `resolved_model` still teach the wrong name through `LearnFrom` at startup, until the next turn with that pick. The newest turn wins, so no migration.

## Tests
The fixture's thread default is now `gpt-6-luna`, matching a typical user config. Picks and the default therefore differ in every test.

## Acceptance Criteria
- [x] With a thread default of Luna, a turn sent with `gpt-6.1-sol` sends `model: gpt-6.1-sol` on `turn/start` and reports `TurnResult.Model = gpt-6.1-sol`. Evidence: `TestCodexDriver_Turn`, `TestCodexDriver_PickedModelOverThreadDefault`.
- [x] A Default turn after a Sol turn sends the thread's default model and reports it. Evidence: second half of `TestCodexDriver_PickedModelOverThreadDefault`. `TestCodexDriver_ResumesThreadAfterAgentEnds` checks that a Default turn with no earlier override sends no model and reports the default.
- [x] A `model/rerouted` notification for the turn makes the turn report `toModel`. Evidence: `TestCodexDriver_ModelRerouted`.
- [x] Regression test from selection to turn to chat meta: `TestCodexDriver_PickedModelOverThreadDefault` checks `meta.ResolvedModel = gpt-6.1-sol` and that the catalog names the Sol option "GPT-6.1-Sol". It failed before the fix with "GPT-6-Luna".
- [x] `make test` passes (exit 0).
