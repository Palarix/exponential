# Exponential

The git-native issue tracker for humans and AI agents. Issues are committed alongside the code that resolves them, the board is a single binary you can run anywhere, and AI agents talk to it through a native MCP server — no SaaS, no API keys.

## Why xpo

- **Git-native** — issues live in `.xpo/` and are tracked with your code. Branches, merges, and history apply to your backlog the same way they apply to everything else.
- **Single binary** — `xpo` ships the CLI, the web UI, and the MCP server in one Go executable. No Node.js, no Docker, no database to run.
- **Agent-first** — a built-in MCP server gives AI coding agents structured tools for reading, writing, and linking issues. No more shelling out to a CLI and praying about quoting.
- **Event-sourced** — every change is an append-only event with a full audit trail. Merge conflicts are rare; "who changed what, when, and why" is always answerable.
- **Local web UI** — `xpo board` opens a Kanban in your browser. Drag-and-drop, sub-issues, keyboard shortcuts, command palette.

## Quickstart

```bash
# 1. Install (see below for other methods)
go install github.com/palarix/xpo/cmd/xpo@latest

# 2. Initialize in your project
cd path/to/your/repo
xpo init

# 3. Capture some work
xpo add "Wire up login form" --label feature

# 4. Open the board
xpo board
```

That's the whole loop. Everything else is a refinement of these four steps (see below)

## Installation

### Binary download

Grab the latest release for your platform from [GitHub Releases](https://github.com/palarix/xpo/releases) and put it on your `PATH`.

### Go install

```bash
go install github.com/palarix/xpo/cmd/xpo@latest
```

Requires Go 1.25+. This builds the CLI without the embedded web UI. For the full build (CLI + web UI), build from source.

### From source

Requires Go 1.25+ and [bun](https://bun.sh).

```bash
git clone https://github.com/palarix/xpo.git
cd xpo
make && make install
```

`make` builds the frontend assets and the Go binary. `make install` copies the resulting `xpo` executable to `/usr/local/bin/xpo` (it will ask for `sudo` permission).

### Docker

```bash
docker build -t xpo:latest .
docker run -v /path/to/repo:/data -p 8080:8080 xpo:latest
```

The image runs `xpo serve` by default — see [Distributed mode](#distributed-mode) below.

## CLI reference

### Daily workflow

```bash
xpo init                                # one-time, inside a git repo
xpo list                                # see the board (alias: xpo ls)
xpo show <id>                           # full details for one issue
xpo add "Title" --label feature         # create
xpo start <id>                          # move to DOING + create branch
xpo done  <id>                          # move to DONE
xpo comment <id> "Fixed in auth.go"     # markdown comment
```

### Status transitions

```bash
xpo planned <id>                        # → PLANNED
xpo start   <id>                        # → DOING (creates a feature branch)
xpo blocked <id> --by <other-id>        # → BLOCKED
xpo done    <id>                        # → DONE
```

### Filtering

```bash
xpo list --status PLANNED
xpo list --label bug
xpo list --assignee "Alice"
xpo list --parent <epic-id>             # children of an epic
xpo list --since 1w                     # updated in the last week
xpo list --match "login"                # full-text search
xpo list --mine                         # issues assigned to you
xpo list --all                          # include archived DONE
```

### Relationships and estimation

```bash
xpo add "Big initiative" --label epic              # epics group work
xpo add "Subtask"        --label feature -p <epic-id>
xpo link <a> <b> --type blocks                     # also: depends_on,
                                                     # blocked_by, duplicates,
                                                     # relates_to
xpo estimate <id> 5                                # story points
xpo update   <id> --assignee "Alice"
```

### Branch workflows

```bash
xpo start <id>                          # creates branch, moves to DOING
xpo review <id>                         # unified diff from the terminal
xpo merge <id>                          # squash-merge + close
```

### Inbox

```bash
xpo inbox                               # events relevant to you
xpo inbox --since 2d                    # last 2 days only
```

### Cycles

```bash
xpo cycle init                          # configure sprint cadence
xpo cycle                               # current cycle info
xpo cycle next                          # next cycle info
xpo cycle history                       # past cycles with completion stats
```

### Audit, maintenance, troubleshooting

```bash
xpo history <id>      # event log for one issue
xpo archive           # move old DONE issues to cold storage
xpo doctor            # validate issues.db, offer fixes
xpo config            # show / edit local config
xpo migrate           # upgrade data model to latest version
```

Run `xpo --help` or `xpo <command> --help` for the full surface.

## Web UI

`xpo board` spins up a local web server (default port `8080`), embeds the React frontend from the binary, and opens your browser. The UI is 

- Drag-and-drop across status columns (hold `Cmd`/`Ctrl` for precise-slot mode).
- Hold `Alt` while dragging onto another issue to nest it as a sub-issue.
- `Cmd`/`Ctrl` + `K` opens a command palette for jump-to-issue and global actions.
- Sub-issues are first-class: epics show progress, children render inline.
- Dashboard with velocity charts, flow metrics, cycle burndowns, and workload distribution.
- Inbox with grouped notifications and read/unread state.

If port 8080 is already in use, xpo picks the next free port automatically. Run `xpo board` from any number of projects in parallel — every running instance registers in a shared local registry, and each sidebar lists the other live boards as one-click destinations.

The server binds to `127.0.0.1` only — no remote access, no auth needed.

## Distributed mode

For teams that need a shared server, xpo can run in client-server mode with SSH key authentication.

### Server setup

```bash
# Start the headless server (auto-generates signing key on first run)
xpo serve --addr :8080
```

The server reads `.xpo/authorized_keys` for user verification. Add team members' SSH public keys (one per line, same format as `~/.ssh/authorized_keys`):

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
xpo login https://xpo.example.com

# All commands now work against the remote server
xpo list
xpo add "Remote issue" --label feature

# Check your identity
xpo whoami

# Disconnect
xpo logout
```

### Board

In distributed mode `xpo board` will proxy to your remote so you get the full web UI locally while data lives on the server:

### Docker deployment

```bash
docker build -t xpo:latest .
docker run -d \
  -v /path/to/repo:/data \
  -p 8080:8080 \
  xpo:latest
```

The container runs `xpo serve` by default, exposing port 8080.

## Configuration

Settings live in `.xpo/config.yaml` and are created by `xpo init`. Project-level settings take precedence over user-level settings in `~/.config/xpo/config.yaml`. Environment variables prefixed with `XPO_` override both (dots become underscores, e.g. `XPO_AUTO_COMMIT=true`).

A full example with explanations:

```yaml
# Issue ID prefix — IDs become <prefix>1, <prefix>2, etc.
prefix: myproject-

# Configuration format version, managed by `xpo migrate`
version: 3

# Automatically git-commit every xpo write (add, update, comment, etc.).
# When false, changes accumulate in the working tree for you to commit manually.
auto_commit: false

# Story-point scale used by `xpo estimate`.
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

Exponential ships an MCP (Model Context Protocol) server so AI agents can manage issues with structured tool calls instead of shelling out. The server is a xpo subcommand:

```bash
xpo mcp
```

It speaks the MCP stdio protocol; wire it into a compatible client through that client's config.

### Setup for Coding Agents

Pick whichever fits your workflow:

**1. Project-local (recommended) — `.mcp.json` at the repo root.** Checked in alongside the code, so every contributor with `xpo` on their `PATH` gets the same tools without setup.

```json
{
  "mcpServers": {
    "xpo": {
      "command": "xpo",
      "args": ["mcp"]
    }
  }
}
```

**2. User-level — `claude mcp add`.** Fastest path if you don't want a checked-in file:

```bash
claude mcp add xpo xpo mcp
```

This appends an entry under `mcpServers` in your user-level Claude Code settings. Applies to every project you open until you remove it with `claude mcp remove xpo`.

**3. Manual user-level edit.** Add the snippet from option 1 to `~/.claude/settings.json` under `mcpServers`.

After any of these, restart Claude Code (or run `/mcp` in-session) and the seven MCP tools become available to the agent.

### Available tools

| Tool | What it does |
|------|-------------|
| `list` | List issues with filters (status, label, assignee, parent, free-text match) |
| `show` | Fetch one issue with description, dependencies, comments, optional events |
| `history` | Return the event audit trail for an issue |
| `add` | Create an issue, including status/labels/parent/story_points/links in one call |
| `update` | Patch any subset of fields; status transitions go through here |
| `comment` | Add a markdown comment |
| `link` | Add a dependency/relationship between two existing issues |
| `start` | Start work on an issue (set to DOING + create branch) |
| `merge` | Merge an issue's branch and close the issue |

### Reducing permission prompts

By default Claude Code asks for confirmation before each MCP tool call. Allow-list the xpo tools in your project's `.claude/settings.json` to skip those prompts:

```json
{
  "permissions": {
    "allow": [
      "mcp__xpo__*"
    ]
  }
}
```

All xpo MCP writes are append-only events on a git-tracked file, so a mistaken call is recoverable via `git`. The wildcard is the recommended default for any project using xpo as its primary tracker.

### Agent identity

Writes performed through MCP are recorded with this precedence:

1. `$XPO_AGENT_IDENTITY` (e.g. `Claude Code <agent@my-machine.local>`) — operator-controlled, wins if set.
2. MCP `clientInfo` from the initialize handshake — best-effort attribution, e.g. `claude-code/1.x <agent@mcp>`.
3. The configured xpo user — same default the CLI uses.

Set `XPO_AGENT_IDENTITY` in the `env` block of `.mcp.json` to fix the identity per agent:

```json
{
  "mcpServers": {
    "xpo": {
      "command": "xpo",
      "args": ["mcp"],
      "env": {
        "XPO_AGENT_IDENTITY": "Claude Code <agent@alices-workstation.local>"
      }
    }
  }
}
```

## Architecture

Issues live in `.xpo/issues.db` as an append-only JSONL event log. Reads are accelerated by a local-only snapshot (`.xpo/issues.snapshot.json`, gitignored) that caches the projection up to a given event. Old completed issues spill to `.xpo/archive.jsonl` to keep the hot log small. The CLI, web UI, and MCP server are all thin clients over the same in-memory projection of these files.

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
cmd/xpo/          CLI commands (Cobra)
internal/
  auth/             JWT signing/verification, SSH key auth, middleware
  xpo/            Core business logic — transports, projections, workflows
  config/           Config loading, cycles, permissions
  mcpserver/        MCP protocol handler and tool definitions
  model/            Domain types (Issue, Event, Dependency, etc.)
  server/           HTTP server, handlers, SSE, metrics
  storage/          Event log I/O (append, read, archive, snapshot)
web/                React + Vite frontend (bun)
```

Issues for xpo are tracked in xpo itself. Run `xpo ls` in this repo to see what's open.

## License

[MIT](LICENSE)
