package exponential

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// AgentConfig defines an AI agent and its instruction file location
type AgentConfig struct {
	Name     string // Agent name (e.g., "Cursor", "GitHub Copilot")
	File     string // Primary file path (e.g., ".cursorrules")
	Binary   string // CLI binary name for PATH detection (empty = not detectable via PATH)
	SkillDir string // Skill directory for this harness (empty = no skill support)
	Format   string // "markdown" or "plain"
	NeedsDir bool   // If true, parent directory must be created
}

// AgentRegistry contains all supported AI agent configurations
var AgentRegistry = []AgentConfig{
	{Name: "Generic Agent", File: "AGENTS.md", Format: "markdown"},
	{Name: "Claude Code", File: "CLAUDE.md", Binary: "claude", SkillDir: ".claude/skills", Format: "markdown"},
	{Name: "Gemini", File: "GEMINI.md", Binary: "gemini", SkillDir: ".gemini/skills", Format: "markdown"},
	{Name: "Cursor", File: ".cursorrules", Binary: "cursor", SkillDir: ".cursor/skills", Format: "plain"},
	{Name: "Windsurf", File: ".windsurfrules", Binary: "windsurf", Format: "plain"},
	{Name: "GitHub Copilot", File: ".github/copilot-instructions.md", Format: "markdown", NeedsDir: true},
	{Name: "Cline", File: ".clinerules", Format: "markdown"},
	{Name: "Roo Code", File: ".roorules", Format: "markdown"},
	{Name: "Aider", File: "CONVENTIONS.md", Binary: "aider", Format: "markdown"},
	{Name: "Continue", File: ".continue/rule./xpo.md", Format: "markdown", NeedsDir: true},
}

// AgentDetectionResult contains information about a detected agent file
type AgentDetectionResult struct {
	Agent                AgentConfig
	Exists               bool
	HasExponentialConfig bool
}

// DetectInstalledAgents returns agents whose CLI binary is found on PATH.
// Falls back to the Generic Agent if no specific agent is detected.
func DetectInstalledAgents() []AgentConfig {
	var detected []AgentConfig
	for _, agent := range AgentRegistry {
		if agent.Binary == "" {
			continue
		}
		if _, err := exec.LookPath(agent.Binary); err == nil {
			detected = append(detected, agent)
		}
	}
	if len(detected) == 0 {
		detected = append(detected, AgentRegistry[0]) // Generic Agent fallback
	}
	return detected
}

// DetectAgentFiles scans for existing agent instruction files
func DetectAgentFiles() []AgentDetectionResult {
	var results []AgentDetectionResult

	for _, agent := range AgentRegistry {
		result := AgentDetectionResult{Agent: agent}

		content, err := os.ReadFile(agent.File)
		if err == nil {
			result.Exists = true
			result.HasExponentialConfig = strings.Contains(string(content), "# Exponential Agent Instructions")
		}

		results = append(results, result)
	}

	return results
}

// GenerateAgentStub generates the thin always-on stub for agent instruction files.
// This contains only hard invariants and a directive to load the xpo-workflow skill.
func GenerateAgentStub(prefix string) string {
	_ = prefix
	return `# Exponential Agent Instructions

This project uses ` + "`xpo`" + ` (Exponential) via the MCP server registered in ` + "`.mcp.json`" + `.
Always use the MCP tools — never shell out to the ` + "`xpo`" + ` CLI.

## Hard Rules

1. Every code change must be backed by an xpo issue transitioned to DOING before any file is modified.
2. Never start work on a BACKLOG issue without explicit user approval to transition it.
3. Bugs discovered during implementation may be filed and fixed without approval — file the issue, link it to the current work, and fix it.
4. Before beginning any implementation task, load the ` + "`xpo-workflow`" + ` skill and follow it.
5. If an MCP tool call fails, report the error to the user. Never fall back to the CLI.

## Agent Identity

Set the ` + "`assignee`" + ` field to yourself when transitioning an issue to DOING. Use the form
` + "`<Agent Name> <agent@<host>.local>`" + ` — e.g. ` + "`Claude Code <agent@macbook.local>`" + `.
`
}

// GenerateAgentDocs generates the full xpo agent documentation for agents without skill support.
// This is the complete document combining stub + workflow + references inline.
func GenerateAgentDocs(prefix string) string {
	return GenerateAgentStub(prefix) + `
## Workflow

The full xpo development workflow follows these steps. See the ` + "`xpo-workflow`" + ` skill for
details if your agent harness supports skills.

` + generateWorkflowBody() + `
` + generateMCPToolsBody()
}

// GenerateSkillMD generates the SKILL.md content for the xpo-workflow skill.
func GenerateSkillMD() string {
	return `---
name: xpo-workflow
description: >-
  Issue-tracking workflow for this repository using xpo (Exponential).
  Use before planning, implementing, fixing, or completing any code
  change; when creating or updating issues, specs, or walkthroughs;
  or when the user mentions xpo, issues, epics, or the board.
metadata:
  author: exponential
  version: "1.0"
---

# xpo Development Workflow

` + generateWorkflowBody()
}

// GenerateSpecGuide generates the spec-guide.md reference content.
func GenerateSpecGuide() string {
	return `# Spec Writing Guide

## Why specs exist

Without a spec, the agent rolls the dice at every design decision — picking the most probable
path from its training data, not the right one for this project. The spec collapses that space:
it makes branch points explicit so the user can steer instead of discovering unwanted choices
after the code is written. It also documents what will change, why, and how, so the user can
evaluate the proposed plan and course-correct before implementation begins.

A spec that merely restates the issue title has failed.

## Scaling the spec to the task

### Bug fix or small change (brief spec)

- What is the problem / what needs to change
- Why (root cause / motivation)
- How (the fix approach, noting any alternatives considered)
- Acceptance criteria (how to verify it worked)

### Feature or complex change (full spec)

- **What** — 2-3 sentence concrete description of what this delivers
- **Why** — motivation, link to parent epic/context
- **Acceptance Criteria** — observable, testable outcomes
- **Flow** — numbered implementation steps; name concrete files, functions, endpoints
- **Decisions** — choices made during design: "X, not Y" with rationale and alternatives
- **Edge Cases** — classified by risk:
  - HIGH (developer decides) — wrong default causes data loss, security holes, or broken UX
  - MEDIUM (agent proposes handling, developer confirms)
  - LOW (agent handles silently)
- **Assumptions** — what you are taking for granted; the user can correct these
- **Open Questions** — unresolved branch points needing answers before implementation
- **Agent Decisions** (non-interactive mode only) — choices the agent made on behalf of the user
  where the answer was not obvious. For each: the question, the choice made, the reasoning, and
  what to revisit if the choice was wrong. This section is the user's entry point for catching
  assumptions they would not have made.

## Key principles

- Draft concrete content. Never present blank templates or ask the user to fill in sections.
- Surface your assumptions explicitly — it is always easier for the user to react to a draft
  than to author from scratch.
- When you encounter open questions that would significantly change the implementation,
  stop and ask the user before proceeding.
- The spec is the source of truth for implementation. If you later discover it is wrong or
  incomplete, update the spec first, then change code.
`
}

// GenerateMCPToolsRef generates the mcp-tools.md reference content.
func GenerateMCPToolsRef() string {
	return generateMCPToolsBody()
}

func generateWorkflowBody() string {
	return `### 1. Discover

- Call ` + "`list`" + ` on the xpo MCP server to see the board; use the ` + "`match`" + ` parameter for free-text search.
- Call ` + "`show`" + ` for full details on candidate issues.
- Before creating a new issue, always search to ensure no existing issue covers the work.

### 2. Plan

- **Features/design work:** propose the idea to the user. Only create an issue after the user approves.
- **Bugs found during implementation:** file immediately, link back to the originating issue. No approval needed.
- Always set at least one label (` + "`bug`" + `, ` + "`feature`" + `, ` + "`epic`" + `, or something more specific).
- Keep titles under 100 characters.
- Descriptions are Markdown. Use headings, lists, code blocks, bold/italic. Write literal newlines, not ` + "`\\n`" + ` escape sequences.
- When a new issue belongs to an epic, set ` + "`parent`" + ` to the epic's ID.

### 3. Spec (Think Before You Build)

Before implementing, ensure the issue has an up-to-date spec. If none exists, write one using the
xpo MCP server's ` + "`spec`" + ` tool. See ` + "`references/spec-guide.md`" + ` for the full spec template and scaling rules.

The spec is a thinking tool — it collapses the design space so the user can steer decisions
instead of discovering unwanted choices after the code is written. Scale the depth to the task:
a bug fix gets a brief What/Why/How/AC spec; a feature gets the full template.

**Interactive mode** (user present — CLI, IDE plugin): after writing the spec, surface open questions
and uncertainties to the user and wait for answers before implementing. The spec is a conversation
artifact — the user should confirm or redirect before code is written.

**Non-interactive mode** (headless — xpo drive, piped prompts): implement from the spec without
waiting, BUT the spec must explicitly mark where you made judgment calls. For every open question,
document the question, the answer you chose, and your reasoning. Use a dedicated section:

- **Agent Decisions** — choices made on behalf of the user where the answer was not obvious.
  Format each as: the question, the choice made, the reasoning, and what to revisit if the
  choice was wrong.

### 4. Start

- Only issues with PLANNED status are eligible for work.
- If the issue you want to work on is in BACKLOG, ask the user if it is okay to transition it. Do not start without approval.
- Check the issue's dependencies. If any ` + "`depends_on`" + ` or ` + "`blocked_by`" + ` targets are not DONE, stop and ask the user how to proceed.
- If the issue is already in DOING and assigned to someone else, stop and ask the user before taking it over.
- Transition to DOING and set ` + "`assignee`" + ` to yourself **before touching any file**.

### 5. Implement & Test

- Follow the spec. The flow steps, decisions, and edge cases are your requirements.
- If you encounter a decision not covered by the spec, make a reasonable choice and note it in the completion comment. If the decision is significant (would surprise the user or constrain future work), stop and ask.
- If you realize the spec has a significant gap or is wrong, stop. Explain the issue, propose a spec update, and wait for approval.
- Use build commands appropriate to the language, framework, or technology of this project.
- Use test commands that ideally execute lint, style checks, and unit tests appropriate for the language, framework, or technology of this project.
- Use the ` + "`artifact`" + ` tool for supplemental files (test outputs, design diagrams, logs).

### 6. Completion Comment

When implementation is done (evidenced by passing tests), add a brief comment:

- **Summary** — what changed (use backtick code spans for file/function names)
- **Rationale** — why this approach
- **Decisions made** — any choices not in the original spec
- **Status** — tests passing, ready for review

The comment says "ready." The walkthrough says "how it works." Do not duplicate one into the other.

### 7. Review & Spec Update

When the user top-hats and requests changes, update the spec to reflect the corrections **before**
modifying code. The spec must always match the final implementation. If the user's feedback changes
the approach, acceptance criteria, or decisions, those changes belong in the spec — not just in the
code diff. Without this step, the spec drifts from reality and future agents reading it will build
the wrong thing.

### 8. Walkthrough (After User Review)

After the user reviews and approves (including any correction rounds), write a walkthrough using the
xpo MCP server's ` + "`walkthrough`" + ` tool.

The walkthrough is the **durable implementation record** — written from the perspective of a senior
engineer explaining the changes to a junior developer:

- What was built and why
- How the pieces fit together
- Key decisions and their rationale (including any that emerged during review)
- Anything non-obvious that a future reader would need to understand

Write the walkthrough **after** any user-requested corrections are applied, so it reflects the final state.

### 9. Complete

Do NOT transition to DONE without a walkthrough. If the user asks to close the issue, write the
walkthrough first — it is the only durable record of what was built and why. Without it, all
context about the implementation is lost when the conversation ends.

Transition to DONE only after: the user approves, tests pass, and the walkthrough is attached.

## Status Graph

` + "`BACKLOG → PLANNED → DOING → DONE; DOING ⇄ BLOCKED`" + `

BLOCKED is not a step on the way to DONE — it is a side-state an issue can enter and leave.

## Prioritization

When choosing what to work on next (and dependency links do not resolve the order):

1. **Blockers** — issues blocking other work
2. **Bugs** — correctness problems in existing functionality
3. **Planned features** — by dependency order, then by story points (smaller first)

## Linking

Use the ` + "`link`" + ` tool to express relationships. Supported types: ` + "`blocks`" + `, ` + "`blocked_by`" + `,
` + "`depends_on`" + `, ` + "`dependency_of`" + `, ` + "`duplicates`" + `, ` + "`duplicated_by`" + `, ` + "`relates_to`" + `.
When a task spawns follow-up work, link the new issue back to the originating one.

## Issue Creation

- Always set at least one label.
- Descriptions and comments render as Markdown. Write literal newlines, not ` + "`\\n`" + ` escape sequences.
- Markdown checklists (` + "`- [ ] Title`" + `) can sub-divide task steps.
`
}

func generateMCPToolsBody() string {
	return `## MCP Tools Reference

The xpo MCP server exposes the tools listed below. Exact tool identifiers depend on your
agent harness (e.g. Claude Code surfaces them as ` + "`mcp__xpo__<tool>`" + `; other harnesses
may use different naming conventions). The table uses the base tool names.

| Tool | Description |
|---|---|
| ` + "`list`" + ` | List or search issues; use ` + "`match`" + ` for free-text search |
| ` + "`show`" + ` | Read one issue with details, dependencies, and comments |
| ` + "`add`" + ` | Create a new issue |
| ` + "`update`" + ` | Update fields including status transitions (BACKLOG/PLANNED/DOING/BLOCKED/DONE) |
| ` + "`comment`" + ` | Add a markdown comment to an issue |
| ` + "`link`" + ` | Add a relationship between two issues |
| ` + "`history`" + ` | View the audit trail for an issue |
| ` + "`spec`" + ` | Read, write, or delete the design spec for an issue |
| ` + "`walkthrough`" + ` | Read, write, or delete the implementation walkthrough for an issue |
| ` + "`artifact`" + ` | Manage generic artifacts attached to an issue |
`
}

// WriteAgentSkill writes the xpo-workflow skill to the agent's skill directory.
// Returns the skill directory path, or empty string if the agent has no skill support.
func WriteAgentSkill(agent AgentConfig) (string, error) {
	if agent.SkillDir == "" {
		return "", nil
	}

	skillDir := filepath.Join(agent.SkillDir, "xpo-workflow")
	refsDir := filepath.Join(skillDir, "references")
	if err := os.MkdirAll(refsDir, 0755); err != nil {
		return "", fmt.Errorf("could not create skill directory %s: %w", refsDir, err)
	}

	files := map[string]string{
		filepath.Join(skillDir, "SKILL.md"):              GenerateSkillMD(),
		filepath.Join(refsDir, "spec-guide.md"):          GenerateSpecGuide(),
		filepath.Join(refsDir, "mcp-tools.md"):           GenerateMCPToolsRef(),
	}

	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return "", fmt.Errorf("could not write %s: %w", path, err)
		}
	}

	return skillDir, nil
}

// MCPConfigStatus describes the state of .mcp.json in the project root.
type MCPConfigStatus struct {
	Exists         bool
	HasExponential bool
}

const mcpConfigFile = ".mcp.json"

// DetectMCPConfig checks whether .mcp.json exists and contains a xpo entry.
func DetectMCPConfig() MCPConfigStatus {
	data, err := os.ReadFile(mcpConfigFile)
	if err != nil {
		return MCPConfigStatus{}
	}
	var doc map[string]json.RawMessage
	if json.Unmarshal(data, &doc) != nil {
		return MCPConfigStatus{Exists: true}
	}
	servers, ok := doc["mcpServers"]
	if !ok {
		return MCPConfigStatus{Exists: true}
	}
	var serversMap map[string]json.RawMessage
	if json.Unmarshal(servers, &serversMap) != nil {
		return MCPConfigStatus{Exists: true}
	}
	_, hasExponential := serversMap["xpo"]
	return MCPConfigStatus{Exists: true, HasExponential: hasExponential}
}

// EnsureMCPConfig creates or updates .mcp.json to include a xpo MCP server entry.
func EnsureMCPConfig() error {
	var doc map[string]interface{}

	data, err := os.ReadFile(mcpConfigFile)
	if err == nil {
		if json.Unmarshal(data, &doc) != nil {
			return fmt.Errorf("could not parse %s: invalid JSON", mcpConfigFile)
		}
	} else if os.IsNotExist(err) {
		doc = make(map[string]interface{})
	} else {
		return fmt.Errorf("could not read %s: %w", mcpConfigFile, err)
	}

	servers, _ := doc["mcpServers"].(map[string]interface{})
	if servers == nil {
		servers = make(map[string]interface{})
	}
	servers["xpo"] = map[string]interface{}{
		"command": "xpo",
		"args":    []string{"mcp"},
	}
	doc["mcpServers"] = servers

	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("could not marshal %s: %w", mcpConfigFile, err)
	}
	out = append(out, '\n')
	return os.WriteFile(mcpConfigFile, out, 0644)
}

// AppendAgentInstructions appends xpo instructions to an agent file.
// For agents with skill support, writes the thin stub; otherwise writes the full docs.
func AppendAgentInstructions(agent AgentConfig, prefix string) error {
	var xpoSection string
	if agent.SkillDir != "" {
		xpoSection = GenerateAgentStub(prefix)
	} else {
		xpoSection = GenerateAgentDocs(prefix)
	}

	if agent.NeedsDir {
		dir := filepath.Dir(agent.File)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("could not create directory %s: %w", dir, err)
		}
	}

	content, err := os.ReadFile(agent.File)
	if err == nil {
		f, err := os.OpenFile(agent.File, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("could not open %s: %w", agent.File, err)
		}
		defer f.Close()

		if !strings.HasSuffix(string(content), "\n") {
			f.WriteString("\n")
		}
		f.WriteString("\n" + xpoSection)
	} else if os.IsNotExist(err) {
		if err := os.WriteFile(agent.File, []byte(xpoSection), 0644); err != nil {
			return fmt.Errorf("could not create %s: %w", agent.File, err)
		}
	} else {
		return fmt.Errorf("could not read %s: %w", agent.File, err)
	}

	return nil
}
