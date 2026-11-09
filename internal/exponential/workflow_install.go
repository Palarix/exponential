package exponential

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// The workflow skill belongs to the project. xpo writes it only when asked:
// on first init (InstallWorkflow) or with `xpo init --workflow`
// (SetupWorkflow). Re-init and doctor never read, create or update it.

// legacyWorkflowHeading marks an xpo SKILL.md from before the reference and
// the workflow were split (xpo-f19bfa).
const legacyWorkflowHeading = "# xpo Development Workflow"

// WorkflowSkillDir is the agent's workflow skill directory, or "" for agents
// without skill support.
func WorkflowSkillDir(agent AgentConfig) string {
	if agent.SkillDir == "" {
		return ""
	}
	return filepath.Join(agent.SkillDir, WorkflowSkillName)
}

func workflowSkillFile(agent AgentConfig) string {
	return filepath.Join(WorkflowSkillDir(agent), "SKILL.md")
}

// HasWorkflowSkill reports whether the agent has a workflow skill.
func HasWorkflowSkill(agent AgentConfig) bool {
	if agent.SkillDir == "" {
		return false
	}
	_, err := os.Stat(workflowSkillFile(agent))
	return err == nil
}

func skillPath(agent AgentConfig) string {
	return filepath.Join(agent.SkillDir, "xpo", "SKILL.md")
}

// HadLegacyWorkflow reports whether the agent's xpo skill is still the
// combined skill from before the split, which held the workflow. Re-init
// checks it before updating the skill, to point the user at
// `xpo init --workflow`.
func HadLegacyWorkflow(agent AgentConfig) bool {
	if agent.SkillDir == "" {
		return false
	}
	data, err := os.ReadFile(skillPath(agent))
	if err != nil {
		return false
	}
	body, _, _, _ := ParseSkillFile(string(data))
	return strings.Contains(body, legacyWorkflowHeading)
}

// InstallWorkflow writes the template as the agent's workflow skill on first
// init, unless the agent already has one. tpl nil means no workflow.
func InstallWorkflow(agent AgentConfig, tpl *WorkflowTemplate) (bool, error) {
	if tpl == nil || agent.SkillDir == "" || HasWorkflowSkill(agent) {
		return false, nil
	}
	if err := writeNewFile(workflowSkillFile(agent), tpl.Skill()); err != nil {
		return false, err
	}
	return true, nil
}

// WorkflowTargets returns the agents that are set up in this project: they
// have the xpo MCP server, a managed block, or the xpo skill.
func WorkflowTargets() []AgentConfig {
	var out []AgentConfig
	for _, agent := range AgentRegistry {
		if CheckAgentHealth(agent, "", "").Configured {
			out = append(out, agent)
		}
	}
	return out
}

// ExistingWorkflowDirs returns the workflow skill directories of the agents
// that already have one: SetupWorkflow replaces them.
func ExistingWorkflowDirs(agents []AgentConfig) []string {
	var out []string
	seen := map[string]bool{}
	for _, a := range agents {
		dir := WorkflowSkillDir(a)
		if dir == "" || seen[dir] {
			continue
		}
		seen[dir] = true
		if _, err := os.Stat(dir); err == nil {
			out = append(out, dir)
		}
	}
	return out
}

// WorkflowSetupResult describes what SetupWorkflow did.
type WorkflowSetupResult struct {
	Written     []string // workflow skill files written
	InlineAdded []string // instruction files that got a Workflow section
	InlineKept  []string // instruction files that already had one, left alone
}

// SetupWorkflow writes the template as the workflow of the given agents, for
// `xpo init --workflow`. Each existing workflow skill directory is deleted
// first. Agents without skill support get the workflow below their managed
// block, unless their file already has a Workflow section.
func SetupWorkflow(agents []AgentConfig, tpl *WorkflowTemplate) (WorkflowSetupResult, error) {
	var res WorkflowSetupResult
	seen := map[string]bool{}
	for _, a := range agents {
		dir := WorkflowSkillDir(a)
		if dir == "" || seen[dir] {
			continue
		}
		seen[dir] = true
		if err := os.RemoveAll(dir); err != nil {
			return res, fmt.Errorf("could not remove %s: %w", dir, err)
		}
		if err := writeNewFile(workflowSkillFile(a), tpl.Skill()); err != nil {
			return res, err
		}
		res.Written = append(res.Written, workflowSkillFile(a))
	}
	for _, a := range InlineWorkflowTargets(agents) {
		added, err := AppendInlineWorkflow(a, tpl)
		if err != nil {
			return res, fmt.Errorf("%s: %w", a.File, err)
		}
		if added {
			res.InlineAdded = append(res.InlineAdded, a.File)
		} else {
			res.InlineKept = append(res.InlineKept, a.File)
		}
	}
	return res, nil
}

// InlineWorkflowTargets returns the agents without skill support whose
// instruction file should get the workflow inline: one per file, and only if
// no agent with skill support shares that file.
func InlineWorkflowTargets(agents []AgentConfig) []AgentConfig {
	skillFiles := map[string]bool{}
	for _, a := range agents {
		if a.SkillDir != "" {
			skillFiles[a.File] = true
		}
	}
	seen := map[string]bool{}
	var out []AgentConfig
	for _, a := range agents {
		if a.SkillDir != "" || skillFiles[a.File] || seen[a.File] {
			continue
		}
		seen[a.File] = true
		out = append(out, a)
	}
	return out
}

// hasInlineWorkflow reports whether the file has a Workflow section.
func hasInlineWorkflow(content string) bool {
	return strings.HasPrefix(content, "## Workflow\n") || strings.Contains(content, "\n## Workflow\n")
}

// AppendInlineWorkflow appends the template's inline workflow below the
// agent's managed block. A file that already has a Workflow section is left
// alone; added reports whether the section was appended.
func AppendInlineWorkflow(agent AgentConfig, tpl *WorkflowTemplate) (added bool, err error) {
	data, err := os.ReadFile(agent.File)
	if err != nil {
		return false, err
	}
	s := string(data)
	if hasInlineWorkflow(s) {
		return false, nil
	}
	s = strings.TrimRight(s, "\n") + "\n\n" + strings.TrimRight(tpl.Inline(), "\n") + "\n"
	return true, os.WriteFile(agent.File, []byte(s), 0644)
}

// writeNewFile writes content to path, creating parent directories.
func writeNewFile(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("could not create directory for %s: %w", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("could not write %s: %w", path, err)
	}
	return nil
}
