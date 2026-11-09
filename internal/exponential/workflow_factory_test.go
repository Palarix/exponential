package exponential

import (
	"path/filepath"
	"strings"
	"testing"
)

// The factory workflow template (xpo-82ee77).

func factory(t *testing.T) *WorkflowTemplate {
	t.Helper()
	tpl, ok := LookupWorkflow("factory")
	if !ok {
		t.Fatal("factory not registered")
	}
	return &tpl
}

func TestFactory_Registered(t *testing.T) {
	if WorkflowTemplates[0].Name != "guided" {
		t.Fatal("guided stays first, the default")
	}
	tpl := factory(t)
	if tpl.Title != "Factory" {
		t.Errorf("title = %q", tpl.Title)
	}
	if !strings.Contains(tpl.Description, "sub-agents") {
		t.Errorf("init prompt should say it needs sub-agents: %q", tpl.Description)
	}
	found := false
	for _, n := range WorkflowNames() {
		found = found || n == "factory"
	}
	if !found {
		t.Error("factory should be an accepted --workflow value")
	}
}

func TestFactory_References(t *testing.T) {
	if refs := guided(t).References; refs != nil && len(refs()) != 0 {
		t.Error("guided has no reference files")
	}
	refs := factory(t).References()
	for _, name := range []string{"open-questions.md", "code-review.md"} {
		if refs[name] == "" {
			t.Errorf("factory is missing references/%s", name)
		}
	}
}

func TestFactory_Skill(t *testing.T) {
	s := factory(t).Skill()
	if !strings.HasPrefix(s, "---\nname: xpo-workflow\n") {
		t.Errorf("factory skill should start with xpo-workflow frontmatter:\n%s", s[:80])
	}
	for _, h := range []string{
		"### Requirements",
		"### Rules",
		"### Public by default",
		"### Planning model",
		"### When to stop",
		"### 1. Orient",
		"### 2. Plan the milestone",
		"### 3. Spec",
		"### 4. Start",
		"### 5. Implement & test",
		"### 6. Adversarial review",
		"### 7. Spec stays true",
		"### 8. Walkthrough, commit, merge",
		"### 9. Next issue, or hand off",
		"### 10. Milestone completion",
		"### After the tophat",
		"### Backlog Review",
		"### Prioritization",
	} {
		if !strings.Contains(s, h) {
			t.Errorf("factory skill is missing %q", h)
		}
	}
	f := flat(s)
	for _, want := range []string{
		"load the `xpo` skill",
		"`references/open-questions.md`",
		"`references/code-review.md`",
		"a fresh sub-agent, not a fork of your context",
		"Agent Decisions",
	} {
		if !strings.Contains(f, want) {
			t.Errorf("factory skill is missing %q", want)
		}
	}
}

func TestFactory_RequirementsAndRules(t *testing.T) {
	s := factory(t).Skill()
	req := section(t, s, "### Requirements", "\n### ")
	for _, want := range []string{"start a sub-agent with a fresh context", "stop and tell the user"} {
		if !strings.Contains(req, want) {
			t.Errorf("Requirements is missing %q:\n%s", want, req)
		}
	}
	rules := section(t, s, "### Rules", "\n### ")
	for _, want := range []string{
		"Every code change must be backed by an xpo issue",
		"A milestone starts only after the human has approved its backlog",
		"BACKLOG issue outside the started milestone",
		"Bugs found during implementation need no approval",
		"link it back",
	} {
		if !strings.Contains(rules, want) {
			t.Errorf("Rules is missing %q:\n%s", want, rules)
		}
	}
}

func TestFactory_PlanningModel(t *testing.T) {
	s := factory(t).Skill()
	f := flat(s)
	if strings.Contains(strings.ToLower(f), "milestone epic") {
		t.Error("the factory uses no milestone epic")
	}
	model := section(t, s, "### Planning model", "\n### ")
	for _, want := range []string{
		"**Done when**",
		"5 story points or fewer",
		"over 5 story points becomes an `epic`",
		"child issues",
		"Do not start an epic",
		"BACKLOG issue in no milestone",
	} {
		if !strings.Contains(model, want) {
			t.Errorf("Planning model is missing %q:\n%s", want, model)
		}
	}

	plan := section(t, s, "### 2. Plan the milestone", "\n### ")
	for _, want := range []string{
		"has no issues",
		"stories in BACKLOG",
		"set every BACKLOG issue in it to PLANNED",
		"as PLANNED in the milestone",
	} {
		if !strings.Contains(plan, want) {
			t.Errorf("step 2 is missing %q:\n%s", want, plan)
		}
	}

	orient := section(t, s, "### 1. Orient", "\n### ")
	if !strings.Contains(orient, "no story in it is PLANNED or DOING") {
		t.Errorf("step 1 should read awaiting-tophat from the board:\n%s", orient)
	}

	stop := section(t, s, "### When to stop", "\n### ")
	for _, want := range []string{"approve the milestone backlog", "milestone completion", "BLOCKED"} {
		if !strings.Contains(stop, want) {
			t.Errorf("When to stop is missing %q:\n%s", want, stop)
		}
	}

	tophat := section(t, s, "### 10. Milestone completion", "\n### ")
	for _, want := range []string{"**Delivered:**", "**How to tophat:**", "**Agent Decisions:**", "**Proposed decision records:**", "**Deferred:**"} {
		if !strings.Contains(tophat, want) {
			t.Errorf("tophat report is missing %q:\n%s", want, tophat)
		}
	}
	after := section(t, s, "### After the tophat", "\n### ")
	if !strings.Contains(after, "`complete`") {
		t.Errorf("after approval the milestone is completed:\n%s", after)
	}
}

func TestFactory_ReviewRounds(t *testing.T) {
	review := section(t, factory(t).Skill(), "### 6. Adversarial review", "\n### ")
	for _, want := range []string{
		"3 rounds", "BLOCKED", "Do not give it your own assessment", "the project's test and lint commands",
		"Only a **blocker** starts another round",
		"fix it or file it as a follow-up issue",
		"not realistic in normal use of the product",
		"not valid in the strict areas",
		"only the fixes and the code they touched",
		"the fix diff",
		"Do not ask it to review the whole change again",
	} {
		if !strings.Contains(review, want) {
			t.Errorf("step 6 is missing %q:\n%s", want, review)
		}
	}
	for _, gone := range []string{"Fix every **blocker** and **major**", "a preference"} {
		if strings.Contains(review, gone) {
			t.Errorf("step 6 still contains %q", gone)
		}
	}
	tophat := section(t, factory(t).Skill(), "### 10. Milestone completion", "\n### ")
	if !strings.Contains(tophat, "rejected review findings, with reasons") {
		t.Errorf("the tophat report lists rejected findings:\n%s", tophat)
	}
}

func TestFactory_CodeReviewIsCalibrated(t *testing.T) {
	cr := flat(factory(t).References()["code-review.md"])
	for _, want := range []string{
		"A false alarm has a cost",
		"**PASS** with no findings is a normal result",
		"realistic trigger",
		"in normal use of the product",
		"several unlikely conditions",
		"**Strict areas:**",
		"authentication", "permissions", "secrets", "data loss",
		"need no realistic trigger",
		"[ROUND 2+ ONLY:",
		"[FIX DIFF]",
		"Check only these fixes and the code they touched",
	} {
		if !strings.Contains(cr, want) {
			t.Errorf("code-review brief is missing %q", want)
		}
	}
	for _, gone := range []string{"a problem you miss ships", "inputs it does not", "find what is wrong with it"} {
		if strings.Contains(cr, gone) {
			t.Errorf("code-review brief still contains %q", gone)
		}
	}
	// The mechanical checks come before the judgment checks.
	if strings.Index(cr, "**Acceptance criteria.**") > strings.Index(cr, "**Correctness.**") {
		t.Error("mechanical checks should come first")
	}
}

// factoryDocs is everything the factory template writes.
func factoryDocs(t *testing.T) map[string]string {
	tpl := factory(t)
	docs := map[string]string{"skill": tpl.Skill(), "inline": tpl.Inline()}
	for name, c := range tpl.References() {
		docs[name] = c
	}
	return docs
}

func TestFactory_NothingProjectSpecific(t *testing.T) {
	for name, doc := range factoryDocs(t) {
		for _, gone := range []string{
			"make test", "make lint", "docs/adr", "plan.md", "CLAUDE.md", "ADR 0", "ADR",
			"fabulous", "general-purpose", "both themes", "internal/", "web/src",
			"Claude Code", "Codex", "OpenCode", "Cursor",
			"metadata:", "<!-- xpo:skill", "## Reference",
			"prioritis", "Prioritis", "behaviour",
		} {
			if strings.Contains(doc, gone) {
				t.Errorf("%s contains project-specific or non-generic text %q", name, gone)
			}
		}
	}
	f := flat(factory(t).Skill())
	for _, want := range []string{
		"the project's test and lint commands",
		"the project's instructions",
		"the project's decision records",
		"If the project keeps no decision records",
	} {
		if !strings.Contains(f, want) {
			t.Errorf("factory skill is missing %q", want)
		}
	}
}

func TestFactory_Briefs(t *testing.T) {
	refs := factory(t).References()
	oq := flat(refs["open-questions.md"])
	for _, want := range []string{"[ISSUE-ID]", "[SPEC TEXT]", "[MILESTONE GOAL]", "[NUMBERED QUESTIONS]",
		"**Answer:**", "**Evidence:**", "**Alternatives rejected:**", "**Confidence:**", "**Human's call?**",
		"Do not edit any file"} {
		if !strings.Contains(oq, want) {
			t.Errorf("open-questions brief is missing %q", want)
		}
	}
	cr := flat(refs["code-review.md"])
	for _, want := range []string{"[ISSUE-ID]", "[WORKTREE PATH]", "[BASE SHA]", "[SPEC TEXT]", "[MILESTONE GOAL]",
		"**Acceptance criteria.**", "**Correctness.**", "**Decision records and project rules.**", "**Tests.**", "**Scope.**",
		"the project's test and lint commands", "**Verdict:**", "**AC table:**", "blocker", "major", "minor",
		"Do not edit any file", "not a fork"} {
		if !strings.Contains(cr, want) {
			t.Errorf("code-review brief is missing %q", want)
		}
	}
	// Mechanical checks first (no realistic trigger needed), then judgment.
	order := []string{"**Acceptance criteria.**", "**Decision records and project rules.**", "**Tests.**", "**Correctness.**", "**Scope.**"}
	for i := 1; i < len(order); i++ {
		if strings.Index(cr, order[i-1]) > strings.Index(cr, order[i]) {
			t.Errorf("%s should come before %s", order[i-1], order[i])
		}
	}
}

func TestFactory_Inline(t *testing.T) {
	s := factory(t).Inline()
	if !strings.HasPrefix(s, "## Workflow\n") {
		t.Errorf("inline factory should start with a Workflow heading:\n%s", s[:40])
	}
	for _, want := range []string{"### Requirements", "### 10. Milestone completion", "### Brief: open questions", "### Brief: adversarial code review", "the xpo section above"} {
		if !strings.Contains(s, want) {
			t.Errorf("inline factory is missing %q", want)
		}
	}
	for _, gone := range []string{"references/", "name: xpo-workflow", "load the `xpo` skill"} {
		if strings.Contains(s, gone) {
			t.Errorf("inline factory should not contain %q", gone)
		}
	}
	if strings.Index(s, "### Brief: open questions") < strings.Index(s, "### Prioritization") {
		t.Error("the briefs come after the workflow")
	}
	// The briefs' own "---" separators must not leak into the inline section.
	if strings.Contains(s, "\n---\n") {
		t.Error("inline factory should not contain a horizontal rule")
	}
}

func TestFactory_InstallWritesReferences(t *testing.T) {
	setupProject(t, "test")
	if _, err := InstallWorkflow(testClaudeCode, factory(t)); err != nil {
		t.Fatal(err)
	}
	refDir := filepath.Join(testClaudeCode.SkillDir, "xpo-workflow", "references")
	for name, content := range factory(t).References() {
		if got := readFile(t, filepath.Join(refDir, name)); got != content {
			t.Errorf("references/%s differs from the template", name)
		}
	}
	plan := ComputeInitPlan([]AgentConfig{testCodex}, factory(t))
	for name := range factory(t).References() {
		p := filepath.Join(testCodex.SkillDir, "xpo-workflow", "references", name)
		if !planHas(plan, p, "create") {
			t.Errorf("plan should list %s, got %v", p, plan.Changes)
		}
	}
}

func TestFactory_SetupWritesReferences(t *testing.T) {
	setupProject(t, "test")
	writeFile(t, workflowPath(testClaudeCode), "old")
	if _, err := SetupWorkflow([]AgentConfig{testClaudeCode}, factory(t)); err != nil {
		t.Fatal(err)
	}
	if readFile(t, workflowPath(testClaudeCode)) != factory(t).Skill() {
		t.Fatal("skill should be the factory template")
	}
	for name := range factory(t).References() {
		if !exists(filepath.Join(testClaudeCode.SkillDir, "xpo-workflow", "references", name)) {
			t.Errorf("references/%s not written", name)
		}
	}
}
