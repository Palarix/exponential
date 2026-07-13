# Walkthrough: xpo init should create .mcp.json and agent instructions

## What changed

`xpo init` previously only created the `.xpo` directory (`config.yaml`, `issues.db`, `.gitignore` entries). Setting up `.mcp.json` and agent instruction files was deferred to `xpo doctor`, which required an interactive terminal session. Now `xpo init` handles everything in one step.

## How it works

Three things were added to the init command in `cmd/exponential/init.go`:

1. **MCP config** — calls `EnsureMCPConfig()` (already existed, used by `xpo doctor`) to create or update `.mcp.json` with the xpo MCP server entry.

2. **Agent detection and instruction files** — calls `DetectInstalledAgents()` (new function in `internal/exponential/agents.go`) which checks PATH for known agent CLI binaries using `exec.LookPath()`. Each detected agent gets its instruction file created via `AppendAgentInstructions()`. If no agents are detected, falls back to a generic `AGENTS.md`.

3. **Config hint** — prints a line at the end pointing users to `.xpo/config.yaml` for customization.

## Agent detection

A `Binary` field was added to the `AgentConfig` struct. The registry maps:

| Agent | Binary | File |
|-------|--------|------|
| Claude Code | `claude` | `CLAUDE.md` |
| Gemini | `gemini` | `GEMINI.md` |
| Cursor | `cursor` | `.cursorrules` |
| Windsurf | `windsurf` | `.windsurfrules` |
| Aider | `aider` | `CONVENTIONS.md` |

Agents without a `Binary` (IDE extensions like Cline, Roo Code, Continue, GitHub Copilot) can't be auto-detected and are handled by `xpo doctor` if their config files are found.

`exec.LookPath()` is cross-platform — it respects `PATHEXT` on Windows and standard `PATH` on Linux/macOS.

## Idempotency

On re-init (`xpo init --force`), the init command reads each agent file and checks for the `# Exponential Agent Instructions` header before appending. If already present, it skips and reports "already has xpo instructions". `EnsureMCPConfig()` is inherently idempotent (it merges into existing JSON).

## Key decision

Config is reloaded after `InitProject()` returns because the global `cfg` variable is `nil` during init (the `PersistentPreRunE` in `main.go` allows init to proceed without a loaded config). The freshly written `config.yaml` is read via `config.LoadConfig()` to get the correct prefix for the agent docs.
