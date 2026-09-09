# Walkthrough: Separate `xpo init` from skill installation

## What was built

Split the monolithic `xpo init` command into three focused subcommands, each owning one concern:

- **`xpo init`** — project data only (`.xpo/` directory, `config.yaml`, `issues.db`, `.gitignore`, `.gitattributes`). Idempotent and safe to re-run. Prints hints for next steps when MCP or skills are missing.
- **`xpo init mcp`** — per-harness MCP server configuration. Supports four different config formats across harnesses (JSON with `mcpServers` key, JSON with `mcp` key, JSON with `local-array` entry style, and TOML).
- **`xpo init skill`** — interactive workflow skill + agent instruction file installation. Supports local (project-level) and global (`~/.config/xpo/skills/` with symlinks) installation. Handles existing agent files gracefully: appends without `--force`, replaces only the xpo section with `--force`.

Refactored `xpo doctor` into three clearly headed sections (Project Configuration, MCP Configuration, Agent Integration), each pointing to its fix command.

## How the pieces fit together

### Agent registry (`internal/exponential/agents.go`)

The `AgentConfig` struct gained two new fields:
- `GlobalSkillDir` — where to create the symlink for global skill installs (relative to `$HOME`)
- `MCPConfig MCPConfigSpec` — how this harness expects MCP configuration

`MCPConfigSpec` has five fields: `File`, `ServerKey`, `Format` ("json"/"toml"), `EntryStyle` ("standard"/"local-array"), and `NeedsDir`. This captures all the per-harness differences in one struct.

The registry was trimmed from 10 to 6 harnesses:

| Harness | Instruction File | MCP Config | Skills |
|---|---|---|---|
| Generic Agent | `AGENTS.md` | — | — |
| Claude Code | `CLAUDE.md` | `.mcp.json` (JSON/mcpServers) | `.claude/skills` |
| GitHub Copilot | `.github/copilot-instructions.md` | — | — |
| Cursor | `.cursorrules` | `.cursor/mcp.json` (JSON/mcpServers) | `.cursor/skills` |
| Codex | `AGENTS.md` | `.codex/config.toml` (TOML) | `.codex/skills` |
| OpenCode | `AGENTS.md` | `opencode.json` (JSON/mcp, local-array) | `.opencode/skills` |

### MCP config functions

`EnsureMCPConfigFor(spec)` and `DetectMCPConfigFor(spec)` replaced the hardcoded `.mcp.json` functions. JSON handling is parameterized by `ServerKey` and `EntryStyle`. TOML handling uses string-based append (no TOML library dependency) — it checks for `[<serverKey>.xpo]` and appends the section if missing.

The `mcpServerEntry(entryStyle)` helper generates the right JSON entry: standard `{command, args}` for most harnesses, or `{type: "local", command: ["xpo", "mcp"]}` for OpenCode's format.

Backward-compatible `EnsureMCPConfig()`/`DetectMCPConfig()` wrappers delegate to the parameterized versions with `DefaultMCPSpec`.

### Skill installation

`WriteAgentSkill(agent, globalBaseDir)` now accepts an optional base directory. When set, files go to `<globalBaseDir>/xpo-workflow/` and a symlink is created from the harness's global skill directory. `DetectSkillInstall(agent)` checks both local and global paths, including broken symlinks.

Global installation is macOS/Linux only (`--global` errors on Windows).

### Command allowlist (`cmd/exponential/main.go`)

`PersistentPreRunE` now uses `commandOrAncestorAllowed()` which walks the parent chain. This means `xpo init skill` and `xpo init mcp` are automatically allowed because their parent `init` is in the allowlist — no risk of collision with the existing `xpo mcp` server command.

## Key decisions

1. **TOML via string manipulation, not a library** — Codex's `.codex/config.toml` is handled by checking for and appending `[mcp_servers.xpo]` sections. This preserves existing content, avoids a new dependency, and is sufficient for the structured config we write.

2. **`EntryStyle` for OpenCode** — OpenCode uses `{"type": "local", "command": ["xpo", "mcp"]}` instead of the standard `{"command": "xpo", "args": ["mcp"]}`. The `EntryStyle` field on `MCPConfigSpec` selects the right format without complicating the common path.

3. **Shared `AGENTS.md`** — Codex, OpenCode, and Generic Agent all use `AGENTS.md`. The instruction content is identical across them; the per-harness differences are in MCP config and skill directories. `AppendAgentInstructions` is idempotent, so writing the file for multiple harnesses produces the same result.

4. **Parent-chain allowlist** — instead of adding "mcp" and "skill" to the flat command name allowlist (which would collide with `xpo mcp`), `commandOrAncestorAllowed()` checks whether any ancestor command is allowed.

## Acceptance Criteria

- [x] `xpo init` writes config only — no side effects on MCP, agent instruction files, or skill files
- [x] `xpo init` is idempotent — safe to re-run; never destroys existing `.xpo/` data (verified by `TestScenario_Init_ExistingProject_Idempotent`)
- [x] `xpo init mcp` writes per-harness MCP config in the correct format/location (verified against official docs for all 4 harnesses)
- [x] `xpo init skill` is interactive — detects installed agents, prompts for local/global scope
- [x] `xpo init skill` writes agent instruction files, respecting existing content (verified by `TestScenario_InitSkill_ExistingFile_NoXpoSection` and `TestScenario_InitSkill_ExistingXpoSection_Replaced`)
- [x] Supported harnesses: Claude Code, GitHub Copilot, Cursor, Codex, OpenCode (+ Generic Agent fallback)
- [x] `--force` required to overwrite existing xpo sections
- [x] Global skill install with symlinks (macOS/Linux only)
- [x] `xpo doctor` three-section health check with per-section fix commands
- [x] First run prints actionable hints
- [x] Backward compatible — existing projects work unchanged
- [x] `make test` passes (lint + 255 frontend tests + Go tests including 20 new scenario tests)
