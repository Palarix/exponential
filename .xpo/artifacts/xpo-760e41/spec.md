# Demo seed: no events after `now`

## What
`demo.Generate` must not write an event with `created_at` later than the time it runs.

## Why
The largest seeded offset is 1439h, about 1 hour before `now`. `snapToWorkWeek` moves weekend events forward by up to 3 days. The per-issue ordering pass adds 1 minute per collision. So the tail of the log can land up to 3 days in the future.

`stampAfter` stamps a new write `prev + 1ns` when the clock is not after the last event. Until the real clock passes the latest seeded event, every new change gets a seeded timestamp.

## How
1. Split `Generate(outputDir)` into `Generate` and `generateAt(outputDir, now time.Time)`. `Generate` calls `generateAt` with `time.Now().UTC()`. Tests pass a fixed `now`.
2. Add `capAtNow(events, now)`. It runs after `snapToWorkWeek`, on the sorted slice.
3. `capAtNow` walks the slice from the end. It keeps a limit that starts at `now` minus 1 second. When an event is later than the limit, it sets the event to the limit. Then it moves the limit 1 second earlier. It stops at the first event that is already at or before the limit.

## Decisions
- **Clamp the tail, do not shift the whole timeline.** A shift of up to 3 days moves Monday planning events onto weekends across the whole 60 days. A clamp changes only the few tail events. They end up just before `now`, which still reads as "recent".
- **1 second between clamped events, and the last one 1 second before `now`.** It keeps the log sorted and each issue's events in strict time order. The real clock is already after the last seeded event.
- **Weekend runs.** When `now` is a Saturday or Sunday, clamped events land on that weekend. The weekend rule cannot hold without moving events into the future, so `created_at ≤ now` wins.

## Acceptance criteria
- [ ] A test calls `generateAt` with `now` on each day of a week, Monday and Tuesday included. It asserts that every event's `created_at` is ≤ `now`.
- [ ] The same test asserts that the log is sorted and each issue's events are in strict time order.
- [ ] A test asserts the last seeded event is before `now`, so `stampAfter` keeps the real time of a change made right after seeding.
- [ ] The existing demo tests pass.
