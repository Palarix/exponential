<!-- xpo:begin 1.5.0 sha256:d976cfa6f6c0 -->
## Exponential (xpo)

This project uses `xpo` (Exponential) via the MCP server registered in `.mcp.json`.
Always use the MCP tools — never shell out to the `xpo` CLI. If an MCP tool call fails,
report the error to the user. Never fall back to the CLI.

Issue IDs in this project use the prefix `xpo` (e.g. `xpo-a1b2c3`).

If the project has an `xpo-workflow` skill, load it before you plan, file or change anything, and follow it.
For how the xpo tools work, load the `xpo` skill.

<!-- xpo:end -->

### Keeping the Changelog actual

- When a user asks for making a commit, ensure you update the CHANGELOG.md
- If work was done based on a story or issue, reference the issue id in the changelog entry

## Building this project

- `make cli` — compile the `xpo` binary.
- `make frontend` — build the web application assets.
- `make build` — build the CLI with the web assets embedded.
- `make test` — run the test suite (includes lint).
- `make lint` — run `go vet`, frontend eslint and the frontend type check (`tsc -b`).
- always use `bun` and `bunx` over `npm` and `npmx` when available

## Before declaring implementation done

`make test` must pass. It runs lint (`go vet` + `bun run lint` + `bun run typecheck`), then frontend tests, then Go tests.
