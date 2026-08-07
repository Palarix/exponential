# Walkthrough: MCP list sort order fix

## What changed

Two files were modified:

### 1. `internal/exponential/projection.go` — `SortIssues` function

The function had two `sort.Slice` calls that compared issues by `CreatedAt`. Both were changed to compare by `SortOrder` (lexicographic string comparison on fractional-index keys) first, falling back to `CreatedAt` only as a tiebreaker.

**Root sorting (line ~324):** Previously `rootI.CreatedAt.Before(rootJ.CreatedAt)`. Now checks `SortOrder` first — if the two roots have different sort orders, the lexicographically smaller one comes first. Only if sort orders are identical does it fall back to creation time.

**Child sorting (line ~348):** Same change. The existing parent-first guard (lines 339-344) is preserved — a parent always sorts before its own children. Among siblings, `SortOrder` now determines order instead of `CreatedAt`.

This works because `backfillSortOrder` (called at the end of `ProjectIssues`) already ensures every issue has a non-empty `SortOrder` value. Legacy issues without explicit sort orders get backfilled keys in creation-time order, so the tiebreaker path is mainly for theoretical edge cases.

### 2. `internal/exponential/sort_test.go` — strengthened and added tests

**`TestSortIssues_SortOrderWithinGroup`** — Previously only checked that both issues were present. Now asserts the actual order: issue "b" (sort order "a") must come before issue "a" (sort order "b").

**`TestSortIssues_ReorderedSortKeysReflected`** (new) — Three issues with `CreatedAt` in one order (x, y, z) but `SortOrder` in a different order (y, z, x). Asserts the output follows `SortOrder`, proving that drag-and-drop reordering is respected even when it contradicts creation time.