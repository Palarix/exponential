# Walkthrough: `xpo pulse` — CLI analytical health check

## What was built

A new `xpo pulse` command that prints a compact terminal dashboard of project health metrics, reading directly from the local event log without starting a web server.

```
Exponential Pulse
╭──────────────────────────────────────╮
│   Velocity:    14 pts/wk (+10)       │
│   Cycle Time:  12m median (n=262)    │
│   Lead Time:   1.9h median (n=291)   │
│   WIP:         1 active              │
│   Blockers:    0                     │
│   Throughput:  11 issues/wk (+10)    │
╰──────────────────────────────────────╯
```

Also: `xpo init` now writes a `name:` field into `config.yaml` (derived from the folder name), which is used as the pulse heading and by the board command's registry.

## How it works

### Exported metrics (`internal/server/metrics.go`)

The existing `computePulseMetrics` function and all its types were exported (capitalized) so the CLI can call them directly. No logic changed — just visibility: `PulseMetrics`, `ComputePulseMetrics`, `AttentionItem`, `WorkloadEntry`, `EpicProgress`, etc.

### CLI command (`cmd/exponential/pulse.go`)

The command:
1. Reads events via `storage.ReadEvents()`
2. Projects issues via `exponential.ProjectIssuesWithConfig(events, cfg)`
3. Calls `server.ComputePulseMetrics(issues, time.Now())`
4. Maps the result into a `ui.PulseData` struct and renders it

The `--json` flag bypasses rendering and dumps the full `PulseMetrics` struct as indented JSON.

### Renderer (`internal/ui/pulse.go`)

A `PulseData` struct holds the six display values. This avoids an import cycle (`ui` → `server` → `exponential` → `ui`) — the CLI command maps from `server.PulseMetrics` to `ui.PulseData`.

Rendering uses lipgloss with color thresholds:
- **Green**: healthy defaults (velocity stable/up, zero blockers, zero stale WIP)
- **Yellow**: velocity declining, stale WIP items present
- **Red**: blockers present

Duration formatting (`formatDuration`) auto-scales: minutes for < 1h, hours for < 24h, days otherwise.

### Project name in init (`internal/exponential/setup.go`)

`xpo init` now writes `name: <folderName>` as the first field in `config.yaml`. The pulse command uses `cfg.Name` for the heading, falling back to "Project" if unset.

## Key decisions

- **Separate `PulseData` struct** rather than having `ui` import `server` — breaks the import cycle cleanly without a shared types package.
- **"Project" as generic fallback** instead of guessing from cwd — avoids the worktree directory name problem. The right fix is setting `name:` in config.
- **Throughput added as 6th row** — complements velocity (points vs issue count), data was already computed.
- **Agent/Human split omitted** — not computed by `ComputePulseMetrics`; would expand scope.
