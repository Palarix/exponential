package chat

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

//go:embed prompt.md
var systemPrompt string

// Models and efforts the composer offers. Empty means "use the default".
var (
	models  = []string{"", "haiku", "sonnet", "opus"}
	efforts = []string{"", "low", "medium", "high", "xhigh", "max"}
)

// ValidModel reports whether m is a model alias the chat accepts.
func ValidModel(m string) bool { return slices.Contains(models, m) }

// ValidEffort reports whether e is an effort level the chat accepts.
func ValidEffort(e string) bool { return slices.Contains(efforts, e) }

// turnOptions are the per-turn inputs to the claude command line. The prompt
// itself goes over stdin.
type turnOptions struct {
	MCPConfigPath string
	SessionID     string
	Model         string
	Effort        string
}

// claudeArgs builds the `claude` arguments for one chat turn. The agent gets
// the read-only code tools plus the xpo MCP tools, and nothing else: there is
// no permission prompt in -p mode, so any other tool call is denied.
func claudeArgs(o turnOptions) []string {
	args := []string{
		"-p",
		"--output-format", "stream-json",
		"--verbose",
		"--include-partial-messages",
		"--tools", "Read,Grep,Glob",
		"--allowedTools", "mcp__xpo__*",
		"--mcp-config", o.MCPConfigPath,
		"--strict-mcp-config",
		"--append-system-prompt", systemPrompt,
	}
	if o.SessionID != "" {
		args = append(args, "--resume", o.SessionID)
	}
	if o.Model != "" {
		args = append(args, "--model", o.Model)
	}
	if o.Effort != "" {
		args = append(args, "--effort", o.Effort)
	}
	return args
}

// MCPTarget says where the chat agent finds the xpo MCP tools: the stdio
// server of the running xpo binary (local mode), or the remote server's
// /mcp endpoint (remote mode, RemoteURL set).
type MCPTarget struct {
	Executable  string
	RemoteURL   string
	RemoteToken string
}

func mcpConfig(t MCPTarget) ([]byte, error) {
	server := map[string]any{"command": t.Executable, "args": []string{"mcp"}}
	if t.RemoteURL != "" {
		server = map[string]any{
			"type": "http",
			"url":  strings.TrimSuffix(t.RemoteURL, "/") + "/mcp",
		}
		if t.RemoteToken != "" {
			server["headers"] = map[string]string{"Authorization": "Bearer " + t.RemoteToken}
		}
	}
	return json.Marshal(map[string]any{"mcpServers": map[string]any{"xpo": server}})
}

// buildPrompt prefixes the user's text with what they had open in the UI.
func buildPrompt(text string, c *Context) string {
	if c == nil || (c.View == "" && c.IssueID == "") {
		return text
	}
	var b strings.Builder
	b.WriteString("<xpo-context>\n")
	if c.View != "" {
		fmt.Fprintf(&b, "view: %s\n", c.View)
	}
	if c.IssueID != "" {
		fmt.Fprintf(&b, "open issue: %s %q\n", c.IssueID, c.IssueTitle)
	}
	b.WriteString("</xpo-context>\n\n")
	b.WriteString(text)
	return b.String()
}
