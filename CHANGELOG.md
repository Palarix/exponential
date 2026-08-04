# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Review view for branches with uncommitted changes — shows working-tree diff in the merge view even before committing, with merge disabled and guidance to commit first (xpo-4295bf)
- Hub-rooted storage layer — all `issues.db` and artifact reads/writes now resolve through the primary checkout ("hub") via `git rev-parse --git-common-dir`, enabling future worktree-based concurrent workflows (xpo-580061)
- Worktree-based `xpo start` — creates a git worktree at `.xpo/worktrees/<branch>/` by default instead of checking out in the primary checkout; `--no-wt` flag and `worktrees: false` config for fallback; `worktree_setup` hook for post-creation build steps (xpo-1765ca)
- Worktree-aware `xpo merge` — skips branch checkout when hub is on main, removes worktree after merge, relaxes clean check to ignore uncommitted `.xpo/` events (xpo-f96832)

### Improved

- Markdown rendering: tables now have full grid borders, cell padding, and distinct header row; headings use graduated top margins for visual hierarchy; lists are nearly flush with body text; overall vertical rhythm between paragraphs, code blocks, and blockquotes increased for better readability (xpo-dbd580)
- Issue detail scrollbar moved to the right edge of the viewport; property sidebar is now sticky-positioned and scrolls independently on short screens (xpo-a8be39)
- Activity timeline system entries (status changes, merges, artifact updates) now render with tighter vertical spacing; comment cards retain generous margins (xpo-13153f)

### Fixed

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
