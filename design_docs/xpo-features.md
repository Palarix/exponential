# Exponential — Product Feature Map

> A snapshot of what xpo does today, written for product/market-fit analysis and roadmap planning.
> Generated 2026-06-08.

---

## What Exponential Is

Exponential is a **local-first, git-native project management tool**. It ships as a single binary that stores all data in a `.xpo/` folder alongside your code. No accounts, no SaaS, no infrastructure — `xpo init` and you're running.

It serves three interfaces from the same data:

- **CLI** — for developers who live in the terminal
- **Web UI** — a local board (`xpo board`) for visual planning
- **MCP server** — so AI coding agents can read and write issues natively

---

## Core Product Features

### Issue Tracking

The foundation. Create, update, search, and manage issues entirely from the command line or browser.

- **Statuses**: BACKLOG → PLANNED → DOING → BLOCKED → DONE
- **Fields**: title, description (markdown), priority, assignee, labels, story points, parent, cycle
- **Search**: free-text across all fields, plus structured filters (status, label, assignee, date range, parent)
- **Soft delete** with optional hard archival to keep the active dataset lean

### Hierarchies (Epics & Sub-Issues)

Any issue can be a parent. Child issues nest underneath, creating natural epic → story → task trees.

- Parent/child relationships via `parent_id`
- Status inference: a parent's status can reflect its children's aggregate progress
- Cycle inheritance: children inherit the parent's cycle when not explicitly assigned
- Web UI displays sub-issues inline; drag-and-drop nesting on the board

### Dependencies & Linking

Six relationship types between issues:

- `blocks` / `blocked_by`
- `depends_on` / `dependency_of`
- `duplicates` / `duplicated_by`
- `relates_to`

The web UI includes a dependency graph visualization.

### Estimation

Four estimation systems, configurable per project:

| System | Values |
|---|---|
| Fibonacci | 1, 2, 3, 5, 8 |
| Exponential | 1, 2, 4, 8, 16 |
| Linear | 1, 2, 3, 4, 5 |
| Shirt sizes | XS, S, M, L, XL |

### Cycles (Sprints)

Time-boxed iterations with configurable cadence (1–4 weeks).

- Anchor date support to lock cycle boundaries
- Current/next cycle shortcuts in the CLI
- Per-cycle progress tracking: scope, started, completed, days remaining
- Dashboard visualization of cycle progress and trends

### Labels

Flexible tagging with color support.

- Built-in defaults: `bug`, `feature`, `epic`, `improvement`
- Custom labels with hex color picker in the web UI
- Label distribution analytics on the dashboard

### Comments & Audit Trail

Every mutation is an immutable event. Comments are first-class.

- Markdown comments on any issue
- Full event history: who changed what field, when, and from what value
- Accessible via `xpo history`, the web UI activity timeline, and the MCP server

### Automations

Configurable rules that reduce manual status bookkeeping:

- Auto-complete parent when all children are done
- Cascade-close sub-issues when parent closes
- Auto-progress children when parent advances
- Auto-progress parent based on children's status

---

## Interfaces

### CLI

Full-featured terminal interface with rich shell completion (color-coded statuses and labels in zsh). Every operation is a subcommand:

| Command | What it does |
|---|---|
| `add` | Create issue (flags, stdin, JSON, or interactive) |
| `list` / `ls` | Filtered, searchable issue listing |
| `show` | Full issue detail with description, deps, comments |
| `update` | Modify fields (flags, editor, JSON) |
| `start` | → DOING + auto-create git branch |
| `done` / `planned` / `blocked` | Status shortcuts |
| `comment` / `comments` | Add or view comments |
| `estimate` | Set story points |
| `link` | Create issue relationships |
| `history` | Full audit trail |
| `delete` / `archive` | Cleanup operations |
| `config` / `doctor` / `init` | Setup and diagnostics |

### Web UI

A React single-page app served locally by `xpo board`. Views:

- **Dashboard** — stats, attention items (blockers, stale WIP), label/assignee/priority distribution charts, activity feed, pulse metrics
- **Backlog** — sortable, filterable list with drag-and-drop ordering and sub-issue nesting
- **Board** — kanban columns (PLANNED, DOING, BLOCKED, DONE) with drag-and-drop status transitions
- **Cycles** — current, upcoming, and past cycles with progress indicators
- **Dependencies** — relationship graph visualization
- **Labels** — management interface with color picker and usage stats
- **Issue Detail** — inline editing of all fields, markdown description editor, comments, activity timeline, sub-issues table
- **Command Palette** — `Ctrl/Cmd+K` for global search and quick navigation

The web UI uses a draft/pending model: changes buffer locally and commit to git via "Save & Sync."

### MCP Server

Seven tools that give AI agents (Claude Code, Cursor, etc.) full read/write access to issues:

`xpo_list`, `xpo_show`, `xpo_history`, `xpo_add`, `xpo_update`, `xpo_comment`, `xpo_link`

This is the primary interface for AI-assisted development workflows — the agent reads the board, picks up work, and documents what it did, all through MCP.

---

## Architecture & Storage

### Event Sourcing

All data is an append-only JSONL event log (`.xpo/issues.db`). Current state is a projection computed from replaying events. This gives:

- Complete audit trail by design
- Minimal merge conflicts (appending lines to a file)
- Portable — it's a text file in your repo

A local snapshot cache (`.xpo/issues.snapshot.json`, gitignored) accelerates reads.

### Git-Native

- `.xpo/` is committed alongside code — issues travel with the repo
- `xpo start` auto-creates a branch from the issue ID and title
- Optional `auto_commit` mode commits `issues.db` on every write
- The web UI's "Save & Sync" batches changes into a single git commit

### Single Binary

The CLI, web server, MCP server, and embedded frontend assets all ship in one Go binary. No runtime dependencies.

### Configuration

Layered config: built-in defaults → user `~/.config/xpo/config.yaml` → project `.xpo/config.yaml` → environment variables. Key settings:

- Estimation system, cycle cadence, default labels
- Auto-commit behavior, editor preference
- Automation rules
- Contributor list, label colors, UI theme

---

## Integration Points

### Git

- Branch creation on `xpo start`
- Auto-commit of issue data
- Issues live in the repo and are cloned/forked/branched with the code

### AI Agents

- MCP server for native agent integration
- Agent identity tracking (`CreatedBy` field)
- `xpo doctor` auto-detects and configures agent instruction files (CLAUDE.md, AGENTS.md, GEMINI.md)
- Duplicate detection on issue creation

### Multi-Instance

- Local registry tracks running `xpo board` instances across projects
- Sidebar links between project boards
- Auto port selection to avoid conflicts

---

## What's NOT in Exponential Today

Gaps and absent capabilities, for roadmap context:

- **No hosted/multi-user mode** — single-user, local only (epic `xpo-075586` covers this)
- **No commit-to-issue linking** — no automatic association between git commits and issues beyond branch naming
- **No CI/CD integration** — no build status, deployment tracking, or webhook receivers
- **No notifications** — no alerts for status changes, mentions, or blockers
- **No permissions / access control** — anyone with repo access has full issue access
- **No time tracking** — no logged hours or time-based reporting
- **No roadmap / timeline view** — no Gantt chart or date-based planning
- **No cross-repo issues** — each `.xpo/` is scoped to one repository
- **No API for external integrations** — REST API is local-only, no stable public contract
- **No import/export** — no migration path from Jira, Linear, GitHub Issues, etc.
- **No templates** — no issue templates or recurring issue patterns
- **No custom fields** — fixed schema, no user-defined metadata
- **No search saved views** — filters are ephemeral, not persistable
