package exponential

import (
	"strings"
	"testing"
)

// flat folds line wraps to single spaces, so assertions don't depend on where
// a template wraps its lines.
func flat(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

var workflowStepHeadings = []string{
	"### 1. Check existing work",
	"### 2. Plan",
	"### 3. Spec",
	"### 4. Start",
	"### 5. Implement & Test",
	"### 6. Handoff",
	"### 7. Revisions",
	"### 8. Walkthrough",
	"### 9. Complete",
}

func TestReferenceSkill_HasNoWorkflow(t *testing.T) {
	s := GenerateSkillMD()
	if !strings.Contains(s, "name: xpo\n") {
		t.Error("reference skill should be named xpo")
	}
	for _, h := range workflowStepHeadings {
		if strings.Contains(s, h) {
			t.Errorf("reference skill still contains workflow step %q", h)
		}
	}
	for _, process := range []string{"Stop here and wait", "### Backlog Review", "### Prioritization", "Hard Rules"} {
		if strings.Contains(s, process) {
			t.Errorf("reference skill still contains process text %q", process)
		}
	}
	if !strings.Contains(s, "`xpo-workflow`") {
		t.Error("reference skill should point to the xpo-workflow skill")
	}
}

func TestReferenceSkill_CoversMechanics(t *testing.T) {
	s := flat(GenerateSkillMD())
	for _, want := range []string{
		"BACKLOG → PLANNED → DOING → DONE",
		"primary label",
		"`link`",
		"`overwrite: true`",
		"references/spec-guide.md",
		"`walkthrough`",
		"assigns it to the person you are working for",
		"Never set `assignee` to yourself",
		"without a `strategy`",
		"It refuses to run while the worktree has uncommitted or untracked files.",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("reference skill is missing %q", want)
		}
	}
}

// The merge-conflict recovery is tool mechanics, so it lives in the reference
// (xpo-8e922f, moved by xpo-f19bfa).
func TestReferenceSkill_RecoversFromMergeConflicts(t *testing.T) {
	s := flat(GenerateSkillMD())
	for _, want := range []string{
		"If it fails because the default branch moved on (conflicts)",
		"`git rebase <default-branch>`, or `git merge <default-branch>`",
		"re-run the tests, commit, and call `merge` again",
		"never run `git push`, or merge, rebase or commit on the default branch yourself",
		"Rebasing the issue's branch onto the default branch (or merging the default branch into it) from the worktree is allowed.",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("reference skill is missing %q", want)
		}
	}
}

// Public by default is policy, so it lives in the workflow, not the
// managed reference.
func TestPublicByDefault_InWorkflowNotReference(t *testing.T) {
	for _, tpl := range WorkflowTemplates {
		for name, doc := range map[string]string{
			tpl.Name + " skill":  tpl.Skill(),
			tpl.Name + " inline": tpl.Inline(),
		} {
			d := flat(doc)
			for _, want := range []string{"### Public by default", "local file paths", "private repositories or projects", "internal hostnames or URLs"} {
				if !strings.Contains(d, want) {
					t.Errorf("%s is missing %q", name, want)
				}
			}
			if strings.Contains(doc, "<!-- xpo:skill") {
				t.Errorf("%s must not carry an xpo:skill marker", name)
			}
		}
	}
	for name, doc := range map[string]string{
		"skill":      GenerateSkillMD(),
		"docs":       GenerateAgentDocs("xpo"),
		"spec-guide": GenerateSpecGuide(),
	} {
		if strings.Contains(doc, "Public by default") {
			t.Errorf("%s should not carry the Public by default rule", name)
		}
	}
}

func TestAgentStub_OnlyUniversalRules(t *testing.T) {
	s := flat(GenerateAgentStub("pay"))
	for _, gone := range []string{
		"Every code change must be backed",
		"Never start work on a BACKLOG issue",
		"Bugs discovered during implementation",
		"load the `xpo` skill and follow it",
	} {
		if strings.Contains(s, gone) {
			t.Errorf("stub still contains workflow rule %q", gone)
		}
	}
	for _, want := range []string{
		"`pay-a1b2c3`",
		"never shell out to the `xpo` CLI",
		"report the error to the user",
		"If the project has an `xpo-workflow` skill, load it before you plan, file or change anything, and follow it.",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("stub is missing %q", want)
		}
	}
}

func TestAgentDocs_ReferenceOnly(t *testing.T) {
	s := flat(GenerateAgentDocs("pay"))
	if !strings.Contains(s, "If this file has a Workflow section, follow it.") {
		t.Error("inline docs should point to a Workflow section")
	}
	if strings.Contains(s, "`xpo-workflow` skill") {
		t.Error("inline docs should not point skill-less agents at a skill")
	}
	if strings.Contains(s, "### Workflow") {
		t.Error("inline docs should not contain the workflow")
	}
	for _, h := range workflowStepHeadings {
		if strings.Contains(s, h) {
			t.Errorf("inline docs still contain workflow step %q", h)
		}
	}
	if !strings.Contains(s, "MCP Tools Reference") {
		t.Error("inline docs should keep the MCP tools reference")
	}
}

func TestGuidedWorkflow_Skill(t *testing.T) {
	tpl, ok := LookupWorkflow("guided")
	if !ok {
		t.Fatal("guided workflow not registered")
	}
	s := tpl.Skill()
	if !strings.HasPrefix(s, "---\nname: xpo-workflow\n") {
		t.Errorf("guided skill should start with xpo-workflow frontmatter:\n%s", s[:80])
	}
	for _, h := range workflowStepHeadings {
		if !strings.Contains(s, h) {
			t.Errorf("guided workflow is missing step %q", h)
		}
	}
	f := flat(s)
	for _, want := range []string{
		"Every code change must be backed by an xpo issue",
		"Never start work on a BACKLOG issue without explicit user approval",
		"Bugs found during implementation need no approval",
		"### Backlog Review",
		"### Prioritization",
		"Stop here and wait.",
		"load the `xpo` skill",
	} {
		if !strings.Contains(f, want) {
			t.Errorf("guided skill is missing %q", want)
		}
	}
	if strings.Contains(s, "<!-- xpo:skill") {
		t.Error("guided workflow must not carry an xpo:skill marker")
	}
}

func TestGuidedWorkflow_Inline(t *testing.T) {
	tpl, _ := LookupWorkflow("guided")
	s := tpl.Inline()
	if !strings.HasPrefix(s, "## Workflow\n") {
		t.Errorf("inline workflow should start with a Workflow heading:\n%s", s[:40])
	}
	if strings.Contains(s, "name: xpo-workflow") {
		t.Error("inline workflow should have no frontmatter")
	}
	if strings.Contains(s, "load the `xpo` skill") {
		t.Error("inline workflow should not point skill-less agents at a skill")
	}
	if !strings.Contains(s, "### 9. Complete") {
		t.Error("inline workflow is missing its steps")
	}
}

// Step 1 searches issues and prior rationale before any add, and reports
// what it found (xpo-ed312f).
func TestGuidedStep1SearchesBoardAndRationaleBeforeAdd(t *testing.T) {
	tpl, _ := LookupWorkflow("guided")
	s := tpl.Skill()
	start := strings.Index(s, "### 1. Check existing work")
	end := strings.Index(s, "### 2. Plan")
	if start == -1 || end == -1 {
		t.Fatal("guided workflow has no step 1")
	}
	step := flat(s[start:end])
	for _, want := range []string{
		"Before any `add`",
		"`list` with `match`",
		"`rationale`",
		"tell the user what the searches found, or that they found nothing",
	} {
		if !strings.Contains(step, want) {
			t.Errorf("step 1 is missing %q:\n%s", want, step)
		}
	}
}

func TestLookupWorkflow(t *testing.T) {
	if _, ok := LookupWorkflow("GUIDED"); !ok {
		t.Error("lookup should be case-insensitive")
	}
	if _, ok := LookupWorkflow("none"); ok {
		t.Error("none is not a template")
	}
	if _, ok := LookupWorkflow("nope"); ok {
		t.Error("unknown name should not resolve")
	}
}
