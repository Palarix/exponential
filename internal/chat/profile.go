package chat

import (
	"encoding/json"
	"path/filepath"
	"strings"
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
	Root    string // Work Mode: the project root, where .xpo/artifacts lives
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

// WorkProfile implements issueID in worktree. root is the project root that
// holds the issue's artifacts. merge asks.
func WorkProfile(issueID, worktree, root string) Profile {
	return Profile{Mode: ModeWork, Dir: worktree, Root: root, IssueID: issueID}
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

// Claude Code's built-in tools beyond the code tools (xpo-80bf05). Both
// modes look things up: WebSearch, WebFetch, and LSP for code navigation.
// Work Mode also keeps a task list and starts subagents. A subagent gets only
// the turn's tools, and its permission asks reach the user as cards.
// "Agent" is the subagent tool; Claude Code 2.1.300 also accepts "Task".
var (
	claudeLookupTools = []string{"WebSearch", "WebFetch", "LSP"}
	claudeWorkTools   = []string{"TaskCreate", "TaskUpdate", "TaskList", "TaskGet", "Agent"}
)

// Skill, in both modes, lets the agent load the project's xpo workflow.
func (p Profile) claudeTools() string {
	tools := []string{"Read", "Grep", "Glob"}
	if p.work() {
		tools = append(tools, "Edit", "Write", "Bash")
	}
	tools = append(tools, "Skill")
	tools = append(tools, claudeLookupTools...)
	if p.work() {
		tools = append(tools, claudeWorkTools...)
	}
	return strings.Join(tools, ",")
}

// claudeSettings holds the ask rules. An ask rule wins over the xpo
// wildcard in the allow list. WebFetch asks in both modes: the model picks
// the URL, so a fetch can leak data in it or bring back prompt injection.
func (p Profile) claudeSettings() string {
	ask := append(mcpNames(p.askTools()), "WebFetch")
	data, _ := json.Marshal(map[string]any{"permissions": map[string]any{"ask": ask}})
	return string(data)
}

// claudeAllowed is the allow list. Loading a skill only adds instructions,
// so it never asks. WebSearch only sends a query, and LSP only reads code.
// In Work Mode, the task list, subagents, edits inside the worktree, edits
// to the bound issue's artifacts (the spec tool hands out spec.md's path,
// outside the worktree), and every Bash command run without a card. "//"
// starts an absolute path in a Claude Code rule, and an Edit rule covers
// Write too.
func (p Profile) claudeAllowed() []string {
	allowed := []string{"mcp__xpo__*", "Skill", "WebSearch", "LSP"}
	if !p.work() {
		return allowed
	}
	allowed = append(allowed, claudeWorkTools...)
	allowed = append(allowed, "Bash")
	allowed = append(allowed, editRules(p.Dir, "")...)
	if p.Root != "" && p.IssueID != "" {
		allowed = append(allowed, editRules(p.Root, filepath.Join(".xpo", "artifacts", p.IssueID))...)
	}
	return allowed
}

// editRules allows edits under base/sub. The rule is added for the resolved
// base as well, so a symlinked dir matches either way. Only base is resolved:
// sub may not exist yet, such as an issue's artifacts before its first spec.
func editRules(base, sub string) []string {
	rules := []string{"Edit(/" + filepath.Join(base, sub) + "/**)"}
	if real, err := filepath.EvalSymlinks(base); err == nil && real != base {
		rules = append(rules, "Edit(/"+filepath.Join(real, sub)+"/**)")
	}
	return rules
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
