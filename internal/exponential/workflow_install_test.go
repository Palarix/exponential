package exponential

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/palarix/exponential/internal/version"
)

func guided(t *testing.T) *WorkflowTemplate {
	t.Helper()
	tpl, ok := LookupWorkflow("guided")
	if !ok {
		t.Fatal("guided not registered")
	}
	return &tpl
}

func workflowPath(agent AgentConfig) string {
	return filepath.Join(agent.SkillDir, "xpo-workflow", "SKILL.md")
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

var testCopilot = AgentConfig{Name: "GitHub Copilot", File: ".github/copilot-instructions.md", Format: "markdown", NeedsDir: true}
var testGeneric = AgentConfig{Name: "Generic Agent", File: "AGENTS.md", Format: "markdown"}

// legacySkill is an xpo SKILL.md from before the split: workflow and
// reference in one file.
const legacySkill = `---
name: xpo
description: >-
  Issue-tracking workflow.
---

# xpo Development Workflow

### 1. Check existing work

Look at the board.
`

// --- First init ---

func TestInstallWorkflow_WritesOnceWithoutMarker(t *testing.T) {
	setupProject(t, "test")

	written, err := InstallWorkflow(testClaudeCode, guided(t))
	if err != nil || !written {
		t.Fatalf("expected write, got written=%v err=%v", written, err)
	}
	s := readFile(t, workflowPath(testClaudeCode))
	if s != guided(t).Skill() || strings.Contains(s, "xpo:skill") {
		t.Fatal("workflow skill should be the template, without a marker")
	}

	writeFile(t, workflowPath(testClaudeCode), "my own workflow")
	written, err = InstallWorkflow(testClaudeCode, guided(t))
	if err != nil || written || readFile(t, workflowPath(testClaudeCode)) != "my own workflow" {
		t.Fatalf("first init must not overwrite an existing workflow, got written=%v err=%v", written, err)
	}
}

func TestInstallWorkflow_NoneOrNoSkillDir(t *testing.T) {
	setupProject(t, "test")
	if written, _ := InstallWorkflow(testClaudeCode, nil); written || exists(workflowPath(testClaudeCode)) {
		t.Fatal("none writes nothing")
	}
	if written, _ := InstallWorkflow(testCopilot, guided(t)); written {
		t.Fatal("skill-less agent gets no workflow skill")
	}
}

func TestInlineWorkflowTargets(t *testing.T) {
	got := InlineWorkflowTargets([]AgentConfig{testGeneric, testCopilot, testClaudeCode})
	if len(got) != 2 {
		t.Fatalf("expected Generic and Copilot, got %v", got)
	}
	got = InlineWorkflowTargets([]AgentConfig{testGeneric, testCodex, testCopilot})
	if len(got) != 1 || got[0].Name != "GitHub Copilot" {
		t.Fatalf("AGENTS.md is shared with Codex, which has the skill; got %v", got)
	}
}

func TestAppendInlineWorkflow_Once(t *testing.T) {
	setupProject(t, "test")
	if _, err := WriteAgentInstructions(testCopilot, "test", false); err != nil {
		t.Fatal(err)
	}
	if added, err := AppendInlineWorkflow(testCopilot, guided(t)); err != nil || !added {
		t.Fatalf("expected append, got %v %v", added, err)
	}
	if added, err := AppendInlineWorkflow(testCopilot, guided(t)); err != nil || added {
		t.Fatalf("second append must be skipped, got %v %v", added, err)
	}
	s := readFile(t, testCopilot.File)
	if strings.Count(s, "\n## Workflow\n") != 1 || strings.Index(s, "<!-- xpo:end") > strings.Index(s, "## Workflow") {
		t.Fatalf("one workflow, below the managed block:\n%s", s)
	}

	// Refreshing the block keeps the workflow, and health ignores it.
	if _, err := WriteAgentInstructions(testCopilot, "test", true); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(readFile(t, testCopilot.File), "\n## Workflow\n") {
		t.Fatal("refreshing the block removed the workflow")
	}
	if h := CheckAgentHealth(testCopilot, version.CLIVersion, "test"); h.HasProblems() {
		t.Fatalf("the inline workflow must not be reported, got %v", h.AllProblems())
	}
}

func TestComputeInitPlan_ListsWorkflowOnlyForFirstInit(t *testing.T) {
	setupProject(t, "test")

	plan := ComputeInitPlan([]AgentConfig{testClaudeCode}, guided(t))
	if !planHas(plan, workflowPath(testClaudeCode), "create") {
		t.Fatalf("plan should create the workflow, got %v", plan.Changes)
	}
	plan = ComputeInitPlan([]AgentConfig{testClaudeCode}, nil)
	for _, c := range plan.Changes {
		if strings.Contains(c.Path, "xpo-workflow") {
			t.Fatalf("re-init and doctor never plan the workflow: %v", c)
		}
	}
	writeFile(t, workflowPath(testClaudeCode), "mine")
	if planHas(ComputeInitPlan([]AgentConfig{testClaudeCode}, guided(t)), workflowPath(testClaudeCode), "create") {
		t.Fatal("existing workflow is not created again")
	}
}

func planHas(plan *ChangePlan, path, action string) bool {
	for _, c := range plan.Changes {
		if c.Path == path && c.Action == action {
			return true
		}
	}
	return false
}

// --- Re-init: no migration, instruction files only between the markers ---

func TestReinit_LeavesWorkflowAlone(t *testing.T) {
	setupProject(t, "test")
	writeFile(t, skillPath(testClaudeCode), WrapSkillFile(legacySkill, "1.5.0"))
	writeFile(t, workflowPath(testClaudeCode), "mine")

	if _, err := WriteAgentSkill(testClaudeCode, "", false); err != nil {
		t.Fatal(err)
	}
	if readFile(t, workflowPath(testClaudeCode)) != "mine" {
		t.Fatal("the skill update must not touch xpo-workflow")
	}
	if !strings.Contains(readFile(t, skillPath(testClaudeCode)), "# xpo Reference") {
		t.Fatal("the old skill should become the reference")
	}
}

func TestHadLegacyWorkflow(t *testing.T) {
	setupProject(t, "test")
	if HadLegacyWorkflow(testClaudeCode) {
		t.Fatal("no skill, no legacy workflow")
	}
	writeFile(t, skillPath(testClaudeCode), WrapSkillFile(legacySkill, "1.5.0"))
	if !HadLegacyWorkflow(testClaudeCode) {
		t.Fatal("old combined skill should be detected")
	}
	WriteAgentSkill(testClaudeCode, "", false)
	if HadLegacyWorkflow(testClaudeCode) {
		t.Fatal("the reference is not a legacy workflow")
	}
}

func TestWriteAgentInstructions_OnlyBetweenMarkers(t *testing.T) {
	setupProject(t, "test")
	old := GenerateAgentDocs("test") + "\n### Workflow\n\n### 1. Check existing work\n"
	writeFile(t, testCopilot.File, "# Copilot\n\n"+WrapManagedBlock(old, "1.5.0", "markdown")+"\n\nMy notes.\n")

	if _, err := WriteAgentInstructions(testCopilot, "test", false); err != nil {
		t.Fatal(err)
	}
	s := readFile(t, testCopilot.File)
	if strings.Contains(s, "## Workflow") {
		t.Fatalf("the update must not write a workflow outside the markers:\n%s", s)
	}
	if !strings.HasPrefix(s, "# Copilot\n\n") || !strings.HasSuffix(s, "\n\nMy notes.\n") {
		t.Fatalf("text outside the markers must stay as it was:\n%s", s)
	}
}

// --- xpo init --workflow on an initialized project ---

func TestSetupWorkflow_ReplacesFoldersAndAppendsInline(t *testing.T) {
	setupProject(t, "test")
	writeFile(t, workflowPath(testClaudeCode), "old")
	writeFile(t, filepath.Join(testClaudeCode.SkillDir, "xpo-workflow", "references", "extra.md"), "extra")
	if _, err := WriteAgentInstructions(testCopilot, "test", false); err != nil {
		t.Fatal(err)
	}

	agents := []AgentConfig{testClaudeCode, testCodex, testCopilot}
	if got := ExistingWorkflowDirs(agents); len(got) != 1 || got[0] != filepath.Join(testClaudeCode.SkillDir, "xpo-workflow") {
		t.Fatalf("expected Claude's folder, got %v", got)
	}

	res, err := SetupWorkflow(agents, guided(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range []AgentConfig{testClaudeCode, testCodex} {
		if readFile(t, workflowPath(a)) != guided(t).Skill() {
			t.Fatalf("%s should have the template", a.Name)
		}
	}
	if exists(filepath.Join(testClaudeCode.SkillDir, "xpo-workflow", "references", "extra.md")) {
		t.Fatal("the old folder should be replaced as a whole")
	}
	if len(res.Written) != 2 || len(res.InlineAdded) != 1 || len(res.InlineKept) != 0 {
		t.Fatalf("unexpected result: %+v", res)
	}

	// Running it again keeps the existing inline workflow.
	res, _ = SetupWorkflow(agents, guided(t))
	if len(res.InlineKept) != 1 || strings.Count(readFile(t, testCopilot.File), "\n## Workflow\n") != 1 {
		t.Fatalf("existing inline workflow should be kept: %+v", res)
	}
}

func TestWorkflowTargets_SetUpAgentsOnly(t *testing.T) {
	setupProject(t, "test")
	setupAgentFull(t, testClaudeCode, "test")
	for _, a := range WorkflowTargets() {
		if a.Name != "Claude Code" {
			t.Fatalf("only set-up agents are targets, got %s", a.Name)
		}
	}
	if len(WorkflowTargets()) != 1 {
		t.Fatalf("expected Claude Code, got %v", WorkflowTargets())
	}
}
