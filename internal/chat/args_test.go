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
	args := claudeArgs(turnOptions{MCPConfigPath: "/tmp/mcp.json"})

	// The prompt goes over stdin: a message that starts with "-" must not
	// parse as a flag.
	if args[0] != "-p" {
		t.Fatalf("args must start with -p: %v", args)
	}
	for flag, want := range map[string]string{
		"--output-format":        "stream-json",
		"--tools":                "Read,Grep,Glob",
		"--allowedTools":         "mcp__xpo__*",
		"--mcp-config":           "/tmp/mcp.json",
		"--append-system-prompt": systemPrompt,
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

func TestBuildPrompt(t *testing.T) {
	if got := buildPrompt("hi", nil); got != "hi" {
		t.Errorf("no context: %q", got)
	}
	if got := buildPrompt("hi", &Context{}); got != "hi" {
		t.Errorf("empty context: %q", got)
	}
	got := buildPrompt("fix this", &Context{View: "board", IssueID: "xpo-1", IssueTitle: "Login <times> out"})
	want := "<xpo-context>\nview: board\nopen issue: xpo-1 \"Login <times> out\"\n</xpo-context>\n\nfix this"
	if got != want {
		t.Errorf("got %q\nwant %q", got, want)
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

func TestValidModelAndEffort(t *testing.T) {
	for _, m := range []string{"", "haiku", "sonnet", "opus"} {
		if !ValidModel(m) {
			t.Errorf("ValidModel(%q) = false", m)
		}
	}
	for _, m := range []string{"gpt-5", "opus; rm -rf", "--dangerously-skip-permissions"} {
		if ValidModel(m) {
			t.Errorf("ValidModel(%q) = true", m)
		}
	}
	for _, e := range []string{"", "low", "medium", "high", "xhigh", "max"} {
		if !ValidEffort(e) {
			t.Errorf("ValidEffort(%q) = false", e)
		}
	}
	if ValidEffort("ultra") {
		t.Error("ValidEffort(ultra) = true")
	}
}
