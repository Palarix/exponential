package exponential

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/palarix/exponential/internal/version"
)

// MCPConfigSpec defines how a harness expects its MCP server configuration.
type MCPConfigSpec struct {
	File       string // config file path relative to project root (e.g. ".mcp.json")
	ServerKey  string // key for the server map (e.g. "mcpServers", "mcp", "mcp_servers")
	Format     string // "json" or "toml"
	EntryStyle string // "standard" (default): {command, args}; "local-array": {type: "local", command: [binary, args...]}
	NeedsDir   bool   // create parent directory if missing
}

// HasMCPConfig returns true if the spec defines an MCP config location.
func (s MCPConfigSpec) HasMCPConfig() bool {
	return s.File != ""
}

// AgentConfig defines an AI agent and its instruction file location
type AgentConfig struct {
	Name           string        // Agent name (e.g., "Cursor", "GitHub Copilot")
	File           string        // Primary file path (e.g., ".cursorrules")
	Binary         string        // CLI binary name for PATH detection (empty = not detectable via PATH)
	SkillDir       string        // Project-local skill directory (empty = no skill support)
	GlobalSkillDir string        // User-level skill directory relative to $HOME (empty = no global support)
	Format         string        // "markdown" or "plain"
	NeedsDir       bool          // If true, parent directory must be created
	MCPConfig      MCPConfigSpec // How this harness expects MCP config (zero value = no MCP support)
}

// AgentRegistry contains all supported AI agent configurations
var AgentRegistry = []AgentConfig{
	{Name: "Generic Agent", File: "AGENTS.md", Format: "markdown"},
	{Name: "Claude Code", File: "CLAUDE.md", Binary: "claude", SkillDir: ".claude/skills", GlobalSkillDir: ".claude/skills", Format: "markdown", MCPConfig: MCPConfigSpec{File: ".mcp.json", ServerKey: "mcpServers", Format: "json"}},
	{Name: "GitHub Copilot", File: ".github/copilot-instructions.md", Format: "markdown", NeedsDir: true},
	{Name: "Cursor", File: ".cursorrules", Binary: "cursor", SkillDir: ".cursor/skills", GlobalSkillDir: ".cursor/skills", Format: "plain", MCPConfig: MCPConfigSpec{File: ".cursor/mcp.json", ServerKey: "mcpServers", Format: "json", NeedsDir: true}},
	{Name: "Codex", File: "AGENTS.md", Binary: "codex", SkillDir: ".codex/skills", GlobalSkillDir: ".codex/skills", Format: "markdown", MCPConfig: MCPConfigSpec{File: ".codex/config.toml", ServerKey: "mcp_servers", Format: "toml", NeedsDir: true}},
	{Name: "OpenCode", File: "AGENTS.md", Binary: "opencode", SkillDir: ".opencode/skills", GlobalSkillDir: ".config/opencode/skills", Format: "markdown", MCPConfig: MCPConfigSpec{File: "opencode.json", ServerKey: "mcp", Format: "json", EntryStyle: "local-array"}},
}

// AgentDetectionResult contains information about a detected agent file
type AgentDetectionResult struct {
	Agent                AgentConfig
	Exists               bool
	HasExponentialConfig bool
}

var legacyHeadings = []string{
	"## Exponential (xpo)",
	"# Agent Instructions",
	"# Exponential Agent Instructions",
}

// HasAgentInstructions checks whether content contains an xpo instructions section
// (current or any legacy heading).
func HasAgentInstructions(content string) bool {
	return findAgentInstructionsOffset(content) != -1
}

func findAgentInstructionsOffset(content string) int {
	for _, heading := range legacyHeadings {
		if idx := strings.Index(content, heading); idx != -1 {
			return idx
		}
	}
	return -1
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
			result.HasExponentialConfig = HasAgentInstructions(string(content))
		}

		results = append(results, result)
	}

	return results
}

// md turns ¦ into backticks, so templates can be written as raw strings.
func md(s string) string {
	return strings.ReplaceAll(s, "¦", "`")
}

// GenerateAgentStub generates the thin always-on stub for agent instruction
// files. It holds only the rules that are true for every workflow, and points
// to the project's workflow skill if there is one.
func GenerateAgentStub(prefix string) string {
	return agentStubHead(prefix) + md(`
If the project has an ¦xpo-workflow¦ skill, load it before you plan, file or change anything, and follow it.
For how the xpo tools work, load the ¦xpo¦ skill.
`)
}

func agentStubHead(prefix string) string {
	return md(`## Exponential (xpo)

This project uses ¦xpo¦ (Exponential) via the MCP server registered in ¦.mcp.json¦.
Always use the MCP tools — never shell out to the ¦xpo¦ CLI. If an MCP tool call fails,
report the error to the user. Never fall back to the CLI.

Issue IDs in this project use the prefix ¦` + prefix + `¦ (e.g. ¦` + prefix + `-a1b2c3¦).
`)
}

// GenerateAgentDocs generates the inline xpo reference for agents without
// skill support. The workflow is not part of it: xpo init writes that once,
// below the managed block.
func GenerateAgentDocs(prefix string) string {
	return agentStubHead(prefix) + `
If this file has a Workflow section, follow it.

` + generateReferenceBody("###") + `
` + generateMCPToolsBody()
}

// GenerateSkillMD generates the SKILL.md content for the xpo reference skill.
func GenerateSkillMD() string {
	return md(`---
name: xpo
description: >-
  How the xpo (Exponential) MCP tools work: statuses, labels, links,
  milestones, specs, walkthroughs, and starting and merging work. Use
  when you create, update, start or merge issues, write specs or
  walkthroughs, or when the user mentions xpo, issues, epics,
  milestones, or the board.
---

# xpo Reference

How the xpo tools behave: statuses, labels, links, milestones, specs, walkthroughs, start and merge.
When to use them is the project's call: if the project has an ¦xpo-workflow¦ skill, load it and follow it.

`) + generateReferenceBody("##")
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
- **Agent Decisions** (when no user is present to answer) — choices the agent made on behalf of
  the user where the answer was not obvious. For each: the question, the choice made, the
  reasoning, and what to revisit if the choice was wrong. This section is the user's entry point
  for catching assumptions they would not have made.

## Key principles

- Draft concrete content. Never present blank templates or ask the user to fill in sections.
- Surface your assumptions explicitly — it is always easier for the user to react to a draft
  than to author from scratch.
- Open questions that would significantly change the implementation need an answer before
  implementing: ask the user, or record them as Agent Decisions when no user is present.
- The spec is the source of truth for implementation, so it must match the final code.
`
}

// GenerateMCPToolsRef generates the mcp-tools.md reference content.
func GenerateMCPToolsRef() string {
	return generateMCPToolsBody()
}

// generateReferenceBody is how xpo works, shared by the reference skill and
// the inline docs. It holds no workflow: no steps, gates or stops. level is
// the heading prefix for its sections ("##" in the skill, "###" inline).
func generateReferenceBody(level string) string {
	return strings.ReplaceAll(md(`@@ Status Graph

¦BACKLOG → PLANNED → DOING → DONE; DOING ⇄ BLOCKED; any → CANCELED; any → DUPLICATE¦

| Status      | Meaning                                           |
| ----------- | ------------------------------------------------- |
| ¦BACKLOG¦   | Captured but not yet prioritized                  |
| ¦PLANNED¦   | Prioritized and ready to be worked on             |
| ¦DOING¦     | Actively being worked on                          |
| ¦BLOCKED¦   | Waiting on something external                     |
| ¦DONE¦      | Completed                                         |
| ¦CANCELED¦  | Will not be done (obsolete, out of scope, etc.)   |
| ¦DUPLICATE¦ | Duplicate of another issue — link to the original |

@@ Finding Work

¦list¦ shows issues: ¦match¦ searches ids, titles, statuses, labels and assignees, and ¦milestone¦
narrows the list to one milestone. ¦show¦ gives one issue's description, links and comments, and
¦history¦ its events. ¦rationale¦ searches the specs and walkthroughs of earlier work for design
decisions, and ¦code_history¦ finds the issues that touched a file or lines.
Full tool list: ¦references/mcp-tools.md¦.

@@ Issue Creation

- Always set a primary label (see **Issue Labels**). Add secondary labels for area/domain when useful.
- Descriptions and comments render as Markdown. Write literal newlines, not ¦\n¦ escape sequences.
- Markdown checklists (¦- [ ] Title¦) can sub-divide task steps.
- Keep titles under 100 characters.
- When a new issue belongs to an epic, set ¦parent¦ to the epic's ID.

@@ Issue Labels

Every issue MUST have exactly one **primary label** that classifies the type of work:

| Primary label | When to use                                         |
| ------------- | --------------------------------------------------- |
| ¦bug¦         | Something is broken or behaving incorrectly         |
| ¦feature¦     | A new capability that doesn't exist yet             |
| ¦task¦        | A concrete piece of work (refactor, migration, etc) |
| ¦epic¦        | A high-level goal that will be broken into subtasks |

An issue MAY also carry **secondary labels** for area or domain (e.g. ¦backend¦, ¦frontend¦, ¦ux¦, ¦infra¦).
Keep the set small — reuse existing labels before inventing new ones.

@@ Linking

Use the ¦link¦ tool to express relationships. Supported types: ¦blocks¦, ¦blocked_by¦,
¦depends_on¦, ¦dependency_of¦, ¦duplicates¦, ¦duplicated_by¦, ¦relates_to¦.
Links are bidirectional: ¦A blocks B¦ also shows on B as ¦blocked_by A¦. Two issues can have at
most one link, so pick the most specific kind. Linking an already-linked pair fails; pass
¦overwrite: true¦ to replace the existing link. ¦unlink¦ removes a link, whichever side stores it.

Moving an issue to DOING is refused while an issue that blocks it is not DONE, whichever side
stores the link. xpo treats the other kinds as informational, but a workflow may gate on them too
(for example on ¦depends_on¦).

@@ Milestones

A milestone is a scope-bounded group of issues that are done together, numbered ¦#1¦, ¦#2¦, …
with an optional label. Current and Next are the first two open milestones, in order. The
¦milestone¦ tool lists, creates, updates, reorders, completes and deletes them; to complete a
milestone moves its open issues to the next one. To put an issue in a milestone, set ¦milestone¦
on ¦add¦ or ¦update¦: a number (¦#7¦), a label, ¦@current¦, ¦@next¦, or ¦none¦. A new child issue
copies its parent's open milestone.

@@ Specs

A spec is the design of an issue, written before the code with the ¦spec¦ tool. See
¦references/spec-guide.md¦ for the template and how to scale it to the task. ¦rationale¦ finds the
specs and walkthroughs of earlier work in the same area.

@@ Walkthroughs

A walkthrough is the durable record of what was built, written after the implementation with the
¦walkthrough¦ tool. Write it as a senior engineer explaining the changes to a junior developer:

- What was built and why
- How the pieces fit together
- Key decisions and their rationale (including any that emerged during review)
- Anything non-obvious that a future reader would need to understand

Write it from the final state of the code.

@@ Starting Work

¦start¦ transitions an issue to DOING, assigns it to the person you are working for, and creates an
isolated git worktree on a new branch for it. Never set ¦assignee¦ to yourself — xpo records that
an agent did the work. ¦start¦ refuses issues in BACKLOG or BLOCKED, and issues that are blocked
(see **Linking**). ¦force: true¦ takes over an issue someone else already has in DOING.

All file reads, edits, builds, and test runs must happen inside the worktree path returned by
¦start¦ — not the main checkout. If you are already working in the issue's worktree, don't call
¦start¦ again.

@@ Merging

Commit everything in the issue's worktree first: ¦git add -A && git commit -m "<issue-id>: <issue title>"¦.
Then call the ¦merge¦ tool without a ¦strategy¦ (the user may ask for one). It squash-merges the
branch into the default branch, records a MERGE event, closes the issue, and removes the worktree and
branch. It refuses to run while the worktree has uncommitted or untracked files.

If it fails because the default branch moved on (conflicts), bring the default branch into the
issue's branch from the worktree (¦git rebase <default-branch>¦, or ¦git merge <default-branch>¦),
resolve any conflicts, re-run the tests, commit, and call ¦merge¦ again.

Committing on the issue's worktree is your job. Merging is ¦merge¦'s job: never run ¦git push¦,
or merge, rebase or commit on the default branch yourself. Rebasing the issue's branch onto the
default branch (or merging the default branch into it) from the worktree is allowed.
`), "@@ ", level+" ")
}

func generateMCPToolsBody() string {
	return `## MCP Tools Reference

The xpo MCP server exposes the tools listed below. Exact tool identifiers depend on your
agent harness (e.g. Claude Code surfaces them as ` + "`mcp__xpo__<tool>`" + `; other harnesses
may use different naming conventions). The table uses the base tool names.

| Tool | Description |
|---|---|
| ` + "`list`" + ` | List or search issues; use ` + "`match`" + ` for free-text search and ` + "`milestone`" + ` to filter by milestone |
| ` + "`show`" + ` | Read one issue with details, dependencies, and comments |
| ` + "`add`" + ` | Create a new issue. Set ` + "`before`" + ` or ` + "`after`" + ` to an issue ID to place it directly above or below that issue |
| ` + "`update`" + ` | Update fields including status transitions (BACKLOG/PLANNED/DOING/BLOCKED/DONE/CANCELED/DUPLICATE) and milestone. Reorder with ` + "`before`" + ` or ` + "`after`" + ` set to an issue ID |
| ` + "`start`" + ` | Start working on an issue: transitions to DOING and creates a git worktree. Returns the worktree path |
| ` + "`merge`" + ` | Squash-merge an issue branch into the default branch, record a MERGE event, close the issue, and clean up the worktree. All changes must be committed in the worktree first. Call it without a ` + "`strategy`" + ` unless the user asks for one |
| ` + "`comment`" + ` | Add a markdown comment to an issue |
| ` + "`link`" + ` | Add a relationship between two issues |
| ` + "`unlink`" + ` | Remove one relationship between two issues, whichever side stores it |
| ` + "`history`" + ` | View the audit trail for an issue |
| ` + "`milestone`" + ` | List, create, update, reorder, complete or delete milestones |
| ` + "`spec`" + ` | Read, write, or delete the design spec for an issue |
| ` + "`walkthrough`" + ` | Read, write, or delete the implementation walkthrough for an issue |
| ` + "`rationale`" + ` | Search across specs and walkthroughs for prior design decisions related to a topic |
| ` + "`artifact`" + ` | Manage generic artifacts attached to an issue |
| ` + "`git_status`" + ` | Show an issue branch's ahead/behind state and its worktree's uncommitted files (read-only) |
| ` + "`git_diff`" + ` | Show what an issue's branch changed, including uncommitted changes (read-only) |
| ` + "`git_log`" + ` | List the commits of an issue's branch or of the default branch (read-only) |
| ` + "`git_show`" + ` | Show one commit's metadata, message and patch (read-only) |
| ` + "`code_history`" + ` | Find the commits and issues that touched a file or a line range (read-only) |
`
}

// GlobalSkillCanonicalDir is the canonical location for globally installed skills.
const GlobalSkillCanonicalDir = ".config/xpo/skills"

// WriteAgentSkill writes the xpo skill to the agent's skill directory.
// When globalBaseDir is non-empty, files are written to that canonical location
// and a symlink is created from the agent's global skill directory.
// Files with local edits are left alone and reported in the result unless
// force is set. Result.Dir is empty if the agent has no skill support.
func WriteAgentSkill(agent AgentConfig, globalBaseDir string, force bool) (SkillWriteResult, error) {
	if agent.SkillDir == "" {
		return SkillWriteResult{}, nil
	}

	var skillDir string
	if globalBaseDir != "" {
		skillDir = filepath.Join(globalBaseDir, "xpo")
	} else {
		skillDir = filepath.Join(agent.SkillDir, "xpo")
	}
	result := SkillWriteResult{Dir: skillDir}

	refsDir := filepath.Join(skillDir, "references")
	if err := os.MkdirAll(refsDir, 0755); err != nil {
		return result, fmt.Errorf("could not create skill directory %s: %w", refsDir, err)
	}

	for _, f := range generatedSkillFiles(skillDir) {
		if !force && CheckSkillFile(f.Path, f.Content) == SkillFileEdited {
			data, _ := os.ReadFile(f.Path)
			body, _, _, _ := ParseSkillFile(string(data))
			result.Edited = append(result.Edited, EditedSkillFile{
				Path:       f.Path,
				OldContent: strings.TrimSpace(body),
				NewContent: strings.TrimSpace(f.Content),
			})
			continue
		}
		if err := os.WriteFile(f.Path, []byte(WrapSkillFile(f.Content, version.CLIVersion)), 0644); err != nil {
			return result, fmt.Errorf("could not write %s: %w", f.Path, err)
		}
	}

	if globalBaseDir != "" && agent.GlobalSkillDir != "" {
		if err := createSkillSymlink(agent, skillDir); err != nil {
			return result, fmt.Errorf("skill files written but symlink failed: %w", err)
		}
	}

	return result, nil
}

// createSkillSymlink creates a symlink from the agent's global skill directory to the canonical location.
func createSkillSymlink(agent AgentConfig, canonicalDir string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("could not determine home directory: %w", err)
	}

	symlinkParent := filepath.Join(home, agent.GlobalSkillDir)
	if err := os.MkdirAll(symlinkParent, 0755); err != nil {
		return fmt.Errorf("could not create directory %s: %w", symlinkParent, err)
	}

	symlinkPath := filepath.Join(symlinkParent, "xpo")

	if info, err := os.Lstat(symlinkPath); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			os.Remove(symlinkPath)
		} else {
			return fmt.Errorf("%s exists and is not a symlink", symlinkPath)
		}
	}

	return os.Symlink(canonicalDir, symlinkPath)
}

// SkillInstallStatus describes how a skill is installed for a harness.
type SkillInstallStatus struct {
	Local         bool
	Global        bool
	BrokenSymlink bool
}

// DetectSkillInstall checks whether the xpo skill is installed for an agent.
func DetectSkillInstall(agent AgentConfig) SkillInstallStatus {
	var status SkillInstallStatus

	if agent.SkillDir != "" {
		skillPath := filepath.Join(agent.SkillDir, "xpo", "SKILL.md")
		if _, err := os.Stat(skillPath); err == nil {
			status.Local = true
		}
	}

	if agent.GlobalSkillDir != "" {
		home, err := os.UserHomeDir()
		if err == nil {
			symlinkPath := filepath.Join(home, agent.GlobalSkillDir, "xpo")
			info, err := os.Lstat(symlinkPath)
			if err == nil && info.Mode()&os.ModeSymlink != 0 {
				if _, statErr := os.Stat(symlinkPath); statErr != nil {
					status.BrokenSymlink = true
				} else {
					status.Global = true
				}
			} else if err == nil {
				canonicalSkill := filepath.Join(symlinkPath, "SKILL.md")
				if _, statErr := os.Stat(canonicalSkill); statErr == nil {
					status.Global = true
				}
			}
		}
	}

	return status
}

// MCPConfigStatus describes the state of an MCP config file.
type MCPConfigStatus struct {
	Exists         bool
	HasExponential bool
}

// DefaultMCPSpec is the MCPConfigSpec for Claude Code (.mcp.json with mcpServers key).
var DefaultMCPSpec = MCPConfigSpec{File: ".mcp.json", ServerKey: "mcpServers", Format: "json"}

// DetectMCPConfig checks whether .mcp.json exists and contains a xpo entry (Claude Code default).
func DetectMCPConfig() MCPConfigStatus {
	return DetectMCPConfigFor(DefaultMCPSpec)
}

// DetectMCPConfigFor checks whether the MCP config file for the given spec exists and contains a xpo entry.
func DetectMCPConfigFor(spec MCPConfigSpec) MCPConfigStatus {
	if spec.File == "" {
		return MCPConfigStatus{}
	}

	data, err := os.ReadFile(spec.File)
	if err != nil {
		return MCPConfigStatus{}
	}

	if spec.Format == "toml" {
		return detectMCPConfigTOML(data, spec.ServerKey)
	}
	return detectMCPConfigJSON(data, spec.ServerKey)
}

func detectMCPConfigJSON(data []byte, serverKey string) MCPConfigStatus {
	var doc map[string]json.RawMessage
	if json.Unmarshal(data, &doc) != nil {
		return MCPConfigStatus{Exists: true}
	}
	servers, ok := doc[serverKey]
	if !ok {
		return MCPConfigStatus{Exists: true}
	}
	var serversMap map[string]json.RawMessage
	if json.Unmarshal(servers, &serversMap) != nil {
		return MCPConfigStatus{Exists: true}
	}
	_, hasXpo := serversMap["xpo"]
	return MCPConfigStatus{Exists: true, HasExponential: hasXpo}
}

func detectMCPConfigTOML(data []byte, serverKey string) MCPConfigStatus {
	content := string(data)
	sectionHeader := fmt.Sprintf("[%s.xpo]", serverKey)
	return MCPConfigStatus{
		Exists:         true,
		HasExponential: strings.Contains(content, sectionHeader),
	}
}

// EnsureMCPConfig creates or updates .mcp.json to include a xpo MCP server entry (Claude Code default).
func EnsureMCPConfig() error {
	return EnsureMCPConfigFor(DefaultMCPSpec)
}

// EnsureMCPConfigFor creates or updates the MCP config file for the given spec.
func EnsureMCPConfigFor(spec MCPConfigSpec) error {
	if spec.File == "" {
		return nil
	}

	if spec.NeedsDir {
		dir := filepath.Dir(spec.File)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("could not create directory %s: %w", dir, err)
		}
	}

	if spec.Format == "toml" {
		return ensureMCPConfigTOML(spec)
	}
	return ensureMCPConfigJSON(spec)
}

func ensureMCPConfigJSON(spec MCPConfigSpec) error {
	var doc map[string]interface{}

	data, err := os.ReadFile(spec.File)
	if err == nil {
		if json.Unmarshal(data, &doc) != nil {
			return fmt.Errorf("could not parse %s: invalid JSON", spec.File)
		}
	} else if os.IsNotExist(err) {
		doc = make(map[string]interface{})
	} else {
		return fmt.Errorf("could not read %s: %w", spec.File, err)
	}

	servers, _ := doc[spec.ServerKey].(map[string]interface{})
	if servers == nil {
		servers = make(map[string]interface{})
	}
	servers["xpo"] = mcpServerEntry(spec.EntryStyle)
	doc[spec.ServerKey] = servers

	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("could not marshal %s: %w", spec.File, err)
	}
	out = append(out, '\n')
	return os.WriteFile(spec.File, out, 0644)
}

func mcpServerEntry(entryStyle string) map[string]interface{} {
	if entryStyle == "local-array" {
		return map[string]interface{}{
			"type":    "local",
			"command": []string{"xpo", "mcp"},
		}
	}
	return map[string]interface{}{
		"command": "xpo",
		"args":    []string{"mcp"},
	}
}

func ensureMCPConfigTOML(spec MCPConfigSpec) error {
	sectionHeader := fmt.Sprintf("[%s.xpo]", spec.ServerKey)
	entry := fmt.Sprintf("%s\ncommand = \"xpo\"\nargs = [\"mcp\"]\n", sectionHeader)

	data, err := os.ReadFile(spec.File)
	if os.IsNotExist(err) {
		return os.WriteFile(spec.File, []byte(entry), 0644)
	}
	if err != nil {
		return fmt.Errorf("could not read %s: %w", spec.File, err)
	}

	content := string(data)
	if strings.Contains(content, sectionHeader) {
		return nil
	}

	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	content += "\n" + entry
	return os.WriteFile(spec.File, []byte(content), 0644)
}

// LookupAgent finds an agent in the registry by name or binary (case-insensitive).
func LookupAgent(query string) (AgentConfig, bool) {
	q := strings.ToLower(query)
	for _, agent := range AgentRegistry {
		if strings.ToLower(agent.Name) == q || (agent.Binary != "" && strings.ToLower(agent.Binary) == q) {
			return agent, true
		}
	}
	return AgentConfig{}, false
}

// AgentRegistryNames returns the names of all agents in the registry.
func AgentRegistryNames() []string {
	names := make([]string, len(AgentRegistry))
	for i, a := range AgentRegistry {
		names[i] = a.Name
	}
	return names
}

// InstructionWriteResult describes what happened when writing agent instructions.
type InstructionWriteResult struct {
	Action     string // "created", "appended", "updated", "skipped"
	WasEdited  bool   // true if the existing managed block had local edits (hash mismatch)
	OldContent string // previous managed block content (for diff display)
	NewContent string // new managed block content
}

// AppendAgentInstructions writes xpo instructions to an agent file using managed blocks.
// If the file already contains a managed block, it is replaced. If a legacy heading-based
// section exists (no markers), it is migrated to a managed block. Otherwise the block is appended.
// For agents with skill support, writes the thin stub; otherwise writes the full docs.
func AppendAgentInstructions(agent AgentConfig, prefix string) error {
	res, _ := WriteAgentInstructions(agent, prefix, false)
	if res.Action == "skipped" {
		return fmt.Errorf("managed block has local edits (use force to overwrite)")
	}
	return nil
}

// GeneratedInstructions returns the managed-block content xpo writes for an
// agent: a short stub when the agent loads the skill, the full docs otherwise.
func GeneratedInstructions(agent AgentConfig, prefix string) string {
	if agent.SkillDir != "" {
		return GenerateAgentStub(prefix)
	}
	return GenerateAgentDocs(prefix)
}

// WriteAgentInstructions writes xpo instructions and returns detailed result.
// When force is false and the managed block has been edited, it returns a result
// with Action="skipped" and WasEdited=true so the caller can prompt the user.
func WriteAgentInstructions(agent AgentConfig, prefix string, force bool) (InstructionWriteResult, error) {
	xpoSection := GeneratedInstructions(agent, prefix)

	cliVersion := version.CLIVersion

	if agent.NeedsDir {
		dir := filepath.Dir(agent.File)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return InstructionWriteResult{}, fmt.Errorf("could not create directory %s: %w", dir, err)
		}
	}

	content, err := os.ReadFile(agent.File)
	if os.IsNotExist(err) {
		wrapped := WrapManagedBlock(xpoSection, cliVersion, agent.Format)
		if err := os.WriteFile(agent.File, []byte(wrapped+"\n"), 0644); err != nil {
			return InstructionWriteResult{}, err
		}
		return InstructionWriteResult{Action: "created", NewContent: xpoSection}, nil
	}
	if err != nil {
		return InstructionWriteResult{}, fmt.Errorf("could not read %s: %w", agent.File, err)
	}

	s := string(content)

	// Check for existing managed block
	if block := FindManagedBlock(s, agent.Format); block != nil {
		if BlockWasEdited(block) && !force {
			return InstructionWriteResult{
				Action:     "skipped",
				WasEdited:  true,
				OldContent: block.Content,
				NewContent: xpoSection,
			}, nil
		}
		result := ReplaceManagedBlock(s, block, xpoSection, cliVersion, agent.Format)
		if err := os.WriteFile(agent.File, []byte(result), 0644); err != nil {
			return InstructionWriteResult{}, err
		}
		return InstructionWriteResult{Action: "updated", NewContent: xpoSection}, nil
	}

	// Legacy migration: heading-based section without managed block markers
	if idx := findAgentInstructionsOffset(s); idx != -1 {
		before := strings.TrimRight(s[:idx], " \t\n")
		if before != "" {
			before += "\n\n"
		}
		wrapped := WrapManagedBlock(xpoSection, cliVersion, agent.Format)
		if err := os.WriteFile(agent.File, []byte(before+wrapped+"\n"), 0644); err != nil {
			return InstructionWriteResult{}, err
		}
		return InstructionWriteResult{Action: "updated", NewContent: xpoSection}, nil
	}

	// No existing section — append
	wrapped := WrapManagedBlock(xpoSection, cliVersion, agent.Format)
	f, err := os.OpenFile(agent.File, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return InstructionWriteResult{}, fmt.Errorf("could not open %s: %w", agent.File, err)
	}
	defer f.Close()

	if !strings.HasSuffix(s, "\n") {
		f.WriteString("\n")
	}
	_, err = f.WriteString("\n" + wrapped + "\n")
	if err != nil {
		return InstructionWriteResult{}, err
	}
	return InstructionWriteResult{Action: "appended", NewContent: xpoSection}, nil
}
