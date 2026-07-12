# Walkthrough: Dashboard Velocity Card — Current Week Running Total

## What changed

The Velocity card on the dashboard now shows two values in a table layout matching the Cycle Time and Lead Time cards, with a trend footer:

- **Left column** ("LAST WEEK"): Total story points completed in the last full Mon–Sun week.
- **Right column** ("CURRENT"): Running total of story points completed so far this week (Monday through today).
- **Footer**: An arrow (up/down, colored green/amber) with the delta comparing last week to the week before that. Shows "no change vs prior week" if delta is zero, or "no prior week data" if there's nothing to compare against.

Additionally:
- All table cards (Velocity, Cycle Time, Lead Time) now use inner vertical dividers only (no outer border), using `--color-border-default` for visibility.
- The Staleness card footer was moved out of its wrapper div so `mt-auto` correctly pushes it to the card bottom, aligning with the other cards.

## Files modified

### `internal/server/metrics.go`

**Struct** (line 95): Added `CurrentWeekPoints int` field (`"current_week_points"`).

**Accumulation** (line 219): New branch counts points for issues completed on or after `thisWeekStart` (Monday of the current week):

```go
if !doneAt.Before(thisWeekStart) {
    m.Velocity.CurrentWeekPoints += points
} else if ...
```

Existing `last_7d_points` / `prior_7d_points` / `delta` logic unchanged.

### `internal/server/metrics_test.go`

Added assertion in `TestMetrics_VelocityExcludesCurrentWeek` verifying that the "this_week" issue (estimate=10) populates `CurrentWeekPoints`.

### `web/src/api/client.ts`

Added `current_week_points: number` to the `PulseMetrics.velocity` interface.

### `web/src/components/Dashboard/Dashboard.tsx`

- **Velocity card**: Replaced the old single-number display with a two-column table (headers "Last Week" / "Current"), vertical inner dividers only, and a footer showing the trend arrow with delta.
- **Cycle Time / Lead Time cards**: Removed outer table borders, kept only inner `border-r` dividers using `--color-border-default`.
- **Staleness card**: Extracted the footer `<p>` from inside the wrapping `<div>` into a fragment so `mt-auto` aligns it to the card bottom.
- Removed unused `Sparkline` import.

## Design decisions

- `current_week_points` is purely additive — no existing API fields changed, so other consumers remain unaffected.
- The trend arrow compares two *completed* weeks (last vs prior), never a partial week, so the comparison stays meaningful.
- Table borders use `--color-border-default` (#282828) instead of `--color-border-subtle` (#1e1e1e) which was invisible against the elevated card background.
