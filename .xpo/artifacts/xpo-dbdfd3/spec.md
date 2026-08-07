# Spec: Fix story point aggregation in status group headers

## Problem

The status group header story points sum includes DONE children's estimates for parent issues, and can double-count for non-epic parents with children in the same group.

## Root cause

The group header computation (useBacklogRows.ts lines 76-84) uses `i.estimate` directly from the API. The backend overwrites a parent's `estimate` with the sum of ALL children's estimates regardless of status. The frontend only excludes "epic"-labeled parents to avoid double-counting, missing other parent types.

## Fix

Replace the `epicIds` / raw-estimate logic with a computation that mirrors the parent row badge: for any issue with children, use the incomplete children's points (`childPointsTotal - childPointsDone`) instead of the backend-inflated `i.estimate`. For leaf issues, use `i.estimate || 1` as before.

Also exclude ghost parent rows from the count entirely — their children are already counted individually in the group.

## Acceptance Criteria

- [ ] Group header shows story points only from incomplete work.
- [ ] No double-counting between parent and children in the same group.
- [ ] Ghost parents do not contribute to the group total.
- [ ] `make build` and `make test` pass.
