package chat

import (
	"encoding/json"
	"path/filepath"
)

// Chat modes. A chat starts in Chat Mode; Work Mode implements one issue in
// its worktree. Only the user changes the mode, and only between turns.
const (
	ModeChat = "chat"
	ModeWork = "work"
)

// WorkBinding is the issue a chat in Work Mode works on, and its worktree.
type WorkBinding struct {
	IssueID  string `json:"issue_id"`
	Worktree string `json:"worktree"`
}

// Profile is what one turn may do: its working directory and, per agent,
// its tools and the calls that ask the user first. The runner builds it from
// the chat's Meta before each turn.
type Profile struct {
	Mode    string
	Dir     string // the agent's working directory
	IssueID string // Work Mode: the bound issue
	// CanWork lets a Chat Mode turn call start, behind an approval card.
	// False in remote mode, where start runs on the server (xpo-a60065).
	CanWork bool
}

// ChatProfile reads the project at root and changes issues. merge is
// denied; start asks, and entering Work Mode follows it.
func ChatProfile(root string, canWork bool) Profile {
	return Profile{Mode: ModeChat, Dir: root, CanWork: canWork}
}

// WorkProfile implements issueID in worktree. merge asks.
func WorkProfile(issueID, worktree string) Profile {
	return Profile{Mode: ModeWork, Dir: worktree, IssueID: issueID}
}

func (p Profile) work() bool { return p.Mode == ModeWork }

// askTools are the xpo tools that ask the user before they run.
func (p Profile) askTools() []string {
	switch {
	case p.work():
		return []string{"merge"}
	case p.CanWork:
		return []string{"start"}
	}
	return nil
}

// deniedTools are the xpo tools the agent cannot call at all.
func (p Profile) deniedTools() []string {
	switch {
	case p.work():
		return nil
	case p.CanWork:
		return []string{"merge"}
	}
	return []string{"start", "merge"}
}

func mcpNames(tools []string) []string {
	out := make([]string, len(tools))
	for i, t := range tools {
		out[i] = "mcp__xpo__" + t
	}
	return out
}

// Skill, in both modes, lets the agent load the project's xpo workflow.
func (p Profile) claudeTools() string {
	if p.work() {
		return "Read,Grep,Glob,Edit,Write,Bash,Skill"
	}
	return "Read,Grep,Glob,Skill"
}

// claudeSettings holds the ask rules. An ask rule wins over the xpo
// wildcard in the allow list.
func (p Profile) claudeSettings() string {
	ask := mcpNames(p.askTools())
	data, _ := json.Marshal(map[string]any{"permissions": map[string]any{"ask": ask}})
	return string(data)
}

// claudeAllowed is the allow list. Loading a skill only adds instructions,
// so it never asks. In Work Mode, edits inside the worktree
// and every Bash command run without a card. "//" starts an absolute path in
// a Claude Code rule, and an Edit rule covers Write too. The rule is added
// for the resolved path as well, so a symlinked worktree matches either way.
func (p Profile) claudeAllowed() []string {
	allowed := []string{"mcp__xpo__*", "Skill"}
	if !p.work() {
		return allowed
	}
	allowed = append(allowed, "Bash", "Edit(/"+p.Dir+"/**)")
	if real, err := filepath.EvalSymlinks(p.Dir); err == nil && real != p.Dir {
		allowed = append(allowed, "Edit(/"+real+"/**)")
	}
	return allowed
}

func (p Profile) claudeDisallowed() []string { return mcpNames(p.deniedTools()) }

// codexSandbox keeps Chat Mode read-only. In Work Mode Codex may write in
// the worktree, its working directory.
func (p Profile) codexSandbox() string {
	if p.work() {
		return "workspace-write"
	}
	return "read-only"
}

// codexApprovalPolicy is "on-request" in both modes. Under "never", Codex
// fails an MCP tool whose approval mode is "prompt" instead of asking
// (codex-cli 0.162.1). It also lets a turn ask before it leaves the sandbox.
func (p Profile) codexApprovalPolicy() string { return "on-request" }

// contextLines tell the agent the turn's mode in the <xpo-context> block.
// The mode lives in the message, not the system prompt, so a mode change
// keeps the system prompt, and the prompt cache with it.
func (p Profile) contextLines() []string {
	if p.work() {
		return []string{"mode: work on " + p.IssueID, "worktree: " + p.Dir}
	}
	return []string{"mode: chat"}
}
