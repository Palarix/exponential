# Walkthrough: demo seed has no events after `now`

## What changed
- `internal/demo/demo.go`: `Generate` calls the new `generateAt(outputDir, now)`. Tests pass a fixed `now`.
- `internal/demo/demo.go`: the new `capAtNow(events, now)` runs after `snapToWorkWeek`. It walks the sorted events from the end. It sets each event later than the limit to the limit. The limit starts at `now - 1s` and moves 1 second earlier per clamped event. The walk stops at the first event at or before the limit.
- `internal/demo/demo_test.go`: the new `TestGenerateNoEventsAfterNow` and the helper `readDemoEvents`.

## Why
The weekend snap moved the last seeded events up to 3 days past `now`. `stampAfter` then stamped every new write `prev + 1ns`, so new changes got seeded times. With the last seeded event before `now`, `stampAfter` keeps the real time of a new write.

## Decisions
- The fix clamps only the tail. A shift of the whole timeline would move Monday planning events onto weekends.
- On a Saturday or Sunday run, clamped events land on that weekend. `created_at ≤ now` wins over the weekend rule.

## Acceptance criteria
- [x] A test calls `generateAt` with `now` on each day of a week, Monday and Tuesday included. It asserts that every event's `created_at` is ≤ `now`. Evidence: before the fix, `TestGenerateNoEventsAfterNow` failed on Monday, Tuesday, Saturday and Sunday. After the fix, it passes on all 7 days.
- [x] The same test asserts that the log is sorted and each issue's events are in strict time order. Evidence: the same test passes.
- [x] A test asserts the last seeded event is before `now`. Evidence: the tail check in `TestGenerateNoEventsAfterNow` passes.
- [x] The existing demo tests pass. Evidence: `make test` exits 0.
