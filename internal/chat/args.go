package chat

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
)

//go:embed prompt.md
var systemPrompt string

// codeAccess tells each agent how it can read the project's code. Both are
// read-only: Claude through its tool allow-list, Codex through its sandbox.
var codeAccess = map[string]string{
	ProviderClaude: "You can read and search the project's code with Read, Grep, and Glob. You cannot edit files or run commands.",
	ProviderCodex:  "You can read and search the project's code with read-only shell commands such as `rg`, `sed -n` and `git log`. The sandbox blocks writes and network access, so you cannot edit files.",
}

// systemPromptFor returns the chat system prompt for a provider.
func systemPromptFor(provider string) string {
	return strings.ReplaceAll(systemPrompt, "{{code_access}}", codeAccess[provider])
}

// turnOptions are the per-turn inputs to the claude command line. The prompt
// itself goes over stdin.
type turnOptions struct {
	MCPConfigPath string
	SessionID     string
	Model         string
	Effort        string
	Grants        []string // rules the user allowed for the whole chat
}

// chatSettings makes `merge` ask before it runs, although the xpo wildcard
// allows it: an ask rule wins over an allow rule.
const chatSettings = `{"permissions":{"ask":["mcp__xpo__merge"]}}`

// claudeArgs builds the `claude` arguments for one chat turn. The agent gets
// the read-only code tools plus the xpo MCP tools. A call outside the allow
// list arrives as a control_request on stdout, and the driver asks the user.
// The prompt goes over stdin as a stream-json message, so a message that
// starts with "-" never parses as a flag.
func claudeArgs(o turnOptions) []string {
	args := []string{
		"-p",
		"--output-format", "stream-json",
		"--input-format", "stream-json",
		"--permission-prompt-tool", "stdio",
		"--verbose",
		"--include-partial-messages",
		"--tools", "Read,Grep,Glob",
		"--settings", chatSettings,
		"--allowedTools", "mcp__xpo__*",
	}
	args = append(args, o.Grants...)
	args = append(args,
		"--mcp-config", o.MCPConfigPath,
		"--strict-mcp-config",
		"--append-system-prompt", systemPromptFor(ProviderClaude),
	)
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
