## Walkthrough: Add jsonio.PulseOutput wrapper for pulse --json

## What changed

Moved the `PulseMetrics` type and its 10 sub-types from `internal/server` to
`internal/jsonio`, establishing jsonio as the owner of all JSON I/O types. Also
fixed a pre-existing velocity double-count bug in the CLI rendering.

## Type ownership change

`internal/jsonio/output.go` now owns:
- `PulseOutput` (renamed from `PulseMetrics`)
- `AttentionKind` (type + 3 constants)
- `AttentionItem`, `WorkloadEntry`, `WeeklyTrend`, `AgeBuckets`, `BugAgeBuckets`,
  `TrendsBlock`, `EpicProgress`, `VelocityBucket`, `DailyBucket`

`internal/server/metrics.go` imports from `jsonio` and uses `jsonio.PulseOutput` as
the return type of `ComputePulseMetrics`. No circular dependency — `jsonio` depends on
`model`, `server` imports `jsonio`.

## Bug fix

`pulse.go` computed the CLI velocity display as:
```go
VelocityPts: m.Velocity.CurrentWeekPoints + m.Velocity.Last7dPoints,
```

`Last7dPoints` already includes the current week's points, so this double-counted them.
The JSON output exposed the correct raw values, making the discrepancy visible.
Fixed to use `Last7dPoints` alone.
