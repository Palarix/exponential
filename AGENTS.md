<!-- xpo:begin 1.5.0 sha256:d976cfa6f6c0 -->
## Exponential (xpo)

This project uses `xpo` (Exponential) via the MCP server registered in `.mcp.json`.
Always use the MCP tools — never shell out to the `xpo` CLI. If an MCP tool call fails,
report the error to the user. Never fall back to the CLI.

Issue IDs in this project use the prefix `xpo` (e.g. `xpo-a1b2c3`).

If the project has an `xpo-workflow` skill, load it before you plan, file or change anything, and follow it.
For how the xpo tools work, load the `xpo` skill.

<!-- xpo:end -->
