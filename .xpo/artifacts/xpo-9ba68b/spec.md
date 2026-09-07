# Spec: `xpo pulse` — CLI analytical health check

## What

Add an `xpo pulse` command that prints a compact terminal dashboard of project health metrics.

## Why

The metrics are already computed in `internal/server/metrics.go` (`computePulseMetrics`) for the
web dashboard. This command surfaces the same data directly in the terminal without starting the
web server.

## How

### 1. Export metrics computation

Export `ComputePulseMetrics` and `PulseMetrics` (+ nested types) from `internal/server/metrics.go`
so the CLI can call them directly.

### 2. CLI command (`cmd/exponential/pulse.go`)

- Read events via `storage.ReadEvents()`
- Project issues via `exponential.ProjectIssuesWithConfig(events, cfg)`
- Call `server.ComputePulseMetrics(issues, time.Now())`
- Render via `ui.RenderPulse(metrics, termWidth)`

Supports `--json` flag for machine-readable output.

### 3. Rendering (`internal/ui/pulse.go`)

Box-drawing output using lipgloss, matching the mockup in the issue description:

```
┌──────────────────────────────┐
│  Velocity:    218 pts/wk     │
│  Cycle Time:  12m (median)   │
│  Lead Time:   4h (median)    │
│  WIP:         3 active, 1 stale │
│  Blockers:    2              │
└──────────────────────────────┘
```

Color-coded values: green for healthy, yellow for warning, red for attention.

Thresholds:
- WIP stale > 0 → yellow; blockers > 0 → red
- Velocity delta negative → yellow
- Cycle time shown with human-friendly formatting (minutes/hours/days)

## Acceptance Criteria

- [ ] `xpo pulse` prints the health-check box with all six metric lines
- [ ] `xpo pulse --json` outputs the raw `PulseMetrics` struct as JSON
- [ ] No web server needed — reads directly from `.xpo/issues.db`
- [ ] Colors render correctly (green/yellow/red thresholds)
- [ ] Tests cover the rendering logic
