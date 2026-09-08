# Walkthrough: Fix sort_order not persisted on status transition

## What changed

### Production: `update.go`

Moved the `sort_order` auto-assignment block (lines 57-77) *before* the primary event is built and appended. Previously, the event struct was appended at line 54 (copied into the slice), then `payload.SortOrder` was set and `primaryEvent.Payload` was mutated — but the copy in the slice was untouched, so the generated key was never persisted.

The fix is a pure reorder — no logic changes. The sort_order computation now runs between `pruneUnchangedFields` and the event construction.

### Test: `update_test.go`

`TestUpdateIssue_StatusChange_AssignsSortOrder`:
- Creates two PLANNED issues with known sort orders (`"a0"`, `"a1"`)
- Moves a BACKLOG issue to PLANNED
- Verifies the resulting `SortOrder` is non-empty and sorts after `"a1"`

Verified the test fails against the old code (returns `"a0"` — the stale value) and passes with the fix.
