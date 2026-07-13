# Exponential Skills for Claude Code

These skills teach Claude Code how to use the `xpo` issue tracker in your project.

## Installation

`xpo init` automatically detects Claude Code and installs the `xpo-workflow` skill to `.claude/skills/xpo-workflow/`. No manual setup needed.

To reinstall or update the skill after upgrading xpo:

```bash
xpo init --force
```

The skills below are supplemental — they ship with this repository and can be manually copied into projects that want the additional slash commands.

## Available Skills

### Auto-installed by `xpo init`

| Skill | Description |
|-------|-------------|
| `xpo-workflow/` | Full development lifecycle: discover, plan, spec, implement, walkthrough, complete. Includes spec-writing guide and MCP tools reference. |

### Supplemental (manual install)

| Skill | Command | Description |
|-------|---------|-------------|
| `xpo/` | (auto-invoked) | Core workflow rules and CLI command reference. |
| `xpo-backlog/` | `/xpo-backlog` | Review project state: in-progress, blocked, planned, and backlog. |
| `xpo-work/` | `/xpo-work` | Guided workflow: discover, start, implement, test, document, complete. |
| `xpo-file/` | `/xpo-file` | File a new bug, task, feature, or epic with duplicate checking. |

To install the supplemental skills:

```bash
cp -r path/to/xpo-cli/claude/skills/xpo* .claude/skills/
```

## Architecture

`xpo init` produces a two-layer setup:

1. **Agent instruction file** (`CLAUDE.md`) — thin always-on stub (~15 lines) containing hard invariants: no ghost work, MCP-only, load the workflow skill before implementation.

2. **Workflow skill** (`.claude/skills/xpo-workflow/`) — the full lifecycle procedure, loaded on demand when the agent is about to do tracked work. Includes reference files for spec writing and MCP tool naming.

This keeps always-on context minimal while giving the agent full procedural guidance when it needs it.
