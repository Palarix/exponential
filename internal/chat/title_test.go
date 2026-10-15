package chat

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeTitler records its calls and answers with title, or err. When hold is
// set, it waits for hold to close first.
type fakeTitler struct {
	mu     sync.Mutex
	calls  []titleCall
	title  string
	err    error
	hold   chan struct{}
	called chan struct{} // closed on the first call, when set
}

type titleCall struct{ dir, model, prompt string }

func (f *fakeTitler) Title(ctx context.Context, dir, model, prompt string) (string, error) {
	f.mu.Lock()
	f.calls = append(f.calls, titleCall{dir, model, prompt})
	if f.called != nil && len(f.calls) == 1 {
		close(f.called)
	}
	f.mu.Unlock()
	if f.hold != nil {
		select {
		case <-f.hold:
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
	return f.title, f.err
}

func (f *fakeTitler) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.calls)
}

// withTitler gives the runner f as Claude's titler, and "haiku" as Claude's
// title model.
func withTitler(f *fakeTitler) func(*RunnerConfig) {
	return func(c *RunnerConfig) {
		c.Titlers = map[string]Titler{ProviderClaude: f}
		c.TitleModel = func(provider string) string {
			if provider == ProviderClaude {
				return "haiku"
			}
			return ""
		}
	}
}

func TestRunner_GeneratesTitleAfterFirstTurn(t *testing.T) {
	f := &fakeTitler{title: "Open login issues"}
	r, store, dir := newTestRunner(t, `cat "$FIXTURE"`, withTitler(f))
	m, _ := store.Create()
	events, unsubscribe := r.Subscribe(m.ID)
	defer unsubscribe()

	r.Send(m.ID, SendRequest{Text: "how many open issues?\nmore detail", Model: "sonnet"})
	r.Wait()

	meta, _ := store.Meta(m.ID)
	if meta.Title != "Open login issues" || meta.AutoTitle {
		t.Fatalf("meta title = %q (auto %v), want the generated title", meta.Title, meta.AutoTitle)
	}
	if f.count() != 1 {
		t.Fatalf("titler called %d times, want once", f.count())
	}
	call := f.calls[0]
	if call.model != "haiku" || call.dir != dir {
		t.Errorf("call = %+v, want haiku in the chat's directory", call)
	}
	if !strings.Contains(call.prompt, "how many open issues?\nmore detail") || !strings.Contains(call.prompt, "You have 1 open issue.") {
		t.Errorf("prompt = %q, want the first message and the first reply", call.prompt)
	}
	if !hasEvent(events, EventTitle, "Open login issues") {
		t.Error("no live title event")
	}

	r.Send(m.ID, SendRequest{Text: "and closed?"})
	r.Wait()
	if f.count() != 1 {
		t.Errorf("titler called %d times after the second turn, want once", f.count())
	}
}

// hasEvent drains ch and reports whether it carried an event of typ with
// text.
func hasEvent(ch <-chan Event, typ, text string) bool {
	for {
		select {
		case e := <-ch:
			if e.Type == typ && e.Text == text {
				return true
			}
		default:
			return false
		}
	}
}

func TestRunner_TitleUsesChatModelWithoutTitleModel(t *testing.T) {
	f := &fakeTitler{title: "T"}
	r, store, _ := newTestRunner(t, `cat "$FIXTURE"`, withTitler(f), func(c *RunnerConfig) { c.TitleModel = nil })
	m, _ := store.Create()
	r.Send(m.ID, SendRequest{Text: "hi", Model: "sonnet"})
	r.Wait()
	if f.count() != 1 || f.calls[0].model != "sonnet" {
		t.Fatalf("calls = %+v, want the chat's model", f.calls)
	}
}

func TestRunner_RenameDuringTitleWins(t *testing.T) {
	f := &fakeTitler{title: "Generated", hold: make(chan struct{}), called: make(chan struct{})}
	r, store, _ := newTestRunner(t, `cat "$FIXTURE"`, withTitler(f))
	m, _ := store.Create()
	r.Send(m.ID, SendRequest{Text: "hi"})
	<-f.called
	if _, err := store.Rename(m.ID, "Mine"); err != nil {
		t.Fatal(err)
	}
	close(f.hold)
	r.Wait()
	if meta, _ := store.Meta(m.ID); meta.Title != "Mine" {
		t.Fatalf("title = %q, want the user's rename", meta.Title)
	}
}

func TestRunner_TitleErrorKeepsPlaceholder(t *testing.T) {
	f := &fakeTitler{err: errors.New("offline")}
	r, store, _ := newTestRunner(t, `cat "$FIXTURE"`, withTitler(f))
	m, _ := store.Create()
	r.Send(m.ID, SendRequest{Text: "hi there"})
	r.Wait()
	meta, events, _ := store.Get(m.ID)
	if meta.Title != "hi there" || meta.AutoTitle {
		t.Fatalf("meta = %q (auto %v), want the placeholder, tried once", meta.Title, meta.AutoTitle)
	}
	if countType(events, EventError) != 0 {
		t.Errorf("events = %+v, want no error in the thread", events)
	}
	r.Send(m.ID, SendRequest{Text: "again"})
	r.Wait()
	if f.count() != 1 {
		t.Errorf("titler called %d times, want once per chat", f.count())
	}
}

func TestRunner_EmptyTitleKeepsPlaceholder(t *testing.T) {
	f := &fakeTitler{title: "  \n"}
	r, store, _ := newTestRunner(t, `cat "$FIXTURE"`, withTitler(f))
	m, _ := store.Create()
	r.Send(m.ID, SendRequest{Text: "hi there"})
	r.Wait()
	if meta, _ := store.Meta(m.ID); meta.Title != "hi there" {
		t.Fatalf("title = %q, want the placeholder", meta.Title)
	}
}

func TestRunner_FailedFirstTurnGeneratesNoTitle(t *testing.T) {
	f := &fakeTitler{title: "Generated"}
	r, store, _ := newTestRunner(t, failOnceBody(`cat "$FIXTURE"`), withTitler(f))
	m, _ := store.Create()
	r.Send(m.ID, SendRequest{Text: "hi"})
	r.Wait()
	if f.count() != 0 {
		t.Fatalf("titler called after a failed turn")
	}
	r.Send(m.ID, SendRequest{Text: "again"})
	r.Wait()
	if meta, _ := store.Meta(m.ID); meta.Title != "Generated" || f.count() != 1 {
		t.Fatalf("title = %q after %d calls, want a title after the first finished turn", meta.Title, f.count())
	}
	if !strings.Contains(f.calls[0].prompt, "hi") {
		t.Errorf("prompt = %q, want the first user message", f.calls[0].prompt)
	}
}

// A chat saved before generated titles has no auto_title: its title stays.
func TestRunner_OldChatKeepsItsTitle(t *testing.T) {
	f := &fakeTitler{title: "Generated"}
	r, store, _ := newTestRunner(t, `cat "$FIXTURE"`, withTitler(f))
	m, _ := store.Create()
	m.Title = "Old title"
	store.SaveMeta(m)
	r.Send(m.ID, SendRequest{Text: "hi"})
	r.Wait()
	if meta, _ := store.Meta(m.ID); meta.Title != "Old title" || f.count() != 0 {
		t.Fatalf("title = %q after %d calls, want the old title", meta.Title, f.count())
	}
}

func TestRunner_WorkChatTitleHint(t *testing.T) {
	f := &fakeTitler{title: "Login fix"}
	r, store, dir := newModeRunner(t)
	withTitler(f)(&r.cfg)
	r.cfg.IssueTitle = func(id string) string { return "Fix the login redirect" }
	wt := makeWorktree(t, dir, "xpo-abc")
	m, _ := store.Create()
	store.SetAutoTitle(m.ID, "Fix the login redirect")
	if _, err := r.BindWork(m.ID, WorkBinding{IssueID: "xpo-abc", Worktree: wt}); err != nil {
		t.Fatal(err)
	}
	r.Send(m.ID, SendRequest{Text: "go"})
	r.Wait()
	if meta, _ := store.Meta(m.ID); meta.Title != "Login fix" {
		t.Fatalf("title = %q, want the generated title with no prefix", meta.Title)
	}
	if f.count() != 1 || !strings.Contains(f.calls[0].prompt, `xpo-abc "Fix the login redirect"`) || f.calls[0].dir != wt {
		t.Fatalf("calls = %+v, want the issue hint, in the worktree", f.calls)
	}
}

func TestRunner_ShutdownCancelsTitle(t *testing.T) {
	f := &fakeTitler{title: "Generated", hold: make(chan struct{}), called: make(chan struct{})}
	r, store, _ := newTestRunner(t, `cat "$FIXTURE"`, withTitler(f))
	m, _ := store.Create()
	r.Send(m.ID, SendRequest{Text: "hi"})
	<-f.called
	done := make(chan struct{})
	go func() { r.Shutdown(); close(done) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Shutdown waited for the title call")
	}
	if meta, _ := store.Meta(m.ID); meta.Title != "hi" {
		t.Fatalf("title = %q, want the placeholder", meta.Title)
	}
}

func TestTitlePrompt(t *testing.T) {
	long := strings.Repeat("a", 3000)
	p := titlePrompt("", "", "first message", long)
	if !strings.Contains(p, "first message") || strings.Contains(p, long) || !strings.Contains(p, strings.Repeat("a", 2000)+"…") {
		t.Errorf("prompt = %q, want the message and the reply cut to 2,000 characters", p)
	}
	if strings.Contains(p, "issue") {
		t.Errorf("chat prompt has an issue hint: %q", p)
	}
	p = titlePrompt("xpo-abc", "Fix login", "go", "ok")
	if !strings.Contains(p, `xpo-abc "Fix login"`) {
		t.Errorf("work prompt = %q, want the issue hint", p)
	}
}

func TestCleanTitle(t *testing.T) {
	tests := map[string]string{
		"Open login issues":                      "Open login issues",
		"  \"Open login issues.\"  \nsecond line": "Open login issues",
		"Title: Fix the spinner":                 "Fix the spinner",
		"**Fix the spinner**":                    "Fix the spinner",
		"'Quoted'":                               "Quoted",
		"“Curly”":                                "Curly",
		"":                                       "",
		strings.Repeat("word ", 30):              strings.TrimSpace(strings.Repeat("word ", 16)) + "…",
	}
	for in, want := range tests {
		if got := cleanTitle(in); got != want {
			t.Errorf("cleanTitle(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestClaudeTitleArgs(t *testing.T) {
	args := claudeTitleArgs("haiku")
	for _, want := range [][]string{
		{"-p"}, {"--model", "haiku"}, {"--output-format", "json"}, {"--no-session-persistence"},
		{"--tools", ""}, {"--strict-mcp-config"}, {"--setting-sources", ""}, {"--disable-slash-commands"},
	} {
		if !containsRun(args, want) {
			t.Errorf("args %q lack %q", args, want)
		}
	}
	if !slices.Contains(args, "--system-prompt") {
		t.Errorf("args %q lack a system prompt", args)
	}
	if containsRun(claudeTitleArgs(""), []string{"--model"}) {
		t.Error("an empty model still passes --model")
	}
}

func TestCodexTitleArgs(t *testing.T) {
	args := codexTitleArgs("gpt-6-luna", "/tmp/instr.md")
	for _, want := range [][]string{
		{"exec"}, {"--ephemeral"}, {"--skip-git-repo-check"}, {"--ignore-rules"}, {"-s", "read-only"},
		{"-m", "gpt-6-luna"}, {"-c", `model_reasoning_effort="low"`}, {"-c", `model_instructions_file="/tmp/instr.md"`},
		{"-c", `web_search="disabled"`}, {"--json"},
	} {
		if !containsRun(args, want) {
			t.Errorf("args %q lack %q", args, want)
		}
	}
	if args[len(args)-1] != "-" {
		t.Errorf("args %q do not read the prompt from stdin", args)
	}
	if containsRun(codexTitleArgs("", "/tmp/i"), []string{"-m"}) {
		t.Error("an empty model still passes -m")
	}
}

// containsRun reports whether want appears in args as consecutive items.
func containsRun(args, want []string) bool {
	for i := 0; i+len(want) <= len(args); i++ {
		if slices.Equal(args[i:i+len(want)], want) {
			return true
		}
	}
	return false
}

func TestParseClaudeTitle(t *testing.T) {
	if got, err := parseClaudeTitle([]byte(`{"type":"result","is_error":false,"result":"Shield spinner"}`)); err != nil || got != "Shield spinner" {
		t.Errorf("parse = %q, %v", got, err)
	}
	if _, err := parseClaudeTitle([]byte(`{"type":"result","is_error":true,"result":"Not logged in"}`)); err == nil || !strings.Contains(err.Error(), "Not logged in") {
		t.Errorf("error result: err = %v", err)
	}
}

func TestParseCodexTitle(t *testing.T) {
	out := `{"type":"thread.started","thread_id":"t"}
{"type":"turn.started"}
{"type":"item.completed","item":{"id":"item_0","type":"agent_message","text":"Shield spinner"}}
{"type":"turn.completed","usage":{"input_tokens":9506,"output_tokens":11}}
`
	if got, err := parseCodexTitle([]byte(out)); err != nil || got != "Shield spinner" {
		t.Errorf("parse = %q, %v", got, err)
	}
	failed := `{"type":"turn.failed","error":{"message":"model not supported"}}`
	if _, err := parseCodexTitle([]byte(failed)); err == nil || !strings.Contains(err.Error(), "model not supported") {
		t.Errorf("failed turn: err = %v", err)
	}
}

// The real titlers run their CLI with the prompt on stdin, in dir.
func TestClaudeTitler_RunsTheCLI(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake claude is a shell script")
	}
	dir := t.TempDir()
	script := filepath.Join(dir, "claude")
	os.WriteFile(script, []byte("#!/bin/sh\ncat > \""+dir+"/stdin.txt\"\npwd -P > \""+dir+"/pwd.txt\"\necho '{\"type\":\"result\",\"result\":\"Fake title\"}'\n"), 0o755)
	got, err := (&ClaudeTitler{Binary: script}).Title(context.Background(), dir, "haiku", "the prompt")
	if err != nil || got != "Fake title" {
		t.Fatalf("title = %q, %v", got, err)
	}
	if stdin, _ := os.ReadFile(filepath.Join(dir, "stdin.txt")); string(stdin) != "the prompt" {
		t.Errorf("stdin = %q", stdin)
	}
}

func TestCodexTitler_RunsTheCLI(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake codex is a shell script")
	}
	dir := t.TempDir()
	script := filepath.Join(dir, "codex")
	os.WriteFile(script, []byte("#!/bin/sh\nfor a in \"$@\"; do printf '%s\\n' \"$a\"; done > \""+dir+"/args.txt\"\ncat > \""+dir+"/stdin.txt\"\necho '{\"type\":\"item.completed\",\"item\":{\"type\":\"agent_message\",\"text\":\"Fake title\"}}'\n"), 0o755)
	got, err := (&CodexTitler{Binary: script}).Title(context.Background(), dir, "gpt-6-luna", "the prompt")
	if err != nil || got != "Fake title" {
		t.Fatalf("title = %q, %v", got, err)
	}
	if stdin, _ := os.ReadFile(filepath.Join(dir, "stdin.txt")); string(stdin) != "the prompt" {
		t.Errorf("stdin = %q", stdin)
	}
	args, _ := os.ReadFile(filepath.Join(dir, "args.txt"))
	var instr string
	for _, line := range strings.Split(string(args), "\n") {
		if v, ok := strings.CutPrefix(line, "model_instructions_file="); ok {
			instr = strings.Trim(v, `"`)
		}
	}
	if instr == "" {
		t.Fatalf("args = %s, want an instructions file", args)
	}
	if _, err := os.Stat(instr); !os.IsNotExist(err) {
		t.Errorf("instructions file %s left behind", instr)
	}
}

func TestCatalog_TitleModel(t *testing.T) {
	c := NewCatalog("", lookPathFor("claude", "codex"))
	if got := c.TitleModel(ProviderClaude); got != "haiku" {
		t.Errorf("claude title model = %q, want haiku", got)
	}
	if got := c.TitleModel(ProviderCodex); got != "" {
		t.Errorf("codex title model before model/list = %q, want none", got)
	}
	c.SetModels(ProviderCodex, []ModelOption{
		{ID: "", Label: "Default model"},
		{ID: "gpt-6.1-sol", Label: "Sol"},
		{ID: "gpt-6-luna", Label: "Luna", Fast: true},
	})
	if got := c.TitleModel(ProviderCodex); got != "gpt-6-luna" {
		t.Errorf("codex title model = %q, want gpt-6-luna", got)
	}
}

func TestCodexModelOptions_MarksFastModels(t *testing.T) {
	raw := []byte(`{"data":[
{"id":"gpt-6.1-sol","displayName":"Sol","description":"Latest workhorse model for coding and everyday work.","isDefault":true},
{"id":"gpt-6-luna","displayName":"Luna","description":"Fast and affordable model for easier tasks."},
{"id":"gpt-5.6-luna","displayName":"Old Luna","description":"Older fast and efficient model."}]}`)
	opts, err := codexModelOptions(raw)
	if err != nil {
		t.Fatal(err)
	}
	var fast []string
	for _, o := range opts {
		if o.Fast {
			fast = append(fast, o.ID)
		}
	}
	if want := []string{"gpt-6-luna", "gpt-5.6-luna"}; !slices.Equal(fast, want) {
		t.Errorf("fast = %v, want %v", fast, want)
	}
}
