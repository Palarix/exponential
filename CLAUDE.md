# Agent Instructions

This repository uses the `xpo` issue tracker (Exponential) as the persistent project memory and to manage development tasks. As an AI agent, you should use `xpo` to understand the current state of the project, plan your work, record new findings, and to document the rationale and work done to implement your tasks.

## How to interact with `xpo`

An `xpo` MCP server is registered in [.mcp.json](.mcp.json). **Always use the MCP tools** — do not shell out to the `xpo` CLI for the agent workflow.

| Action | MCP tool |
|---|---|
| List / search issues | `mcp__xpo__xpo_list` |
| Read one issue | `mcp__xpo__xpo_show` |
| Create an issue | `mcp__xpo__xpo_add` |
| Update fields (incl. **status transitions**, labels, assignee, story_points, parent) | `mcp__xpo__xpo_update` |
| Add a comment | `mcp__xpo__xpo_comment` |
| Add a dependency link | `mcp__xpo__xpo_link` |
| View audit trail | `mcp__xpo__xpo_history` |

Status transitions (`BACKLOG` → `PLANNED` → `DOING` → `BLOCKED` → `DONE`) are done by calling `xpo_update` with the `status` field. When you write descriptions or comments with the mcp_xpo_* tools, do not escape non-printing characters.

> The CLI command reference under [claude/skills/](claude/skills/) is supplemental material for downstream users of `xpo` to install in their own projects. It is not the interface this repository's agents should use.

## Development Workflow

1. **Discover** — `xpo_list` to see the board; `xpo_show` for details on candidate issues.
2. **Plan** — if no issue covers the work, create one with `xpo_add` (only after the user approves the design).
3. **Start** — `xpo_update` with `status: "DOING"` before editing any code.
4. **Implement & test** — make changes, then run `make test` / `make build` to check against the test suite.
5. **Document** — `xpo_comment` with a markdown summary of what changed and why.
6. **Complete** — `xpo_update` with `status: "DONE"` once the user approves.

## Strict Workflow Rules

1. **No "ghost" work** — every code change MUST be backed by an xpo issue.
2. **Only pick up planned work** — do not start work on issues with `BACKLOG` status. Issues must be `PLANNED` to be eligible.
3. **Check dependencies first** — before picking up an issue, inspect its `dependencies` array via `xpo_show`. If any `depends_on` or `blocked_by` targets are not `DONE`, flag the unresolved blockers before starting work.
4. **Missing tasks** — if no issue exists for your current objective, create it first, but only _after_ the user approves your design/plan.
5. **In-progress before edits** — before touching any file, transition the issue to `DOING` via `xpo_update`.
6. **Comment before complete** — add a summary comment via `xpo_comment` _before_ transitioning to `DONE`, and only do so after the user approves the final walkthrough.
7. **File what you find** — bugs or follow-up work discovered during a task must be filed as new issues (linked to the current one via `xpo_link`), not left as TODOs in code.

## Agent Identity

When the tracker records who made a change, identify yourself as an agent. Use the form `<Agent Name> <agent@<host>.local>` — e.g. `Claude Code <agent@nicbet-wsl.local>`. The host portion helps distinguish contributions from different execution environments. (Note: per stored memory, `GIT_AUTHOR_NAME` / `GIT_AUTHOR_EMAIL` env vars do not affect `xpo` identity.)

## Usage Notes

### Discovery

- Before creating a new issue, search with `xpo_list` (use the `match` parameter for free-text search) to ensure no existing issue already covers the work.
- If an issue looks related, read it fully with `xpo_show` before deciding.

### Creating issues

- Always set at least one label via the `labels` parameter. Prefer the built-in labels (`bug`, `feature`, `epic`) unless something more specific fits.
- Keep titles under 100 characters.
- Descriptions render as **Markdown** in the web UI. Use headings, lists, code blocks, bold/italic, and links. Use double newlines between paragraphs. Markdown checklists (`- [ ] Title`) can sub-divide task steps.
- When a new issue belongs to an Epic, set `parent` to the epic's ID.

### Linking

Use `xpo_link` to express relationships. Supported types: `blocks`, `blocked_by`, `depends_on`, `dependency_of`, `duplicates`, `duplicated_by`, `relates_to`. When a task spawns follow-up work, link the new issue back to the originating one.

### Completion comments

Comments are markdown. A good completion comment includes:

- A short **Summary** section listing what changed (use backtick code spans for file/function names).
- The **rationale** — why this approach, why not the alternatives.
- Anything a future agent reading the issue would need to pick up where you left off.

## Building this project

- `make cli` — compile the `xpo` binary.
- `make frontend` — build the web application assets.
- `make build` — build the CLI with the web assets embedded.
- `make test` — run the test suite.
- always use `bun` and `bunx` over `npm` and `npmx` when available
