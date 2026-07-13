# xpo-ecbf49: xpo init should create .mcp.json and agent instructions

## What
`xpo init` should be a complete one-step setup: create `.xpo` directory, `.mcp.json`, and `AGENTS.md` — not defer agent file and MCP setup to an interactive `xpo doctor` session.

## Why
Users expect `xpo init` to produce a working project. Currently it creates the database but leaves the MCP and agent instruction files for `xpo doctor`, which requires an interactive terminal.

## How
Add two steps to `InitProject()` (or to the init command handler after `InitProject` returns):
1. Call `EnsureMCPConfig()` to create/update `.mcp.json`
2. Call `AppendAgentInstructions()` with the generic agent config to create `AGENTS.md`

Both functions already exist and are called by `xpo doctor`. We just need to also call them during init.

## Acceptance Criteria
- `xpo init` in a fresh directory creates `.mcp.json` with the xpo entry
- `xpo init` in a fresh directory creates `AGENTS.md` with agent instructions
- `xpo init` in a directory with existing `.mcp.json` adds the xpo entry without destroying other entries
- `xpo init` in a directory with existing `AGENTS.md` appends instructions
- `xpo doctor` continues to work for adding instructions to additional agent files
