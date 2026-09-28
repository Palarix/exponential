# Spec: Fix collapse ordering/attribution and repair the event log

## What

1. Fix `storage.AppendEventCollapsed` so collapsed events keep file order and timestamps in step and never mix actors.
2. Add a report-only `xpo doctor` check that `created_at` strictly increases in file order.
3. Repair this repo's `issues.db` once with a **one-time script**. This is not a doctor fixer, and xpo still never rewrites the issue log itself.

## Why

See the issue. File order is correct; timestamps are not. Anything that orders by time (history, metrics, RepoDB in xpo-09f337) builds the wrong board.

## How

### 1. Collapse (`internal/storage/collapse.go`)

**UPDATE merge target rule.** An incoming UPDATE merges only into the **most recent uncommitted event for the same issue**, and only when that event:
- is an UPDATE, and
- has the same `created_by`, `on_behalf_of` and `source`.

Otherwise the incoming event is appended.

> Why not the looser "latest UPDATE by the same actor": web sets PLANNED (pending), an agent sets DOING, then web changes `sort_order`. The looser rule merges into the old web event and moves it after DOING, so the stale PLANNED comes back.

**Move to end.** A merged UPDATE is removed from its position and appended to the end of the uncommitted tail with the incoming `created_at`. Since it is already the issue's latest event, the issue's projected state is unchanged.

**Comment edits.** These keep merging in place (same comment ID) as today, but **keep the original `created_at`**. The comment keeps its original time and position in the thread. Projection lists comments in event order and doesn't dedupe by ID, so moving the event or splitting it by actor would reorder the thread or duplicate the comment.

**No-op pruning uses running state.** `pruneNoopUpdates` compares each uncommitted UPDATE against the state projected from committed events plus the uncommitted events kept before it, not against committed state alone. Without this, "committed PLANNED → agent DOING → web PLANNED" would prune the web revert and leave the issue in DOING. `projectCommittedState` is refactored into a per-event apply helper shared by both.

### 2. Doctor check (`storage.CheckEventOrder`)

- Returns violations `{Line, Kind: out_of_order|tie}` for `issues.db` in file order.
- Doctor passes when there are none. Otherwise it shows a `note` with counts, the first 10 line numbers, and "…and K more". The problem goes into "Cannot fix automatically" with a short explanation. Nothing is rewritten, including under `--fix`.

### 3. One-time repair script

Go program at `scripts/repair-event-timestamps/main.go` in the worktree. It is used to run the repair and is **not committed**; it's attached to the issue as an artifact for the record.

1. Parse `created_at` of every raw line of `.xpo/issues.db`.
2. Keep the longest non-decreasing subsequence of timestamps untouched.
3. Re-time every other line evenly between its kept neighbours. Lines before the first kept line or after the last get ns offsets from their single neighbour.
4. Tie-break in file order: if `t ≤ prev`, set `t = prev + 1ns`.
5. Replace only the top-level `created_at` value on changed lines (the last `"created_at":"…"` occurrence). Re-parse each changed line and assert that only `CreatedAt` differs. Unchanged lines stay byte-identical.
6. Write a backup to the scratchpad first (not into `.xpo/`), then write through a tmp file and rename.
7. Verify and print the results:
   - `CheckEventOrder` is clean.
   - Projection (excluding timestamps and the event list) is identical to the original in file order.
   - Repaired file order and time order give identical projections.

Run it after user approval, just before merge, when no other clone has unpushed `issues.db` edits.

## Acceptance criteria

- [ ] Collapse never merges UPDATEs across `created_by`/`on_behalf_of`/`source`. Test: an agent MCP update followed by a web update to the same issue produces two events with correct attribution.
- [ ] Collapse doesn't merge past a newer event for the same issue. Test: web PLANNED → agent DOING → web sort_order gives final status DOING.
- [ ] A collapsed UPDATE is moved to the end of the uncommitted tail. Test: after interleaved web edits to A and B, timestamps strictly increase in file order.
- [ ] Comment edit collapse keeps the original timestamp and position.
- [ ] No-op pruning still works (A=foo → bar → foo leaves no pending UPDATE) and uses running state (committed PLANNED → agent DOING → web PLANNED ends PLANNED).
- [ ] The doctor check reports out-of-order and tied timestamps with line numbers and never rewrites the log.
- [ ] The one-time script repairs this repo's `issues.db`. Afterwards the check is clean, projection is unchanged apart from timestamps, and file order and time order match.
- [ ] `make test` passes.
