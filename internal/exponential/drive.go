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

	"github.com/charmbracelet/glamour"
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

// driveLogger handles formatted output for xpo drive with phase headers.
type driveLogger struct {
	isTTY      bool
	mu         sync.Mutex
	phaseStart time.Time
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

func (l *driveLogger) phase(title string) {
	l.endPhase()
	l.mu.Lock()
	defer l.mu.Unlock()
	l.phaseStart = time.Now()
	if l.isTTY {
		fmt.Printf("\n%s\n", phaseStyle.Render(title))
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
		fmt.Printf("  %s\n", dimStyle.Render(fmt.Sprintf("completed in %s", formatDuration(elapsed))))
	} else {
		fmt.Printf("  completed in %s\n", formatDuration(elapsed))
	}
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
	if l.isTTY {
		fmt.Printf("  %s %s\n", successStyle.Render("✔"), msg)
	} else {
		fmt.Printf("  ✔ %s\n", msg)
	}
}

func (l *driveLogger) fail(msg string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.isTTY {
		fmt.Printf("  %s %s\n", failStyle.Render("✘"), msg)
	} else {
		fmt.Printf("  ✘ %s\n", msg)
	}
}

func (l *driveLogger) warn(msg string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.isTTY {
		fmt.Printf("  %s %s\n", warnStyle.Render("⚠"), msg)
	} else {
		fmt.Printf("  ⚠ %s\n", msg)
	}
}

func (l *driveLogger) info(msg string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.isTTY {
		fmt.Printf("  %s\n", dimStyle.Render(msg))
	} else {
		fmt.Printf("  %s\n", msg)
	}
}

func (l *driveLogger) walkthrough(text string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	width := 80
	if l.isTTY {
		if w := ui.TerminalWidth(); w > 4 {
			width = w - 4
		}
	}

	if l.isTTY {
		renderer, err := glamour.NewTermRenderer(
			glamour.WithAutoStyle(),
			glamour.WithWordWrap(width),
		)
		if err == nil {
			rendered, err := renderer.Render(strings.TrimSpace(text))
			if err == nil {
				fmt.Print(rendered)
				return
			}
		}
	}

	// Fallback: plain text with word wrap
	wrapped := wordWrap(strings.TrimSpace(text), width)
	for _, line := range strings.Split(wrapped, "\n") {
		fmt.Printf("  %s\n", line)
	}
}

func wordWrap(text string, width int) string {
	if width <= 0 {
		return text
	}
	var out strings.Builder
	for _, paragraph := range strings.Split(text, "\n") {
		if out.Len() > 0 {
			out.WriteByte('\n')
		}
		line := ""
		for _, word := range strings.Fields(paragraph) {
			if line == "" {
				line = word
			} else if len(line)+1+len(word) > width {
				out.WriteString(line)
				out.WriteByte('\n')
				line = word
			} else {
				line += " " + word
			}
		}
		out.WriteString(line)
	}
	return out.String()
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

func (c *Client) DriveIssue(opts DriveOptions) (*DriveResult, error) {
	log := newDriveLogger()
	cfg := c.Config
	supervisor := coalesce(opts.Supervisor, cfg.Drive.Supervisor, "claude")
	coder := coalesce(opts.Coder, cfg.Drive.Coder, "claude")
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

	issueMeta := formatIssueMeta(issue)
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
		log.phase("Pre-flight")
		log.ok(issueMeta)
		log.ok(fmt.Sprintf("Agent: %s", agentLine))
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
	supExec := NewAgentExecutor(supervisor)
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
	log.phase("Preparation")
	log.ok(issueMeta)
	log.ok(fmt.Sprintf("Agent: %s", agentLine))

	branch, _, err := c.StartWork(issue.ID, true)
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

	// Evaluate spec
	var sr specResponse
	for attempt := 0; attempt < 3; attempt++ {
		err := log.spin("Evaluating spec", func() error {
			return supervisorJSON(ctx, supExec, buildSpecPrompt(spec), &sr)
		})
		if err != nil {
			return nil, recoverHint(fmt.Errorf("spec evaluation: %w", err))
		}
		if sr.Ready {
			log.ok("Spec is implementation-ready")
			break
		}
		if sr.RevisedSpec != "" {
			spec = sr.RevisedSpec
			if err := c.WriteSpec(issue.ID, spec); err != nil {
				return nil, recoverHint(fmt.Errorf("update spec: %w", err))
			}
			log.ok("Spec revised, re-evaluating")
		}
		if attempt == 2 {
			return nil, recoverHint(fmt.Errorf("spec not ready after 3 revisions"))
		}
	}

	// Plan implementation
	var cr contextResponse
	err = log.spin("Planning implementation", func() error {
		return supervisorJSON(ctx, supExec, buildContextPrompt(spec), &cr)
	})
	if err != nil {
		return nil, recoverHint(fmt.Errorf("planning: %w", err))
	}
	testCmd := coalesce(opts.TestCmd, cfg.Drive.TestCmd, cr.TestCmd)
	if testCmd == "" {
		return nil, recoverHint(fmt.Errorf("no test command: set --test-cmd, configure [drive] test_cmd, or ensure the supervisor detects one"))
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
		workLabel := fmt.Sprintf("Working on %s %s", dimStyle.Render(issue.ID), issue.Title)
		err := log.spin(workLabel, func() error {
			_, e := coderExec.Run(ctx, coderPrompt)
			return e
		})
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

		var er evalResponse
		err = log.spin("Reviewing implementation", func() error {
			return supervisorJSON(ctx, supExec, buildEvalPrompt(spec, diff, testOut, testErr), &er)
		})
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
	elapsed := time.Since(issue.UpdatedAt).Round(time.Second)
	var walkthrough string
	err = log.spin("Preparing walkthrough", func() error {
		prompt := buildWalkthroughPrompt(spec, diff, testCmd, lastTestOutput)
		r, sErr := supExec.Run(ctx, prompt)
		if sErr == nil {
			walkthrough = r.Output
		}
		return sErr
	})
	if err != nil {
		walkthrough = fmt.Sprintf("Implementation completed in %d attempt(s).", result.Attempts)
	}

	c.WriteWalkthrough(issue.ID, walkthrough)
	status := string(model.StatusDone)
	c.UpdateIssue(issue.ID, model.UpdatePayload{Status: &status}, "drive")
	log.walkthrough(walkthrough)

	// Commit issues.db changes on the feature branch so checkout doesn't fail
	if isGit {
		GitCommit(fmt.Sprintf("xpo: close %s", issue.ID))
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
	issueType := issueTypeLabel(issue.Labels)
	stats := mergeStats(supExec.Stats(), coderExec.Stats())
	result.Status = "done"
	result.Summary = walkthrough
	fmt.Println()
	secondary := fmt.Sprintf("%s · %s cycle · %d pts", wallTime, elapsed, issue.Estimate)
	if usage := formatStats(stats); usage != "" {
		secondary += " · " + usage
	}
	fmt.Printf("%s %s  %s\n",
		successStyle.Render("✔"),
		fmt.Sprintf("%s complete", issueType),
		dimStyle.Render(secondary))
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

func runTestCmd(testCmd string) (string, error) {
	cmd := exec.Command("sh", "-c", testCmd)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func buildSpecPrompt(spec string) string {
	return fmt.Sprintf(`You are evaluating whether an issue spec is ready for an AI coding agent to implement.

Here is the issue spec:

%s

Is this spec implementation-ready? A good spec has:
- Clear acceptance criteria or expected behavior
- Enough detail to know what "done" looks like
- No ambiguous requirements

If the spec is NOT ready, revise it to be implementation-ready while preserving the original intent.

Respond ONLY in JSON: {"ready": true} or {"ready": false, "revised_spec": "...improved spec..."}`, spec)
}

func buildContextPrompt(spec string) string {
	return fmt.Sprintf(`You are preparing context for an AI coding agent that will implement the following spec.

Spec:
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
	if feedback != "" {
		b.WriteString("\n## Feedback from previous attempt\n\n")
		b.WriteString(feedback)
		b.WriteString("\n")
	}
	return b.String()
}

func buildEvalPrompt(spec, diff, testOutput string, testErr error) string {
	testStatus := "PASSED"
	if testErr != nil {
		testStatus = "FAILED"
	}
	return fmt.Sprintf(`You are evaluating whether an implementation meets a spec.

## Spec
%s

## Diff
%s

## Test output (%s)
%s

Did this implementation meet the goal? Consider:
- Does the diff address all requirements in the spec?
- Did tests pass?
- Are there any obvious issues?

Respond ONLY in JSON: {"done": true} or {"done": false, "feedback": "...what needs to change..."}`,
		spec, truncate(diff, 50000), testStatus, truncate(testOutput, 10000))
}

func buildWalkthroughPrompt(spec, diff, testCmd, testOutput string) string {
	testSection := ""
	if testOutput != "" {
		testSection = fmt.Sprintf("\n## Test output\n%s\n", truncate(testOutput, 5000))
	}
	return fmt.Sprintf(`You are writing a walkthrough of an implementation for a code reviewer.
Explain it like a senior engineer walking a colleague through a PR.

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
		spec, truncate(diff, 40000), testSection, testCmd)
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
