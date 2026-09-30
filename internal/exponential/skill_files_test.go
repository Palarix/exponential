package exponential

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/palarix/exponential/internal/version"
)

// --- Step 9 wording ---

func step9(t *testing.T) string {
	t.Helper()
	s := GenerateSkillMD()
	start := strings.Index(s, "### 9. Complete")
	if start == -1 {
		t.Fatal("SKILL.md has no step 9")
	}
	end := strings.Index(s[start:], "\n---\n")
	if end == -1 {
		t.Fatal("step 9 has no terminating rule")
	}
	return s[start : start+end]
}

// Bug: step 9 said "commit, then merge" and "do not use manual git commands to
// commit" in the same paragraph, so agents skipped the commit and merge failed.
func TestSkillMD_Step9_CommitsBeforeMerge(t *testing.T) {
	s := step9(t)

	commit := strings.Index(s, "git add -A && git commit")
	merge := strings.Index(s, "`merge` tool")
	if commit == -1 {
		t.Fatal("step 9 must tell the agent to commit")
	}
	if merge == -1 || commit > merge {
		t.Fatal("step 9 must commit before calling merge")
	}
	if strings.Contains(s, "Do not use manual git commands to merge or commit") {
		t.Fatal("step 9 must not forbid committing")
	}
}

func TestSkillMD_Step9_GateRequiresCommit(t *testing.T) {
	s := step9(t)
	gates := s[:strings.Index(s, "If any are missing")]
	if !strings.Contains(gates, "4. All changes are committed") {
		t.Fatalf("gate list must require committed changes, got:\n%s", gates)
	}
}

func TestSkillMD_Step9_ChangelogIsGeneric(t *testing.T) {
	s := step9(t)
	if !strings.Contains(s, "If the project's instructions require a changelog entry") {
		t.Fatal("step 9 must mention the optional changelog entry before committing")
	}
}

func TestMCPToolsRef_MergeRequiresCommit(t *testing.T) {
	ref := GenerateMCPToolsRef()
	for _, line := range strings.Split(ref, "\n") {
		if strings.HasPrefix(line, "| `merge`") {
			if !strings.Contains(line, "committed") {
				t.Fatalf("merge row must mention the commit requirement: %s", line)
			}
			return
		}
	}
	t.Fatal("mcp-tools reference has no merge row")
}

// --- Marker ---

func TestWrapSkillFile_RoundTrip(t *testing.T) {
	body := GenerateSkillMD()
	wrapped := WrapSkillFile(body, "9.9.9")

	if !strings.HasPrefix(wrapped, "---\n") {
		t.Fatal("frontmatter must stay first in a wrapped SKILL.md")
	}

	gotBody, ver, hash, ok := ParseSkillFile(wrapped)
	if !ok {
		t.Fatal("marker not found in wrapped file")
	}
	if ver != "9.9.9" {
		t.Fatalf("version = %q, want 9.9.9", ver)
	}
	if hash != ComputeBlockHash(body) {
		t.Fatalf("hash = %q, want %q", hash, ComputeBlockHash(body))
	}
	if strings.TrimSpace(gotBody) != strings.TrimSpace(body) {
		t.Fatal("body did not round-trip")
	}
}

func TestParseSkillFile_NoMarker(t *testing.T) {
	body, _, _, ok := ParseSkillFile("# legacy\n\ncontent\n")
	if ok {
		t.Fatal("legacy file must report no marker")
	}
	if body != "# legacy\n\ncontent\n" {
		t.Fatalf("legacy body should be returned unchanged, got %q", body)
	}
}

// --- State ---

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestCheckSkillFile(t *testing.T) {
	dir := t.TempDir()
	current := "# Skill\n\nnew text\n"
	old := "# Skill\n\nold text\n"

	cases := []struct {
		name    string
		content *string
		want    SkillFileState
	}{
		{"missing", nil, SkillFileMissing},
		{"up to date", ptr(WrapSkillFile(current, "1.0.0")), SkillFileUpToDate},
		{"up to date at older version", ptr(WrapSkillFile(current, "0.1.0")), SkillFileUpToDate},
		{"stale", ptr(WrapSkillFile(old, "1.0.0")), SkillFileStale},
		{"legacy without marker", ptr(old), SkillFileStale},
		{"legacy matching template", ptr(current), SkillFileUpToDate},
		{"edited", ptr(strings.Replace(WrapSkillFile(old, "1.0.0"), "old text", "my text", 1)), SkillFileEdited},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(dir, strings.ReplaceAll(tc.name, " ", "-")+".md")
			if tc.content != nil {
				writeFile(t, path, *tc.content)
			}
			if got := CheckSkillFile(path, current); got != tc.want {
				t.Fatalf("CheckSkillFile = %v, want %v", got, tc.want)
			}
		})
	}
}

func ptr(s string) *string { return &s }

// --- Writing ---

func skillPath(agent AgentConfig) string {
	return filepath.Join(agent.SkillDir, "xpo", "SKILL.md")
}

func TestWriteAgentSkill_WritesMarker(t *testing.T) {
	setupProject(t, "test")
	if _, err := WriteAgentSkill(testClaudeCode, "", false); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{
		skillPath(testClaudeCode),
		filepath.Join(testClaudeCode.SkillDir, "xpo", "references", "spec-guide.md"),
		filepath.Join(testClaudeCode.SkillDir, "xpo", "references", "mcp-tools.md"),
	} {
		data, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		if _, ver, _, ok := ParseSkillFile(string(data)); !ok || ver != version.CLIVersion {
			t.Fatalf("%s: marker missing or wrong version (%q)", p, ver)
		}
	}
}

func TestWriteAgentSkill_OverwritesStaleAndLegacy(t *testing.T) {
	setupProject(t, "test")
	path := skillPath(testClaudeCode)

	for _, content := range []string{WrapSkillFile("old", "0.1.0"), "legacy without marker"} {
		writeFile(t, path, content)
		res, err := WriteAgentSkill(testClaudeCode, "", false)
		if err != nil {
			t.Fatal(err)
		}
		if len(res.Edited) != 0 {
			t.Fatalf("stale file reported as edited: %v", res.Edited)
		}
		if got := CheckSkillFile(path, GenerateSkillMD()); got != SkillFileUpToDate {
			t.Fatalf("after write state = %v, want up to date", got)
		}
	}
}

func TestWriteAgentSkill_EditedSkippedWithoutForce(t *testing.T) {
	setupProject(t, "test")
	path := skillPath(testClaudeCode)
	edited := strings.Replace(WrapSkillFile("old body", "0.1.0"), "old body", "my body", 1)
	writeFile(t, path, edited)

	res, err := WriteAgentSkill(testClaudeCode, "", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Edited) != 1 || res.Edited[0].Path != path {
		t.Fatalf("expected %s reported as edited, got %v", path, res.Edited)
	}
	if res.Edited[0].OldContent != "my body" || res.Edited[0].NewContent != strings.TrimSpace(GenerateSkillMD()) {
		t.Fatal("edited result must carry old and new bodies for the diff")
	}
	data, _ := os.ReadFile(path)
	if string(data) != edited {
		t.Fatal("edited file must not be overwritten without force")
	}

	// The other files are still refreshed.
	ref := filepath.Join(testClaudeCode.SkillDir, "xpo", "references", "mcp-tools.md")
	if CheckSkillFile(ref, GenerateMCPToolsRef()) != SkillFileUpToDate {
		t.Fatal("unedited files must still be written")
	}
}

func TestWriteAgentSkill_EditedOverwrittenWithForce(t *testing.T) {
	setupProject(t, "test")
	path := skillPath(testClaudeCode)
	writeFile(t, path, strings.Replace(WrapSkillFile("old body", "0.1.0"), "old body", "my body", 1))

	res, err := WriteAgentSkill(testClaudeCode, "", true)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Edited) != 0 {
		t.Fatalf("force must not report edits, got %v", res.Edited)
	}
	if CheckSkillFile(path, GenerateSkillMD()) != SkillFileUpToDate {
		t.Fatal("force must overwrite the edited file")
	}
}

// --- Health ---

// Bug: template changes shipped without a version bump never reached existing
// projects because doctor only compared CLI versions.
func TestHealth_StaleSkill_SameVersion_IsAutoFix(t *testing.T) {
	setupProject(t, "test")
	setupAgentFull(t, testClaudeCode, "test")
	writeFile(t, skillPath(testClaudeCode), WrapSkillFile("older template", version.CLIVersion))

	h := CheckAgentHealth(testClaudeCode, version.CLIVersion)

	if !containsStr(h.AutoFix, "Skill out of date") {
		t.Fatalf("expected 'Skill out of date' in AutoFix, got %v", h.AutoFix)
	}
	if len(h.Interactive) > 0 {
		t.Fatalf("stale skill must not be interactive, got %v", h.Interactive)
	}
}

func TestHealth_EditedSkill_IsInteractive(t *testing.T) {
	setupProject(t, "test")
	setupAgentFull(t, testClaudeCode, "test")
	path := skillPath(testClaudeCode)
	data, _ := os.ReadFile(path)
	writeFile(t, path, strings.Replace(string(data), "### 1. Check existing work", "### 1. My own step", 1))

	h := CheckAgentHealth(testClaudeCode, version.CLIVersion)

	if !h.SkillEdited() {
		t.Fatal("expected SkillEdited")
	}
	if !containsSubstr(h.Interactive, "skill has local edits") {
		t.Fatalf("expected skill edit in Interactive, got %v", h.Interactive)
	}
	if containsStr(h.AutoFix, "Skill out of date") {
		t.Fatalf("edited skill must not be auto-fixed, got %v", h.AutoFix)
	}
}

func TestHealth_EditedBlockOnly_SkillNotEdited(t *testing.T) {
	setupProject(t, "test")
	setupAgentFull(t, testClaudeCode, "test")

	h := CheckAgentHealth(testClaudeCode, version.CLIVersion)
	if h.SkillEdited() || h.BlockEdited {
		t.Fatal("fresh setup must report no edits")
	}
}

// --- Plan ---

func TestChangePlan_SkipsUpToDateSkill(t *testing.T) {
	setupProject(t, "test")
	setupAgentFull(t, testClaudeCode, "test")

	for _, c := range ComputeInitPlan([]AgentConfig{testClaudeCode}).Changes {
		if strings.HasPrefix(c.Path, ".claude/skills/") {
			t.Fatalf("up-to-date skill must not be in the plan, found %s", c.Path)
		}
	}

	writeFile(t, skillPath(testClaudeCode), "legacy")
	found := false
	for _, c := range ComputeInitPlan([]AgentConfig{testClaudeCode}).Changes {
		if c.Path == skillPath(testClaudeCode) && c.Action == "modify" {
			found = true
		}
	}
	if !found {
		t.Fatal("stale skill must be listed as modify")
	}
}

func containsStr(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func containsSubstr(list []string, s string) bool {
	for _, v := range list {
		if strings.Contains(v, s) {
			return true
		}
	}
	return false
}
