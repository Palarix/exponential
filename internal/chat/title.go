package chat

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
	"unicode/utf8"
)

// Titler asks an agent CLI for a short chat title, in one call with no
// tools and no session.
type Titler interface {
	// Title runs the CLI in dir with model ("" is the CLI's default) and
	// returns its reply to prompt.
	Title(ctx context.Context, dir, model, prompt string) (string, error)
}

// titleTimeout bounds one title call.
const titleTimeout = 60 * time.Second

// titleInstructions is the system prompt of a title call.
const titleInstructions = "You name chat conversations between a user and a coding agent. Reply with the title only."

// maxTitleInput is how much of the first message and of the first reply a
// title prompt includes, in characters.
const maxTitleInput = 2000

// maxGeneratedTitleRunes cuts a generated title that ignored the word limit.
const maxGeneratedTitleRunes = 80

// titlePrompt asks for a title from a chat's first message and first reply.
// In Work Mode, issueID (and issueTitle, when known) hint at what the chat
// works on; the model decides whether the title names it.
func titlePrompt(issueID, issueTitle, firstUser, firstReply string) string {
	var b strings.Builder
	b.WriteString("Write a 3-8 word title for this chat, in sentence case. Reply with the title only: no quotes, no trailing period.\n")
	if issueID != "" {
		hint := issueID
		if issueTitle != "" {
			hint += fmt.Sprintf(" %q", issueTitle)
		}
		fmt.Fprintf(&b, "\nThe chat works on issue %s. Name the issue in the title only when that helps.\n", hint)
	}
	fmt.Fprintf(&b, "\nUser:\n%s\n", cutRunes(strings.TrimSpace(firstUser), maxTitleInput))
	if firstReply != "" {
		fmt.Fprintf(&b, "\nAgent:\n%s\n", cutRunes(strings.TrimSpace(firstReply), maxTitleInput))
	}
	return b.String()
}

// cutRunes cuts s to max characters, and marks the cut with an ellipsis.
func cutRunes(s string, max int) string {
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	return strings.TrimSpace(string([]rune(s)[:max])) + "…"
}

// cleanTitle turns a model's reply into a title: its first non-empty line,
// without a "Title:" label, Markdown emphasis, quotes or a trailing period.
func cleanTitle(s string) string {
	line := ""
	for l := range strings.SplitSeq(s, "\n") {
		if line = strings.TrimSpace(l); line != "" {
			break
		}
	}
	line = strings.TrimSpace(strings.TrimPrefix(line, "Title:"))
	line = strings.Trim(line, "*_`\"'“”‘’ ")
	line = strings.TrimSpace(strings.TrimSuffix(line, "."))
	return cutRunes(line, maxGeneratedTitleRunes)
}

// ClaudeTitler names chats with `claude -p`, without tools, MCP servers,
// settings or a saved session.
type ClaudeTitler struct {
	Binary string // "claude" when empty
}

func claudeTitleArgs(model string) []string {
	args := []string{"-p", "--output-format", "json", "--no-session-persistence",
		"--tools", "", "--strict-mcp-config", "--setting-sources", "", "--disable-slash-commands",
		"--system-prompt", titleInstructions}
	if model != "" {
		args = append(args, "--model", model)
	}
	return args
}

// Title implements Titler.
func (t *ClaudeTitler) Title(ctx context.Context, dir, model, prompt string) (string, error) {
	bin := t.Binary
	if bin == "" {
		bin = "claude"
	}
	out, err := runTitleCLI(ctx, bin, claudeTitleArgs(model), dir, prompt)
	if err != nil {
		return "", err
	}
	return parseClaudeTitle(out)
}

func parseClaudeTitle(out []byte) (string, error) {
	var res struct {
		IsError bool   `json:"is_error"`
		Result  string `json:"result"`
	}
	if err := json.Unmarshal(out, &res); err != nil {
		return "", fmt.Errorf("parse claude reply: %w", err)
	}
	if res.IsError {
		return "", errors.New(res.Result)
	}
	return res.Result, nil
}

// CodexTitler names chats with `codex exec`: an ephemeral, read-only run at
// low effort, with short instructions in place of Codex's own.
type CodexTitler struct {
	Binary string // "codex" when empty
}

func codexTitleArgs(model, instructionsFile string) []string {
	args := []string{"exec", "--ephemeral", "--skip-git-repo-check", "--ignore-rules", "-s", "read-only"}
	if model != "" {
		args = append(args, "-m", model)
	}
	return append(args,
		"-c", "model_reasoning_effort="+tomlValue("low"),
		"-c", "model_instructions_file="+tomlValue(instructionsFile),
		"-c", "web_search="+tomlValue("disabled"),
		"--json", "-")
}

// Title implements Titler.
func (t *CodexTitler) Title(ctx context.Context, dir, model, prompt string) (string, error) {
	bin := t.Binary
	if bin == "" {
		bin = "codex"
	}
	f, err := os.CreateTemp("", "xpo-title-*.md")
	if err != nil {
		return "", err
	}
	defer os.Remove(f.Name())
	_, err = f.WriteString(titleInstructions)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return "", err
	}
	out, err := runTitleCLI(ctx, bin, codexTitleArgs(model, f.Name()), dir, prompt)
	if err != nil {
		return "", err
	}
	return parseCodexTitle(out)
}

// parseCodexTitle returns the last agent message of `codex exec --json`.
func parseCodexTitle(out []byte) (string, error) {
	var title string
	sc := bufio.NewScanner(bytes.NewReader(out))
	sc.Buffer(make([]byte, 64*1024), 4*1024*1024)
	for sc.Scan() {
		var line struct {
			Type string `json:"type"`
			Item struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"item"`
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if json.Unmarshal(sc.Bytes(), &line) != nil {
			continue
		}
		switch {
		case line.Type == "item.completed" && line.Item.Type == "agent_message":
			title = line.Item.Text
		case line.Type == "turn.failed":
			return "", errors.New(line.Error.Message)
		}
	}
	return title, sc.Err()
}

// runTitleCLI runs bin with prompt on stdin and returns its stdout. A
// cancelled ctx kills its process group.
func runTitleCLI(ctx context.Context, bin string, args []string, dir, prompt string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader(prompt)
	setProcessGroup(cmd)
	cmd.Cancel = func() error {
		killGroup(cmd.Process.Pid)
		return nil
	}
	cmd.WaitDelay = outputGrace
	stderr := &tailBuffer{max: 2048}
	cmd.Stderr = stderr
	out, err := cmd.Output()
	if err != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return out, fmt.Errorf("%w: %s", err, msg)
		}
		return out, err
	}
	return out, nil
}

// titleInput returns a chat's first user message and the agent's text of
// the first turn that finished, from the settled events.
func titleInput(events []Event) (firstUser, firstReply string) {
	var reply strings.Builder
	for _, e := range events {
		switch e.Type {
		case EventUserMessage:
			if firstUser == "" {
				firstUser = e.Text
			}
			reply.Reset()
		case EventAssistantText:
			if reply.Len() > 0 {
				reply.WriteString("\n\n")
			}
			reply.WriteString(e.Text)
		case EventTurnDone:
			return firstUser, reply.String()
		}
	}
	return firstUser, reply.String()
}

// generateTitle asks the chat's agent for a title and replaces the
// placeholder with it, unless the user renamed the chat in the meantime.
// It runs once per chat: the caller has cleared AutoTitle.
func (r *Runner) generateTitle(id string, m Meta, placeholder string) {
	defer r.titles.Done()
	titler := r.cfg.Titlers[m.Agent]
	_, events, err := r.cfg.Store.Get(id)
	if err != nil {
		return
	}
	firstUser, firstReply := titleInput(events)
	var issueID, issueTitle string
	if m.Mode == ModeWork && m.Work != nil {
		issueID = m.Work.IssueID
		if r.cfg.IssueTitle != nil {
			issueTitle = r.cfg.IssueTitle(issueID)
		}
	}
	model := m.Model
	if r.cfg.TitleModel != nil {
		if fast := r.cfg.TitleModel(m.Agent); fast != "" {
			model = fast
		}
	}

	ctx, cancel := context.WithTimeout(r.titleCtx, titleTimeout)
	defer cancel()
	reply, err := titler.Title(ctx, r.profileFor(m).Dir, model, titlePrompt(issueID, issueTitle, firstUser, firstReply))
	if err != nil {
		if ctx.Err() == nil || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			log.Printf("chat %s: generate title: %v", id, err)
		}
		return
	}
	title := cleanTitle(reply)
	if title == "" {
		return
	}
	if ok, err := r.cfg.Store.ReplaceAutoTitle(id, placeholder, title); err == nil && ok {
		r.live(id, Event{Type: EventTitle, Text: title})
	}
}
