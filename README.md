# Beats

The git-native issue tracker for humans and AI agents. Issues are committed alongside the code that resolves them, the board is a single binary you can run anywhere, and AI agents talk to it through a native MCP server — no SaaS, no API keys.

## Why beats

- **Git-native** — issues live in `.beats/` and are tracked with your code. Branches, merges, and history apply to your backlog the same way they apply to everything else.
- **Single binary** — `beats` ships the CLI, the web UI, and the MCP server in one Go executable. No Node.js, no Docker, no database to run.
- **Agent-first** — a built-in MCP server gives AI coding agents structured tools for reading, writing, and linking issues. No more shelling out to a CLI and praying about quoting.
- **Event-sourced** — every change is an append-only event with a full audit trail. Merge conflicts are rare; "who changed what, when, and why" is always answerable.
- **Local web UI** — `beats board` opens a Kanban in your browser. Drag-and-drop, sub-issues, keyboard shortcuts, command palette.

## Quickstart

```bash
# 1. Install (see below for other methods)
go install github.com/palarix/beats/cmd/beats@latest

# 2. Initialize in your project
cd path/to/your/repo
beats init

# 3. Capture some work
beats add "Wire up login form" --label feature

# 4. Open the board
beats board
```

That's the whole loop. Everything else is a refinement of these four steps (see below)

## Installation

### Binary download

Grab the latest release for your platform from [GitHub Releases](https://github.com/palarix/beats/releases) and put it on your `PATH`.

### Go install

```bash
go install github.com/palarix/beats/cmd/beats@latest
```

Requires Go 1.25+. This builds the CLI without the embedded web UI. For the full build (CLI + web UI), build from source.

### From source

Requires Go 1.25+ and [bun](https://bun.sh).

```bash
git clone https://github.com/palarix/beats.git
cd beats
make && make install
```

`make` builds the frontend assets and the Go binary. `make install` copies the resulting `beats` executable to `/usr/local/bin/beats` (it will ask for `sudo` permission).

### Docker

```bash
docker build -t beats:latest .
docker run -v /path/to/repo:/data -p 8080:8080 beats:latest
```

The image runs `beats serve` by default — see [Distributed mode](#distributed-mode) below.

## CLI reference

### Daily workflow

```bash
beats init                                # one-time, inside a git repo
beats list                                # see the board (alias: beats ls)
beats show <id>                           # full details for one issue
beats add "Title" --label feature         # create
beats start <id>                          # move to DOING + create branch
beats done  <id>                          # move to DONE
beats comment <id> "Fixed in auth.go"     # markdown comment
```

### Status transitions

```bash
beats planned <id>                        # → PLANNED
beats start   <id>                        # → DOING (creates a feature branch)
beats blocked <id> --by <other-id>        # → BLOCKED
beats done    <id>                        # → DONE
```

### Filtering

```bash
beats list --status PLANNED
beats list --label bug
beats list --assignee "Alice"
beats list --parent <epic-id>             # children of an epic
beats list --since 1w                     # updated in the last week
beats list --match "login"                # full-text search
beats list --mine                         # issues assigned to you
beats list --all                          # include archived DONE
```

### Relationships and estimation

```bash
beats add "Big initiative" --label epic              # epics group work
beats add "Subtask"        --label feature -p <epic-id>
beats link <a> <b> --type blocks                     # also: depends_on,
                                                     # blocked_by, duplicates,
                                                     # relates_to
beats estimate <id> 5                                # story points
beats update   <id> --assignee "Alice"
```

### Branch workflows

```bash
beats start <id>                          # creates branch, moves to DOING
beats review <id>                         # unified diff from the terminal
beats merge <id>                          # squash-merge + close
```

### Inbox

```bash
beats inbox                               # events relevant to you
beats inbox --since 2d                    # last 2 days only
```

### Cycles

```bash
beats cycle init                          # configure sprint cadence
beats cycle                               # current cycle info
beats cycle next                          # next cycle info
beats cycle history                       # past cycles with completion stats
```

### Audit, maintenance, troubleshooting

```bash
beats history <id>      # event log for one issue
beats archive           # move old DONE issues to cold storage
beats doctor            # validate issues.db, offer fixes
beats config            # show / edit local config
beats migrate           # upgrade data model to latest version
```

Run `beats --help` or `beats <command> --help` for the full surface.

## Web UI

`beats board` spins up a local web server (default port `8080`), embeds the React frontend from the binary, and opens your browser. The UI is 

- Drag-and-drop across status columns (hold `Cmd`/`Ctrl` for precise-slot mode).
- Hold `Alt` while dragging onto another issue to nest it as a sub-issue.
- `Cmd`/`Ctrl` + `K` opens a command palette for jump-to-issue and global actions.
- Sub-issues are first-class: epics show progress, children render inline.
- Dashboard with velocity charts, flow metrics, cycle burndowns, and workload distribution.
- Inbox with grouped notifications and read/unread state.

If port 8080 is already in use, beats picks the next free port automatically. Run `beats board` from any number of projects in parallel — every running instance registers in a shared local registry, and each sidebar lists the other live boards as one-click destinations.

The server binds to `127.0.0.1` only — no remote access, no auth needed.

## Distributed mode

For teams that need a shared server, beats can run in client-server mode with SSH key authentication.

### Server setup

```bash
# Start the headless server (auto-generates signing key on first run)
beats serve --addr :8080
```

The server reads `.beats/authorized_keys` for user verification. Add team members' SSH public keys (one per line, same format as `~/.ssh/authorized_keys`):

```
ssh-ed25519 AAAAC3Nza... alice@example.com
ssh-ed25519 AAAAC3Nza... bob@example.com
```

Endpoints exposed:

| Endpoint | Auth | Description |
|----------|------|-------------|
| `POST /auth/challenge` | No | Get a nonce for SSH key auth |
| `POST /auth/verify` | No | Exchange signed nonce for a JWT |
| `GET /healthz` | No | Health check |
| `/api/*` | JWT | REST API |
| `/mcp` | JWT | MCP endpoint |

### Client setup

```bash
# Authenticate (discovers your SSH keys, signs a challenge, caches JWT)
beats login https://beats.example.com

# All commands now work against the remote server
beats list
beats add "Remote issue" --label feature

# Check your identity
beats whoami

# Disconnect
beats logout
```

### Board

In distributed mode `beats board` will proxy to your remote so you get the full web UI locally while data lives on the server:

### Docker deployment

```bash
docker build -t beats:latest .
docker run -d \
  -v /path/to/repo:/data \
  -p 8080:8080 \
  beats:latest
```

The container runs `beats serve` by default, exposing port 8080.

## Configuration

Settings live in `.beats/config.yaml` and are created by `beats init`. Project-level settings take precedence over user-level settings in `~/.config/beats/config.yaml`. Environment variables prefixed with `BEATS_` override both (dots become underscores, e.g. `BEATS_AUTO_COMMIT=true`).

A full example with explanations:

```yaml
# Issue ID prefix — IDs become <prefix>1, <prefix>2, etc.
prefix: myproject-

# Configuration format version, managed by `beats migrate`
version: 3

# Automatically git-commit every beats write (add, update, comment, etc.).
# When false, changes accumulate in the working tree for you to commit manually.
auto_commit: false

# Story-point scale used by `beats estimate`.
#   fibonacci    — 1, 2, 3, 5, 8       (default)
#   exponential  — 1, 2, 4, 8, 16
#   linear       — 1, 2, 3, 4, 5
#   shirt        — XS, S, M, L, XL     (stored as 1, 2, 3, 5, 8 internally)
estimation_system: fibonacci

# Include unestimated children in a parent's story-point rollup.
# When false, only estimated children contribute to the total.
count_unestimated: true

# Sprint / iteration cycles
cycles:
  enabled: true
  duration: 2w              # cycle length: 1w, 2w, 3w, or 4w
  start_day: monday         # day of the week each cycle begins
  # Optional reference date (YYYY-MM-DD) to pin cycle numbering.
  # If omitted, anchors to the nearest start_day before now.
  anchor_date: "2025-01-06"

# Labels shown in the UI label picker (defaults to [bug, feature, epic, improvement])
default_labels:
  - bug
  - feature
  - epic
  - improvement

# Custom label colors as CSS hex values. Overrides built-in colors for that label.
labels:
  bug: "#e11d48"
  feature: "#2563eb"
  epic: "#7c3aed"
  improvement: "#0891b2"

# Team members available for assignment. Each entry: "Name <email>"
contributors:
  - "Alice <alice@example.com>"
  - "Bob <bob@example.com>"

# Automatic state propagation between parent and child issues (all default to true)
automations:
  auto_complete_parent: false       # move parent to DONE when all children are DONE
  auto_close_sub_issues: false      # move children to DONE when parent moves to DONE
  auto_progress_sub_issues: false   # advance children when parent moves forward
  auto_progress_parent: false       # advance parent when a child moves forward

# Role-based permissions (only enforced in distributed / server mode).
# When this block is absent, all operations are allowed.
permissions:
  roles:
    admin: ["*"]            # "*" grants all capabilities
    member: ["issue.read", "issue.create", "issue.update", "issue.comment", "issue.start"]
    viewer: ["issue.read"]
  # Assign users to roles by email
  users:
    alice@example.com: admin
    bob@example.com: member
```

## Agent integration (MCP)

Beats ships an MCP (Model Context Protocol) server so AI agents can manage issues with structured tool calls instead of shelling out. The server is a beats subcommand:

```bash
beats mcp
```

It speaks the MCP stdio protocol; wire it into a compatible client through that client's config.

### Setup for Coding Agents

Pick whichever fits your workflow:

**1. Project-local (recommended) — `.mcp.json` at the repo root.** Checked in alongside the code, so every contributor with `beats` on their `PATH` gets the same tools without setup.

```json
{
  "mcpServers": {
    "beats": {
      "command": "beats",
      "args": ["mcp"]
    }
  }
}
```

**2. User-level — `claude mcp add`.** Fastest path if you don't want a checked-in file:

```bash
claude mcp add beats beats mcp
```

This appends an entry under `mcpServers` in your user-level Claude Code settings. Applies to every project you open until you remove it with `claude mcp remove beats`.

**3. Manual user-level edit.** Add the snippet from option 1 to `~/.claude/settings.json` under `mcpServers`.

After any of these, restart Claude Code (or run `/mcp` in-session) and the seven `beats_*` tools become available to the agent.

### Available tools

| Tool | What it does |
|------|-------------|
| `beats_list` | List issues with filters (status, label, assignee, parent, free-text match) |
| `beats_show` | Fetch one issue with description, dependencies, comments, optional events |
| `beats_history` | Return the event audit trail for an issue |
| `beats_add` | Create an issue, including status/labels/parent/story_points/links in one call |
| `beats_update` | Patch any subset of fields; status transitions go through here |
| `beats_comment` | Add a markdown comment |
| `beats_link` | Add a dependency/relationship between two existing issues |
| `beats_start` | Start work on an issue (set to DOING + create branch) |
| `beats_merge` | Merge an issue's branch and close the issue |

### Reducing permission prompts

By default Claude Code asks for confirmation before each MCP tool call. Allow-list the beats tools in your project's `.claude/settings.json` to skip those prompts:

```json
{
  "permissions": {
    "allow": [
      "mcp__beats__beats_*"
    ]
  }
}
```

All beats MCP writes are append-only events on a git-tracked file, so a mistaken call is recoverable via `git`. The wildcard is the recommended default for any project using beats as its primary tracker.

### Agent identity

Writes performed through MCP are recorded with this precedence:

1. `$BEATS_AGENT_IDENTITY` (e.g. `Claude Code <agent@my-machine.local>`) — operator-controlled, wins if set.
2. MCP `clientInfo` from the initialize handshake — best-effort attribution, e.g. `claude-code/1.x <agent@mcp>`.
3. The configured beats user — same default the CLI uses.

Set `BEATS_AGENT_IDENTITY` in the `env` block of `.mcp.json` to fix the identity per agent:

```json
{
  "mcpServers": {
    "beats": {
      "command": "beats",
      "args": ["mcp"],
      "env": {
        "BEATS_AGENT_IDENTITY": "Claude Code <agent@alices-workstation.local>"
      }
    }
  }
}
```

## Architecture

Issues live in `.beats/issues.db` as an append-only JSONL event log. Reads are accelerated by a local-only snapshot (`.beats/issues.snapshot.json`, gitignored) that caches the projection up to a given event. Old completed issues spill to `.beats/archive.jsonl` to keep the hot log small. The CLI, web UI, and MCP server are all thin clients over the same in-memory projection of these files.

For the full design (event sourcing, transport abstraction, auth, storage, MCP), see [docs/design.md](docs/design.md).

## Contributing

```bash
make           # full build (frontend + Go binary)
make cli       # Go binary only (skip frontend)
make test      # run Go tests
make lint      # go vet
make clean     # remove build artifacts
make release VERSION=x.y.z  # bump version, stamp changelog, tag
```

### Project structure

```
cmd/beats/          CLI commands (Cobra)
internal/
  auth/             JWT signing/verification, SSH key auth, middleware
  beats/            Core business logic — transports, projections, workflows
  config/           Config loading, cycles, permissions
  mcpserver/        MCP protocol handler and tool definitions
  model/            Domain types (Issue, Event, Dependency, etc.)
  server/           HTTP server, handlers, SSE, metrics
  storage/          Event log I/O (append, read, archive, snapshot)
web/                React + Vite frontend (bun)
```

Issues for beats are tracked in beats itself. Run `beats ls` in this repo to see what's open.

## License

[MIT](LICENSE)
