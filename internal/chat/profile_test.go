package chat

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// flagValues returns the values after flag up to the next flag.
func flagValues(args []string, flag string) []string {
	i := slices.Index(args, flag)
	if i < 0 {
		return nil
	}
	var out []string
	for _, a := range args[i+1:] {
		if strings.HasPrefix(a, "--") {
			break
		}
		out = append(out, a)
	}
	return out
}

// The built-in tools of each mode (xpo-80bf05). Chat Mode adds the web and
// LSP tools to the read-only ones; Work Mode adds editing, Bash, the task
// list, and subagents. Both modes ask the user questions (xpo-447687).
const (
	chatTools = "Read,Grep,Glob,Skill,WebSearch,WebFetch,LSP,AskUserQuestion"
	workTools = "Read,Grep,Glob,Edit,Write,Bash,Skill,WebSearch,WebFetch,LSP,AskUserQuestion,TaskCreate,TaskUpdate,TaskList,TaskGet,Agent"
)

func askRules(t *testing.T, args []string) []string {
	t.Helper()
	var settings struct {
		Permissions struct {
			Ask []string `json:"ask"`
		} `json:"permissions"`
	}
	if err := json.Unmarshal([]byte(flagValue(args, "--settings")), &settings); err != nil {
		t.Fatalf("--settings = %q: %v", flagValue(args, "--settings"), err)
	}
	return settings.Permissions.Ask
}

func TestChatProfile_Claude(t *testing.T) {
	args := claudeArgs(turnOptions{MCPConfigPath: "m", Profile: ChatProfile("/repo", true)})

	// Skill lets the agent load the project's xpo workflow (xpo-a058b4).
	if got := flagValue(args, "--tools"); got != chatTools {
		t.Errorf("--tools = %q", got)
	}
	if got := flagValues(args, "--allowedTools"); !slices.Equal(got, []string{"mcp__xpo__*", "Skill", "WebSearch", "LSP"}) {
		t.Errorf("--allowedTools = %v", got)
	}
	// A deny rule holds in every permission mode, so no later permission
	// picker can lift it.
	if got := flagValues(args, "--disallowedTools"); !slices.Equal(got, []string{"mcp__xpo__merge"}) {
		t.Errorf("--disallowedTools = %v, want merge denied", got)
	}
	// start asks: its approval card is the way into Work Mode. WebFetch
	// asks: the model picks the URL.
	if got := askRules(t, args); !slices.Equal(got, []string{"mcp__xpo__start", "WebFetch"}) {
		t.Errorf("ask rules = %v, want start and WebFetch to ask", got)
	}
}

func TestChatProfile_ClaudeWithoutWork(t *testing.T) {
	args := claudeArgs(turnOptions{MCPConfigPath: "m", Profile: ChatProfile("/repo", false)})
	got := flagValues(args, "--disallowedTools")
	if !slices.Contains(got, "mcp__xpo__merge") || !slices.Contains(got, "mcp__xpo__start") {
		t.Errorf("--disallowedTools = %v, want start and merge denied in remote mode", got)
	}
}

func TestWorkProfile_Claude(t *testing.T) {
	wt := t.TempDir()
	args := claudeArgs(turnOptions{MCPConfigPath: "m", Profile: WorkProfile("xpo-1", wt, "/repo")})

	if got := flagValue(args, "--tools"); got != workTools {
		t.Errorf("--tools = %q", got)
	}
	allowed := flagValues(args, "--allowedTools")
	// "//" starts an absolute path in a Claude Code rule; an Edit rule also
	// covers Write. Measured with Claude Code 2.1.296.
	real, _ := filepath.EvalSymlinks(wt)
	for _, want := range []string{"mcp__xpo__*", "Skill", "Bash", "WebSearch", "LSP", "TaskCreate", "TaskUpdate", "TaskList", "TaskGet", "Agent", "Edit(/" + real + "/**)"} {
		if !slices.Contains(allowed, want) {
			t.Errorf("--allowedTools = %v, missing %q", allowed, want)
		}
	}
	if slices.Contains(args, "--disallowedTools") {
		t.Errorf("work profile denies nothing: %v", args)
	}
	if got := askRules(t, args); !slices.Equal(got, []string{"mcp__xpo__merge", "WebFetch"}) {
		t.Errorf("ask rules = %v, want merge and WebFetch to ask", got)
	}
}

// WebFetch asks in every mode: an allow rule would skip its card.
func TestProfile_WebFetchNeverAllowed(t *testing.T) {
	for _, p := range []Profile{ChatProfile("/repo", true), ChatProfile("/repo", false), WorkProfile("xpo-1", "/wt", "/repo")} {
		args := claudeArgs(turnOptions{MCPConfigPath: "m", Profile: p})
		if allowed := flagValues(args, "--allowedTools"); slices.ContainsFunc(allowed, func(r string) bool { return strings.HasPrefix(r, "WebFetch") }) {
			t.Errorf("%s: --allowedTools = %v allows WebFetch", p.Mode, allowed)
		}
		if !slices.Contains(askRules(t, args), "WebFetch") {
			t.Errorf("%s: WebFetch does not ask", p.Mode)
		}
	}
}

// AskUserQuestion must reach the user as a card. On the allow list, Claude
// Code runs it without asking, and there is no answer to return.
func TestProfile_AskUserQuestionNeverAllowed(t *testing.T) {
	for _, p := range []Profile{ChatProfile("/repo", true), ChatProfile("/repo", false), WorkProfile("xpo-1", "/wt", "/repo")} {
		args := claudeArgs(turnOptions{MCPConfigPath: "m", Profile: p})
		if !slices.Contains(strings.Split(flagValue(args, "--tools"), ","), ToolAskUserQuestion) {
			t.Errorf("%s: --tools = %q, missing AskUserQuestion", p.Mode, flagValue(args, "--tools"))
		}
		if allowed := flagValues(args, "--allowedTools"); slices.ContainsFunc(allowed, func(r string) bool { return strings.HasPrefix(r, ToolAskUserQuestion) }) {
			t.Errorf("%s: --allowedTools = %v allows AskUserQuestion", p.Mode, allowed)
		}
	}
}

// Chat Mode stays read-only: no editing, no shell, no subagents, no task list.
func TestChatProfile_NoWorkTools(t *testing.T) {
	tools := strings.Split(ChatProfile("/repo", true).claudeTools(), ",")
	for _, tool := range []string{"Edit", "Write", "Bash", "Agent", "Task", "TaskCreate", "TaskUpdate", "TaskList", "TaskGet"} {
		if slices.Contains(tools, tool) {
			t.Errorf("chat mode has %s", tool)
		}
	}
}

func TestWorkProfile_EditRuleForSymlinkedPath(t *testing.T) {
	real := t.TempDir()
	link := filepath.Join(t.TempDir(), "wt")
	if err := os.Symlink(real, link); err != nil {
		t.Skip(err)
	}
	resolved, _ := filepath.EvalSymlinks(real)
	allowed := WorkProfile("xpo-1", link, "/repo").claudeAllowed()
	for _, want := range []string{"Edit(/" + link + "/**)", "Edit(/" + resolved + "/**)"} {
		if !slices.Contains(allowed, want) {
			t.Errorf("allowed = %v, missing %q", allowed, want)
		}
	}
}

// The spec tool hands the agent the spec's path, outside the worktree. Edits
// to the bound issue's own artifacts run without a card (xpo-2c724e).
func TestWorkProfile_EditRuleForIssueArtifacts(t *testing.T) {
	root := t.TempDir()
	resolved, _ := filepath.EvalSymlinks(root)
	allowed := WorkProfile("xpo-1", filepath.Join(root, ".xpo", "worktrees", "x"), root).claudeAllowed()
	for _, want := range []string{
		"Edit(/" + root + "/.xpo/artifacts/xpo-1/**)",
		"Edit(/" + resolved + "/.xpo/artifacts/xpo-1/**)",
	} {
		if !slices.Contains(allowed, want) {
			t.Errorf("allowed = %v, missing %q", allowed, want)
		}
	}
	for _, rule := range allowed {
		if strings.HasSuffix(rule, "/.xpo/**)") || strings.HasSuffix(rule, "/.xpo/artifacts/**)") {
			t.Errorf("allowed = %v, %q covers more than the issue's artifacts", allowed, rule)
		}
	}
}

func TestWorkProfile_NoArtifactRuleWithoutRoot(t *testing.T) {
	for _, rule := range WorkProfile("xpo-1", "/wt", "").claudeAllowed() {
		if strings.Contains(rule, ".xpo/artifacts") {
			t.Errorf("rule %q without a root", rule)
		}
	}
}

func TestChatProfile_NoArtifactRule(t *testing.T) {
	for _, rule := range ChatProfile("/repo", true).claudeAllowed() {
		if strings.HasPrefix(rule, "Edit(") {
			t.Errorf("chat mode allows %q", rule)
		}
	}
}

func TestProfile_Dir(t *testing.T) {
	if got := ChatProfile("/repo", true).Dir; got != "/repo" {
		t.Errorf("chat dir = %q", got)
	}
	if got := WorkProfile("xpo-1", "/repo/.xpo/worktrees/x", "/repo").Dir; got != "/repo/.xpo/worktrees/x" {
		t.Errorf("work dir = %q", got)
	}
}

func TestProfile_Codex(t *testing.T) {
	cases := []struct {
		name            string
		p               Profile
		sandbox, policy string
		disabled        []string
		prompt          []string // tools that ask
	}{
		{"chat", ChatProfile("/repo", true), "read-only", "on-request", []string{"merge"}, []string{"start"}},
		{"chat remote", ChatProfile("/repo", false), "read-only", "on-request", []string{"start", "merge"}, nil},
		{"work", WorkProfile("xpo-1", "/wt", "/repo"), "workspace-write", "on-request", nil, []string{"merge"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if c.p.codexSandbox() != c.sandbox || c.p.codexApprovalPolicy() != c.policy {
				t.Errorf("sandbox %q policy %q", c.p.codexSandbox(), c.p.codexApprovalPolicy())
			}
			args := strings.Join(codexArgs(MCPTarget{Executable: "/bin/xpo"}, c.p), "\n")
			if c.disabled != nil {
				want := "mcp_servers.xpo.disabled_tools=" + tomlValue(c.disabled)
				if !strings.Contains(args, want) {
					t.Errorf("args %s\nmissing %s", args, want)
				}
			} else if strings.Contains(args, "disabled_tools") {
				t.Errorf("args %s: nothing is disabled", args)
			}
			for _, tool := range c.prompt {
				want := "mcp_servers.xpo.tools." + tool + ".approval_mode=\"prompt\""
				if !strings.Contains(args, want) {
					t.Errorf("args %s\nmissing %s", args, want)
				}
			}
			if strings.Count(args, ".approval_mode=") != len(c.prompt) {
				t.Errorf("args %s: want %d per-tool approval modes", args, len(c.prompt))
			}
		})
	}
}

func TestModeContext(t *testing.T) {
	if got := ChatProfile("/repo", true).contextLines(); !slices.Equal(got, []string{"mode: chat"}) {
		t.Errorf("chat = %v", got)
	}
	got := WorkProfile("xpo-1", "/wt", "/repo").contextLines()
	if !slices.Equal(got, []string{"mode: work on xpo-1", "worktree: /wt"}) {
		t.Errorf("work = %v", got)
	}
}
