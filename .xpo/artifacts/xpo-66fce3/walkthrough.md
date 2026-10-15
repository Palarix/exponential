# Walkthrough: chat cost totals add Claude's cumulative cost every turn

## What was wrong

A chat keeps one Claude session across its turns. Each turn's `result` line carries `total_cost_usd`, which is the **running total for the whole session**, not the cost of that turn. The total also carries over when the process restarts and resumes the session, for example after a mode switch. In the chat logs, the value never drops.

xpo saved that number as `turn_done.cost_usd`, and `Usage.add` added it to the chat's total each turn. Every turn re-added all earlier costs, so the total grew roughly with the square of the number of turns. A 30-turn chat that cost about $12.33 showed $178.73. Short chats looked about right, which is why the bug was easy to miss.

Token counts were not affected. The result's `usage` counts only that turn: the logged values go up and down from turn to turn.

## The fix

### 1. Count differences, not totals (`internal/chat/event.go`)

`Usage` now remembers `CostReported`, the last running total it saw. `add` works out the turn's cost:

- total ≥ last total: the turn cost `total − last`
- total < last total: a new session that starts from zero, so the turn cost `total`

Events with no cost (Codex, or a turn that reported none) leave the cost fields alone, so `CostKnown` still means "some turn reported a price".

### 2. The event keeps the raw value

`turn_done.cost_usd` still stores exactly what the agent reported, and the field's comment now says so. This is deliberate: old and new logs mean the same thing, so one rule rebuilds the total for any chat. If the event stored the turn's own cost instead, a log could mix both kinds of value with no way to tell them apart.

### 3. Rebuild wrong totals once (`internal/chat/store.go`)

`Usage` has a `Rev` field and the package defines `usageRev = 1`. `Store.Meta` already rebuilt totals from the log when `Usage` was nil. It now also does so when `Rev < usageRev`, and saves the result, so each chat's log is read only once. `Store.Create` starts new chats at the current revision. If the counting rule changes again, raising `usageRev` rebuilds every chat.

`Runner.endTurn` already calls `Store.Meta` before it logs a new `turn_done`. An old chat is therefore rebuilt before the new turn is added, and that turn is not counted twice.

## Known limit

If a session resets and its new total passes the old one before a turn ends, the reset can't be seen. Part of that turn's cost is then missed. The error only makes the total too low, and the case is unlikely, because Claude carries the cost over on resume.

## Acceptance criteria

- [x] Running totals 0.2, 0.5, 1.0 in one chat add up to `cost_usd` 1.0. Evidence: `TestUsage_AddsCostDifferences/running_total` (0.25, 0.5, 1 → 1).
- [x] A drop in the total counts as a reset. Evidence: `TestUsage_AddsCostDifferences/reset_when_the_total_drops` (0.5, 1, 0.25, 0.5 → 1.5).
- [x] Codex turns (cost 0) leave `cost_usd` at 0 and `cost_known` false. Evidence: `TestUsage_AddsCostDifferences/no_cost_reported`.
- [x] A saved meta without `rev` is recomputed from its log on read and saved with `rev: 1`. Evidence: `TestStore_MetaRecomputesOldUsage`; the summed total of 1.75 becomes 1, and a later read does not rebuild it again.
- [x] A runner test covers several turns in one live session. Evidence: `TestRunner_TurnsAddUsage` runs 3 turns in one process (it checks the start count) with totals 0.01, 0.02 and 0.03, and expects 0.03. This test used to expect the summed value, so it had locked in the bug.
- [x] `make test` passes (lint, 1081 frontend tests, Go tests).
