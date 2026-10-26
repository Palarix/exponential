# Spec: Codex reports Luna after GPT-6.1-Sol is selected

## What
A Codex chat with an explicit model pick (GPT-6.1-Sol) records `resolved_model = gpt-6-luna`, and the composer then names the pick "GPT-6-Luna".

## Why (root cause)
Codex does run the picked model: `startTurn` sends `model` on `turn/start`, which app-server documents as "Override the model for this turn and subsequent turns". But the driver reports the wrong one. `codexSession.model` is set once, from the `thread/start` / `thread/resume` response, and that is the thread's default from `~/.codex/config.toml` (Luna). `turn/completed` returns that as `TurnResult.Model`. The runner then stores it as `resolved_model`, and `Catalog.Learn` maps the pick `gpt-6.1-sol` to `gpt-6-luna`. From then on, `Status()` names the Sol option "GPT-6-Luna".

The turn objects in app-server 0.164 do not carry a model. The only other source is the `model/rerouted` notification (`fromModel`, `toModel`, `turnId`). Codex sends it when it moves a turn to another model.

## How
In `internal/chat/codex.go`:
1. Keep the thread's default model (`defaultModel`, from the thread response) apart from the model the thread runs now (`model`).
2. `startTurn`: when the turn names a model, send it and set `s.model` to it. When the turn picks Default (`""`) and an earlier turn overrode the model, send `defaultModel`, because the override sticks to the thread. That way Default runs the default again, and `s.model` becomes `defaultModel`.
3. Handle `model/rerouted` for the current turn: set `s.model = toModel`, so the turn reports the model that really answered.
4. `turn/completed` keeps reporting `s.model`.

Effort is not changed. Its "Default" has the same sticky-override behaviour, but nothing reports it wrong. I can file it separately if you want.

Chats saved with the wrong `resolved_model` keep teaching the wrong name at startup (`LearnFrom`) until the next Sol turn. The newest turn wins, so one turn fixes it. No migration.

## Acceptance Criteria
- [ ] With a thread default of Luna, a turn sent with `gpt-6.1-sol` sends `model: gpt-6.1-sol` on `turn/start` and reports `TurnResult.Model = gpt-6.1-sol`.
- [ ] A Default turn after a Sol turn sends the thread's default model and reports it.
- [ ] A `model/rerouted` notification for the turn makes the turn report `toModel`.
- [ ] Regression test from selection to turn to chat meta: a runner chat with `Model = gpt-6.1-sol` on a Codex thread whose default is Luna ends with `meta.ResolvedModel = gpt-6.1-sol`, and the catalog names the Sol option "GPT-6.1-Sol".
- [ ] `make test` passes.
