package exponential

import (
	"os"
	"sort"
	"strings"
)

// FileChange represents one file that will be created or modified.
type FileChange struct {
	Path   string // relative file path
	Action string // "create" or "modify"
}

// ChangePlan holds the list of file changes that init will make.
type ChangePlan struct {
	Changes []FileChange
}

// ComputeInitPlan determines which files will be created or modified by init.
func ComputeInitPlan(agents []AgentConfig) *ChangePlan {
	plan := &ChangePlan{}

	// .xpo/ directory
	if _, err := os.Stat(".xpo"); os.IsNotExist(err) {
		plan.add(".xpo/", "create")
	}

	// .gitignore
	plan.addFile(".gitignore")

	// .gitattributes
	plan.addFile(".gitattributes")

	// Per-agent files
	seen := map[string]bool{}
	for _, agent := range agents {
		// MCP config
		if agent.MCPConfig.HasMCPConfig() && !seen[agent.MCPConfig.File] {
			seen[agent.MCPConfig.File] = true
			plan.addFile(agent.MCPConfig.File)
		}

		// Agent instruction file
		if !seen[agent.File] {
			seen[agent.File] = true
			plan.addFile(agent.File)
		}

		// Skill files: only those that differ from the current template
		for _, f := range AgentSkillStates(agent) {
			if seen[f.Path] || f.State == SkillFileUpToDate {
				continue
			}
			seen[f.Path] = true
			plan.addFile(f.Path)
		}
	}

	plan.sort()
	return plan
}

func (p *ChangePlan) sort() {
	sort.Slice(p.Changes, func(i, j int) bool {
		a, b := p.Changes[i].Path, p.Changes[j].Path
		aDot := strings.HasPrefix(a, ".")
		bDot := strings.HasPrefix(b, ".")
		if aDot != bDot {
			return aDot
		}
		return a < b
	})
}

func (p *ChangePlan) addFile(path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		p.add(path, "create")
	} else {
		p.add(path, "modify")
	}
}

func (p *ChangePlan) add(path, action string) {
	p.Changes = append(p.Changes, FileChange{Path: path, Action: action})
}
