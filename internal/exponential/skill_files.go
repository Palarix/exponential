package exponential

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/palarix/exponential/internal/version"
)

// Generated skill files end with a marker line recording the xpo version and a
// hash of the content above it:
//
//	<!-- xpo:skill 1.2.1 sha256:0123456789ab -->
//
// The marker sits at the end so SKILL.md's YAML frontmatter stays first. The
// hash lets doctor/init tell a hand-edited file (hash mismatch) apart from one
// that is merely out of date (body differs from the current template).
var skillMarkerRe = regexp.MustCompile(`\n*<!-- xpo:skill(?:\s+(\S+))?\s+sha256:([a-f0-9]+) -->\s*$`)

// SkillFileState classifies an installed skill file against the current template.
type SkillFileState int

const (
	SkillFileMissing  SkillFileState = iota // file does not exist
	SkillFileUpToDate                       // body matches the current template
	SkillFileStale                          // body differs and was not edited by hand
	SkillFileEdited                         // marker hash does not match the body
)

func (s SkillFileState) String() string {
	switch s {
	case SkillFileMissing:
		return "missing"
	case SkillFileUpToDate:
		return "up to date"
	case SkillFileStale:
		return "stale"
	case SkillFileEdited:
		return "edited"
	}
	return fmt.Sprintf("SkillFileState(%d)", int(s))
}

// WrapSkillFile appends the xpo:skill marker to generated skill content.
func WrapSkillFile(content, version string) string {
	body := strings.TrimRight(content, " \t\n")
	return fmt.Sprintf("%s\n\n<!-- xpo:skill %s sha256:%s -->\n", body, version, ComputeBlockHash(body))
}

// ParseSkillFile splits a skill file into its body and marker fields. Files
// without a marker (installed before markers existed) return the whole content
// as the body and ok=false.
func ParseSkillFile(fileContent string) (body, version, hash string, ok bool) {
	loc := skillMarkerRe.FindStringSubmatchIndex(fileContent)
	if loc == nil {
		return fileContent, "", "", false
	}
	if loc[2] != -1 {
		version = fileContent[loc[2]:loc[3]]
	}
	hash = fileContent[loc[4]:loc[5]]
	return fileContent[:loc[0]], version, hash, true
}

// CheckSkillFile compares the skill file at path with the generated template.
// Content is compared rather than versions, so template changes shipped
// without a version bump are still detected.
func CheckSkillFile(path, generated string) SkillFileState {
	data, err := os.ReadFile(path)
	if err != nil {
		return SkillFileMissing
	}
	body, _, hash, ok := ParseSkillFile(string(data))
	if strings.TrimSpace(body) == strings.TrimSpace(generated) {
		return SkillFileUpToDate
	}
	if ok && hash != ComputeBlockHash(body) {
		return SkillFileEdited
	}
	return SkillFileStale
}

// skillFile is one generated file of the xpo skill.
type skillFile struct {
	Path    string
	Content string
}

// generatedSkillFiles lists the files that make up the xpo skill in skillDir.
func generatedSkillFiles(skillDir string) []skillFile {
	refsDir := filepath.Join(skillDir, "references")
	return []skillFile{
		{filepath.Join(skillDir, "SKILL.md"), GenerateSkillMD()},
		{filepath.Join(refsDir, "spec-guide.md"), GenerateSpecGuide()},
		{filepath.Join(refsDir, "mcp-tools.md"), GenerateMCPToolsRef()},
	}
}

// SkillFileStatus is the state of one installed skill file.
type SkillFileStatus struct {
	Path  string
	State SkillFileState
}

// AgentSkillStates reports the state of each file of the agent's local skill
// install. Returns nil for agents without skill support.
func AgentSkillStates(agent AgentConfig) []SkillFileStatus {
	if agent.SkillDir == "" {
		return nil
	}
	var out []SkillFileStatus
	for _, f := range generatedSkillFiles(filepath.Join(agent.SkillDir, "xpo")) {
		out = append(out, SkillFileStatus{Path: f.Path, State: CheckSkillFile(f.Path, f.Content)})
	}
	return out
}

// EditedSkillFile describes a skill file that was left alone because it has
// local edits. Contents are marker-free bodies, for diff display.
type EditedSkillFile struct {
	Path       string
	OldContent string
	NewContent string
}

// SkillWriteResult describes what WriteAgentSkill did.
type SkillWriteResult struct {
	Dir    string            // skill directory, empty if the agent has no skill support
	Edited []EditedSkillFile // files skipped because they have local edits
}

// ReplaceSkillFile overwrites an edited skill file with the current template.
func ReplaceSkillFile(f EditedSkillFile) error {
	return os.WriteFile(f.Path, []byte(WrapSkillFile(f.NewContent, version.CLIVersion)), 0644)
}
