# Agent chat: chat cost totals add Claude's cumulative cost every turn

## What

A chat's `Usage.CostUSD` adds up the cost of each turn correctly. Chats saved with the wrong totals are fixed once, from their event logs.

## Why

Claude's `result.total_cost_usd` is a running total for the whole Claude session, not the cost of one turn. The total carries over when the process restarts on resume, for example after a mode switch. xpo saves it as `turn_done.cost_usd`, and `Usage.add` adds each value to the chat total. The total grows roughly with the square of the number of turns.

Evidence, chat `a21a526f…` (30 turns, one Claude session, one mode switch):

- `turn_done.cost_usd`: 0.21, 0.28, 0.55 … 11.58, 12.33. The values only go up, including across the mode switch.
- Saved `usage.cost_usd`: **$178.73**. The real cost is about **$12.33**.

Token counts are fine. `tokens_in` and `tokens_out` come from the result's `usage`, which counts only that turn. The values go up and down from turn to turn (for example 6.7M, then 614K, then 307K).

## How

1. **Event meaning stays the same.** `turn_done.cost_usd` keeps the value the agent reports. For Claude that is the running session total. Document this on the field. Old and new logs then mean the same thing, so the same rule can recompute any chat.
2. **`Usage` counts the difference.** Add `CostReported float64` (`json:"cost_reported,omitempty"`), the last running total seen. In `Usage.add`, when `e.CostUSD > 0`:
   - `e.CostUSD >= CostReported`: add `e.CostUSD - CostReported`.
   - `e.CostUSD < CostReported`: the total was reset (new session, or a resume that did not carry the cost over). Add `e.CostUSD`.
   - Set `CostReported = e.CostUSD` and `CostKnown = true`.
   
   Codex reports 0, so nothing changes for Codex.
3. **Recompute old totals once.** Add `Rev int` (`json:"rev,omitempty"`) to `Usage`, with `const usageRev = 1`. `Store.Meta` now rebuilds usage from the log when `Usage == nil` **or** `Usage.Rev < usageRev`. The rebuild is the existing `backfillUsage`, which sets `Rev`. `Store.Create` sets `Rev: usageRev`.
4. No frontend change: the UI already reads `usage.cost_usd` and `cost_known`.

## Edge cases

- **Stopped turn** (no `turn_done`): its cost is in the next running total, so it ends up in the next turn's difference. That is correct.
- **Reset followed by a total above the old one**: the reset can't be detected, so part of the cost is missed. This is an unlikely case and only makes the total too low.
- **Turn ended with an error**: there is no `turn_done`, so the same as a stopped turn.

## Acceptance criteria

- [ ] Running totals 0.2, 0.5, 1.0 in one chat add up to `cost_usd` 1.0.
- [ ] A drop in the total (1.0, then 0.3) counts as a reset: total 1.3.
- [ ] Codex turns (cost 0) leave `cost_usd` at 0 and `cost_known` false.
- [ ] A saved meta without `rev` (wrong old total) is recomputed from its log on read and saved with `rev: 1`.
- [ ] A test in the runner covers several turns in one live session and checks the chat's `usage.cost_usd`.
- [ ] `make test` passes.
