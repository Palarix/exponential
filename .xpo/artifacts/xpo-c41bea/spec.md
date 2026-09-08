# Spec: Fix sort_order not persisted on status transition

## What

`buildUpdate` appends the primary event at line 54 (copying the struct into the slice), then computes `sort_order` at lines 57-75 and mutates the local `primaryEvent` variable — but the copy in `eventsToAppend[0]` is unchanged.

## How

Move the sort_order computation before the append. The primary event is built with the final payload (including `SortOrder` if applicable), then appended once.

## Test

Add `TestUpdateIssue_StatusChange_AssignsSortOrder`:
- Create two issues in PLANNED with known sort orders
- Move a third issue from BACKLOG to PLANNED
- Verify the resulting issue has a non-empty `SortOrder` that sorts after the existing keys

## AC

- [ ] Status transition assigns a persisted `SortOrder`
- [ ] Existing tests still pass
- [ ] `make test` passes
