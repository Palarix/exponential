package exponential

import (
	"fmt"
	"os"

	"github.com/palarix/exponential/internal/version"
)

// AgentHealth describes the diagnostic state of one agent's xpo integration.
type AgentHealth struct {
	Agent      AgentConfig
	HasMCP     bool     // has xpo MCP entry
	HasInstrs  bool     // has managed block or legacy instructions
	HasSkill   bool     // has skill installed locally
	Configured bool     // at least one of the above is true (agent was set up)
	AutoFix    []string // fixable silently (missing MCP, missing skill, stale version)
	Interactive []string // fixable with user prompt (edited block, legacy format)
}

// HasProblems returns true if the agent has any issues (auto-fixable or interactive).
func (h *AgentHealth) HasProblems() bool {
	return len(h.AutoFix) > 0 || len(h.Interactive) > 0
}

// AllProblems returns all issues combined for display.
func (h *AgentHealth) AllProblems() []string {
	return append(h.AutoFix, h.Interactive...)
}

// CheckAgentHealth examines one agent's integration state and classifies issues.
func CheckAgentHealth(agent AgentConfig, integrationVer string) AgentHealth {
	h := AgentHealth{Agent: agent}

	if agent.MCPConfig.HasMCPConfig() {
		status := DetectMCPConfigFor(agent.MCPConfig)
		h.HasMCP = status.HasExponential
	}

	content, err := os.ReadFile(agent.File)
	if err == nil {
		block := FindManagedBlock(string(content), agent.Format)
		if block != nil {
			h.HasInstrs = true
			if BlockWasEdited(block) {
				h.Interactive = append(h.Interactive, fmt.Sprintf("%s: xpo section has local edits", agent.File))
			} else if integrationVer != "" && integrationVer != version.CLIVersion &&
				version.CompareVersions(integrationVer, version.CLIVersion) < 0 {
				h.AutoFix = append(h.AutoFix, fmt.Sprintf("Update available (%s → %s)", integrationVer, version.CLIVersion))
			}
		} else if HasAgentInstructions(string(content)) {
			h.HasInstrs = true
			h.Interactive = append(h.Interactive, fmt.Sprintf("%s: legacy format", agent.File))
		}
	}

	if agent.SkillDir != "" {
		status := DetectSkillInstall(agent)
		if status.Local {
			h.HasSkill = true
		} else if status.BrokenSymlink {
			h.AutoFix = append(h.AutoFix, "Skill symlink broken")
		}
	}

	h.Configured = h.HasMCP || h.HasInstrs || h.HasSkill

	// Only report missing pieces for agents that are already configured
	if h.Configured {
		if agent.MCPConfig.HasMCPConfig() && !h.HasMCP {
			h.AutoFix = append(h.AutoFix, "MCP missing")
		}
		if !h.HasInstrs {
			h.AutoFix = append(h.AutoFix, "Instructions missing")
		}
		if agent.SkillDir != "" && !h.HasSkill {
			h.AutoFix = append(h.AutoFix, "Skill not installed")
		}
	}

	return h
}
