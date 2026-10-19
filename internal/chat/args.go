package chat

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
)

//go:embed prompt.md
var systemPrompt string

// codeAccess tells each agent how it can reach the project's code in each
// mode: Claude through its tool lists, Codex through its sandbox. The system
// prompt describes both modes, so it stays the same when the mode changes.
var codeAccess = map[string]map[string]string{
	ProviderClaude: {
		ModeChat: "You can read and search the code with Read, Grep, Glob, and LSP, and search the web with WebSearch. WebFetch asks the user before it fetches a page. You cannot edit files or run commands. To ask the user to choose between options, use AskUserQuestion.",
		ModeWork: "You can read, edit, and write files, and run shell commands with Bash. Keep every change inside the worktree: an edit outside it asks the user. You can also navigate code with LSP, search the web with WebSearch, track your steps with TaskCreate and TaskUpdate, and hand self-contained work to subagents with Agent. WebFetch asks the user before it fetches a page. To ask the user to choose between options, use AskUserQuestion.",
	},
	ProviderCodex: {
		ModeChat: "You can read and search the code with read-only shell commands such as `rg`, `sed -n` and `git log`. The sandbox blocks writes and network access, so you cannot edit files.",
		ModeWork: "You can edit files and run commands in the worktree. The sandbox blocks writes outside it.",
	},
}

// systemPromptFor returns the chat system prompt for a provider.
func systemPromptFor(provider string) string {
	return strings.NewReplacer(
		"{{chat_access}}", codeAccess[provider][ModeChat],
		"{{work_access}}", codeAccess[provider][ModeWork],
	).Replace(systemPrompt)
}

// turnOptions are the per-turn inputs to the claude command line. The prompt
// itself goes over stdin.
type turnOptions struct {
	MCPConfigPath string
	SessionID     string
	ResumeAt      string // keep the session up to this assistant message; needs SessionID
	Model         string
	Effort        string
	Grants        []string // rules the user allowed for the whole chat
	Profile       Profile  // the turn's tools and rules
}

// claudeArgs builds the `claude` arguments for one chat turn. The profile
// picks the code tools, the allow, ask and deny rules. A call outside the
// allow list arrives as a control_request on stdout, and the driver asks the
// user.
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
		"--tools", o.Profile.claudeTools(),
		"--settings", o.Profile.claudeSettings(),
		"--allowedTools",
	}
	args = append(args, o.Profile.claudeAllowed()...)
	args = append(args, o.Grants...)
	if denied := o.Profile.claudeDisallowed(); len(denied) > 0 {
		args = append(args, "--disallowedTools")
		args = append(args, denied...)
	}
	args = append(args,
		"--mcp-config", o.MCPConfigPath,
		"--strict-mcp-config",
		"--append-system-prompt", systemPromptFor(ProviderClaude),
	)
	if o.SessionID != "" {
		args = append(args, "--resume", o.SessionID)
		if o.ResumeAt != "" {
			args = append(args, "--resume-session-at", o.ResumeAt)
		}
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
// server of the running xpo binary.
type MCPTarget struct {
	Executable string
}

func mcpConfig(t MCPTarget) ([]byte, error) {
	server := map[string]any{"command": t.Executable, "args": []string{"mcp"}}
	return json.Marshal(map[string]any{"mcpServers": map[string]any{"xpo": server}})
}

// promptFile is an attachment as the prompt lists it.
type promptFile struct {
	Name  string
	Path  string // absolute
	Image bool   // also sent as image input
}

// buildPrompt prefixes the user's text with the turn's mode and what they
// had open in the UI, and lists the attachments with their paths after it,
// so the agent can read them with its own tools.
func buildPrompt(text string, c *Context, p Profile, files []promptFile) string {
	var b strings.Builder
	b.WriteString("<xpo-context>\n")
	for _, line := range p.contextLines() {
		b.WriteString(line + "\n")
	}
	if c == nil {
		c = &Context{}
	}
	if c.View != "" {
		fmt.Fprintf(&b, "view: %s\n", c.View)
	}
	if c.IssueID != "" {
		fmt.Fprintf(&b, "open issue: %s %q\n", c.IssueID, c.IssueTitle)
	}
	b.WriteString("</xpo-context>\n\n")
	b.WriteString(withAttachments(text, files))
	return b.String()
}

// withAttachments is the user's text with the attachments listed after it.
func withAttachments(text string, files []promptFile) string {
	if len(files) == 0 {
		return text
	}
	var b strings.Builder
	b.WriteString(text)
	if text != "" {
		b.WriteString("\n\n")
	}
	b.WriteString("<attachments>\n")
	for _, f := range files {
		kind := ""
		if f.Image {
			kind = " (image, attached)"
		}
		fmt.Fprintf(&b, "- %s%s: %s\n", f.Name, kind, f.Path)
	}
	b.WriteString("</attachments>")
	return b.String()
}
