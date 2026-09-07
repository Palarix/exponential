# Walkthrough: `xpo history` — unified timeline command

## What was built

Restructured `xpo history` from a flat table of event types into a rich, visual activity timeline that interleaves issue events with git commits. One command, two modes:

- **`xpo history <id>`** — single-issue timeline with branch commits
- **`xpo history`** — project-wide timeline (issue events + default branch commits)

Both support `--json`, `--reverse`, `--since <duration>`, `--limit N`, and `--no-pager`.

## How the pieces fit together

### Timeline rendering (`internal/ui/timeline.go`)

The renderer accepts `[]model.TimelineEntry` — a unified type that represents both issue events (`kind: "issue_event"`) and git commits (`kind: "commit"`). Each entry gets a typed dot on a vertical rail:

- `●` blue — issue events (create, status change, assignment)
- `●` cyan — comments
- `○` amber — git commits
- `◆` green — merges
- `☰` purple — artifacts (spec, walkthrough)
- `│` dim connector between entries

Comments render with glamour (the charmbracelet markdown renderer already used in `details.go` and `review.go`), indented under the rail. UPDATE events expand to show specific field changes ("changed status to DOING and set estimate to 3") rather than the old generic "Updated issue".

### Data flow

**Single-issue mode:** `FindIssue` → `FillLocalBranchStats` (finds the issue's branch) → collects issue events as `TimelineEntry` + branch commits via `ListBranchCommitsDetailed` → sorts chronologically.

**Global mode:** `ListIssues` → collects all events + builds issue map → calls `BuildTimeline` (existing function from `internal/exponential/timeline.go`, already used by the web server's `/api/timeline` endpoint). `BuildTimeline` reads recent commits from the default branch via `git log`, links them to issues when the commit message contains an issue ID, and interleaves everything sorted by timestamp.

### Auto-pager (`cmd/exponential/history.go`)

When stdout is a terminal (detected via `go-isatty`, already a dependency), output pipes through `$PAGER` or `less -R`. Skipped for `--json`, piped output, or `--no-pager`. The pager is a `pagerPipe` wrapping `exec.Cmd` — `Close()` closes stdin then waits for the pager process to exit.

### Duration parser (`cmd/exponential/duration.go`)

`parseDuration` extends Go's `time.ParseDuration` with `d` (days) and `w` (weeks) suffixes. Used by `--since`.

### Type relocation

`TimelineEntry` was moved from `internal/exponential` to `internal/model` to break an import cycle (`ui` → `exponential` → `ui`). All references in `timeline.go`, `history.go`, and the server handler updated. `BuildTimeline` now returns `[]model.TimelineEntry`.

## Key decisions

- **One command with `--json` instead of separate `xpo history` / `xpo log`.** Matches existing flag patterns (`pulse --json`, `comment --json`).
- **Newest-first default, `--reverse` for chronological.** More useful for the "what happened recently?" use case.
- **`AccentColor` bumped from `#6f42c1` to `#b392f0`.** The original dark purple was unreadable on dark terminals. This affects issue IDs project-wide — a deliberate improvement, not just scoped to the timeline.
- **Glamour for comments, no extra border.** The timeline rail already provides visual continuity, so the old `│`-bordered comment block was redundant. Comment text is indented under the rail with glamour-rendered markdown.
- **Removed `internal/ui/history.go` entirely.** The old table renderer (`RenderHistory`) had no remaining callers — `show` has its own inline history in `details.go`.

## Files changed

- `cmd/exponential/history.go` — rewired: two modes, five flags, pager, `BuildTimeline` integration
- `cmd/exponential/duration.go` — new: `parseDuration` for `--since`
- `cmd/exponential/duration_test.go` — new: 7 tests
- `internal/ui/timeline.go` — new: `RenderTimeline`, `RenderTimelineJSON`, `DescribeEvent`, `FormatActor`, visual rail rendering
- `internal/ui/timeline_test.go` — new: 18 tests (event types, commits, mixed entries, JSON, markdown)
- `internal/ui/helpers_test.go` — added `captureStdout` (moved from deleted `history_test.go`)
- `internal/ui/styles.go` — `AccentColor` lightened for dark terminal readability
- `internal/model/types.go` — `TimelineEntry` struct added (moved from `exponential`)
- `internal/exponential/timeline.go` — `TimelineEntry` struct removed, references updated to `model.TimelineEntry`
- `internal/ui/history.go` — deleted
- `internal/ui/history_test.go` — deleted
