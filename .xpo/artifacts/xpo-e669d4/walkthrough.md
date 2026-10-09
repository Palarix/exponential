# Walkthrough: Event timestamps strictly increase in file order

## The problem
`issues.db` is an append-only log, and projection replays it in **file order**. xpo-e9a2eb established that `created_at` must also strictly increase in file order: anything that orders by time (history, metrics, the planned RepoDB in xpo-09f337) otherwise sees a different story, and that issue added a `doctor` check for it. While preparing 1.3.0 (xpo-e60c26), `doctor` reported 27 out-of-order lines and 1 tie, all written after that repair.

Classifying the 28 lines showed three causes:

1. **Merge (26 lines).** `recordMerge` called `buildUpdate` for the DONE transition first, which stamped it with `time.Now()`, and only then built the MERGE event with a later `time.Now()`. It appended MERGE first, so every merge left a DONE update about 100µs *older* than the MERGE line above it.
2. **Batch updates (1 tie).** `buildUpdate` stamps every event it produces (the primary update, link-reconcile events, parent auto-start and auto-close) with one shared `timestamp`. The tie was a child closing its parent through the "last completed" automation.
3. **A stale ARTIFACT (1 line).** It was stamped 2026-09-30 but entered the hub file between 10-04 and 10-08. I checked every commit's `issues.db` from 09-29 onward, and each one is a byte prefix of the next, so collapse rewrites didn't cause it. It came in outside the normal append path, most likely an old stash or union re-apply. No current code path does this.

## The fix, in two layers

### 1. Storage enforces the invariant (`internal/storage/event_time.go`)
This is the root fix. Every writer on the machine goes through `storage`, so the invariant is enforced there, once:

- **`lastEventTime(path)`** returns the last complete line's `created_at`. It reads backwards from the end of the file in chunks that double in size (starting at 8 KB), so an append never scans the whole log. A torn final line is skipped, matching `ReadEvents`. A missing or empty file gives the zero time.
- **`stampAfter(prev, at)`** returns `at` when it is later than `prev`, and `prev + 1ns` otherwise.
- **`AppendEvent`** calls `lastEventTime`, re-stamps the event and only then marshals it, all inside `withEventsLock`. The lock matters: the last line can't change between reading it and appending.
- **`appendEventCollapsed`** runs `stampInOrder` over the rewritten uncommitted tail, starting from the last committed event's time. Collapse moves merged events to the end with the incoming time (xpo-e9a2eb), and that incoming time can be older than lines already in the tail. Comment-edit collapse keeps its original time and position, and that already satisfies the rule, so nothing changes there.

As a result, a caller's clock only ever moves an event *later*, never earlier, and only by as many nanoseconds as needed. It also covers the wall clock stepping backwards (an NTP correction), and a genuinely stale event gets its append time. That agrees with file order, which is authoritative.

### 2. Call sites build honest times (`internal/exponential`)
The guard would hide these bugs, but the events a function builds should still be ordered correctly on their own.

- **`buildMergeEvents`** (new, extracted from `recordMerge`) builds and stamps the MERGE event *before* building the DONE transition, then returns `[MERGE, DONE…]`. `recordMerge` just appends them in that order.
- **`sequenceTimes(events)`** bumps any event whose time isn't after the previous one by 1ns. `buildUpdate` applies it on both return paths: the full one and the link-only early return. `buildMergeEvents` applies it too, because two consecutive `time.Now()` calls can return the same value on coarse clocks such as Windows.

## Tests
- `TestAppendEvent_KeepsTimestampsStrictlyIncreasing` covers an earlier time, an equal time, a later time (kept unchanged) and a tie with the last line.
- `TestCollapseKeepsTimestampsStrictlyIncreasing` feeds collapse incoming events older than the committed tail.
- `TestLastEventTime` covers an empty file, one line, a last line longer than the first 8 KB chunk, a torn final line, a file that is only a torn line, and a missing file.
- `TestBuildUpdate_CascadeTimesStrictlyIncrease`, `TestBuildUpdate_LinkOnlyTimesStrictlyIncrease` and `TestBuildMergeEvents_MergeBeforeDone` assert on the *built* events, not on the file, so the storage guard can't hide a regression at the call sites.

## Data repair
The xpo-e9a2eb script (`repair-event-timestamps.go`, attached to that issue) was run once from a temporary, uncommitted directory. It keeps the longest in-order run of lines untouched, spreads the other lines' times evenly between their kept neighbours, and nudges ties by 1ns. Only the `created_at` bytes change.

- **Dry run on a copy:** 3,665 events, 27 re-timed and 1 tie nudged, 28 lines changed. Projection is identical apart from timestamps, file order equals time order, and a second pass changes nothing. The stale 09-30 ARTIFACT now reads 2026-10-06T13:20:11Z, between its neighbours.
- **Applied to the hub `issues.db`:** same result. The backup was kept outside the repo. `xpo doctor` then reported "✓ Event timestamps are in file order".

## Caveat for this repo
`merge` runs in the xpo MCP server that is already running. Until that binary is reinstalled with this fix and the server restarted, each merge still writes its DONE update before its MERGE event. That includes the merge of this issue. Install the new binary and restart the MCP server, then re-run the check before releasing 1.3.0.

## Acceptance Criteria
- [x] `AppendEvent` stamps an event whose time is earlier than or equal to the last line's at `last + 1ns`, and writes a later time unchanged. Evidence: `TestAppendEvent_KeepsTimestampsStrictlyIncreasing`.
- [x] A collapsed rewrite produces a tail whose times strictly increase, even when an incoming event is older than the last committed event. Evidence: `TestCollapseKeepsTimestampsStrictlyIncreasing`.
- [x] Merge: the MERGE event is earlier than the DONE update. Evidence: `TestBuildMergeEvents_MergeBeforeDone` (first event is MERGE, times strictly increase).
- [x] `buildUpdate` with a cascade gives strictly increasing times. Evidence: `TestBuildUpdate_CascadeTimesStrictlyIncrease`, plus the link-only path in `TestBuildUpdate_LinkOnlyTimesStrictlyIncrease`.
- [x] Repair dry run: 28 lines changed, verification passed, and a second pass changed nothing.
- [x] Repair applied to the hub `issues.db`, and `xpo doctor` reports "✓ Event timestamps are in file order" (3,666 events).
- [x] `make test` passes.
