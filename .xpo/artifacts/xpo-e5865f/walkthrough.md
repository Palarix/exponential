# Walkthrough: Improved Agent Instructions Out of the Box

## What was built

A complete redesign of the agent instructions that `xpo init` generates, replacing a single monolithic document with a portable two-layer architecture informed by the AiSE kit's principles of deliberate software development.

## Architecture

### Layer 1: Agent instruction file (CLAUDE.md, AGENTS.md, etc.)

A thin always-on stub (~17 lines) containing only hard invariants that must fire every turn:

- Use MCP tools, never CLI
- Every code change needs an issue in DOING
- BACKLOG needs user approval to transition
- Bugs can be filed and fixed without approval
- Load the `xpo-workflow` skill before implementation
- Never fall back to CLI on MCP failure
- Set `assignee` when transitioning to DOING

This keeps always-on context minimal. ~90% of the old content no longer loads on every turn.

### Layer 2: Portable workflow skill

Written to the Agent Skills spec (frontmatter with `name`, `description`, `metadata` only — no harness-specific extensions). The skill loads on demand when the agent is about to do tracked work.

**Directory structure:**
```
<harness>/skills/xpo-workflow/
├── SKILL.md              # 9-step lifecycle procedure
└── references/
    ├── spec-guide.md     # spec template + scaling rules
    └── mcp-tools.md      # tool table with harness-naming notes
```

### Agent detection

`xpo init` detects installed agents via `exec.LookPath()` (cross-platform) and writes to the correct files:

| Binary | Instruction file | Skill directory |
|--------|-----------------|-----------------|
| `claude` | `CLAUDE.md` | `.claude/skills/` |
| `gemini` | `GEMINI.md` | `.gemini/skills/` |
| `cursor` | `.cursorrules` | `.cursor/skills/` |
| `aider` | `CONVENTIONS.md` | (none — full inline docs) |
| (none found) | `AGENTS.md` | (none — full inline docs) |

Agents with a `SkillDir` get the thin stub + skill. Agents without get the full document inline. This ensures every agent gets the workflow regardless of skill support.

## Key design decisions

### Specs as thinking tools (from AiSE kit)

The core insight ported from the AiSE kit: specs exist to collapse the design tree, not to document what the agent already decided. Without a spec, the agent makes probabilistic choices at every branch point. The spec makes those branch points explicit so the user can steer.

Specs scale to the task: a bug fix gets 4 lines (What/Why/How/AC), a feature gets the full template (What/Why/AC/Flow/Decisions/Edge Cases/Assumptions/Open Questions).

### Interactive vs non-interactive mode

In interactive mode (CLI, IDE), the agent surfaces open questions and waits for answers. In non-interactive mode (xpo drive, piped prompts), the agent proceeds but must explicitly document an **Agent Decisions** section in the spec — what it guessed, what it chose, and what to revisit.

### Spec drift prevention

A new step 7 (Review & Spec Update) requires updating the spec before modifying code when the user requests changes during top-hat. This prevents the spec from drifting from the implementation — without it, the only record of corrections is buried in the walkthrough (or lost entirely).

### Walkthrough enforcement

Step 9 makes the walkthrough non-negotiable: "Do NOT transition to DONE without a walkthrough." The walkthrough is the durable record — written after all correction rounds, it captures the final truth.

### Comment vs walkthrough separation

The completion comment says "ready" (brief signal). The walkthrough says "how it works" (durable artifact). The instructions explicitly prohibit duplicating one into the other.

### Harness-neutral MCP tool naming

The skill uses "call `list` on the xpo MCP server" instead of `mcp__xpo__list`. A reference file explains that exact identifiers depend on the harness. This makes the skill portable across Claude Code, Codex, Gemini CLI, etc.

### Status graph fix

Changed from the misleading linear `BACKLOG → PLANNED → DOING → BLOCKED → DONE` to the correct `BACKLOG → PLANNED → DOING → DONE; DOING ⇄ BLOCKED`.

## Files changed

- `internal/exponential/agents.go` — complete rewrite of `GenerateAgentDocs()`, new functions `GenerateAgentStub()`, `GenerateSkillMD()`, `GenerateSpecGuide()`, `GenerateMCPToolsRef()`, `WriteAgentSkill()`. Added `SkillDir` and `Binary` fields to `AgentConfig`. Added `DetectInstalledAgents()`.
- `cmd/exponential/init.go` — init now creates `.mcp.json`, detects installed agents, writes instruction files (stub or full), writes workflow skills, shows config hint. Idempotent on re-init.
- `README.md` — expanded `xpo init` description
- `claude/skills/README.md` — rewritten for auto-install architecture
- `design_docs/xpo-features.md` — updated agent setup description

## Related issues

- **xpo-ecbf49** (DONE) — `xpo init` should create `.mcp.json` and agent instructions
- **xpo-2ba758** (BACKLOG) — `GenerateAgentDocs()` ignores its prefix parameter
- **xpo-cd0f62** (BACKLOG) — drag-and-drop reorder drops parent relationship
- **xpo-2792c3** (BACKLOG) — MCP show tool does not reflect backlog sort order
