# Beats

The git-native issue tracker for humans and AI agents. Issues are committed alongside the code that resolves them, the board is a single binary you can run anywhere, and AI agents talk to it through a native MCP server — no SaaS, no API keys.

## Why beats

- **Git-native** — issues live in `.beats/` and are tracked with your code. Branches, merges, and history apply to your backlog the same way they apply to everything else.
- **Single binary** — `beats` ships the CLI, the web UI, and the MCP server in one Go executable. No Node.js, no Docker, no database to run.
- **Agent-first** — a built-in MCP server gives AI coding agents structured tools for reading, writing, and linking issues. No more shelling out to a CLI and praying about quoting.
- **Event-sourced** — every change is an append-only event with a full audit trail. Merge conflicts are rare; "who changed what, when, and why" is always answerable.
- **Local web UI** — `beats board` opens a Linear-style Kanban in your browser. Drag-and-drop, sub-issues, keyboard shortcuts, command palette.

## Quickstart

```bash
# 1. Build and install
make && make install

# 2. Initialize in your project
cd path/to/your/repo
beats init

# 3. Capture some work
beats add "Wire up login form" --label feature

# 4. Open the board
beats board
```

That's the whole loop. Everything else is a refinement of these four steps.

## Installation

### From source (requires Go 1.25+ and [bun](https://bun.sh))

```bash
git clone https://github.com/kuyio/beats.git
cd beats
make && make install
```

`make` builds the frontend assets and the Go binary. `make install` copies the resulting `beats` executable to `/usr/local/bin/beats` (it will ask for `sudo` permission).

To rebuild later:

```bash
make           # full rebuild (frontend + Go)
make cli       # Go binary only, skip frontend
make test      # run the test suite
make clean     # nuke build artifacts
```

## CLI usage

### Daily workflow

```bash
beats init                                # one-time, inside a git repo
beats list                                # see the board (alias: beats ls)
beats show <id>                           # full details for one issue
beats add "Title" --label feature         # create
beats start <id>                          # move to DOING
beats done  <id>                          # move to DONE
beats comment <id> "Fixed in auth.go"     # markdown comment
```

### Status transitions

```bash
beats planned <id>                        # → PLANNED
beats start   <id>                        # → DOING
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

### Relationships

```bash
beats add "Big initiative" --label epic              # epics group work
beats add "Subtask"        --label feature -p <epic-id>
beats link <a> <b> --type blocks                     # also: depends_on,
                                                     # blocked_by, duplicates,
                                                     # relates_to
beats estimate <id> 5                                # story points
beats update   <id> --assignee "Alice"
```

### Audit, maintenance, troubleshooting

```bash
beats history <id>      # event log for one issue
beats archive           # move old DONE issues to cold storage
beats doctor            # validate issues.db, offer fixes
beats config            # show / edit local config
```

Run `beats --help` or `beats <command> --help` for the full surface.

## Web UI

`beats board` spins up a local web server (default port `8080`), embeds the React frontend from the binary, and opens your browser. The UI is Linear-inspired:

- Drag-and-drop across status columns (hold `Cmd`/`Ctrl` for precise-slot mode).
- Hold `Alt` while dragging onto another issue to nest it as a sub-issue.
- `Cmd`/`Ctrl` + `K` opens a command palette for jump-to-issue and global actions.
- Sub-issues are first-class: epics show progress, children render inline.

If port 8080 is already in use, beats picks the next free port automatically. Run `beats board` from any number of projects in parallel — every running instance registers in a shared local registry, and each sidebar lists the other live boards as one-click destinations. Clean shutdowns unregister automatically; hard-killed instances are pruned the next time a peer refreshes.

The server binds to `127.0.0.1` only — no remote access, no auth needed.

## Agent integration (MCP)

Beats ships an MCP (Model Context Protocol) server so AI agents can manage issues with structured tool calls instead of shelling out. The server is a beats subcommand:

```bash
beats mcp
```

It speaks the MCP stdio protocol; wire it into a compatible client through that client's config.

### Setup for Claude Code

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

| Tool             | What it does                                                                |
| ---------------- | --------------------------------------------------------------------------- |
| `beats_list`     | List issues with filters (status, label, assignee, parent, free-text match) |
| `beats_show`     | Fetch one issue with description, dependencies, comments, optional events   |
| `beats_history`  | Return the event audit trail for an issue                                   |
| `beats_add`      | Create an issue, including status/labels/parent/story_points/links in one call |
| `beats_update`   | Patch any subset of fields; status transitions go through here              |
| `beats_comment`  | Add a markdown comment                                                      |
| `beats_link`     | Add a dependency/relationship between two existing issues                   |

Status shortcuts (`beats_start`, `beats_done`, etc.) are intentionally absent — agents transition status via `beats_update` with `status: "DOING"` etc., which keeps the tool surface tight.

### Reducing permission prompts

By default Claude Code asks for confirmation before each MCP tool call. Allow-list the beats tools in your project's `.claude/settings.json` to skip those prompts. Wildcard everything:

```json
{
  "permissions": {
    "allow": [
      "mcp__beats__beats_*"
    ]
  }
}
```

…or allow individual tools if you'd rather keep writes gated and only auto-allow reads:

```json
{
  "permissions": {
    "allow": [
      "mcp__beats__beats_list",
      "mcp__beats__beats_show",
      "mcp__beats__beats_history",
      "mcp__beats__beats_add",
      "mcp__beats__beats_update",
      "mcp__beats__beats_comment",
      "mcp__beats__beats_link"
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
        "BEATS_AGENT_IDENTITY": "Claude Code <agent@nicspc.local>"
      }
    }
  }
}
```

## Configuration

Settings are loaded in order (highest precedence first):

1. Environment variables (`BEATS_*`)
2. Project config: `.beats/config.yaml`
3. User config: `~/.config/beats/config.yaml`
4. Built-in defaults

| Key            | Environment variable     | Default                   | Description                                        |
| -------------- | ------------------------ | ------------------------- | -------------------------------------------------- |
| `editor`       | `BEATS_EDITOR`           | `$EDITOR` or `vim`        | Editor for entering long-form text.                |
| `auto_commit`  | `BEATS_AUTO_COMMIT`      | `false`                   | Auto-commit `issues.db` changes after each write.  |
| `style.theme`  | `BEATS_STYLE_THEME`      | `default`                 | UI theme (currently only `default`).               |

Example `.beats/config.yaml`:

```yaml
editor: nano
auto_commit: true
style:
  theme: default
```

## Architecture (in one paragraph)

Issues live in `.beats/issues.db` as an append-only JSONL event log. Reads are accelerated by a local-only snapshot (`.beats/issues.snapshot.json`, gitignored) that caches the projection up to a given event. Old completed issues spill to `.beats/archive.jsonl` to keep the hot log small. The CLI, web UI, and MCP server are all thin clients over the same in-memory projection of these files.

For the full design (event sourcing, snapshotting, merge-conflict strategy, ID generation), see [design_docs/DESIGN.md](design_docs/DESIGN.md).

## Building & contributing

```bash
make           # full build (frontend + Go binary)
make cli       # Go binary only
make test      # run Go tests
make lint      # go vet
make clean     # remove build artifacts
```

Frontend lives under [web/](web/) (React + Vite, package-managed with bun). Backend is under [internal/](internal/) and [cmd/beats/](cmd/beats/). MCP server is under [internal/mcpserver/](internal/mcpserver/).

Issues for beats are tracked in beats itself (eat your own dog food). Run `beats ls` in this repo to see what's open.
