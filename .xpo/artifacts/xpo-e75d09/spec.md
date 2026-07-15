# CycleID never validated during create or update

## Problem

The `cycle_id` field is stored without validation. Invalid values like "not-a-date" are accepted.

## Fix

Add cycle_id validation to `ValidateCreatePayload` and `ValidateUpdatePayload` in `internal/exponential/client.go`:

- Non-empty cycle_id with cycles not enabled → error "cycles are not enabled"
- Non-empty cycle_id with cycles enabled → validate via `CycleConfig.CycleForID()` (checks YYYY-MM-DD format)
- Empty cycle_id → no validation (clearing or not set)

## Acceptance Criteria

- [ ] Creating/updating with an invalid cycle_id returns an error.
- [ ] Creating/updating with a cycle_id when cycles aren't enabled returns an error.
- [ ] Valid cycle_id values are accepted.
- [ ] Empty cycle_id (clearing) is accepted.
- [ ] Tests pass.
