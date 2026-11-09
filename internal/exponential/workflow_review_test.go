package exponential

import (
	"strings"
	"testing"
)

// Review round on xpo-f19bfa: the reference says how, the workflow says when.

func section(t *testing.T, s, from, to string) string {
	t.Helper()
	start := strings.Index(s, from)
	if start == -1 {
		t.Fatalf("no %q in:\n%s", from, s)
	}
	rest := s[start+len(from):]
	end := len(rest)
	if to != "" {
		if i := strings.Index(rest, to); i != -1 {
			end = i
		}
	}
	return flat(rest[:end])
}

func TestReference_DescriptionAndIntro(t *testing.T) {
	s := GenerateSkillMD()
	if !strings.Contains(s, "description: >-\n  How the xpo (Exponential) MCP tools work") {
		t.Errorf("description should say how:\n%s", s[:300])
	}
	if strings.Contains(s, "metadata:") || strings.Contains(s, "version:") {
		t.Error("the xpo:skill marker carries the version; no metadata block")
	}
	if !strings.Contains(flat(s), "How the xpo tools behave: statuses, labels, links, milestones, specs, walkthroughs, start and merge. When to use them is the project's call: if the project has an `xpo-workflow` skill, load it and follow it.") {
		t.Error("intro is missing")
	}
}

var referenceSections = []string{"Status Graph", "Finding Work", "Issue Creation", "Issue Labels", "Linking", "Milestones", "Specs", "Walkthroughs", "Starting Work", "Merging"}

func TestReference_HeadingLevels(t *testing.T) {
	skill := GenerateSkillMD()
	docs := GenerateAgentDocs("xpo")
	for _, name := range referenceSections {
		if !strings.Contains(skill, "\n## "+name+"\n") {
			t.Errorf("skill should have ## %s", name)
		}
		if !strings.Contains(docs, "\n### "+name+"\n") {
			t.Errorf("inline docs should have ### %s", name)
		}
	}
}

func TestReference_NoWhenContent(t *testing.T) {
	for name, doc := range map[string]string{
		"skill":      GenerateSkillMD(),
		"docs":       GenerateAgentDocs("xpo"),
		"spec-guide": GenerateSpecGuide(),
	} {
		d := flat(doc)
		for _, gone := range []string{
			"update the spec first",
			"search ¦rationale¦ first",
			"Work Mode",
			"Only `blocked_by` prevents",
			"non-interactive",
			"Non-interactive",
			"stop and ask",
		} {
			if strings.Contains(d, md(gone)) {
				t.Errorf("%s still contains %q", name, gone)
			}
		}
	}
}

func TestReference_LinkingAndStartDescribeTheGate(t *testing.T) {
	s := GenerateSkillMD()
	linking := section(t, s, "## Linking", "\n## ")
	for _, want := range []string{
		"Moving an issue to DOING is refused while an issue that blocks it is not DONE",
		"whichever side stores the link",
		"a workflow may gate on them too",
	} {
		if !strings.Contains(linking, want) {
			t.Errorf("Linking is missing %q:\n%s", want, linking)
		}
	}
	start := section(t, s, "## Starting Work", "\n## ")
	if !strings.Contains(start, "`start` refuses issues in BACKLOG or BLOCKED") {
		t.Errorf("Starting Work should say what start refuses:\n%s", start)
	}
}

func TestReference_Milestones(t *testing.T) {
	m := section(t, GenerateSkillMD(), "## Milestones", "\n## ")
	for _, want := range []string{"`milestone`", "Current", "Next", "`@current`", "copies its parent's open milestone", "complete"} {
		if !strings.Contains(m, want) {
			t.Errorf("Milestones is missing %q:\n%s", want, m)
		}
	}
}

func TestReference_LinksToolList(t *testing.T) {
	f := section(t, GenerateSkillMD(), "## Finding Work", "\n## ")
	if !strings.Contains(f, "`references/mcp-tools.md`") {
		t.Errorf("Finding Work should link the tool list:\n%s", f)
	}
}

func TestMCPToolsRef_ListsAllTools(t *testing.T) {
	ref := GenerateMCPToolsRef()
	for _, tool := range []string{"list", "show", "add", "update", "start", "merge", "comment", "link", "unlink", "history", "spec", "walkthrough", "rationale", "artifact", "milestone", "git_status", "git_diff", "git_log", "git_show", "code_history"} {
		if !strings.Contains(ref, "| `"+tool+"` |") {
			t.Errorf("mcp-tools.md is missing %s", tool)
		}
	}
	for _, line := range strings.Split(ref, "\n") {
		if strings.HasPrefix(line, "| `update`") {
			for _, st := range []string{"BACKLOG", "PLANNED", "DOING", "BLOCKED", "DONE", "CANCELED", "DUPLICATE"} {
				if !strings.Contains(line, st) {
					t.Errorf("update row is missing %s: %s", st, line)
				}
			}
		}
	}
}

func TestSpecGuide_NeutralWording(t *testing.T) {
	g := flat(GenerateSpecGuide())
	for _, want := range []string{
		"when no user is present to answer",
		"or record them as Agent Decisions when no user is present",
	} {
		if !strings.Contains(g, want) {
			t.Errorf("spec guide is missing %q", want)
		}
	}
}

func TestUSSpelling(t *testing.T) {
	docs := map[string]string{
		"skill":      GenerateSkillMD(),
		"spec-guide": GenerateSpecGuide(),
	}
	for _, tpl := range WorkflowTemplates {
		docs[tpl.Name] = tpl.Skill()
		docs[tpl.Name+" inline"] = tpl.Inline()
		if tpl.References != nil {
			for file, c := range tpl.References() {
				docs[tpl.Name+" "+file] = c
			}
		}
	}
	for name, doc := range docs {
		if strings.Contains(strings.ToLower(doc), "prioritis") {
			t.Errorf("%s uses UK spelling", name)
		}
	}
}

func TestGuided_RulesAndSteps(t *testing.T) {
	tpl, _ := LookupWorkflow("guided")
	s := tpl.Skill()

	rules := section(t, s, "### Rules", "\n### ")
	for _, want := range []string{
		"Never start work on a BACKLOG issue without explicit user approval",
		"in the scope of the current issue",
		"out of scope",
		"link it back",
	} {
		if !strings.Contains(rules, want) {
			t.Errorf("Rules is missing %q:\n%s", want, rules)
		}
	}

	if step4 := section(t, s, "### 4. Start", "\n### "); strings.Contains(step4, "BACKLOG") {
		t.Errorf("step 4 should not repeat the BACKLOG rule:\n%s", step4)
	}
	if step5 := section(t, s, "### 5. Implement & Test", "\n### "); !strings.Contains(step5, "Rule 3") {
		t.Errorf("step 5 should defer bugs to Rule 3:\n%s", step5)
	}

	review := section(t, s, "### Backlog Review", "\n### ")
	if !strings.Contains(review, "no recent event in its `history`") {
		t.Errorf("Backlog Review should define stale:\n%s", review)
	}

	public := section(t, s, "### Public by default", "\n### ")
	if !strings.Contains(public, `"another project"`) || strings.Contains(public, "forked the skill") {
		t.Errorf("Public by default example should be generic:\n%s", public)
	}
}
