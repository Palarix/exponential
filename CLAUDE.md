<!-- xpo:begin 1.2.1 sha256:04bcb9830f10 -->
## Exponential (xpo)

This project uses `xpo` (Exponential) via the MCP server registered in `.mcp.json`.
Always use the MCP tools — never shell out to the `xpo` CLI.

Issue IDs in this project use the prefix `xpo` (e.g. `xpo-a1b2c3`).

### Hard Rules

1. Every code change must be backed by an xpo issue transitioned to DOING before any file is modified.
2. Never start work on a BACKLOG issue without explicit user approval to transition it.
3. Bugs discovered during implementation may be filed and fixed without approval — file the issue, link it to the current work, and fix it.
4. Before beginning any implementation task, load the `xpo` skill and follow it.
5. If an MCP tool call fails, report the error to the user. Never fall back to the CLI.

### Agent Identity

Set the `assignee` field to yourself when transitioning an issue to DOING. Use the form
`<Agent Name> <agent@<host>.local>` — e.g. `Claude Code <agent@macbook.local>`.

<!-- xpo:end -->

### Keeping the Changelog actual

- When a user asks for making a commit, ensure you update the CHANGELOG.md
- If work was done based on a story or issue, reference the issue id in the changelog entry

## Building this project

- `make cli` — compile the `xpo` binary.
- `make frontend` — build the web application assets.
- `make build` — build the CLI with the web assets embedded.
- `make test` — run the test suite (includes lint).
- `make lint` — run `go vet` and frontend eslint.
- always use `bun` and `bunx` over `npm` and `npmx` when available

## Before declaring implementation done

`make test` must pass. It runs lint (`go vet` + `bun run lint`) then Go tests.
