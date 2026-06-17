# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed

- **Rebrand**: Beats → Exponential (`xpo` binary). Module path `github.com/palarix/exponential`, data dir `.xpo/`, env prefix `XPO_*`, MCP tools `xpo_*`. Clean break — no backward compatibility.

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
- **MCP server**: `xpo_merge` tool; `.mcp.json` auto-detection in `xpo init`/`xpo doctor`
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
