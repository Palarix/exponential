# Spec: Event timestamps must strictly increase in file order

## What
`xpo doctor` on main reports `27 out-of-order and 1 tied timestamps` in `issues.db`. Two live writer bugs cause 27 of the 28; the last one is a historical one-off. Fix the writers so the invariant "`created_at` strictly increases in file order" can't break again on this machine, and repair the existing 28 lines so doctor is clean for the 1.3.0 release (xpo-e60c26 is blocked on this).

## Why
The invariant was established in xpo-e9a2eb, which also added the doctor check. Anything that orders events by time (history, metrics, the planned RepoDB in xpo-09f337) disagrees with file-order projection when the invariant is broken. Shipping 1.3.0 with a merge path that breaks it on every run is not acceptable.

## Findings (from the hub `issues.db`)
| Pattern | Count | Cause | Still happening? |
|---|---|---|---|
| MERGE → DONE UPDATE about 100µs older | 26 | `recordMerge` (`merge.go`) calls `buildUpdate`, which stamps the DONE events with `time.Now()`. Only afterwards does it build the MERGE event with a later `time.Now()`, and it appends MERGE first. | Yes, on every merge since xpo-9fef67 |
| Tie, L3585/L3586 | 1 | `buildUpdate` stamps every event it produces (primary update, link events, parent auto-start/auto-close) with one shared `timestamp`. Here the "last completed" automation closed parent xpo-570a07 in the same call as child xpo-1657f1. | Yes, on every cascade or link reconcile |
| Stale ARTIFACT, L3414 (xpo-0cbecc spec, 09-30) | 1 | The event is stamped 09-30 16:00 UTC, but it isn't in commit `92ed543` (10-04, 3,413 lines) and first appears in `4a67de3` (10-08), placed right after the 10-04 lines. Every commit's `issues.db` from 09-29 to now is a byte prefix of the next commit's, and today's appends go straight to the hub under the lock. So it entered the file outside the normal append path (most likely a stash or union re-apply under the old merge code). | No known current path. The writer guard below would stamp it with its append time. |

## How

### 1. Writer guard in storage (the root fix)
- Inside `withEventsLock`, read the last event's `created_at` in `issues.db`. This means seeking back from the end of the file, not reading the whole file. Skip a torn final line, as readers already do.
- **`AppendEvent`:** if `event.CreatedAt <= last`, set it to `last + 1ns` before marshalling.
- **`AppendEventCollapsed` / `rewriteFile`:** walk the rewritten uncommitted tail in order and apply the same rule to each event, starting from the last committed event's time. Comment-edit collapse keeps its original time and position (xpo-e9a2eb), so it already satisfies the rule.
- Every writer on the machine (CLI, MCP, web, `xpo serve`, `drive`) goes through these functions, so the invariant holds no matter how a caller stamps its events. This also covers the wall clock stepping backwards after an NTP correction.
- Adjustments are at most a few ns in the normal case. A genuinely stale event is re-stamped with its append time, which agrees with file order, and file order is authoritative.

### 2. Call-site fixes (so stamped times are honest without the guard)
- **`recordMerge`:** build the MERGE event first, then build the DONE events. The order of events then matches the order of time.
- **`buildUpdate`:** give each additional event in a batch a strictly later time than the one before it (`timestamp.Add(i ns)` in append order).

### 3. One-time data repair
- Reuse `repair-event-timestamps.go` from xpo-e9a2eb as a throwaway script in the scratchpad, not committed:
  1. Keep the longest in-order run of events untouched.
  2. Re-time the rest evenly between their kept neighbours, and nudge ties by 1 ns.
  3. Rewrite only the `created_at` bytes on changed lines.
  4. Verify that projected state, with timestamps masked, is identical before and after, and identical between file order and time order.
  5. Write a backup first.
- Dry-run it on a copy, then run it on the hub `issues.db` just before merge so the merge commit includes it. Nothing else should be writing at that point; I'll check with you first.
- Expected change: 28 lines (27 re-timed, 1 tie nudge). The stale 09-30 ARTIFACT gets a time between L3413 (10-04 19:29:40) and L3415 (10-08 07:10:43).

## Acceptance Criteria
- [ ] `AppendEvent` with a `CreatedAt` earlier than or equal to the last line's is written at `last + 1ns`. With a later `CreatedAt`, it is written unchanged.
- [ ] A collapsed rewrite produces a tail whose timestamps strictly increase, even when an incoming event's time is earlier than the last committed event's.
- [ ] `recordMerge`: the MERGE event's time is earlier than the DONE update's (test through `Merge` and `CheckEventOrder` on the resulting file).
- [ ] `buildUpdate` with a cascade (the last child closes the parent) returns events with strictly increasing times.
- [ ] Repair dry run: 28 lines changed, verification passes, and a second pass changes nothing.
- [ ] Repair applied to the hub `issues.db`. `xpo doctor` shows "Event timestamps are in file order" and no other warnings.
- [ ] `make test` passes.

## Out of scope
Ordering across clones: git union merges of `issues.db` from different machines can still interleave events. The doctor check reports this, and xpo-09f337 is the long-term answer.

## Decisions
1. **Both the storage guard (§1) and the call-site fixes (§2)** (confirmed by the user). The guard enforces the invariant in one place, and the call-site fixes keep the stamped times honest even without it.
