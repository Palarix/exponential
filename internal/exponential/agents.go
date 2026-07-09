package exponential

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// AgentConfig defines an AI agent and its instruction file location
type AgentConfig struct {
	Name     string // Agent name (e.g., "Cursor", "GitHub Copilot")
	File     string // Primary file path (e.g., ".cursorrules")
	Format   string // "markdown" or "plain"
	NeedsDir bool   // If true, parent directory must be created
}

// AgentRegistry contains all supported AI agent configurations
var AgentRegistry = []AgentConfig{
	{Name: "Generic Agent", File: "AGENTS.md", Format: "markdown"},
	{Name: "Gemini", File: "GEMINI.md", Format: "markdown"},
	{Name: "Cursor", File: ".cursorrules", Format: "plain"},
	{Name: "Windsurf", File: ".windsurfrules", Format: "plain"},
	{Name: "GitHub Copilot", File: ".github/copilot-instructions.md", Format: "markdown", NeedsDir: true},
	{Name: "Cline", File: ".clinerules", Format: "markdown"},
	{Name: "Roo Code", File: ".roorules", Format: "markdown"},
	{Name: "Aider", File: "CONVENTIONS.md", Format: "markdown"},
	{Name: "Continue", File: ".continue/rule./xpo.md", Format: "markdown", NeedsDir: true},
}

// AgentDetectionResult contains information about a detected agent file
type AgentDetectionResult struct {
	Agent          AgentConfig
	Exists         bool
	HasExponentialConfig bool
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

// GenerateAgentDocs generates the xpo agent documentation content
func GenerateAgentDocs(prefix string) string {
	_ = prefix
	return `# Exponential Agent Instructions

This repository uses ` + "`xpo`" + ` (Exponential), a local issue tracker, to manage development tasks. As an AI agent, you should use ` + "`xpo`" + ` to understand the current state of the project, plan your work, record new findings, and document the rationale and work done to implement your tasks.

## How to Interact with xpo

An ` + "`xpo`" + ` MCP server is registered in ` + "`.mcp.json`" + `. **Always use the MCP tools** — do not shell out to the ` + "`xpo`" + ` CLI.

| Action | MCP tool |
|---|---|
| List / search issues | ` + "`mcp__xpo__list`" + ` |
| Read one issue | ` + "`mcp__xpo__show`" + ` |
| Create an issue | ` + "`mcp__xpo__add`" + ` |
| Update fields (incl. **status transitions**, labels, assignee, story_points, parent) | ` + "`mcp__xpo__update`" + ` |
| Add a comment | ` + "`mcp__xpo__comment`" + ` |
| Add a dependency link | ` + "`mcp__xpo__link`" + ` |
| View audit trail | ` + "`mcp__xpo__history`" + ` |
| Read/write/delete a spec | ` + "`mcp__xpo__spec`" + ` |
| Read/write/delete a walkthrough | ` + "`mcp__xpo__walkthrough`" + ` |
| Manage generic artifacts | ` + "`mcp__xpo__artifact`" + ` |

Status transitions (` + "`BACKLOG`" + ` → ` + "`PLANNED`" + ` → ` + "`DOING`" + ` → ` + "`BLOCKED`" + ` → ` + "`DONE`" + `) are done by calling ` + "`update`" + ` with the ` + "`status`" + ` field.

## Development Workflow

1. **Discover** — ` + "`list`" + ` to see the board; ` + "`show`" + ` for details on candidate issues.
2. **Plan** — if no issue covers the work, create one with ` + "`add`" + ` (only after the user approves the design).
3. **Start** — ` + "`update`" + ` with ` + "`status: \"DOING\"`" + ` before editing any code.
4. **Implement & test** — make changes, then verify against the project's test suite.
5. **Document** — ` + "`comment`" + ` with a markdown summary of what changed and why.
6. **Complete** — ` + "`update`" + ` with ` + "`status: \"DONE\"`" + ` once the user approves.

## Strict Workflow Rules

1. **No "ghost" work** — every code change MUST be backed by an xpo issue.
2. **Only pick up planned work** — do not start work on issues with ` + "`BACKLOG`" + ` status. Issues must be ` + "`PLANNED`" + ` to be eligible.
3. **Check dependencies first** — before picking up an issue, inspect its dependencies via ` + "`show`" + `. If any ` + "`depends_on`" + ` or ` + "`blocked_by`" + ` targets are not ` + "`DONE`" + `, flag the unresolved blockers before starting work.
4. **Missing tasks** — if no issue exists for your current objective, create it first, but only _after_ the user approves your design/plan.
5. **In-progress before edits** — before touching any file, transition the issue to ` + "`DOING`" + ` via ` + "`update`" + `.
6. **Comment before complete** — add a summary comment via ` + "`comment`" + ` _before_ transitioning to ` + "`DONE`" + `, and only do so after the user approves the final walkthrough.
7. **File what you find** — bugs or follow-up work discovered during a task must be filed as new issues (linked to the current one via ` + "`link`" + `), not left as TODOs in code.

## Agent Identity

When the tracker records who made a change, identify yourself as an agent. Use the form ` + "`<Agent Name> <agent@<host>.local>`" + ` — e.g. ` + "`Claude Code <agent@myhost.local>`" + `. The host portion helps distinguish contributions from different execution environments.

## Usage Notes

### Discovery

- Before creating a new issue, search with ` + "`list`" + ` (use the ` + "`match`" + ` parameter for free-text search) to ensure no existing issue already covers the work.
- If an issue looks related, read it fully with ` + "`show`" + ` before deciding.

### Creating Issues

- Always set at least one label via the ` + "`labels`" + ` parameter. Prefer the built-in labels (` + "`bug`" + `, ` + "`feature`" + `, ` + "`epic`" + `) unless something more specific fits.
- Keep titles under 100 characters.
- Descriptions render as **Markdown** in the web UI. Use headings, lists, code blocks, bold/italic, and links. Use double newlines between paragraphs. Markdown checklists (` + "`- [ ] Title`" + `) can sub-divide task steps.
- When a new issue belongs to an Epic, set ` + "`parent`" + ` to the epic's ID.

### Linking

Use ` + "`link`" + ` to express relationships. Supported types: ` + "`blocks`" + `, ` + "`blocked_by`" + `, ` + "`depends_on`" + `, ` + "`dependency_of`" + `, ` + "`duplicates`" + `, ` + "`duplicated_by`" + `, ` + "`relates_to`" + `. When a task spawns follow-up work, link the new issue back to the originating one.

### Specs and Walkthroughs

- **Before starting implementation**, read the issue's spec (if any) via ` + "`spec`" + ` with ` + "`operation: \"read\"`" + `. The spec captures requirements, acceptance criteria, and design decisions agreed upon before coding begins.
- **After implementation**, write a walkthrough via ` + "`walkthrough`" + ` with ` + "`operation: \"write\"`" + ` summarizing what changed and why.
- Use ` + "`artifact`" + ` for attaching supplemental files (test outputs, design diagrams, logs) that support the issue but don't fit into spec or walkthrough.

### Completion Comments

Comments are markdown. A good completion comment includes:

- A short **Summary** section listing what changed (use backtick code spans for file/function names).
- The **rationale** — why this approach, why not the alternatives.
- Anything a future agent reading the issue would need to pick up where you left off.
`
}

// MCPConfigStatus describes the state of .mcp.json in the project root.
type MCPConfigStatus struct {
	Exists    bool
	HasExponential  bool
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

// AppendAgentInstructions appends xpo instructions to an agent file
func AppendAgentInstructions(agent AgentConfig, prefix string) error {
	xpoSection := GenerateAgentDocs(prefix)

	// Create directory if needed
	if agent.NeedsDir {
		dir := filepath.Dir(agent.File)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("could not create directory %s: %w", dir, err)
		}
	}

	content, err := os.ReadFile(agent.File)
	if err == nil {
		// File exists - append
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
		// Create new file
		if err := os.WriteFile(agent.File, []byte(xpoSection), 0644); err != nil {
			return fmt.Errorf("could not create %s: %w", agent.File, err)
		}
	} else {
		return fmt.Errorf("could not read %s: %w", agent.File, err)
	}

	return nil
}
