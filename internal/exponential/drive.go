package exponential

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/charmbracelet/lipgloss"
	"github.com/palarix/exponential/internal/model"
	"github.com/palarix/exponential/internal/ui"
	"golang.org/x/term"
)

type DriveOptions struct {
	IssueID    string
	Filter     string
	TestCmd    string
	DryRun     bool
	NoMerge    bool
	Resume     bool
	MaxRetries int
	Supervisor string
	Coder      string
}

type DriveResult struct {
	IssueID  string
	Title    string
	Status   string // "done", "blocked", "no-work"
	Attempts int
	Summary  string
	Branch   string
	Messages []string
}

type specResponse struct {
	Ready       bool   `json:"ready"`
	RevisedSpec string `json:"revised_spec,omitempty"`
}

type contextResponse struct {
	Context string   `json:"context"`
	Files   []string `json:"files"`
	TestCmd string   `json:"test_cmd,omitempty"`
}

type evalResponse struct {
	Done     bool   `json:"done"`
	Feedback string `json:"feedback,omitempty"`
}

// driveLogger handles formatted output for xpo drive with collapsible phases.
type driveLogger struct {
	isTTY      bool
	mu         sync.Mutex
	phaseStart time.Time
	phaseTitle string
	lineCount  int // step lines printed in current phase (excl. header)
}

func newDriveLogger() *driveLogger {
	return &driveLogger{
		isTTY: term.IsTerminal(int(os.Stdout.Fd())),
	}
}

var (
	phaseStyle   = lipgloss.NewStyle().Bold(true)
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#198754"))
	failStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#dc3545"))
	warnStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#ffc107"))
	dimStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#6c757d"))
)

func (l *driveLogger) header(issueID, title, agent string) {
	line := fmt.Sprintf("Driving %s %s (%s)", issueID, title, agent)
	if l.isTTY {
		fmt.Printf("\n%s\n", phaseStyle.Render(line))
	} else {
		fmt.Printf("\n%s\n", line)
	}
}

func (l *driveLogger) phase(title string) {
	l.endPhase()
	l.mu.Lock()
	defer l.mu.Unlock()
	l.phaseStart = time.Now()
	l.phaseTitle = title
	l.lineCount = 0
	if l.isTTY {
		fmt.Printf("\n%s %s\n", dimStyle.Render("●"), phaseStyle.Render(title))
	} else {
		fmt.Printf("\n%s\n", title)
	}
}

func (l *driveLogger) endPhase() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.phaseStart.IsZero() {
		return
	}
	elapsed := time.Since(l.phaseStart).Round(time.Millisecond)
	l.phaseStart = time.Time{}

	if l.isTTY {
		// Erase all step lines + phase header + leading blank line
		totalLines := l.lineCount + 2
		for i := 0; i < totalLines; i++ {
			fmt.Printf("\033[A\r\033[K")
		}
		fmt.Printf("%s %s %s\n",
			successStyle.Render("✔"),
			l.phaseTitle,
			dimStyle.Render(fmt.Sprintf("(%s)", formatDuration(elapsed))))
	} else {
		fmt.Printf("completed in %s\n", formatDuration(elapsed))
	}
	l.lineCount = 0
}

func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	m := int(d.Minutes())
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%dm%ds", m, s)
}

func (l *driveLogger) ok(msg string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.lineCount++
	if l.isTTY {
		fmt.Printf("  %s %s\n", successStyle.Render("✔"), msg)
	} else {
		fmt.Printf("  ✔ %s\n", msg)
	}
}

func (l *driveLogger) fail(msg string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.lineCount++
	if l.isTTY {
		fmt.Printf("  %s %s\n", failStyle.Render("✘"), msg)
	} else {
		fmt.Printf("  ✘ %s\n", msg)
	}
}

func (l *driveLogger) warn(msg string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.lineCount++
	if l.isTTY {
		fmt.Printf("  %s %s\n", warnStyle.Render("⚠"), msg)
	} else {
		fmt.Printf("  ⚠ %s\n", msg)
	}
}

func (l *driveLogger) info(msg string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.lineCount++
	if l.isTTY {
		fmt.Printf("  %s\n", dimStyle.Render(msg))
	} else {
		fmt.Printf("  %s\n", msg)
	}
}


// spin runs a braille spinner while fn executes, then clears itself.
// The caller is responsible for printing the result line.
func (l *driveLogger) spin(label string, fn func() error) error {
	if !l.isTTY {
		fmt.Printf("  · %s\n", label)
		return fn()
	}

	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	done := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		i := 0
		for {
			select {
			case <-done:
				fmt.Printf("\r\033[K")
				return
			default:
				fmt.Printf("\r  %s %s", dimStyle.Render(frames[i%len(frames)]), dimStyle.Render(label))
				i++
				time.Sleep(80 * time.Millisecond)
			}
		}
	}()

	err := fn()
	close(done)
	wg.Wait()
	return err
}

var toolLabels = map[string]string{
	"Read":         "Reading",
	"Edit":         "Editing",
	"Write":        "Writing",
	"Bash":         "",
	"WebSearch":    "Searching",
	"WebFetch":     "Fetching",
	"LSP":          "Analyzing",
	"TaskCreate":   "Planning",
	"TaskUpdate":   "Tracking",
	"ToolSearch":   "Loading tools",
	"NotebookEdit": "Editing notebook",
}

func toolLabel(name string) string {
	if label, ok := toolLabels[name]; ok {
		return label
	}
	if strings.HasPrefix(name, "mcp__") {
		parts := strings.SplitN(name, "__", 3)
		if len(parts) == 3 {
			return fmt.Sprintf("Using %s", parts[2])
		}
	}
	return fmt.Sprintf("Using %s", name)
}

func termWidth() int {
	w, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || w <= 0 {
		return 80
	}
	return w
}

var thinkingVerbs = []string{
	"Thinking",
	"Pondering",
	"Mulling",
	"Considering",
	"Tinkering",
	"Deliberating",
	"Noodling",
	"Iterating",
	"Puzzling",
	"Hacking",
	"Brewing",
	"Wrangling",
	"Cooking",
	"Sketching",
	"Assembling",
}

type thinkingCycler struct {
	idx int
}

func (tc *thinkingCycler) next() string {
	verb := thinkingVerbs[tc.idx%len(thinkingVerbs)]
	tc.idx++
	return verb
}

// newStreamDisplay creates a streaming display with a background ticker for
// smooth spinner animation. The callback updates state; the ticker renders.
func (l *driveLogger) newStreamDisplay(label string, showText bool) (*streamDisplay, StreamCallback) {
	sd := &streamDisplay{
		isTTY:     l.isTTY,
		label:     label,
		showText:  showText,
		frames:    []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		idleLabel: thinkingVerbs[0],
		stop:      make(chan struct{}),
	}
	return sd, sd.callback
}

type streamDisplay struct {
	isTTY         bool
	label         string
	showText      bool
	compact       bool
	frames        []string
	frameIdx      int
	headerPrinted bool
	mu            sync.Mutex
	activity      string
	detail        string
	textBuf       string
	thinking      thinkingCycler
	isThinking    bool
	thinkingNext  time.Time
	idleLabel     string
	idleNext      time.Time
	stop          chan struct{}
	wg            sync.WaitGroup
}

func (sd *streamDisplay) callback(ev StreamEvent) {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	switch ev.Kind {
	case StreamToolStart:
		if ev.Text != "" {
			sd.isThinking = false
			sd.activity = toolLabel(ev.Text)
			if ev.ToolArg != "" {
				sd.detail = shortenPath(ev.ToolArg)
			}
			sd.textBuf = ""
		}
	case StreamToolDone:
		sd.activity = ""
		sd.detail = ""
	case StreamThinking:
		sd.isThinking = true
		sd.thinkingNext = time.Now()
		sd.activity = sd.thinking.next()
		sd.detail = ""
		sd.textBuf = ""
	case StreamText:
		if sd.showText {
			sd.textBuf += ev.Text
			if len(sd.textBuf) > 80 {
				sd.textBuf = sd.textBuf[len(sd.textBuf)-80:]
			}
		}
	case StreamStatus:
		// Ignore low-level statuses like "requesting"
	}
}

// start begins the background render ticker.
func (sd *streamDisplay) start() {
	if !sd.isTTY {
		fmt.Printf("  · %s\n", sd.label)
		return
	}
	sd.wg.Add(1)
	go func() {
		defer sd.wg.Done()
		for {
			select {
			case <-sd.stop:
				return
			default:
				sd.mu.Lock()
				sd.render()
				sd.mu.Unlock()
				time.Sleep(40 * time.Millisecond)
			}
		}
	}()
}

func (sd *streamDisplay) render() {
	frame := sd.frames[sd.frameIdx%len(sd.frames)]
	sd.frameIdx++

	if sd.isThinking && time.Now().After(sd.thinkingNext) {
		sd.activity = sd.thinking.next()
		sd.thinkingNext = time.Now().Add(3 * time.Second)
	}

	if !sd.headerPrinted {
		if !sd.compact {
			fmt.Printf("  %s %s\n", dimStyle.Render("○"), sd.label)
		}
		sd.headerPrinted = true
	}

	// Build the detail sub-line
	// Cycle the idle label every 3 seconds
	if sd.activity == "" && sd.detail == "" && time.Now().After(sd.idleNext) {
		sd.idleLabel = sd.thinking.next()
		sd.idleNext = time.Now().Add(3 * time.Second)
	}

	var detail string
	switch {
	case sd.activity != "" && sd.detail != "":
		detail = sd.activity + " " + sd.detail
	case sd.activity != "":
		detail = sd.activity
	case sd.detail != "":
		detail = sd.detail
	default:
		detail = sd.idleLabel
	}
	if sd.textBuf != "" {
		snippet := strings.TrimSpace(sd.textBuf)
		snippet = strings.ReplaceAll(snippet, "\n", " ")
		if len(snippet) > 50 {
			snippet = "…" + snippet[len(snippet)-49:]
		}
		if detail != "" {
			detail += " · " + snippet
		} else {
			detail = snippet
		}
	}

	line := frame + " " + detail
	maxLen := termWidth() - 5
	if maxLen > 0 && len(line) > maxLen {
		line = line[:maxLen-1] + "…"
	}
	fmt.Printf("\r\033[K  %s %s", dimStyle.Render("└"), dimStyle.Render(line))
}

// shortenPath trims a file path to just the last two components for display.
func shortenPath(s string) string {
	if len(s) <= 40 {
		return s
	}
	parts := strings.Split(s, "/")
	if len(parts) <= 2 {
		return s
	}
	return "…/" + strings.Join(parts[len(parts)-2:], "/")
}

func (sd *streamDisplay) clear() {
	close(sd.stop)
	sd.wg.Wait()
	if !sd.isTTY {
		return
	}
	if sd.compact {
		// Compact mode: only the detail line to clear
		fmt.Printf("\r\033[K")
	} else if sd.headerPrinted {
		// Clear detail line, move up, clear header
		fmt.Printf("\r\033[K\033[A\r\033[K")
	}
}

// spinStreaming runs an agent call with streaming progress display.
func (l *driveLogger) spinStreaming(label string, exec AgentExecutor, ctx context.Context, prompt string) (*AgentResult, error) {
	sd, cb := l.newStreamDisplay(label, false)
	sd.start()
	result, err := exec.RunStreaming(ctx, prompt, cb)
	sd.clear()
	return result, err
}

// spinStreamingCompact runs an agent call showing only the detail line
// directly, with no step header. Used when a phase has a single step.
func (l *driveLogger) spinStreamingCompact(exec AgentExecutor, ctx context.Context, prompt string) (*AgentResult, error) {
	sd, cb := l.newStreamDisplay("", false)
	sd.compact = true
	sd.start()
	result, err := exec.RunStreaming(ctx, prompt, cb)
	sd.clear()
	return result, err
}

// spinStreamingJSON runs a supervisor agent call with streaming progress,
// parses the JSON result, and suppresses text deltas (which are raw JSON).
func (l *driveLogger) spinStreamingJSON(label string, exec AgentExecutor, ctx context.Context, prompt string, out interface{}) error {
	sd, cb := l.newStreamDisplay(label, false)
	sd.start()
	err := supervisorStreamJSON(ctx, exec, prompt, out, cb)
	sd.clear()
	return err
}

func (c *Client) DriveIssue(opts DriveOptions) (*DriveResult, error) {
	log := newDriveLogger()
	cfg := c.Config
	supervisor := coalesce(opts.Supervisor, cfg.Drive.Supervisor.Agent, "claude")
	supervisorModel := cfg.Drive.Supervisor.Model
	coder := coalesce(opts.Coder, cfg.Drive.Coder.Agent, "claude")
	isGit := CheckGitRepo()

	// ── Pre-flight ──────────────────────────────────────────────────
	if c.local == nil {
		return nil, fmt.Errorf("xpo drive requires a local project (not a remote server)")
	}

	issue, err := c.pickIssue(opts)
	if err != nil {
		return nil, err
	}
	if issue == nil {
		return &DriveResult{Status: "no-work", Messages: []string{"Nothing to pick"}}, nil
	}

	result := &DriveResult{IssueID: issue.ID, Title: issue.Title}

	var problems []string
	if isGit && !IsWorkingTreeClean() {
		problems = append(problems, "working tree is not clean")
	}
	if _, err := exec.LookPath(agentBinary(supervisor)); err != nil {
		problems = append(problems, fmt.Sprintf("supervisor %q not in PATH", supervisor))
	}
	if _, err := exec.LookPath(agentBinary(coder)); err != nil {
		problems = append(problems, fmt.Sprintf("coder %q not in PATH", coder))
	}

	agentLine := supervisor
	if coder != supervisor {
		agentLine = fmt.Sprintf("%s (supervisor) + %s (coder)", supervisor, coder)
	}

	if opts.DryRun {
		log.header(issue.ID, issue.Title, agentLine)
		if len(problems) > 0 {
			log.fail(strings.Join(problems, "; "))
		} else {
			log.ok("Ready")
		}
		fmt.Println()
		log.info("xpo drive " + issue.ID)
		fmt.Println()
		result.Status = "dry-run"
		return result, nil
	}

	if len(problems) > 0 {
		return nil, fmt.Errorf("pre-flight failed: %s", strings.Join(problems, "; "))
	}

	driveStart := time.Now()

	// Create agent executors
	supExec := NewAgentExecutor(supervisor, supervisorModel)
	coderExec := NewAgentExecutor(coder)

	// Set agent identity — attribute work to the agent, not the user
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "local"
	}
	c.UserOverride = fmt.Sprintf("%s <agent@%s>", supervisor, hostname)

	timeout := 30 * time.Minute
	if cfg.Drive.Timeout != "" {
		if d, err := time.ParseDuration(cfg.Drive.Timeout); err == nil {
			timeout = d
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	maxRetries := opts.MaxRetries
	if maxRetries <= 0 {
		maxRetries = cfg.Drive.MaxRetries
	}
	if maxRetries <= 0 {
		maxRetries = 3
	}

	// ── Preparation ─────────────────────────────────────────────────
	log.header(issue.ID, issue.Title, agentLine)
	log.phase("Preparation")

	branch, _, _, err := c.StartWork(issue.ID, true)
	if err != nil {
		return nil, fmt.Errorf("start: %w", err)
	}
	result.Branch = branch
	if branch != "" {
		log.ok(fmt.Sprintf("Branch %s created", dimStyle.Render(branch)))
	}

	// From here on, work is in flight — errors include recovery hints
	recoverHint := func(err error) error {
		hint := fmt.Sprintf("\n\nRecovery:\n  Issue %s is DOING.", issue.ID)
		if branch != "" {
			hint += fmt.Sprintf(" Work is on branch %s.", branch)
			hint += "\n  Resume:  xpo drive --resume"
			hint += fmt.Sprintf("\n  Discard: git checkout %s && git branch -D %s && xpo update %s --status PLANNED", DefaultBranch(), branch, issue.ID)
		} else {
			hint += "\n  Resume: xpo drive --resume"
			hint += fmt.Sprintf("\n  Reset:  xpo update %s --status PLANNED", issue.ID)
		}
		return fmt.Errorf("%w%s", err, hint)
	}

	spec := issue.Description

	// Evaluate spec — first pass is always just evaluation
	var sr specResponse
	err = log.spinStreamingJSON("Evaluating spec", supExec, ctx, buildSpecPrompt(spec), &sr)
	if err != nil {
		return nil, recoverHint(fmt.Errorf("spec evaluation: %w", err))
	}
	if sr.Ready {
		log.ok("Spec is implementation-ready")
	} else {
		log.warn("Spec needs work")
		// Revise up to 3 times
		for attempt := 0; attempt < 3; attempt++ {
			if sr.RevisedSpec != "" {
				spec = sr.RevisedSpec
			}
			reviseLabel := "Revising spec"
			if attempt > 0 {
				reviseLabel = fmt.Sprintf("Revising spec (attempt %d)", attempt+1)
			}
			err = log.spinStreamingJSON(reviseLabel, supExec, ctx, buildSpecPrompt(spec), &sr)
			if err != nil {
				return nil, recoverHint(fmt.Errorf("spec revision: %w", err))
			}
			if sr.Ready {
				if sr.RevisedSpec != "" {
					spec = sr.RevisedSpec
				}
				if err := c.WriteSpec(issue.ID, spec); err != nil {
					return nil, recoverHint(fmt.Errorf("save spec: %w", err))
				}
				log.ok("Spec revised and ready")
				break
			}
			if attempt == 2 {
				return nil, recoverHint(fmt.Errorf("spec not ready after 3 revisions"))
			}
		}
	}

	// Plan implementation
	var cr contextResponse
	err = log.spinStreamingJSON("Planning implementation", supExec, ctx, buildContextPrompt(spec), &cr)
	if err != nil {
		return nil, recoverHint(fmt.Errorf("planning: %w", err))
	}
	testCmd := coalesce(opts.TestCmd, cfg.Drive.TestCmd, cr.TestCmd)
	if testCmd != "" {
		if _, err := runTestCmd(testCmd); err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() != 1 {
				// Exit code 1 = tests ran but failed (legitimate).
				// Other codes (2=make error, 126=permission, 127=not found, etc.)
				// indicate the test command itself is broken.
				log.warn(fmt.Sprintf("Test command failed (exit %d): %s — skipping tests", exitErr.ExitCode(), testCmd))
				testCmd = ""
			}
		}
	}
	log.ok(fmt.Sprintf("Context ready — %d files identified", len(cr.Files)))

	// ── Implementation ──────────────────────────────────────────────
	base := DefaultBranch()
	feedback := ""
	lastTestOutput := ""
	for attempt := 1; attempt <= maxRetries; attempt++ {
		result.Attempts = attempt

		if attempt == 1 {
			log.phase("Implementation")
		} else {
			log.phase(fmt.Sprintf("Implementation — retry %d/%d", attempt, maxRetries))
		}

		coderPrompt := buildCoderPrompt(spec, cr.Context, cr.Files, testCmd, branch, feedback)
		_, err := log.spinStreamingCompact(coderExec, ctx, coderPrompt)
		if err != nil {
			log.fail(fmt.Sprintf("Agent error: %v", err))
			feedback = fmt.Sprintf("Agent failed with error: %v", err)
			if attempt == maxRetries {
				break
			}
			continue
		}
		log.ok("Done")

		// ── Testing ─────────────────────────────────────────────────
		log.phase("Testing")
		diff := GetDiffText(branch, base)
		var testOut string
		var testErr error
		if testCmd != "" {
			_ = log.spin("Running tests", func() error {
				testOut, testErr = runTestCmd(testCmd)
				lastTestOutput = testOut
				return nil
			})
			if testErr != nil {
				log.fail("Tests failed")
			} else {
				log.ok("Tests passed")
			}
		} else {
			log.info("No test command — set drive.test_cmd in .xpo/config.yaml or pass --test-cmd")
		}

		testsSkipped := testCmd == ""
		var er evalResponse
		err = log.spinStreamingJSON("Reviewing implementation", supExec, ctx, buildEvalPrompt(spec, diff, testOut, testErr, testsSkipped), &er)
		if err != nil {
			return nil, recoverHint(fmt.Errorf("evaluation: %w", err))
		}

		if er.Done {
			log.ok("Review passed")
			break
		}

		log.fail(ui.Truncate(er.Feedback, 100))
		feedback = er.Feedback

		if attempt == maxRetries {
			log.endPhase()
			status := string(model.StatusBlocked)
			c.UpdateIssue(issue.ID, model.UpdatePayload{Status: &status}, "drive")
			c.AddComment(issue.ID, fmt.Sprintf("## xpo drive — blocked\n\nFailed after %d attempts.\n\n**Last feedback:**\n%s", maxRetries, er.Feedback))
			result.Status = "blocked"
			fmt.Println()
			log.warn(fmt.Sprintf("Blocked after %d attempts. See %s for details.", maxRetries, issue.ID))
			fmt.Println()
			return result, nil
		}
	}

	// ── Walkthrough ─────────────────────────────────────────────────
	log.phase("Walkthrough")
	diff := GetDiffText(branch, base)
	var walkthrough string
	walkthroughResult, err := log.spinStreaming("Preparing walkthrough", supExec, ctx,
		buildWalkthroughPrompt(spec, diff, testCmd, lastTestOutput))
	if err == nil {
		walkthrough = walkthroughResult.Output
	}
	if err != nil {
		walkthrough = fmt.Sprintf("Implementation completed in %d attempt(s).", result.Attempts)
	}

	c.WriteWalkthrough(issue.ID, walkthrough)
	log.ok(fmt.Sprintf("Walkthrough saved → xpo artifact show %s --walkthrough", issue.ID))

	// Commit issues.db changes on the feature branch so checkout doesn't fail
	if isGit {
		GitCommit(fmt.Sprintf("xpo: add walkthrough for %s", issue.ID))
	}

	// ── Cleanup ─────────────────────────────────────────────────────
	if isGit && !opts.NoMerge {
		log.phase("Cleanup")
		mergeResult, err := c.MergeIssue(issue.ID, MergeOptions{
			Strategy:     MergeStrategySquash,
			DeleteBranch: true,
		})
		if err != nil {
			return nil, recoverHint(fmt.Errorf("merge: %w", err))
		}
		for _, m := range mergeResult.Messages {
			log.ok(m)
		}
	} else if isGit && branch != "" {
		log.phase("Cleanup")
		log.info(fmt.Sprintf("Branch %s ready for review", dimStyle.Render(branch)))
	}

	// ── Final ───────────────────────────────────────────────────────
	log.endPhase()
	wallTime := time.Since(driveStart).Round(time.Second)
	stats := mergeStats(supExec.Stats(), coderExec.Stats())
	result.Status = "done"
	result.Summary = walkthrough
	secondary := formatDuration(wallTime)
	if stats.TokensIn > 0 || stats.TokensOut > 0 {
		secondary += fmt.Sprintf(" · %dk in / %dk out", stats.TokensIn/1000, stats.TokensOut/1000)
	}
	if stats.CostUSD > 0 {
		secondary += fmt.Sprintf(" · $%.2f", stats.CostUSD)
	}
	fmt.Printf("%s Done %s\n",
		successStyle.Render("✔"),
		dimStyle.Render("("+secondary+")"))
	fmt.Println()
	return result, nil
}

func formatIssueMeta(issue *model.Issue) string {
	suffix := ""
	if issue.Estimate > 0 {
		suffix += fmt.Sprintf(" [%d pts]", issue.Estimate)
	}
	if len(issue.Labels) > 0 {
		suffix += fmt.Sprintf(" (%s)", strings.Join(issue.Labels, ", "))
	}
	return fmt.Sprintf("%s %s%s", dimStyle.Render(issue.ID), issue.Title, dimStyle.Render(suffix))
}

func issueTypeLabel(labels []string) string {
	for _, l := range labels {
		switch strings.ToLower(l) {
		case "bug":
			return "Bug fix"
		case "feature":
			return "Feature"
		case "improvement":
			return "Improvement"
		case "epic":
			return "Epic"
		case "task":
			return "Task"
		}
	}
	return "Issue"
}

func (c *Client) pickIssue(opts DriveOptions) (*model.Issue, error) {
	if opts.IssueID != "" {
		issue, err := c.GetIssue(opts.IssueID)
		if err != nil {
			return nil, fmt.Errorf("issue %s not found: %w", opts.IssueID, err)
		}
		return issue, nil
	}

	// --resume or default: check for an in-progress DOING issue first
	if opts.Resume {
		doing, err := c.ListIssues(FilterOptions{Statuses: []string{string(model.StatusDoing)}})
		if err != nil {
			return nil, err
		}
		if len(doing) == 0 {
			return nil, fmt.Errorf("no in-progress issues to resume")
		}
		if len(doing) == 1 {
			return doing[0], nil
		}
		return nil, fmt.Errorf("multiple in-progress issues — specify the issue ID as an argument to resume")
	}

	// Without --resume, prefer a DOING issue if exactly one exists (pick up
	// where a previous drive left off), otherwise pick from PLANNED.
	doing, _ := c.ListIssues(FilterOptions{Statuses: []string{string(model.StatusDoing)}})
	if len(doing) == 1 {
		return doing[0], nil
	}

	filter := FilterOptions{
		Statuses: []string{string(model.StatusPlanned)},
	}
	if opts.Filter != "" {
		filter.Label = opts.Filter
	}

	issues, err := c.ListIssues(filter)
	if err != nil {
		return nil, err
	}

	// Filter out epics (they have children) and issues with unresolved blockers
	childOf := make(map[string]bool)
	for _, iss := range issues {
		if iss.ParentID != "" {
			childOf[iss.ParentID] = true
		}
	}
	var eligible []*model.Issue
	for _, iss := range issues {
		if childOf[iss.ID] {
			continue
		}
		if hasUnresolvedBlockers(iss, c) {
			continue
		}
		eligible = append(eligible, iss)
	}

	if len(eligible) == 0 {
		return nil, nil
	}

	// Sort by priority (urgent first: 1=Urgent, 2=High, 3=Medium, 4=Low,
	// 0=none goes last), then by board order (sort_order), then by
	// creation time. Board order is primary within the same priority —
	// the user arranged issues in the order they want them worked.
	sort.SliceStable(eligible, func(i, j int) bool {
		pi, pj := eligible[i].Priority, eligible[j].Priority
		if pi != pj {
			if pi == 0 {
				return false
			}
			if pj == 0 {
				return true
			}
			return pi < pj
		}
		if eligible[i].SortOrder != eligible[j].SortOrder {
			return eligible[i].SortOrder < eligible[j].SortOrder
		}
		return eligible[i].CreatedAt.Before(eligible[j].CreatedAt)
	})

	return eligible[0], nil
}

func hasUnresolvedBlockers(issue *model.Issue, c *Client) bool {
	for _, dep := range issue.Dependencies {
		if dep.Kind != model.DependencyBlockedBy && dep.Kind != model.DependencyDependsOn {
			continue
		}
		blocker, err := c.GetIssue(dep.TargetID)
		if err != nil {
			continue
		}
		if blocker.Status != model.StatusDone {
			return true
		}
	}
	return false
}

// extractJSON finds the first top-level JSON object in text.
func extractJSON(text string) []byte {
	depth := 0
	start := -1
	for i, r := range text {
		switch r {
		case '{':
			if depth == 0 {
				start = i
			}
			depth++
		case '}':
			depth--
			if depth == 0 && start >= 0 {
				return []byte(text[start : i+1])
			}
		}
	}
	return nil
}

// supervisorJSON runs a prompt via the executor and parses structured JSON from the output.
func supervisorJSON(ctx context.Context, exec AgentExecutor, prompt string, out interface{}) error {
	result, err := exec.Run(ctx, prompt)
	if err != nil {
		return err
	}
	blob := extractJSON(result.Output)
	if blob == nil {
		return fmt.Errorf("no JSON found in agent output:\n%s", truncate(result.Output, 500))
	}
	return json.Unmarshal(blob, out)
}

// supervisorStreamJSON is like supervisorJSON but streams progress events via cb.
func supervisorStreamJSON(ctx context.Context, exec AgentExecutor, prompt string, out interface{}, cb StreamCallback) error {
	result, err := exec.RunStreaming(ctx, prompt, cb)
	if err != nil {
		return err
	}
	blob := extractJSON(result.Output)
	if blob == nil {
		return fmt.Errorf("no JSON found in agent output:\n%s", truncate(result.Output, 500))
	}
	return json.Unmarshal(blob, out)
}

func runTestCmd(testCmd string) (string, error) {
	cmd := exec.Command("sh", "-c", testCmd)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func buildSpecPrompt(spec string) string {
	return fmt.Sprintf(`You are a supervisor agent overseeing an AI coding agent. Your job is to evaluate specs, plan work, review implementations, and write walkthroughs. Always respond ONLY in the JSON format requested.

First task: evaluate whether this issue spec is ready for implementation.

## Spec

%s

A good spec has clear acceptance criteria, enough detail to know what "done" looks like, and no ambiguous requirements. If the spec is NOT ready, revise it to be implementation-ready while preserving the original intent.

Respond ONLY in JSON: {"ready": true} or {"ready": false, "revised_spec": "...improved spec..."}`, spec)
}

func buildContextPrompt(spec string) string {
	return fmt.Sprintf(`Next task: prepare context for the coding agent to implement this spec.

## Spec

%s

Examine the repository and identify:
1. The files, modules, and patterns relevant to implementation
2. Key architectural patterns the implementation should follow
3. The appropriate test command for this project

Respond ONLY in JSON:
{"context": "...detailed implementation guidance...", "files": ["path/to/relevant/file1", "path/to/file2"], "test_cmd": "make test"}`, spec)
}

func buildCoderPrompt(spec, context string, files []string, testCmd, branch, feedback string) string {
	var b strings.Builder
	b.WriteString("Implement the following spec on the current branch.\n\n")
	b.WriteString("## Spec\n\n")
	b.WriteString(spec)
	b.WriteString("\n\n## Context\n\n")
	b.WriteString(context)
	if len(files) > 0 {
		b.WriteString("\n\n## Relevant files\n\n")
		for _, f := range files {
			b.WriteString("- ")
			b.WriteString(f)
			b.WriteString("\n")
		}
	}
	b.WriteString("\n\n## Instructions\n\n")
	b.WriteString(fmt.Sprintf("- Work on branch: %s\n", branch))
	b.WriteString(fmt.Sprintf("- Run tests with: %s\n", testCmd))
	b.WriteString("- Commit your changes when tests pass\n")
	b.WriteString("- Do NOT manage issue lifecycle (no status transitions, no comments via xpo tools) — the driver handles that\n")
	if feedback != "" {
		b.WriteString("\n## Feedback from previous attempt\n\n")
		b.WriteString(feedback)
		b.WriteString("\n")
	}
	return b.String()
}

func buildEvalPrompt(spec, diff, testOutput string, testErr error, testsSkipped bool) string {
	var testSection string
	if testsSkipped {
		testSection = "No test command was configured — evaluate based on the diff alone.\n"
	} else {
		testStatus := "PASSED"
		if testErr != nil {
			testStatus = "FAILED"
		}
		testSection = fmt.Sprintf("Test output (%s):\n%s\n", testStatus, truncate(testOutput, 10000))
	}

	return fmt.Sprintf(`Next task: the coding agent has completed its implementation. Review whether it meets the spec.

## Spec
%s

## Diff
%s

## Tests
%s
Does the diff address all requirements in the spec? Are there any obvious issues?

Respond ONLY in JSON: {"done": true} or {"done": false, "feedback": "...what needs to change..."}`,
		spec, truncate(diff, 50000), testSection)
}

func buildWalkthroughPrompt(spec, diff, testCmd, testOutput string) string {
	testSection := ""
	if testOutput != "" {
		testSection = fmt.Sprintf("\n## Test output\n%s\n", truncate(testOutput, 5000))
	}
	verifyCmd := testCmd
	if verifyCmd == "" {
		verifyCmd = "the project's test command"
	}
	return fmt.Sprintf(`Final task: write a walkthrough of the implementation for a code reviewer. Explain it like a senior engineer walking a colleague through a PR.

## Spec
%s

## Diff
%s
%s
Write the walkthrough in markdown with these sections:

### Rationale
Why this approach? What alternatives were considered and why were they rejected? 1-3 sentences.

### Changes
File-by-file walkthrough. For each changed file, one line explaining what changed and why.
Use backtick code spans for file and function names.

### Decisions
Any non-obvious choices or tradeoffs. Skip this section if everything was straightforward.

### How to verify
Concrete steps to test the changes. Include commands to run (e.g. %s),
specific behavior to check, and edge cases to try.

### What to look out for
Anything the reviewer should look closely at. Skip this section if there are no concerns.

Keep it concise — this is a walkthrough, not a novel. No JSON — just markdown.`,
		spec, truncate(diff, 40000), testSection, verifyCmd)
}

func coalesce(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func truncate(s string, max int) string {
	s = strings.TrimRightFunc(s, unicode.IsSpace)
	if len(s) <= max {
		return s
	}
	return s[:max] + "\n... (truncated)"
}
