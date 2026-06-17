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
	binaryName := strings.TrimSuffix(prefix, "-")
	return fmt.Sprintf(`# Exponential Agent Instructions

This repository uses `+"`xpo`"+`, a local JSONL-based issue tracker, to manage development tasks. As an AI agent, you should use `+"`xpo`"+` to understand the current state of the project, plan your work, and record new findings.

## Core Philosophy: Project Memory

`+"`xpo`"+` serves as the persistent memory for the project.
- **Start** by reading the backlog to understand what needs to be done.
- **Update** the status of tasks you are working on.
- **Record** any new tasks or bugs you discover as new issues. Do not just fix them implicitly or leave them as TODO comments in code; create a tracked issue so it can be prioritized.
- **Persist** your planning. If a task is too big, break it down into child tasks in `+"`xpo`"+`.

## Strict Workflow Rules

1. **No "Ghost" Work**: Any work done by an agent MUST be backed by a xpo task/bug/epic.
2. **Missing Tasks**: If no such task exists for your current objective, you must create it.
   - **Timing**: Create the task *after* the user approves your initial design/plan.
3. **Check Dependencies**: Before picking up a task, check its dependencies. If it has unresolved `+"`depends_on`"+` or `+"`blocked_by`"+` links to tasks that are not `+"`DONE`"+`, flag the blockers before starting work.
4. **In-Progress**: Before starting any code work (editing files), you MUST set the corresponding xpo task to `+"`DOING`"+` using `+"`xpo start`"+`.
5. **Completion**: You MUST set the xpo task to `+"`DONE`"+` using `+"`xpo done`"+` *only after* the user approves the final review/walkthrough.

## Agent Identity
When performing actions that modify the tracker (add, update), ensure you are identified as an agent if possible, or use the execution environment's git config.

## Usage Guide

### 1. Discovery (Reading the State)

**List all issues:**
`+"```bash\n./xpo list\n```"+`
Use this to find your assigned task or pick the next prioritized item from the backlog.

**Read a specific issue:**
`+"```bash\n./xpo show <issue-id>\n```"+`
Always read the full details of an issue before starting work. It may contain description, acceptance criteria, or context from previous agents.

### 2. Planning (Creating Issues)

**Create an Epic (High-level goal):**
`+"```bash\n./xpo add \"Refactor Database Layer\" --epic --desc \"Move from SQLite to Postgres\"\n```"+`

**Create a Task (Actionable item):**
`+"```bash\n./xpo add \"Create Migration Script\" -p <status-id-of-epic> --desc \"Write SQL migration\"\n```"+`

**Filing Bugs/Findings:**
If you encounter a bug or necessary refactor while working on something else, file it immediately so it isn't lost.
`+"```bash\n./xpo add \"Bug: Race condition in login\" --desc \"Observed when...\"\n```"+`

### 3. Execution (Updating Status)

**Start a task:**
`+"```bash\n./xpo start <issue-id>\n```"+`

**Mark as Done:**
`+"```bash\n./xpo done <issue-id>\n```"+`

**Update details:**
`+"```bash\n./xpo update <issue-id> --desc \"Updated description with new findings...\"\n```"+`

## Workflow Example for Agents

1. **Context Check**: Run `+"`./xpo list`"+` to see what is PLANNED or DOING.
2. **Backing Task**: Ensure a task exists for your work.
   - If yes: `+"`xpo start <id>`"+`.
   - If no: Plan your work, get approval, then `+"`xpo add \"...\"`"+`, then `+"`xpo start <id>`"+`.
3. **Implementation**: Modify code, tests, docs.
4. **Review**: Present walkthrough/results to user.
5. **Completion**: On approval, `+"`xpo done <id>`"+`.


## Building the project

Use the `+"`make build`"+` command to compile the `+"`%s`"+` binary.
`, binaryName)
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
