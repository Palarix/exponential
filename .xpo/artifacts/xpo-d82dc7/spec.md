# Spec: `xpo history` — unified timeline command

## What

Restructure `xpo history` into a single command with two modes:

1. **`xpo history <issueID>`** — human-readable activity timeline for one issue, modeled after the web UI's ActivityTimeline.
2. **`xpo history`** (no args) — project-wide timeline showing recent activity across all issues.

Both modes support:
- `--json` for JSONL output (one JSON event per line)
- `--reverse` for newest-first ordering
- `--since <duration>` to filter events by age (e.g. `7d`, `24h`, `2w`)

## Why

The current `xpo history` is a flat table with minimal detail — UPDATE says "Updated issue" without showing *what* changed, COMMENT says "Added comment" without the text, and MERGE isn't rendered at all. The web UI already provides a richer experience; the CLI should match it.

Adding the no-arg global timeline gives users a quick "what happened recently?" view — useful for catching up after being away, or seeing what an agent has been doing.

## Acceptance Criteria

- [ ] `xpo history <id>` renders a scannable timeline with:
  - Relative timestamps via `humanize.Time()`
  - Actor names (with "via {principal}" when `on_behalf_of` is set)
  - Rich detail per event type:
    - **CREATE**: "created this issue"
    - **UPDATE**: specific field changes (e.g. "changed status to DOING", "set estimate to 3", "updated the title", "assigned to Alice", "updated labels to [bug, CLI]") joined with " and "
    - **COMMENT**: full comment text, word-wrapped, in a bordered block
    - **DELETE**: "deleted this issue"
    - **MERGE**: "merged `branch-name` via fast-forward"
    - **ARTIFACT**: "created spec.md" / "updated walkthrough.md"
  - Comments rendered as visually distinct blocks (indented, `│`-prefixed lines)
  - Chronological order (oldest first, `--reverse` for newest first)
- [ ] `xpo history` (no args) renders the same timeline format across all issues, with the issue ID shown in each entry
- [ ] `xpo history` defaults to the 50 most recent events; `--limit N` overrides
- [ ] `xpo history --json [id]` outputs JSONL (one JSON object per line per event)
- [ ] `xpo history --reverse [id]` shows newest events first
- [ ] `xpo history --since 7d [id]` filters to events within the given duration
- [ ] `--since` accepts `Nd` (days), `Nh` (hours), `Nw` (weeks)
- [ ] All flags compose (e.g. `--json --since 24h --reverse`)
- [ ] Shell completion for issue IDs
- [ ] Existing tests updated; new tests cover both modes and all flags
- [ ] The `show` command's inline History section (in `details.go`) is unaffected

## Flow

### Step 1: New `internal/ui/timeline.go`

**`RenderTimeline(events []model.Event, termWidth int, showIssueID bool)`**

The human-readable renderer. `showIssueID` controls whether each entry includes the event's issue ID (true for global mode, false for single-issue mode).

Format for single-issue mode:
```
  3 hours ago · Nicolas Bettenburg created this issue

  3 hours ago · Nicolas Bettenburg changed status to DOING

  2 hours ago · Claude Code (via Nicolas Bettenburg) assigned to Claude Code

  1 hour ago · Nicolas Bettenburg commented:
  │ This looks good. Let's also handle the edge case
  │ where the description is empty.

  30 minutes ago · Claude Code (via Nicolas Bettenburg) merged xpo-d82dc7 via fast-forward

  15 minutes ago · Claude Code (via Nicolas Bettenburg) created spec.md
```

Format for global mode — prepends the issue ID:
```
  3 hours ago · xpo-d82dc7 · Nicolas Bettenburg changed status to DOING
  2 hours ago · xpo-a1b2c3 · Alice commented:
  │ Looks ready to ship.
```

Rendering rules:
- Each entry: `  {timestamp} · [{issueID} · ]{actor} {description}`
- Actor: `ExtractName(created_by)`, appending ` (via ExtractName(on_behalf_of))` when present
- Timestamp in `MutedStyle`, issue ID in `AccentStyle`, actor in `WhiteStyle`
- Status values rendered with their `StatusStyle` color
- Comments get a bordered block: lines prefixed with `  │ `, word-wrapped to `termWidth - 6`
- UPDATE events: decode `UpdatePayload`, describe each changed field, join with " and "
- UPDATE events with no recognized field changes: skip silently
- Blank line between entries for readability

**`RenderTimelineJSON(events []model.Event, w io.Writer)`**

One `json.Marshal` per event, written as a line to `w`.

**`FormatActor(createdBy, onBehalfOf string) string`**

Returns `"Name"` or `"Name (via Principal)"`.

**`DescribeEvent(evt model.Event) string`**

Returns the human-readable description for an event. Empty string for unrecognized/empty updates (caller skips).

### Step 2: New `internal/cli/duration.go` — `ParseDuration(s string) (time.Duration, error)`

Parses human-friendly duration strings: `30m`, `24h`, `7d`, `2w`. Go's `time.ParseDuration` handles `m` and `h`; this adds `d` (×24h) and `w` (×7d). Returns an error for unrecognized suffixes.

### Step 3: Update `cmd/exponential/history.go`

- Change `Args` from `cobra.ExactArgs(1)` to `cobra.MaximumNArgs(1)`
- Add flags: `--json` (bool), `--limit` (int, default 50), `--reverse` (bool), `--since` (string)
- When arg provided: `FindIssue(id)` → use `issue.Events`
- When no arg: `ListIssues({})` → collect all events from all issues → sort by `CreatedAt` → take last N
- Apply `--since` filter: parse duration, drop events older than `now - duration`
- Apply `--reverse`: reverse the slice after sorting
- `--json` routes to `RenderTimelineJSON`, otherwise `RenderTimeline`

### Step 4: Remove `internal/ui/history.go` and `internal/ui/history_test.go`

The old table renderer. The `show` command's inline History section (in `details.go:149-156`) has its own rendering and is unaffected.

### Step 5: New `internal/ui/timeline_test.go`

Tests:
- Empty events → "No activity recorded."
- Each event type renders its expected description
- UPDATE with status change shows the status value
- UPDATE with multiple fields shows "X and Y"
- UPDATE with no recognized fields is skipped
- COMMENT text appears with `│` prefix
- `on_behalf_of` renders as "(via Name)"
- `showIssueID=true` includes issue IDs
- JSON mode: each line is valid JSON with expected fields

### Step 6: New `internal/cli/duration_test.go`

Tests for `ParseDuration`: `30m`, `24h`, `7d`, `2w`, invalid input, zero value.

## Decisions

- **One command with `--json`, not two commands.** Simpler UX. The `--json` flag pattern already exists on `pulse`, `comment`, `add`, and `update`.
- **Default oldest-first, `--reverse` for newest-first.** Timeline reads naturally chronologically. `--reverse` mirrors the web UI's toggle.
- **`--since` uses simple duration syntax (`7d`, `24h`, `2w`)** rather than absolute dates. Covers the common "what happened recently?" use case without needing date parsing.
- **Global mode defaults to 50 events with `--limit`.** Prevents flooding the terminal. Single-issue mode shows all events (issues rarely have hundreds of events).
- **Remove `RenderHistory` entirely.** No callers remain after the change.
- **Comment text is word-wrapped, not truncated.** Comments are the most valuable part of the timeline.

## Edge Cases

- **Empty events list** (LOW): Print "No activity recorded."
- **UPDATE with no recognized fields** (LOW): Skip the entry silently
- **Very long comment text** (LOW): Word-wrap to `termWidth - 6`, no truncation
- **Payload type coercion** (MEDIUM): Events from JSONL storage have `Payload` as `map[string]interface{}`. Use `json.Marshal` → `json.Unmarshal` into the concrete type.
- **Global mode with archived issues** (LOW): `ListIssues` returns active issues only. Archived events don't appear. Fine by design.
- **`--since` with no unit suffix** (LOW): Return parse error with usage hint.

## Assumptions

- The `show` command's inline History section (in `details.go:149-156`) is independent and unchanged.
- Shell completion uses the existing `completeIssueIDs` function.
- No Transport interface changes needed.
- No MCP server changes needed.
