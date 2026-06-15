# Agent Instructions

This repository uses the `beats` issue tracker as the persistent project memory and to manage development tasks. As an AI agent, you should use `beats` to understand the current state of the project, plan your work, record new findings, and to document the rationale and work done to implement your tasks.

## How to interact with `beats`

A `beats` MCP server is registered in [.mcp.json](.mcp.json). **Always use the MCP tools** — do not shell out to the `beats` CLI for the agent workflow.

| Action | MCP tool |
|---|---|
| List / search issues | `mcp__beats__beats_list` |
| Read one issue | `mcp__beats__beats_show` |
| Create an issue | `mcp__beats__beats_add` |
| Update fields (incl. **status transitions**, labels, assignee, story_points, parent) | `mcp__beats__beats_update` |
| Add a comment | `mcp__beats__beats_comment` |
| Add a dependency link | `mcp__beats__beats_link` |
| View audit trail | `mcp__beats__beats_history` |

Status transitions (`BACKLOG` → `PLANNED` → `DOING` → `BLOCKED` → `DONE`) are done by calling `beats_update` with the `status` field — there are no separate start/done tools.

> The CLI command reference under [claude/skills/](claude/skills/) is supplemental material for downstream users of `beats` to install in their own projects. It is not the interface this repository's agents should use.

## Development Workflow

1. **Discover** — `beats_list` to see the board; `beats_show` for details on candidate issues.
2. **Plan** — if no issue covers the work, create one with `beats_add` (only after the user approves the design).
3. **Start** — `beats_update` with `status: "DOING"` before editing any code.
4. **Implement & test** — make changes, then run `make test` / `make build` to check against the test suite.
5. **Document** — `beats_comment` with a markdown summary of what changed and why.
6. **Complete** — `beats_update` with `status: "DONE"` once the user approves.

## Strict Workflow Rules

1. **No "ghost" work** — every code change MUST be backed by a beats issue.
2. **Only pick up planned work** — do not start work on issues with `BACKLOG` status. Issues must be `PLANNED` to be eligible.
3. **Check dependencies first** — before picking up an issue, inspect its `dependencies` array via `beats_show`. If any `depends_on` or `blocked_by` targets are not `DONE`, flag the unresolved blockers before starting work.
4. **Missing tasks** — if no issue exists for your current objective, create it first, but only _after_ the user approves your design/plan.
5. **In-progress before edits** — before touching any file, transition the issue to `DOING` via `beats_update`.
6. **Comment before complete** — add a summary comment via `beats_comment` _before_ transitioning to `DONE`, and only do so after the user approves the final walkthrough.
7. **File what you find** — bugs or follow-up work discovered during a task must be filed as new issues (linked to the current one via `beats_link`), not left as TODOs in code.

## Agent Identity

When the tracker records who made a change, identify yourself as an agent. Use the form `<Agent Name> <agent@<host>.local>` — e.g. `Claude Code <agent@nicbet-wsl.local>`. The host portion helps distinguish contributions from different execution environments. (Note: per stored memory, `GIT_AUTHOR_NAME` / `GIT_AUTHOR_EMAIL` env vars do not affect `beats` identity.)

## Usage Notes

### Discovery

- Before creating a new issue, search with `beats_list` (use the `match` parameter for free-text search) to ensure no existing issue already covers the work.
- If an issue looks related, read it fully with `beats_show` before deciding.

### Creating issues

- Always set at least one label via the `labels` parameter. Prefer the built-in labels (`bug`, `feature`, `epic`) unless something more specific fits.
- Keep titles under 100 characters.
- Descriptions render as **Markdown** in the web UI. Use headings, lists, code blocks, bold/italic, and links. Use double newlines between paragraphs. Markdown checklists (`- [ ] Title`) can sub-divide task steps.
- When a new issue belongs to an Epic, set `parent` to the epic's ID.

### Linking

Use `beats_link` to express relationships. Supported types: `blocks`, `blocked_by`, `depends_on`, `dependency_of`, `duplicates`, `duplicated_by`, `relates_to`. When a task spawns follow-up work, link the new issue back to the originating one.

### Completion comments

Comments are markdown. A good completion comment includes:

- A short **Summary** section listing what changed (use backtick code spans for file/function names).
- The **rationale** — why this approach, why not the alternatives.
- Anything a future agent reading the issue would need to pick up where you left off.

## Building this project

- `make cli` — compile the `beats` binary.
- `make frontend` — build the web application assets.
- `make build` — build the CLI with the web assets embedded.
- `make test` — run the test suite.
- always use `bun` and `bunx` over `npm` and `npmx` when available
