# Walkthrough: Collapse ordering/attribution fix and event-log repair

## The problem in one paragraph

`issues.db` is an append-only log, and projection replays it in **file order**. The web UI writes through `storage.AppendEventCollapsed`, which folds rapid edits into one pending (uncommitted) UPDATE so the log doesn't fill up with drag-and-drop noise. That folding had three problems:

1. It bumped the merged event's `created_at` but left the event where it was, so timestamps stopped following file order.
2. Old fields rode along under the new timestamp.
3. It merged edits regardless of who made them, so web edits ended up inside an agent's MCP event.

The board looked fine because it replays in file order. Anything that orders by time (history, metrics, the future RepoDB in xpo-09f337) saw a different, wrong board.

## What changed

### 1. `internal/storage/collapse.go`

`AppendEventCollapsed` now delegates by event type:

- **`collapseUpdate`** finds the issue's **most recent** uncommitted event. It merges only if that event is an UPDATE and `sameActor` (same `created_by`, `on_behalf_of`, `source`). Otherwise the incoming event is appended. The merged event is removed from its slot and appended at the end with the incoming timestamp.
  - *Why "most recent event for the issue" and not "most recent UPDATE by this actor"?* Take web PLANNED (pending) → agent DOING → web sort change. The looser rule would merge into the old web event and move it after DOING, and the stale PLANNED would win. Requiring the target to already be the issue's last event means moving it to the end can't reorder anything that matters for that issue.
- **`collapseComment`** still replaces a same-ID comment in place but **keeps the original `created_at`**. Projection lists comments in event order and doesn't dedupe by comment ID. Moving the event would reorder the thread, and splitting it by actor would show the comment twice. Keeping the original time is the only choice that stays consistent with file order without either side effect.
- **`pruneNoopUpdates`** now compares each UPDATE against *running* state: committed state plus the uncommitted events kept before it. `projectCommittedState` was split so the per-event logic lives in `applyStateEvent`, which both use.
  - *Why:* once actors stop merging, "committed PLANNED → agent DOING → web PLANNED" leaves a web event equal to the committed value. The old pruning would drop it and the issue would stay DOING. Running state recognises it as a real revert. The single-actor round trip (foo → bar → foo) still collapses to nothing.
- `mergeUpdatePayloads` lost its always-true `bool` return.

### 2. `internal/storage/event_order.go` + doctor

`CheckEventOrder` scans `issues.db` and returns `{Line, Kind}` for every line whose `created_at` is earlier than (`out_of_order`) or equal to (`tie`) the line before it. Strictly increasing pairs imply the whole file is sorted.

`xpo doctor` shows a pass line, or a note with counts and the first 10 line numbers (`describeOrderViolations`). The note also goes into "Cannot fix automatically". The check is deliberately **report-only**: doctor promises xpo never rewrites the issue log, and we kept that promise.

### 3. One-time repair (not in the codebase)

The repair was run as a throwaway script, attached to this issue as `repair-event-timestamps.go`:

1. Keep the longest non-decreasing subsequence of timestamps untouched.
2. Spread the rest evenly between their kept neighbours.
3. Nudge ties forward by 1 ns steps.
4. Replace only the `created_at` bytes on changed lines.

It verifies three things before writing:
- Every changed line differs only in `created_at`.
- The file strictly increases.
- Projection (timestamps masked, event list excluded) is identical to the original and identical between file and time order.

Result on this repo's `issues.db`: 3,331 events, 79 re-timed, 221 ties nudged (max 178 ns), 298 lines changed. The three issues that used to differ between file and time order (xpo-70c971, xpo-d23f3e, xpo-834350) now agree. `xpo doctor` reports the timestamps as in file order. The backup was kept outside the repo.

Not repairable: edits already credited to the wrong actor. The original split is lost, but that only affects history, not board state.

## Also fixed along the way

`setupTestRepo` in `collapse_test.go` now calls `ResetHubRoot()`. `HubRoot` is cached in a `sync.Once`, so collapse tests after the first one were resolving against a stale temp directory.

## Acceptance criteria

- [x] Collapse never merges UPDATEs across `created_by`/`on_behalf_of`/`source`. Evidence: `TestCollapseDoesNotMergeAcrossActors` and `TestCollapseDoesNotMergeAcrossOnBehalfOf`. Agent MCP then web update produce two correctly attributed events.
- [x] Collapse doesn't merge past a newer event for the same issue. Evidence: `TestCollapseDoesNotMergePastNewerEventForIssue` (final status DOING, last web event carries no stale status).
- [x] A collapsed UPDATE is moved to the end of the tail. Evidence: `TestCollapseMovesMergedEventToEnd`, where timestamps strictly increase after interleaved A/B edits.
- [x] Comment edit collapse keeps the original timestamp and position. Evidence: `TestCollapseCommentEditKeepsOriginalTimeAndPosition`.
- [x] No-op pruning still works and uses running state. Evidence: `TestCollapseSameActorRoundTripLeavesNoUpdate`, `TestCollapsePruneUsesRunningState`, and the existing `TestCollapsePrunesRoundTrip`/`TestCollapseDefaultBacklogStatus`.
- [x] The doctor check reports out-of-order and tied timestamps with line numbers and never rewrites the log. Evidence: `TestCheckEventOrder*` and `TestDescribeOrderViolations*`. On the real repo before the repair it reported "72 out-of-order and 219 tied timestamps (lines 218, 219, …)".
- [x] The one-time script repaired this repo's `issues.db`. Evidence: the script's verification passed, and afterwards doctor shows "✓ Event timestamps are in file order".
- [x] `make test` passes.
