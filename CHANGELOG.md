# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Keyboard help overlay: added Board section documenting all Board shortcuts; disambiguated key separators (`/` = or, `→` = then, `+` = together); expanded status number shortcuts to show actual status names; widened to 4-column layout (xpo-87503f)
- Optional `default_branch` field in `.xpo/config.yaml` — explicitly sets the base branch for branching, merging, and diff operations, bypassing git-based detection; `xpo init` auto-detects and writes it (xpo-961910)
- Global config accessor (`config.Get()` / `config.Set()`) — package-level functions can now read config without parameter threading (xpo-961910)
- Shared `TopBar` component (`components/ui/TopBar.tsx`) with left/center/right slots — replaces 13 duplicated top-bar implementations with a single composable primitive; includes `cn()` utility (`clsx` + `tailwind-merge`) for safe Tailwind class overrides (xpo-09ebcc)
- Backlog and Dependencies search inputs show `/` keyboard hint when empty and `Esc` when populated (xpo-09ebcc)
- Cycles view now has a proper fixed top bar matching all other views (xpo-09ebcc)
- `xpo start --mode worktree|branch` CLI flag and MCP `mode` parameter — override the global worktree config per-invocation; replaces the old `--no-wt` flag (xpo-00198b)
- Dependencies view: interactive dependency graph — filterable table as default with kind/search/completed filters, click-to-drill-down graph powered by dagre for exploring dependency chains; relationship-aware resolved logic; entry from issue detail property sidebar (xpo-a49ec2)
- `xpo rationale` CLI command and MCP tool — BM25-ranked full-text search across specs and walkthroughs for design rationale; supports `--top N` / `-n N` and `--json`; rich terminal output with colored scores, document types, labels, and clickable artifact paths (xpo-d6b4de)
- `xpo history` now renders a visual activity timeline with typed dot markers (`●` events, `○` commits, `◆` merges, `☰` artifacts) connected by a vertical rail; interleaves git commits with issue events; supports `--json`, `--reverse`, `--since <duration>`, `--limit N`, and auto-paging through `$PAGER` (xpo-d82dc7)
- `xpo history` without arguments shows a project-wide timeline across all issues and recent commits (xpo-d82dc7)
- Git hooks (`.githooks/`) — commit-msg strips AI-agent trailers and enforces prefix convention; pre-commit runs `go vet` and `eslint` on staged files; pre-push runs the test suite. Activated automatically via `make setup`
- `xpo pulse` CLI command — compact terminal dashboard of project health metrics (velocity, cycle time, lead time, WIP, blockers, throughput) with color-coded values; supports `--json` for machine-readable output (xpo-9ba68b)
- `xpo init` now writes a `name:` field to `config.yaml` derived from the folder name, used as the project heading in `xpo pulse` and `xpo board` (xpo-9ba68b)
- Empty state messaging for Backlog, Active, and Done filtered views — shows tab-specific or filter-aware hints instead of a blank screen (xpo-5a17a7)
- Timeline event-type filter menu — replaces single-select tabs with a multi-select checklist popover (Issues, Closed, Comments, Merges, Artifacts, Commits); client-side filtering for instant toggling; renders ARTIFACT events and CANCELED/DUPLICATE status transitions that were previously invisible (xpo-eb76cd)
- Walkthrough tab in the merge view — renders the issue's walkthrough as Markdown and becomes the default tab when present (xpo-3bfbc8)
- Backlog view: `[` / `]` keyboard shortcuts to cycle through sub-tabs (All Issues, Backlog, Active, Done) with wrapping (xpo-e5d2e4)

### Changed

- Parents with no visible children in nested backlog mode show a static right-pointing chevron instead of a clickable expand toggle; ghost parents show a static down-pointing chevron — clicking neither triggers cross-group side effects (xpo-2dd9a8)
- Ghost row opacity increased from 50% to 65% for better readability (xpo-2dd9a8)
- All top bars standardized to `h-12` height with `pl-5 pr-3` asymmetric padding (xpo-09ebcc)
- Backlog search is now always visible in the center of the top bar instead of appearing on `/` keypress (xpo-09ebcc)
- Labels view top bar normalized from `h-13 px-6` to standard TopBar defaults (xpo-09ebcc)
- Timeline kind filter converted from segmented control to underlined tabs on the left (xpo-09ebcc)
- Filter buttons in MyIssues and Notifications changed from text+icon to icon-only with tooltip, matching Backlog/Board style (xpo-09ebcc)
- Backlog filter and view-options icon buttons normalized to `w-7 h-7` bordered style, matching Board (xpo-09ebcc)
- Dependencies search Escape key now clears the search and blurs the input (xpo-09ebcc)
- CLI accent color (issue IDs, labels) lightened from `#6f42c1` to `#b392f0` for dark terminal readability (xpo-d82dc7)
- Branch badges on board/backlog cards now show commit count instead of SHA, with a blue indicator dot for uncommitted changes (xpo-ebb248)
- Detail sidebar shows "No commits" and uncommitted change details instead of the base branch SHA when a branch has no commits (xpo-ebb248)
- SubProgress ring uses Lucide-compatible viewBox for proper text alignment, with a dashed track and brand-color progress arc (xpo-ebb248)
- Artifact count badges unified with branch badge style (rounded-square, bordered) across board and backlog views (xpo-ebb248)

### Fixed

- Markdown checklist checkboxes now visible and flush-left in both rendered markdown and tiptap editor — border uses `--color-border-control` for proper contrast, padding zeroed to align with surrounding list content (xpo-958474)
- Spec and walkthrough content in issue detail panel now live-updates when the file changes on disk — artifact content cache is invalidated on `updated_at` change instead of only on issue navigation (xpo-0b193b)
- `xpo merge` auto-checkout failed in branch mode when `Worktrees` config defaulted to true — `FindWorktreeForBranch` matched the hub checkout itself; now compares against `HubRoot()` (xpo-1fccf2)
- Added 11 merge test cases covering all three strategies, error paths, branch deletion, worktree cleanup, hub-wrong-branch, and issues.db preservation for FF (xpo-7a5bba)
- Backlog drag-and-drop performance — eliminated per-pointermove re-renders by moving DnD visual state (drop indicator, group highlight, nest target) from React state to refs with direct DOM manipulation; extracted `BacklogIssueRow` and `BacklogGroupHeader` as `React.memo` components (xpo-d049dc)
- Cross-group drag over ghost parent rows no longer shows a spurious indicator in the originating group — ghost rows now participate in collision detection, and `findIndex` lookups skip ghost instances (xpo-d049dc)
- Cancelling a drag with Escape no longer triggers a click-through to the issue detail view on mouseup (xpo-d049dc)
- Merge view auto-refreshes when new commits are pushed, with a manual refresh button in the tab bar; `.xpo/` files hidden from files/diff tabs (xpo-af0b09)
- Merge view walkthrough and conversation tabs now constrain to 832px and center, matching the issue detail content width (xpo-2c6297)
- ContextMenu sub-menus now flip left and clamp vertically when near viewport edges instead of clipping (xpo-911a91)
- MergeView no longer crashes when `branch_stats` becomes `undefined` after a merge completes — replaced non-null assertions with optional chaining and an early-return "branch merged" state (xpo-db1b0a)
- FilterMenu and ViewOptionsMenu popovers now center on their trigger button with viewport clamping, instead of right-aligning (xpo-374512)
- Scrollbar appearance no longer shifts layout left — scroll containers use `overflow-y: scroll` with a thin 6px transparent-track scrollbar (xpo-0b2c6c)
- Inline create row in backlog status groups now aligns with normal issue rows — icons wrapped in matching containers, blank spacer replaced with dimmed `xpo-······` placeholder ID (xpo-5d3fb0)
- Fixed all 63 frontend eslint errors across 21 files with no suppression comments — ref writes moved out of render, setState positioning replaced with direct DOM, non-component exports extracted to separate modules, missing hook deps added, unused vars removed, `any` types replaced (xpo-c08d52)
- `make test` now depends on `make lint` (`go vet` + `bun run lint`), ensuring lint errors are caught before declaring implementation done (xpo-c08d52)
- Completed ghost children (DONE/CANCELED/DUPLICATE) no longer appear in non-terminal views by default; "Show done ghosts" toggle in view options to opt back in (xpo-b109e5)
- Merge view diff/files endpoints now target the correct worktree directory instead of the hub checkout, and normalize diff output prefixes for compatibility with `diff.mnemonicPrefix` git config (xpo-2d6b57)
- `computeBranchStats` now detects uncommitted changes in worktree directories, making the merge view button appear for worktree branches with pending changes (xpo-67a1c4)
- `xpo merge` no longer blocks on untracked files (e.g. `idea.md`) on the hub checkout — only modified tracked files that actually conflict with the incoming branch are rejected (xpo-cb6b49)
- Merge error messages now explicitly warn against stashing to prevent agent stash-thrash loops that corrupt `.xpo/issues.db` (xpo-cb6b49)
- Nested hierarchy mode no longer duplicates children across status groups — ghost children under a real parent are skipped when the child's status group is visible on the same tab (xpo-93e234)
- `xpo merge` now auto-checkouts the default branch in branch mode instead of erroring — detects branch vs worktree mode at runtime by checking whether a worktree exists for the issue's branch (xpo-d5c047)
- Web merge view mergeability check updated to use the same relaxed logic (xpo-cb6b49)

## [1.1.0] - 2026-08-31

### Added

- **Terminal statuses: CANCELED and DUPLICATE** (xpo-a18d46) — two new terminal statuses alongside DONE, with `IsTerminal()` / `IsCompleted()` helpers replacing hardcoded DONE checks across the entire codebase. CANCELED uses Lucide `CircleMinus`, DUPLICATE uses `CirclePercent`, both in medium gray. All terminal statuses unblock dependents, trigger parent auto-close, and are rejected by `xpo start`. Only DONE counts toward velocity and burndown. Includes "Show empty groups" toggle in backlog View Options and a column visibility dropdown in the Board view.
- **"Done" view tab** (xpo-65e8e3) — dedicated backlog tab showing only completed issues, sorted by most recently completed first. Tab order: All Issues → Backlog → Active → Done. Supports flat/nested toggle with independent localStorage persistence.
- **Hierarchy toggle for backlog views** (xpo-9091b7) — per-view toggle (nested/flat) in a new View Options dropdown, persisted in localStorage per tab. Nested mode renders full parent-child trees with ghost parents (interleaved by sort order) and ghost children for cross-status context. Flat mode renders all issues at depth 0 grouped by parent with breadcrumbs. Ghost rows use prefixed DnD IDs to avoid duplicate registration. Dragging a parent across status groups shows the cascade "Update sub-issues?" prompt. View options dropdown consolidates layout, expand/collapse, and sort controls behind a single Settings icon.
- Expand All / Collapse All icon buttons in the backlog toolbar for toggling all parent issue nodes at once (xpo-b44c4c)
- **Worktree-based concurrent workflows** (xpo-f7ea12) — `xpo start` and `xpo merge` now use git worktrees by default, making multi-agent and multi-session work safe by design. The primary checkout stays parked on `main` as the integration hub; each `xpo start` creates an isolated worktree at `.xpo/worktrees/<branch>/`; `xpo merge` runs from the hub and cleans up the worktree automatically. All issue events serialize through the hub so the web board reflects real-time state from every agent. Opt out per-command with `--no-wt` or globally with `worktrees: false` in config.
  - Hub-rooted storage layer — all `issues.db` and artifact I/O routes through the primary checkout via `git rev-parse --git-common-dir`, so MCP servers in worktrees read/write the hub's event log (xpo-580061)
  - `xpo start` creates worktrees with `--force` takeover (removes existing worktree), `worktree_setup` config hook for post-creation build steps (e.g. `make deps`), and MCP `start` tool returns `worktree_path` (xpo-1765ca)
  - `xpo merge` verifies hub is on the default branch, skips checkout, removes worktree after merge regardless of `--keep-branch`, and relaxes clean check to allow uncommitted `.xpo/` events (xpo-f96832)
- Review view for branches with uncommitted changes — shows working-tree diff in the merge view even before committing, with merge disabled and guidance to commit first (xpo-4295bf)
- `xpo init` and `xpo doctor` now create/ensure `.gitattributes` with `merge=union` for `issues.db`, preventing merge conflicts on the append-only event log (xpo-c762f7)

- Generated `xpo-workflow` skill now includes worktree isolation guidance (`start`/`merge` tools, worktree path usage, parallel issue isolation) and an explicit review gate after the completion comment — agents must wait for user tophat before writing walkthrough, committing, or merging (xpo-cf75df)
- `xpo init` is now idempotent — re-running on an existing project refreshes agent skill files without overwriting `config.yaml`, `issues.db`, or other project state (xpo-cf75df)

### Improved

- `xpo init` now replaces stale agent instructions instead of skipping files that already have them; heading changed from "Exponential Agent Instructions" to "Agent Instructions"; both old and new headings are detected for backward compatibility (xpo-8f7b7a)
- `xpo merge` via MCP and REST API now deletes the issue branch by default after merge; pass `keep_branch: true` to opt out (xpo-408db9)
- Render cap for Board (50 per column) and Backlog (100 per group) views — reduces DOM nodes for large projects while keeping all counts and stats accurate; "+N more" button reveals the rest (xpo-f16a94)
- Label ordering: metadata labels now sort alphabetically to the left, primary labels (matching `default_labels` config) sort to the right in backlog, board, and detail views (xpo-e928c1)
- Markdown rendering: tables now have full grid borders, cell padding, and distinct header row; headings use graduated top margins for visual hierarchy; lists are nearly flush with body text; overall vertical rhythm between paragraphs, code blocks, and blockquotes increased for better readability (xpo-dbd580)
- Issue detail scrollbar moved to the right edge of the viewport; property sidebar is now sticky-positioned and scrolls independently on short screens (xpo-a8be39)
- Activity timeline system entries (status changes, merges, artifact updates) now render with tighter vertical spacing; comment cards retain generous margins (xpo-13153f)

### Fixed

- `xpo merge` with squash or ff-only strategy no longer overwrites `issues.db` events on `main` when worktrees are disabled — performs a line-based union of both versions before committing, matching the `merge=union` behavior that only fires during 3-way merges (xpo-c58af8)
- `xpo merge` now stages `.xpo/artifacts/<issue-id>/` (spec, walkthrough, generic artifacts) alongside `issues.db` in the merge commit — previously left as untracked files on `main` (xpo-dda39d)
- Backlog rows now dynamically fit labels into available space (35% of row width budget) with `+N` overflow, adapting as the window resizes; board cards cap at 2 (xpo-5f9ee7)
- Sub-issues table rows no longer wrap when issues have many labels; shows first 2 labels with "+N" for the rest (xpo-ff3cb7)
- Notification inbox labels no longer overflow when issues have many labels (xpo-7eff79)
- MCP `list` tool now returns issues in user-defined sort order instead of creation order (xpo-2792c3)
- Epic "Update sub-issues?" dialog no longer resets completed children; detail view now shows the same cascade prompt as the backlog (xpo-886099)
- Sub-issues preview in cascade dialog caps labels at 2 (primary first), truncates long titles, and scrolls when the list is long (xpo-ee99a5)
- Drag-and-drop no longer accidentally changes parent relationships; Alt is now required for nesting and unparenting (xpo-cd0f62)
- Alt parent/unparent now works in Active and Backlog views: nest highlight responds to mid-drag Alt press, ghost parent indicator placement fixed, and "Remove from parent" added to the right-click context menu (xpo-9859c9)
- Ghost parent row no longer duplicates popovers when the real parent is also visible in a different status group (xpo-377101)
- Status group header story points no longer include completed children or double-count parents; DONE parents show delivered points (xpo-dbdfd3)
- Scroll position now resets to top when navigating between issues via arrow keys or prev/next buttons (xpo-a6a17e)

## [1.0.3] - 2026-07-20

### Added

- Discard confirmation dialog when closing the new-issue modal with unsaved content — guards Escape, backdrop click, X button, and Cancel button (xpo-762a91)
- Floating formatting bar on text selection in markdown editors — text type, bold/italic/strikethrough/underline, inline link editor, quote, code, code block, and list options (xpo-df91c4)
- "Issue not found" page when navigating to an invalid issue ID, with a link back to the Backlog (xpo-147414)
- Clickable priority icon in Backlog rows — opens a priority picker popover; `p` keyboard shortcut (xpo-a274c0)

### Fixed

- Checkmark and shortcut number alignment in status, priority, and estimate picker popovers
- Generated agent instructions now include the project's issue ID prefix and an example ID (xpo-2ba758)
- Board view now sorts issues within each status column the same way as the Backlog — by `sort_order` for active statuses, by recency for DONE (xpo-337854)
- Relationship links in issue detail view now resolve correctly — dependency target IDs and parent IDs are resolved to canonical `xpo-` prefixed IDs via `GetIssue()` in MCP add/update tools (xpo-147414)
- Existing dependencies with prefixless IDs matched via suffix fallback in the frontend (xpo-147414)
- CMD+K search now uses substring matching for issue titles instead of fuzzy subsequence — eliminates false positives that buried actual results past the 12-item cap (xpo-d41e32)
- Web UI `handleDraft` endpoint now validates all input — title, status, estimate, parent ID, dependency target IDs, and JSON payload errors (xpo-49cbfe)
- `handleDraft` capability mismatch — each operation (create, update, comment, delete) now checks its specific capability instead of a blanket `issue.create` (xpo-ddb470)
- Estimate validation added to update path and negative estimates rejected in both create and update (xpo-7868ea)
- MCP link tool rejects self-links and duplicate links; deleted dependency targets shown as "(deleted)" instead of broken links (xpo-a90985)
- CycleID validated as YYYY-MM-DD format during create and update; rejected when cycles aren't enabled (xpo-e75d09)
- Length limits on string input fields — title (500), description (100KB), comment (100KB), assignee (200), labels (100), artifact content (1MB) (xpo-75f1cf)

## [1.0.2] - 2026-07-13

### Added

- **Improved agent instructions out of the box**: `xpo init` now generates a two-layer agent setup — a thin always-on stub with hard invariants in the agent instruction file (CLAUDE.md, AGENTS.md, etc.) and a portable `xpo-workflow` skill with the full development lifecycle procedure. Specs are framed as thinking tools that collapse the design space. Includes interactive/non-interactive mode guidance, spec drift prevention, walkthrough enforcement, and a harness-neutral MCP tool reference.
- **Agent auto-detection**: `xpo init` detects installed agents (Claude Code, Gemini, Cursor, Aider, etc.) via PATH lookup and writes to the correct instruction file and skill directory for each. Falls back to generic `AGENTS.md` if no agents are found.
- **MCP and agent setup during init**: `xpo init` now creates `.mcp.json` and agent instruction files automatically — no longer deferred to an interactive `xpo doctor` session.
- **Streaming progress in `xpo drive`**: real-time display of agent activity (tool use, file paths, thinking verbs) replaces static spinner. Phases are collapsible — active phase expands with steps, completed phases collapse to a single `✔ Phase (Xs)` line.
- **Supervisor model override**: `drive.supervisor.model` config key (defaults to `sonnet`) lets supervisor calls use a faster/cheaper model for spec evaluation, planning, review, and walkthrough.
- **Session reuse**: supervisor and coder executors reuse Claude CLI sessions via `--resume` for prompt cache hits, cutting token usage roughly in half.
- **Test command validation**: broken test commands (exit code ≠ 1) are detected during preparation and skipped instead of sending the coder into a retry loop. `xpo init` no longer writes a default test command.
- **Release archives**: GitHub Actions now packages release binaries as `.tar.gz` (Linux/macOS) and `.zip` (Windows) instead of plain binaries.

### Changed

- Drive config restructured: `supervisor` and `coder` are now nested objects with `agent` and `model` subkeys

### Fixed

- Normalize labels case-insensitively to prevent duplicates
- Show current week running total in dashboard velocity card
- Parse pasted text as markdown in MarkdownEditor
- Makefile `frontend` and `clean` targets now preserve `internal/server/static/.gitkeep` so CI test runs don't fail from a missing embed directory

## [1.0.1] - 2026-07-09

### Changed

- Updated agent instructions to MCP-first workflow — MCP tools are now the primary interface, CLI command reference is supplemental
- Updated README with screenshot

### Fixed

- Added placeholder file to `internal/server/static/` so Go embed works without a prior frontend build

## [1.0.0] - 2026-07-08

### Changed

- **Rebrand**: Beats → Exponential (`xpo` binary). Module path `github.com/palarix/exponential`, data dir `.xpo/`, env prefix `XPO_*`, MCP tools `xpo_*`. Clean break — no backward compatibility.
- **Public release**: MIT license under Nicolas Bettenburg, cleaned repository history.

## [0.3.0] - 2026-06-16

### Added

- **Distributed mode**: `xpo serve` runs a headless server; `xpo board` proxies to a remote instance
- **Authentication**: SSH public-key challenge/response auth with JWT tokens (`xpo login`/`logout`/`whoami`)
- **Role-based permissions**: configurable per-user capabilities via `config.yaml`
- **Inbox**: personal notification feed — CLI (`xpo inbox`) and WebUI with grouped notifications and tabbed read state
- **Cycles**: configurable sprint/cycle planning with progress tracking and burndown charts
- **Branch workflows**: `xpo start` creates a feature branch and locks the issue; `xpo merge` squash-merges and closes
- **Code review**: `xpo review` shows a unified diff from the terminal; WebUI diff viewer with per-commit navigation
- **Dashboard**: pulse metrics — velocity charts (weekly + daily), flow stats (cycle time, lead time), WIP tracking, epic progress, attention items, workload distribution, and bug age
- **SSE**: real-time board updates via Server-Sent Events
- **Filtering**: structured backlog filtering with nested menus and status bar chips
- **Drag-and-drop**: reorder issues across status groups with globally consistent sort order (data model v3)
- **Docker**: `Dockerfile` for packaging `xpo serve`
- **MCP server**: `merge` tool; `.mcp.json` auto-detection in `xpo init`/`xpo doctor`
- **Contributors**: configurable contributor list for assignment
- **Empty state**: full-page onboarding for new projects

### Changed

- Data model upgraded to v3 — globally consistent `sort_order` keys with automatic migration from v2
- Error messages audited for user-facing clarity — no raw Go errors or stack traces leak to clients
- Issue cache: projected issues are cached instead of reprojected on every API call
- Request logging: structured JSON logs for `xpo serve`
- Story point display toggle persisted in localStorage

### Fixed

- Sub-issue ordering and dependency view toggle
- Bulk status updates for parents with sub-issues
- Description flicker on save in issue detail view
- Title flicker on edit and link clicks entering edit mode
- Keyboard focus and navigation in filter popover
- Auto-progressed parent status correctly marked as inferred
- Drag-and-drop sorting to top of backlog group
- Unestimated issues count as 1 point; epics excluded from totals
- Epics show remaining story points instead of full count
- Status icons and borderless label badges in new issue dialog
- New issues append at end of backlog
- Login credential and config storage consistency
- Collapse/prune handling of status field in CreatePayload
- Inbox participation filter counts update authors
- Nil-slice JSON responses return `[]` instead of `null`

## [0.2.0] - 2026-04-01

### Added

- **Web UI**: `xpo board` serves a React-based board view with embedded static assets
- **Comments**: add and view comments on issues (`xpo comment`, `xpo comments`)
- **Dependencies**: `xpo link` command to express issue relationships (blocks, depends_on, relates_to, etc.)
- **Archive**: `xpo archive` moves old DONE/deleted issues to a separate store; regular commands search the archive as fallback
- **Delete**: `xpo delete` with cascade option for parent issues
- **Interactive mode**: Bubble Tea TUI for `xpo add` and `xpo update`
- **Autocomplete**: shell completions for issue IDs across all commands
- **Version command**: `xpo version`
- **Filtering**: `--since`, `--before`, `-m` (match) flags on `xpo ls`; hide DONE by default (`-a` to show all)
- **Duplicate detection**: warn when creating issues with similar titles

### Changed

- **Breaking**: renamed `issues.jsonl` to `issues.db`
- **Breaking**: upgraded to v2 data model — single Issue type with labels, assignee, estimation, automations
- Dynamic column widths based on terminal size
- Markdown-rendered descriptions in terminal
- Separated UI and business logic from CLI commands

### Fixed

- First item in list had purple background highlight
- First item in list had extra indentation
- Sorting issues correctly
- Timezone handling (all dates now UTC)
- Interactive add truncating title after first whitespace
- `xpo config` keys missing underscores
- ANSI escape sequences in bash completions

## [0.1.0] - 2026-02-01

### Added

- Initial release
- Event-sourced issue tracker stored in `.xpo/issues.db`
- CLI commands: `xpo init`, `xpo add`, `xpo update`, `xpo ls`, `xpo show`, `xpo history`
- `xpo doctor` for health checks
- `xpo snapshot` for point-in-time state export
- Git auto-commit support (configurable)
- Configurable issue ID prefix
- Parent/child issue hierarchy with auto-sync
- Status workflow: BACKLOG → PLANNED → DOING → BLOCKED → DONE
