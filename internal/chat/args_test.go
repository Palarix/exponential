package chat

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"
)

// flagValue returns the argument after flag, or "" when flag is absent.
func flagValue(args []string, flag string) string {
	i := slices.Index(args, flag)
	if i < 0 || i+1 >= len(args) {
		return ""
	}
	return args[i+1]
}

func TestClaudeArgs_Baseline(t *testing.T) {
	args := claudeArgs(turnOptions{MCPConfigPath: "/tmp/mcp.json", Profile: ChatProfile("/repo", true)})

	// The prompt goes over stdin: a message that starts with "-" must not
	// parse as a flag.
	if args[0] != "-p" {
		t.Fatalf("args must start with -p: %v", args)
	}
	for flag, want := range map[string]string{
		"--output-format":        "stream-json",
		"--tools":                "Read,Grep,Glob,Skill",
		"--allowedTools":         "mcp__xpo__*",
		"--mcp-config":           "/tmp/mcp.json",
		"--append-system-prompt": systemPromptFor(ProviderClaude),
	} {
		if got := flagValue(args, flag); got != want {
			t.Errorf("%s = %q, want %q", flag, got, want)
		}
	}
	// Permission requests come back over stdio as control_request lines, so
	// the prompt goes over stdin as a stream-json message.
	for flag, want := range map[string]string{
		"--input-format":           "stream-json",
		"--permission-prompt-tool": "stdio",
	} {
		if got := flagValue(args, flag); got != want {
			t.Errorf("%s = %q, want %q", flag, got, want)
		}
	}
	for _, flag := range []string{"--verbose", "--include-partial-messages", "--strict-mcp-config"} {
		if !slices.Contains(args, flag) {
			t.Errorf("missing %s", flag)
		}
	}
	for _, flag := range []string{"--resume", "--model", "--effort", "--dangerously-skip-permissions", "--permission-mode", "--setting-sources"} {
		if slices.Contains(args, flag) {
			t.Errorf("unexpected %s in %v", flag, args)
		}
	}
}

func TestClaudeArgs_ResumeModelEffort(t *testing.T) {
	args := claudeArgs(turnOptions{MCPConfigPath: "m", SessionID: "sess-1", Model: "opus", Effort: "high"})
	if flagValue(args, "--resume") != "sess-1" || flagValue(args, "--model") != "opus" || flagValue(args, "--effort") != "high" {
		t.Fatalf("args = %v", args)
	}
}

func TestClaudeArgs_GrantsFollowTheAllowList(t *testing.T) {
	args := claudeArgs(turnOptions{MCPConfigPath: "m", Profile: ChatProfile("/repo", true), Grants: []string{"Bash(echo hi *)", "mcp__xpo__merge"}})
	i := slices.Index(args, "--allowedTools")
	if i < 0 || !slices.Equal(args[i+1:i+5], []string{"mcp__xpo__*", "Skill", "Bash(echo hi *)", "mcp__xpo__merge"}) {
		t.Fatalf("args = %v, want the grants as extra --allowedTools values", args)
	}
	if !strings.HasPrefix(args[i+5], "--") {
		t.Errorf("--allowedTools values must end at the next flag: %v", args)
	}
}

func TestBuildPrompt(t *testing.T) {
	chat := ChatProfile("/repo", true)
	if got, want := buildPrompt("hi", nil, chat), "<xpo-context>\nmode: chat\n</xpo-context>\n\nhi"; got != want {
		t.Errorf("no context: %q", got)
	}
	got := buildPrompt("fix this", &Context{View: "board", IssueID: "xpo-1", IssueTitle: "Login <times> out"}, chat)
	want := "<xpo-context>\nmode: chat\nview: board\nopen issue: xpo-1 \"Login <times> out\"\n</xpo-context>\n\nfix this"
	if got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
	got = buildPrompt("go", nil, WorkProfile("xpo-1", "/wt", "/repo"))
	if want := "<xpo-context>\nmode: work on xpo-1\nworktree: /wt\n</xpo-context>\n\ngo"; got != want {
		t.Errorf("work: %q", got)
	}
}

func TestMCPConfig_Local(t *testing.T) {
	raw, err := mcpConfig(MCPTarget{Executable: "/usr/local/bin/xpo"})
	if err != nil {
		t.Fatal(err)
	}
	var cfg struct {
		MCPServers map[string]struct {
			Type    string            `json:"type"`
			Command string            `json:"command"`
			Args    []string          `json:"args"`
			URL     string            `json:"url"`
			Headers map[string]string `json:"headers"`
		} `json:"mcpServers"`
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		t.Fatal(err)
	}
	xpo := cfg.MCPServers["xpo"]
	if xpo.Command != "/usr/local/bin/xpo" || !slices.Equal(xpo.Args, []string{"mcp"}) || xpo.URL != "" {
		t.Fatalf("local config = %s", raw)
	}
}

func TestMCPConfig_Remote(t *testing.T) {
	raw, err := mcpConfig(MCPTarget{RemoteURL: "https://xpo.example.com/", RemoteToken: "tok"})
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	for _, want := range []string{`"type":"http"`, `"url":"https://xpo.example.com/mcp"`, `"Authorization":"Bearer tok"`} {
		if !strings.Contains(s, want) {
			t.Errorf("remote config %s missing %s", s, want)
		}
	}
	if strings.Contains(s, `"command"`) {
		t.Errorf("remote config must not have a command: %s", s)
	}
}

func TestSystemPromptFor(t *testing.T) {
	for _, p := range []string{ProviderClaude, ProviderCodex} {
		got := systemPromptFor(p)
		if strings.Contains(got, "{{") || !strings.Contains(got, codeAccess[p][ModeChat]) || !strings.Contains(got, codeAccess[p][ModeWork]) {
			t.Errorf("%s prompt = %q", p, got)
		}
		// Both modes name the xpo skill, so the agent follows the project's
		// workflow (xpo-a058b4).
		for _, want := range []string{
			"Before you plan, spec or start work on an issue, load the `xpo` skill.",
			"Before you change anything, load the `xpo` skill",
		} {
			if !strings.Contains(got, want) {
				t.Errorf("%s prompt is missing %q", p, want)
			}
		}
	}
}
