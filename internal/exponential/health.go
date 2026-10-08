package exponential

import (
	"fmt"
	"os"
	"strings"

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
	Interactive []string // fixable with user prompt (edited block, edited skill, legacy format)

	BlockEdited  bool     // managed instruction block has local edits
	BlockStale   bool     // unedited managed block differs from the current template
	SkillStale   bool     // skill files are missing or differ from the current template
	EditedSkills []string // skill files with local edits
}

// SkillEdited returns true if any skill file has local edits.
func (h *AgentHealth) SkillEdited() bool {
	return len(h.EditedSkills) > 0
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
// prefix is the project's issue prefix, which the generated instructions embed;
// when empty (unknown), the block content is not compared with the template.
func CheckAgentHealth(agent AgentConfig, integrationVer, prefix string) AgentHealth {
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
				h.BlockEdited = true
				h.Interactive = append(h.Interactive, fmt.Sprintf("%s: xpo section has local edits", agent.File))
			} else if integrationVer != "" && integrationVer != version.CLIVersion &&
				version.CompareVersions(integrationVer, version.CLIVersion) < 0 {
				h.BlockStale = true
				h.AutoFix = append(h.AutoFix, fmt.Sprintf("Update available (%s → %s)", integrationVer, version.CLIVersion))
			} else if prefix != "" && strings.TrimSpace(block.Content) != strings.TrimSpace(GeneratedInstructions(agent, prefix)) {
				// Template changed without a version bump.
				h.BlockStale = true
				h.AutoFix = append(h.AutoFix, "Instructions out of date")
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
			for _, f := range AgentSkillStates(agent) {
				switch f.State {
				case SkillFileEdited:
					h.EditedSkills = append(h.EditedSkills, f.Path)
					h.Interactive = append(h.Interactive, fmt.Sprintf("%s: skill has local edits", f.Path))
				case SkillFileStale, SkillFileMissing:
					h.SkillStale = true
				}
			}
			if h.SkillStale {
				h.AutoFix = append(h.AutoFix, "Skill out of date")
			}
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
