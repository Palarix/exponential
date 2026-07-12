# Agent Instructions

This repository uses the `xpo` issue tracker (Exponential) as the persistent project memory and to manage development tasks. As an AI agent, you should use `xpo` to understand the current state of the project, plan your work, record new findings, and to document the rationale and work done to implement your tasks.

## How to interact with `xpo`

An `xpo` MCP server is registered in [.mcp.json](.mcp.json). **Always use the MCP tools** — do not shell out to the `xpo` CLI for the agent workflow.

| Action | MCP tool |
|---|---|
| List / search issues | `mcp__xpo__list` |
| Read one issue | `mcp__xpo__show` |
| Create an issue | `mcp__xpo__add` |
| Update fields (incl. **status transitions**, labels, assignee, story_points, parent) | `mcp__xpo__update` |
| Add a comment | `mcp__xpo__comment` |
| Add a dependency link | `mcp__xpo__link` |
| View audit trail | `mcp__xpo__history` |
| Read/write/delete a spec | `mcp__xpo__spec` |
| Read/write/delete a walkthrough | `mcp__xpo__walkthrough` |
| Manage generic artifacts | `mcp__xpo__artifact` |

Status transitions (`BACKLOG` → `PLANNED` → `DOING` → `BLOCKED` → `DONE`) are done by calling `update` with the `status` field. When you write descriptions or comments with the mcp__xpo__* tools, do not escape non-printing characters.

> The CLI command reference under [claude/skills/](claude/skills/) is supplemental material for downstream users of `xpo` to install in their own projects. It is not the interface this repository's agents should use.

## Development Workflow

1. **Discover** — `list` to see the board; `show` for details on candidate issues.
2. **Plan** — if no issue covers the work, create one with `add` (only after the user approves the design).
3. **Start** — `update` with `status: "DOING"` before editing any code.
4. **Implement & test** — make changes, then run `make test` / `make build` to check against the test suite.
5. **Document** — `comment` with a markdown summary of what changed and why.
6. **Complete** — `update` with `status: "DONE"` once the user approves.

## Strict Workflow Rules

1. **No "ghost" work** — every code change MUST be backed by an xpo issue.
2. **Missing tasks** — if no issue exists for your current objective, create it first, but only _after_ the user approves your design/plan.
3. **Only pick up planned work** — do not start work on issues with `BACKLOG` status. Issues must be `PLANNED` to be eligible.
4. **Check dependencies first** — before picking up an issue, inspect its `dependencies` array via `show`. If any `depends_on` or `blocked_by` targets are not `DONE`, flag the unresolved blockers before starting work.
5. **Ensure there is a spec** - before starting your implementation work, ensure there is an up-to-date spec attached to the issue. If none exists create one from the current state of the project. For requirements that are unclear or decisions that require user input, now is the time to loop back and ask the user.
6. **In-progress before edits** — before touching any file, transition the issue to `DOING` via `update`.
7. **Comment when done implementing** - when you finish the implementation (evidenced by passing tests), add a summary comment via `comment`.
8. **Attach a walkthrough before done** — attach a walkthrough document to the issue  _before_ transitioning to `DONE`. The walkthrough shall be written from the perspective of a senior engineer explaining the code changes to a junior developer.
9. **File what you find** — bugs or follow-up work discovered during a task must be filed as new issues (linked to the current one via `link`), not left as TODOs in code.

## Agent Identity

When the tracker records who made a change, identify yourself as an agent. Use the form `<Agent Name> <agent@<host>.local>` — e.g. `Claude Code <agent@nicbet-wsl.local>`. The host portion helps distinguish contributions from different execution environments. (Note: per stored memory, `GIT_AUTHOR_NAME` / `GIT_AUTHOR_EMAIL` env vars do not affect `xpo` identity.)

## Usage Notes

### Discovery

- Before creating a new issue, search with `list` (use the `match` parameter for free-text search) to ensure no existing issue already covers the work.
- If an issue looks related, read it fully with `show` before deciding.

### Creating issues

- Always set at least one label via the `labels` parameter. Prefer the built-in labels (`bug`, `feature`, `epic`) unless something more specific fits.
- Keep titles under 100 characters.
- Descriptions render as **Markdown** in the web UI. Use headings, lists, code blocks, bold/italic, and links. Use double newlines between paragraphs. Markdown checklists (`- [ ] Title`) can sub-divide task steps.
- When a new issue belongs to an Epic, set `parent` to the epic's ID.

### Linking

Use `link` to express relationships. Supported types: `blocks`, `blocked_by`, `depends_on`, `dependency_of`, `duplicates`, `duplicated_by`, `relates_to`. When a task spawns follow-up work, link the new issue back to the originating one.

### Specs and Walkthroughs

- Before starting implementation, read the issue's spec (if any) via `spec` with `operation: "read"`. The spec must capture requirements, acceptance criteria, and design decisions before implementation begins.
- After implementation, write a walkthrough via `walkthrough` with `operation: "write"` summarizing what changed and why.
- Use `artifact` for attaching supplemental files (test outputs, design diagrams, logs) that support the issue but don't fit into spec or walkthrough.

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
