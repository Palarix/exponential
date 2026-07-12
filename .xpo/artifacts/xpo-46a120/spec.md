# Dashboard: Show cumulative story points for current week

## Requirements

Users want two numbers in the Velocity card instead of one:

1. **Left side** — Last completed week total (Mon–Sun), with a trend arrow comparing it to the week before that. Skip the arrow entirely if there isn't enough history (fewer than 2 completed weeks of data).
2. **Right side** — Current week running total (Mon of current week through today).

The sparkline remains unchanged (8-week history).

## Backend changes (`internal/server/metrics.go`)

- Add a new field `current_week_points` to the velocity struct that accumulates points for issues completed `>= thisWeekStart` (i.e. from Monday of the current week up to now).
- The accumulation loop already has `thisWeekStart` — items done on or after `thisWeekStart` get counted into the new field.
- `last_7d_points` and `prior_7d_points` remain unchanged (last completed week / week before that).
- `delta` remains `last_7d_points - prior_7d_points` (trend for completed weeks only).

## Frontend changes (`web/src/components/Dashboard/Dashboard.tsx`)

- Refactor the Velocity `PulseCard` to show two values side-by-side:
  - Left: `last_7d_points` with label "last week" and the trend arrow (only if `prior_7d_points > 0`).
  - Right: `current_week_points` with label "this week" (no arrow).
- Update the TypeScript interface in `web/src/api/client.ts` to include `current_week_points: number`.

## Acceptance criteria

- [ ] API returns `current_week_points` counting Mon-of-current-week through now.
- [ ] Frontend shows last week total on left with trend arrow (hidden if no prior week data).
- [ ] Frontend shows current week running total on right.
- [ ] Sparkline unchanged.
- [ ] `make test` passes.
- [ ] `make build` succeeds.
