# Trail Viewer Component

## Overview

A React component rendered inside the issue detail view that shows drive trail events as a visual timeline. Operates in two modes: **static** for completed runs (fetch the JSONL, render all at once) and **live** for in-progress runs (subscribe to `drive_event` SSE messages, render incrementally).

## Where it appears

In `IssueDetail.tsx`, as a new tab or section alongside the existing description, comments, and artifacts. Visible when the issue has at least one drive trail artifact.

Detection: the issue's artifact list (already available via `GET /api/issues/{id}`) includes entries with `artifact_type: "drive-trail"`. Each entry has the trail filename.

## Data flow

### Static mode (completed runs)

1. Issue detail loads, artifact list includes `drive-a1b2c3d4.jsonl`
2. Component fetches `GET /api/issues/{id}/artifacts/drive-a1b2c3d4.jsonl`
3. Parses JSONL into an array of trail events
4. Renders the full timeline

### Live mode (in-progress runs)

1. Issue detail loads, no completed trail yet (or a trail without `drive_end`)
2. Component subscribes to `drive_event` SSE messages filtered by `issue_id`
3. Each incoming event is appended to the local event array
4. Timeline re-renders with each new event, auto-scrolling to the latest
5. When `drive_end` arrives, live mode transitions to static (final state)

The `useSSE` hook in `App.tsx` already receives all SSE events. Extend it to forward `drive_event` messages to a new callback, or use a dedicated `useTrailSSE(issueId)` hook that filters by issue.

## Timeline layout

Vertical timeline, newest event at the bottom (chat-style, so live events append naturally). Each event is a card/row with:

### Common elements (all events)

- **Timestamp** — relative ("2m ago") while live, absolute when static
- **Event type icon** — distinct icon per event type for scannability

### Event-specific rendering

**`drive_start`**
- Header: "Drive started" with issue title
- Meta: supervisor, coder, max retries, branch name
- Subtle — this is context, not action

**`spec_eval`**
- Ready: green checkmark, "Spec ready"
- Not ready: yellow warning, "Spec needs revision (round N)"
- Show tokens/cost as small secondary text

**`plan`**
- "Implementation planned — N files identified"
- Expandable file list

**`attempt_start`**
- "Attempt N" — prominent, this is a chapter boundary
- If retry: show feedback preview from prior verdict in a muted block

**`coder_done`**
- "Coder finished" with duration and token/cost
- Subtle — the verdict is what matters

**`test_result`**
- Passed: green "Tests passed" with duration
- Failed: red "Tests failed" with summary (last few lines), expandable

**`verdict`** (the dramatic beat — most prominent rendering)
- **Passed**: large green checkmark, "Review passed"
- **Failed**: red X, "Review rejected" with the full feedback text rendered as markdown in a distinct block. This is the content people come to see — don't truncate, don't collapse.

**`drive_end`**
- Summary card with: outcome badge (merged/blocked/error), attempt count, total cost, wall time, commit link (if merged)
- This is the "final score" — render it distinctly from the timeline events above

### Now-playing strip (live mode only)

A sticky header above the timeline showing:
- Current phase (prep / implementation / testing / review)
- Active role (supervisor / coder)
- Attempt N/M
- Running cost counter (ticks up with each `coder_done` and `verdict` event)
- Elapsed time (live clock since `drive_start` timestamp)

Disappears when `drive_end` arrives.

### Cost accumulation

A running total displayed as a small counter that updates with each event carrying token/cost data. Render as "$0.00" format. This is "weirdly magnetic" per the design vision — keep it visible but not dominant.

## Multiple runs

If an issue has multiple trail files (retried drives), show them as a tabbed or accordion list, newest first. Each run is a separate timeline. The run selector shows: date, outcome badge, attempt count, cost.

## Empty state

When an issue has no trail files and no active drive:
- Don't show the trail section at all (no empty tab cluttering the UI)

## Component structure

```
DriveTrails/
  DriveTrails.tsx        — container: fetches trails, manages live subscription
  TrailTimeline.tsx      — renders an array of trail events as the vertical timeline
  TrailEvent.tsx         — renders a single event card (switches on event type)
  NowPlaying.tsx         — the live sticky header strip
  RunSelector.tsx        — tabs/accordion for multiple runs
  useTrailEvents.ts      — hook: merges static trail data with live SSE events
```

## Acceptance Criteria

1. Completed drive runs render as a full timeline in the issue detail view
2. In-progress drives render events as they arrive via SSE, with auto-scroll
3. The "now playing" strip shows current phase, role, attempt count, and running cost
4. Verdict events render the full reviewer feedback as markdown
5. The `drive_end` event renders as a summary card with outcome, attempts, cost, and commit link
6. Multiple runs for the same issue are selectable, newest first
7. Issues with no trails don't show an empty trail section
8. The timeline handles interrupted runs gracefully (no `drive_end` — show "interrupted" state)

## Out of Scope

- Replay with time compression (xpo-2f3900) — that's a distinct rendering mode
- Spectator mode as a standalone page (xpo-a168ae) — builds on this component but has its own routing and layout
- Editing drive configuration from the UI (v1 uses defaults)