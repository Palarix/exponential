# Walkthrough: Terminal statuses — CANCELED and DUPLICATE

## What was built

Added CANCELED and DUPLICATE as first-class terminal statuses alongside DONE, replacing the originally planned resolution-field approach. Two helper functions — `IsTerminal()` (DONE, CANCELED, DUPLICATE) and `IsCompleted()` (DONE only) — replaced ~40 hardcoded `"DONE"` checks across Go and ~25 across TypeScript.

## Backend changes

### Model (`internal/model/types.go`)
- Added `StatusCanceled` and `StatusDuplicate` constants
- Added `IsTerminal(s IssueStatus) bool` and `IsCompleted(s IssueStatus) bool`

### Semantic replacements
Every `StatusDone` check was categorized as either "terminal" (issue is closed) or "completed" (work was shipped):

- **Blocker checks** (`update.go`, `drive.go`): `!= StatusDone` → `!IsTerminal()` — CANCELED/DUPLICATE blockers unblock dependents
- **Auto-close parent** (`update.go`): triggers when all children reach any terminal status; parent always goes to DONE specifically
- **Start rejection** (`start.go`): switched from a `case` per status to `model.IsTerminal()` with a descriptive error showing the actual status
- **Cycle rollover** (`projection.go`): skips all terminal statuses
- **Velocity/burndown** (`server/metrics.go`, `handlers.go`): only counts `IsCompleted()` — CANCELED/DUPLICATE don't inflate throughput
- **Epic progress** (`server/metrics.go`): "done" children use `IsCompleted()`
- **Staleness/bug-age** (`server/metrics.go`): excludes all terminal
- **List filtering** (`list.go`): terminal issues hidden by default unless explicitly requested
- **Archival** (`maintenance.go`): all terminal issues eligible
- **Branch inference** (`branches.go`): terminal issues not inferred as DOING from remote branches

### Validation
All three status validation sites updated — `inputs.ValidateStatus()`, `client.ValidateCreatePayload()`, `client.ValidateUpdatePayload()`, and the MCP `list` tool's status filter.

### CLI styling (`internal/ui/styles.go`)
CANCELED and DUPLICATE get medium gray color (`#8b8b8b`) and strikethrough text, with `⊖` and `⊘` terminal icons.

## Frontend changes

### Foundation
- `constants.ts`: `isTerminal()` and `isCompleted()` helpers, `TERMINAL_STATUSES` set, STATUS_OPTIONS extended
- `index.css`: `--color-status-canceled` and `--color-status-duplicate` CSS variables (medium gray `#8b8b8b`)
- `sort.ts`: `STATUS_ORDER` extended with DONE → CANCELED → DUPLICATE ordering
- `StatusIcon.tsx`: Lucide `CircleMinus` for CANCELED, `CirclePercent` for DUPLICATE (medium gray)

### Backlog (`Backlog.tsx`, `useBacklogRows.ts`)
- All `=== "DONE"` / `!== "DONE"` replaced with `isTerminal()` / `isCompleted()` as appropriate
- TAB_CONFIGS updated: "All Issues" and "Done" tabs include all three terminal statuses
- Status labels, group expansion defaults, cascade prompts, epic estimates all use the helpers
- **"Show empty groups" toggle** added to View Options under Layout heading, persisted per tab in localStorage, defaults to on for "All Issues" and "Done" tabs

### Board (`Board.tsx`, `BoardColumn.tsx`)
- COLUMNS array extended with CANCELED and DUPLICATE (after DONE)
- **View Options dropdown** (Settings2 icon) with per-column visibility toggles, persisted in localStorage
- Issue count reflects only visible columns
- Header layout: Settings2 button → issue count (right-aligned), matching Issues view
- Terminal columns use reduced visible count (5 vs 50)

### Other components
- `PropertySidebar.tsx`: cascade prompt and merge button use `isTerminal()`
- `NewIssueModal.tsx`: parent picker excludes all terminal issues
- `Dashboard.tsx`: "done" stat uses `isCompleted()`, "active" filters exclude all terminal, status distribution includes new statuses

## Tests added

### `internal/model/types_test.go` (new)
- `TestIsTerminal` — exhaustive coverage of all 7 statuses
- `TestIsCompleted` — only DONE returns true

### `internal/exponential/start_test.go`
- `TestStartWork_CanceledIssue` — rejects start on CANCELED
- `TestStartWork_DuplicateIssue` — rejects start on DUPLICATE

### `internal/exponential/update_test.go`
- `TestUpdateIssue_BlockerCanceled_Unblocks` — CANCELED blocker unblocks dependent
- `TestUpdateIssue_BlockerDuplicate_Unblocks` — DUPLICATE blocker unblocks dependent
- `TestUpdateIssue_LastCompletedWithCanceledChild` — auto-close fires when last child goes CANCELED
- `TestUpdateIssue_LastCompletedMixedTerminal` — auto-close with DONE + CANCELED + DUPLICATE children
- `TestUpdateIssue_LastCompletedNotTriggeredByActiveChild` — no auto-close when a child is still active
- `TestUpdateIssue_TerminalToActive` — reopening CANCELED → PLANNED works

### `internal/inputs/inputs_test.go`
- `TestValidateStatus` updated to include CANCELED and DUPLICATE as valid

## Key decisions

- **Single dimension over resolution field**: status alone is simpler to filter, query, and display. The code smell of hardcoded DONE checks was the trigger — replacing them with `IsTerminal()` / `IsCompleted()` makes adding future terminal statuses trivial.
- **Lucide icons**: CANCELED uses `CircleMinus`, DUPLICATE uses `CirclePercent` — standard Lucide components rather than hand-drawn SVGs.
- **Medium gray color**: both non-completed terminal statuses use `#8b8b8b`, visually distinct from DONE's green but clearly "closed".
- **Ordering**: DONE → CANCELED → DUPLICATE everywhere — completed work sorts before non-completed closures.
- **"Show empty groups" toggle**: user-controlled instead of hardcoded per tab, so the Done tab can show CANCELED/DUPLICATE group headers even when empty.