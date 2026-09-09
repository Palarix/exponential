# Spec: Separate `xpo init` from skill installation

## What

Split the monolithic `xpo init` command into focused subcommands:
- `xpo init` — idempotent project data setup (`.xpo/` directory, config, database, git config)
- `xpo init mcp` — per-harness MCP server configuration
- `xpo init skill` — agent harness skill files + agent instruction files

Update `xpo doctor` to be the universal health check across all concerns.

## Why

The current `xpo init` bundles four distinct responsibilities: project data setup, MCP config, agent instruction files, and skill installation. This coupling means:
- Users can't re-install skills or MCP config independently
- There's no way to install skills globally (user-level) vs locally (project-level)
- `xpo doctor` can't distinguish between "project not initialized", "MCP not configured", and "skills not installed"
- Different agent harnesses need different MCP config formats/locations
- Adding new agent harness support requires touching the init flow

## Acceptance Criteria

- [ ] `xpo init` writes config only — no side effects on MCP, agent instruction files, or skill files
- [ ] `xpo init` is idempotent — safe to re-run; never destroys existing `.xpo/` data
- [ ] `xpo init` checks config completeness (all expected keys present) and version (latest data model)
- [ ] `xpo init mcp` writes per-harness MCP config in the correct format/location
- [ ] `xpo init skill` is interactive — detects installed agents, prompts for local/global scope
- [ ] `xpo init skill` writes agent instruction files, respecting existing content in the file
- [ ] Supported harnesses: Claude Code, GitHub Copilot, Cursor, Codex, OpenCode (+ Generic Agent fallback)
- [ ] `xpo init skill --force` required to overwrite an existing xpo section in agent instruction files
- [ ] Global skill install uses `~/.config/xpo/skills/` as canonical location, symlinked into harness dirs (macOS/Linux only)
- [ ] On Windows, `--global` is not available — only project-local installation
- [ ] `xpo doctor` is the universal health check: git repo, config correctness + version, MCP, skills, agent files
- [ ] First run of `xpo init` prints actionable hints (next steps: `xpo init mcp`, `xpo init skill`)
- [ ] Existing projects that already ran the old `xpo init` continue to work

## Agent Harness Registry

The `AgentRegistry` is trimmed to actively supported harnesses:

| Name | File | Binary | SkillDir | Format | MCP Config |
|---|---|---|---|---|---|
| Generic Agent | `AGENTS.md` | — | — | markdown | — |
| Claude Code | `CLAUDE.md` | `claude` | `.claude/skills` | markdown | `.mcp.json` (JSON, `mcpServers` key) |
| GitHub Copilot | `.github/copilot-instructions.md` | — | — | markdown | — |
| Cursor | `.cursorrules` | `cursor` | `.cursor/skills` | plain | `.cursor/mcp.json` (JSON, `mcpServers` key) |
| Codex | `AGENTS.md` | `codex` | `.codex/skills` | markdown | `.codex/config.toml` (TOML, `[mcp_servers.*]`) |
| OpenCode | `AGENTS.md` | `opencode` | `.opencode/skills` | markdown | `opencode.json` (JSON, `mcp` key) |

**Removed:** Windsurf, Gemini, Cline, Roo Code, Aider, Continue.

**Shared instruction file:** Codex, OpenCode, and Generic Agent all use `AGENTS.md`. This is fine — the instruction content is identical. Each harness differs in MCP config location/format and skill directory. `AppendAgentInstructions` is idempotent, so writing `AGENTS.md` multiple times produces the same result.

**Agents without skill support** (GitHub Copilot, Generic Agent): `xpo init skill` writes the full inline agent docs (not just the thin stub) via `GenerateAgentDocs()`, since there's no skill to load.

## Flow

### 1. Refactor `xpo init` (config only, idempotent)

**File:** `cmd/exponential/init.go`

Strip Phase 2 (MCP + agent instructions + skills) from `initCmd.Run`. The command becomes:

1. **Create `.xpo/` directory** if missing (never overwrite if present)
2. **Write `config.yaml`** on fresh init or `--force`. On re-run without `--force`:
   - Check all expected config keys are present; add missing keys with defaults
   - Check data model version; suggest `xpo migrate` if outdated
3. **Create `issues.db`** if missing
4. **Ensure `.gitignore`** entries (additive only)
5. **Ensure `.gitattributes`** merge rule (additive only)
6. **Run checks** — git repo, GitHub workflows, git hooks, shell completion
7. **Print summary + hints:**
   - On first init: "Run `xpo init mcp` to configure MCP, then `xpo init skill` to install agent skills."
   - On re-run: only print hints for components that are missing

The `--force` flag re-writes config from scratch (current behavior).

### 2. New `xpo init mcp` subcommand

**File:** `cmd/exponential/init_mcp.go` (new)

Register as subcommand: `initCmd.AddCommand(initMCPCmd)`.

**Flags:**
- `--harness <name>` — configure MCP for a specific agent harness only. If omitted, auto-detect installed agents
- `--force` — overwrite existing MCP config entries

**Prerequisite:** `.xpo/` directory must exist (error: "run `xpo init` first").

**Steps:**
1. Determine target agents (auto-detect or `--harness`)
2. For each agent with an `MCPConfig` spec:
   a. Check if MCP config file already exists and has an xpo entry
   b. If entry exists and no `--force`: skip with note "already configured, use --force to overwrite"
   c. Write MCP entry in the harness-appropriate format/location
3. Print summary

**Interactive flow:** if a config file exists but has no xpo entry, write it (additive, non-destructive). If the file has an existing xpo entry: require `--force` or confirm interactively.

### 3. New `xpo init skill` subcommand (interactive)

**File:** `cmd/exponential/init_skill.go` (new)

Register as subcommand: `initCmd.AddCommand(initSkillCmd)`.

**Flags:**
- `--harness <name>` — install for a specific agent harness only. If omitted, auto-detect all installed agents
- `--global` — install skill files globally via `~/.config/xpo/skills/` + symlinks (macOS/Linux only; error on Windows)
- `--local` — install skill files into the project (default if neither flag given and non-interactive)
- `--force` — overwrite existing installations

**Prerequisite:** `.xpo/` directory must exist (error: "run `xpo init` first").

**Responsibilities:**
- Write skill files (SKILL.md, references/) — local or global
- Write agent instruction files (CLAUDE.md, AGENTS.md, .cursorrules, etc.) — always project-local

**Interactive flow** (when stdin is a terminal):
1. Detect installed agents via `DetectInstalledAgents()`
2. Show detected agents, let user confirm or deselect
3. Prompt: install skills **globally** or **locally**? (skip on Windows — always local)
4. For each selected agent:
   a. Handle agent instruction file (see edge case rules below)
   b. Write skill files (global or local per user's choice)
5. Print summary

**Agent instruction file handling (CLAUDE.md, AGENTS.md, .cursorrules, etc.):**

| State | Action | `--force`? |
|---|---|---|
| File does not exist | Create with xpo section | No |
| File exists, no xpo section | Append xpo section (confirm interactively) | No |
| File exists, has xpo section | Replace xpo section in place (all other content preserved) | Yes (or interactive confirm) |

**Non-interactive flow** (piped/headless):
- Use auto-detected agents (or `--harness`)
- Default to `--local` if neither `--global` nor `--local` specified
- Require `--force` to overwrite existing xpo sections or skill directories

### 4. Global skill installation via `~/.config/xpo/skills/` (macOS/Linux only)

**Canonical location:** `~/.config/xpo/skills/xpo-workflow/`

Contains the actual skill files:
- `SKILL.md`
- `references/spec-guide.md`
- `references/mcp-tools.md`

**Symlinks into harness directories:**
- Claude Code: `~/.claude/skills/xpo-workflow` → `~/.config/xpo/skills/xpo-workflow`
- Cursor: `~/.cursor/skills/xpo-workflow` → `~/.config/xpo/skills/xpo-workflow`
- Codex: `~/.codex/skills/xpo-workflow` → `~/.config/xpo/skills/xpo-workflow`
- OpenCode: `~/.config/opencode/skills/xpo-workflow` → `~/.config/xpo/skills/xpo-workflow`

**Windows:** `--global` flag prints an error: "Global skill installation is not supported on Windows. Use `--local` instead."

### 5. Per-harness MCP config

**File:** `internal/exponential/agents.go`

Extend `AgentConfig` with MCP config knowledge:

```go
type MCPConfigSpec struct {
    File      string // config file path relative to project root
    ServerKey string // JSON key for server map (e.g. "mcpServers", "mcp")
    Format    string // "json" or "toml"
    NeedsDir  bool   // create parent dir if missing
}
```

Three MCP config formats to support:

**JSON with `mcpServers` key** (Claude Code, Cursor):
```json
{
  "mcpServers": {
    "xpo": { "command": "xpo", "args": ["mcp"] }
  }
}
```

**JSON with `mcp` key** (OpenCode):
```json
{
  "mcp": {
    "xpo": { "command": "xpo", "args": ["mcp"] }
  }
}
```

**TOML** (Codex):
```toml
[mcp_servers.xpo]
command = "xpo"
args = ["mcp"]
```

Refactor `EnsureMCPConfig()` → `EnsureMCPConfigFor(spec MCPConfigSpec) error` to handle all three formats. The current `EnsureMCPConfig()` logic (read file, parse, add entry, write back) generalizes well — the key and file path become parameters, and TOML gets a parallel code path.

### 6. Update `WriteAgentSkill` signature

**File:** `internal/exponential/agents.go`

Change to: `WriteAgentSkill(agent AgentConfig, globalBaseDir string) (string, error)`

- `globalBaseDir` empty → project-local (current behavior using `agent.SkillDir`)
- `globalBaseDir` set → write files to `globalBaseDir/xpo-workflow/`, then create symlink from `agent.SkillDir/xpo-workflow` → `globalBaseDir/xpo-workflow`

### 7. Update `xpo doctor` (universal health check)

**File:** `cmd/exponential/doctor.go`

Reorganize into clearly headed sections:

**Section 1: "Project Configuration"** (fix with `xpo init`)
- `.xpo/` directory exists
- Data model version matches CLI
- Config has all expected keys (report missing ones)
- Git repo detected
- `issues.db` exists
- `.gitignore` entries present
- `.gitattributes` merge rule present

**Section 2: "MCP Configuration"** (fix with `xpo init mcp`)
- Per-harness: MCP config file exists, has xpo entry, format correct

**Section 3: "Agent Integration"** (fix with `xpo init skill`)
- Agent instruction files detected and configured
- Skill installation status per harness:
  - Check project-local path
  - Check global path (`~/.config/xpo/skills/` + symlinks) on macOS/Linux
  - Report: "installed locally", "installed globally", "installed globally (symlink broken)", or "not installed"
- Shell completion configured

Each section header includes the fix command.

### 8. Verify `PersistentPreRunE` allowlist

**File:** `cmd/exponential/main.go`

Ensure `init` and all its subcommands can run without a loaded config. Currently `init` is in the allowlist — verify Cobra subcommand inheritance works here.

## Decisions

1. **Three-way split: `init` / `init mcp` / `init skill`** — each subcommand owns one concern.

2. **Trimmed registry to 6 harnesses** — Claude Code, GitHub Copilot, Cursor, Codex, OpenCode, Generic Agent. Dropped Windsurf, Gemini, Cline, Roo Code, Aider, Continue.

3. **Shared `AGENTS.md`** — Codex, OpenCode, and Generic Agent all use the same file. Writing it is idempotent. No conflict.

4. **Three MCP formats** — JSON/mcpServers (Claude, Cursor), JSON/mcp (OpenCode), TOML (Codex). Handled by `MCPConfigSpec` with a `Format` + `ServerKey` fields.

5. **Agent instruction files belong in `init skill`** — the stub tells the agent to load the skill. Tightly coupled to skill installation.

6. **`~/.config/xpo/skills/` as canonical global location (macOS/Linux)** — single source of truth, symlinked into each harness.

7. **Windows: project-local only** — no global skill installation on Windows.

8. **`--force` semantics for agent instruction files** — creating new or appending (non-destructive) does not require `--force`. Replacing an existing xpo section does.

9. **`xpo init` checks config completeness on re-run** — lightweight config migration/repair without destroying existing data.

## Edge Cases

**HIGH — Existing projects (already ran old `xpo init`):** Everything is already set up. New `xpo init` won't touch agent files or MCP config. `xpo init mcp` and `xpo init skill` without `--force` will detect existing installations and skip with a note.

**HIGH — Existing agent instruction file without xpo section:** User has a `CLAUDE.md` in their repo before adding xpo. `xpo init skill` appends the xpo section. In interactive mode, confirms before modifying. All existing content is preserved.

**HIGH — Existing agent instruction file with xpo section:** Requires `--force` or interactive confirmation to replace. Only the xpo section is touched — all other content preserved.

**HIGH — Multiple harnesses sharing `AGENTS.md`:** If both Codex and OpenCode are detected, `AGENTS.md` is written once (idempotent). MCP and skills are written separately per harness.

**MEDIUM — Broken symlinks:** Global skill dir deleted but symlinks remain. `xpo doctor` detects broken symlinks and reports with fix instructions.

**MEDIUM — `--harness` with unknown name:** Print available harness names from registry and exit with error.

**MEDIUM — No agents detected:** Show the Generic Agent fallback option. Let user choose to install AGENTS.md.

**MEDIUM — TOML dependency:** Codex MCP config is TOML. Need a TOML library — check if one is already in `go.mod`, or add one (e.g. `github.com/BurntSushi/toml`).

**LOW — `~/.config` doesn't exist:** Create with `os.MkdirAll`.

**LOW — Mixed global/local:** User installed globally before, now wants local (or vice versa). `xpo doctor` reports both.
