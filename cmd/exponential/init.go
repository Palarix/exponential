package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"charm.land/huh/v2"
	"github.com/palarix/exponential/internal/config"
	"github.com/palarix/exponential/internal/exponential"
	"github.com/palarix/exponential/internal/ui"
	"github.com/palarix/exponential/internal/version"
	"github.com/spf13/cobra"
)

var (
	initForce    bool
	initYes      bool
	initPrefix   string
	initAgents   string
	initWorkflow string
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Set up Exponential in this project",
	Long: `Initialize Exponential: create the .xpo directory, configure MCP servers,
and install agent skills — all in one step.

The first run also asks which workflow agents follow and writes it as the
xpo-workflow skill. The project owns that skill: re-running xpo init never
changes it.

Re-running in an initialized project refreshes integration files to the
current CLI version. Add --workflow to (re)write the workflow skill from a
template; existing workflow skills are replaced after you confirm.`,
	Run: runInit,
}

func runInit(cmd *cobra.Command, args []string) {
	interactive := isInteractive() && !initYes

	// "--workflow guided" (with a space) leaves the name as an argument.
	if initWorkflow == workflowAsk && len(args) > 0 {
		initWorkflow = args[0]
	}
	// Validate --workflow before any prompt or write.
	if initWorkflow != "" && initWorkflow != workflowAsk {
		if _, err := workflowFromFlag(initWorkflow); err != nil {
			fmt.Printf("%s %v\n", ui.ErrorPrefix, err)
			os.Exit(1)
		}
	}

	// Pre-flight
	alreadyInit := false
	if _, err := os.Stat(".xpo"); err == nil {
		alreadyInit = true
	}

	if alreadyInit {
		if strings.EqualFold(initWorkflow, "none") {
			fmt.Printf("%s --workflow none: nothing to set up. Delete the xpo-workflow skill folders to remove the workflow.\n", ui.ErrorPrefix)
			os.Exit(1)
		}
		runReinit(interactive)
		if initWorkflow != "" {
			runWorkflowSetup(interactive)
		}
		return
	}

	// --- Collect all answers before writing anything ---
	fmt.Printf("\n%s\n", ui.Banner())
	cleanup := withCtrlC("")
	defer cleanup()

	needsGitInit := !exponential.CheckGitRepo()
	if needsGitInit {
		if interactive {
			if !promptConfirm("Exponential needs git. Create a repository here?", true) {
				abortInit("git init && xpo init")
			}
		} else {
			fmt.Printf("%s Not a git repository. Run git init first, or run xpo init interactively.\n", ui.ErrorPrefix)
			os.Exit(1)
		}
	}

	cwd, _ := os.Getwd()
	home, _ := os.UserHomeDir()
	displayPath := cwd
	if home != "" && strings.HasPrefix(cwd, home) {
		displayPath = "~" + cwd[len(home):]
	}
	fmt.Printf("\nProject:   %s\n", displayPath)

	prefix := resolvePrefix(interactive)
	agents := resolveAgentSelection(interactive)
	wf := resolveWorkflowChoice(interactive)

	// Change plan
	plan := exponential.ComputeInitPlan(agents, wf)
	fmt.Printf("\nChanges\n")
	if needsGitInit {
		fmt.Printf("  + .git/\n")
	}
	for _, c := range plan.Changes {
		marker := "~"
		if c.Action == "create" {
			marker = "+"
		}
		fmt.Printf("  %s %s\n", marker, c.Path)
	}

	if interactive {
		if !promptConfirm("Apply changes?", true) {
			abortInit("xpo init")
		}
	} else if !initYes {
		fmt.Printf("\nRun with --yes to apply, or run interactively.\n")
		os.Exit(1)
	}

	// --- Apply: all writes happen here ---
	cleanup()

	if needsGitInit {
		gitCmd := exec.Command("git", "init")
		if out, err := gitCmd.CombinedOutput(); err != nil {
			fmt.Printf("%s git init failed: %v\n%s", ui.ErrorPrefix, err, string(out))
			os.Exit(1)
		}
	}

	applyInit(prefix, agents, wf)

	fmt.Printf("\n%s Exponential is ready\n", ui.OKPrefix)
	fmt.Printf("\nNext\n")
	fmt.Printf("  Commit to share with your team:  git add -A && git commit -m \"Add Exponential\"\n")
	fmt.Printf("  Create your first issue:         xpo new \"…\"\n\n")
}

func resolvePrefix(interactive bool) string {
	if initPrefix != "" {
		return strings.TrimSuffix(initPrefix, "-")
	}

	suggested := exponential.DefaultPrefix()

	if !interactive {
		return suggested
	}

	prefix := promptInput("Issue prefix", suggested, func(val string) string {
		return fmt.Sprintf("Issue IDs will look like %s-ab3ef2", val)
	})

	return strings.TrimSuffix(strings.TrimSpace(prefix), "-")
}

func resolveAgentSelection(interactive bool) []exponential.AgentConfig {
	if initAgents != "" {
		names := strings.Split(initAgents, ",")
		var result []exponential.AgentConfig
		for _, name := range names {
			agent, ok := exponential.LookupAgent(strings.TrimSpace(name))
			if !ok {
				fmt.Printf("%s Unknown agent: %s\n", ui.ErrorPrefix, name)
				fmt.Printf("  Available: %s\n", strings.Join(exponential.AgentRegistryNames(), ", "))
				os.Exit(1)
			}
			result = append(result, agent)
		}
		return result
	}

	allAgents := exponential.AgentRegistry
	detected := exponential.DetectInstalledAgents()
	detectedSet := map[string]bool{}
	for _, d := range detected {
		detectedSet[d.Name] = true
	}

	if !interactive {
		return detected
	}

	// Build multi-select: found agents first (pre-selected), then not-found (muted)
	type agentEntry struct {
		agent    exponential.AgentConfig
		detected bool
	}
	var found, notFound []agentEntry
	for _, agent := range allAgents {
		if agent.Name == "Generic Agent" {
			continue
		}
		if detectedSet[agent.Name] {
			found = append(found, agentEntry{agent, true})
		} else {
			notFound = append(notFound, agentEntry{agent, false})
		}
	}
	sorted := append(found, notFound...)

	var options []huh.Option[string]
	var preSelected []string
	for _, entry := range sorted {
		status := ui.MutedStyle.Render("not found")
		if entry.detected {
			status = "found"
			preSelected = append(preSelected, entry.agent.Name)
		}
		label := fmt.Sprintf("%-16s %s", entry.agent.Name, status)
		if !entry.detected {
			label = ui.MutedStyle.Render(label)
		}
		options = append(options, huh.NewOption(label, entry.agent.Name).Selected(entry.detected))
	}

	fmt.Printf("\n%s Agent integrations\n", ui.AccentStyle.Render("?"))
	var selected []string
	huh.NewMultiSelect[string]().
		Title("Select integrations (space to toggle, enter to confirm)").
		Options(options...).
		Value(&selected).
		Run()

	if len(selected) == 0 {
		selected = preSelected
	}

	// Show what was selected
	for _, entry := range sorted {
		if contains(selected, entry.agent.Name) {
			fmt.Printf("  %s %s\n", ui.OKPrefix, entry.agent.Name)
		}
	}

	var result []exponential.AgentConfig
	selectedSet := map[string]bool{}
	for _, name := range selected {
		selectedSet[name] = true
	}

	needsSharedAGENTS := false
	for _, agent := range allAgents {
		if agent.Name == "Generic Agent" {
			continue
		}
		if !selectedSet[agent.Name] {
			continue
		}
		result = append(result, agent)
		if agent.File == "AGENTS.md" {
			needsSharedAGENTS = true
		}
	}

	if needsSharedAGENTS {
		for _, agent := range allAgents {
			if agent.Name == "Generic Agent" {
				result = append([]exponential.AgentConfig{agent}, result...)
				break
			}
		}
	}

	return result
}

// workflowFromFlag resolves a --workflow value. It returns nil for "none".
func workflowFromFlag(name string) (*exponential.WorkflowTemplate, error) {
	if strings.EqualFold(name, "none") {
		return nil, nil
	}
	tpl, ok := exponential.LookupWorkflow(name)
	if !ok {
		return nil, fmt.Errorf("unknown workflow %q (available: %s)", name, strings.Join(exponential.WorkflowNames(), ", "))
	}
	return &tpl, nil
}

// resolveWorkflowChoice picks the workflow written on first init: the
// --workflow flag, else a prompt, else the first template. Returns nil for
// "none".
func resolveWorkflowChoice(interactive bool) *exponential.WorkflowTemplate {
	if initWorkflow != "" && initWorkflow != workflowAsk {
		tpl, _ := workflowFromFlag(initWorkflow) // validated in runInit
		return tpl
	}
	if !interactive {
		tpl := exponential.WorkflowTemplates[0]
		return &tpl
	}

	var options []huh.Option[int]
	for i, t := range exponential.WorkflowTemplates {
		options = append(options, huh.NewOption(fmt.Sprintf("%-10s %s", t.Title, ui.MutedStyle.Render(t.Description)), i))
	}
	none := len(exponential.WorkflowTemplates)
	options = append(options, huh.NewOption(fmt.Sprintf("%-10s %s", "None", ui.MutedStyle.Render("No workflow: write your own xpo-workflow skill, or let the agent decide")), none))

	fmt.Printf("\n%s Workflow\n", ui.AccentStyle.Render("?"))
	selected := 0
	huh.NewSelect[int]().
		Title("How should agents work on issues? Written once; edit it freely afterwards").
		Options(options...).
		Value(&selected).
		Run()

	if selected == none {
		fmt.Printf("  %s None\n", ui.OKPrefix)
		return nil
	}
	tpl := exponential.WorkflowTemplates[selected]
	fmt.Printf("  %s %s\n", ui.OKPrefix, tpl.Title)
	return &tpl
}

// applyInit writes the integration files. wf is the workflow chosen on first
// init; re-init and doctor pass nil and never touch the workflow.
func applyInit(prefix string, agents []exponential.AgentConfig, wf *exponential.WorkflowTemplate) {
	_, err := exponential.InitProject(initForce, prefix)
	if err != nil {
		fmt.Printf("%s %v\n", ui.ErrorPrefix, err)
		os.Exit(1)
	}

	for _, agent := range agents {
		if !agent.MCPConfig.HasMCPConfig() {
			continue
		}
		status := exponential.DetectMCPConfigFor(agent.MCPConfig)
		if status.HasExponential && !initForce {
			continue
		}
		if err := exponential.EnsureMCPConfigFor(agent.MCPConfig); err != nil {
			fmt.Printf("  %s %s: %v\n", ui.ErrorPrefix, agent.MCPConfig.File, err)
			continue
		}
	}

	writtenFiles := map[string]bool{}
	for _, agent := range agents {
		if writtenFiles[agent.File] {
			continue
		}
		writtenFiles[agent.File] = true

		result, err := exponential.WriteAgentInstructions(agent, prefix, initForce)
		if err != nil {
			fmt.Printf("  %s %s: %v\n", ui.ErrorPrefix, agent.File, err)
			continue
		}
		if result.Action == "skipped" && result.WasEdited {
			handleEditedBlock(agent, prefix)
		}
	}

	if wf != nil {
		for _, agent := range exponential.InlineWorkflowTargets(agents) {
			if _, err := exponential.AppendInlineWorkflow(agent, wf); err != nil {
				fmt.Printf("  %s %s: %v\n", ui.ErrorPrefix, agent.File, err)
			}
		}
	}

	for _, agent := range agents {
		if agent.SkillDir == "" {
			continue
		}
		handleEditedSkills(agent)
		if _, err := exponential.InstallWorkflow(agent, wf); err != nil {
			fmt.Printf("  %s %s workflow: %v\n", ui.ErrorPrefix, agent.Name, err)
		}
	}
}

// handleEditedSkills writes the agent's skill files. Files with local edits are
// replaced without asking in non-interactive mode or with --yes, and prompted
// for otherwise.
func handleEditedSkills(agent exponential.AgentConfig) {
	// A skill from before the split held the workflow; the update turns it
	// into the reference, so point the user at --workflow.
	defer func(legacy bool) {
		if legacy {
			printWorkflowHint()
		}
	}(exponential.HadLegacyWorkflow(agent))

	res, err := exponential.WriteAgentSkill(agent, "", initForce)
	if err != nil {
		fmt.Printf("  %s %s skill: %v\n", ui.ErrorPrefix, agent.Name, err)
		return
	}
	for _, f := range res.Edited {
		if !isInteractive() || initYes {
			if err := exponential.ReplaceSkillFile(f); err != nil {
				fmt.Printf("  %s %s: %v\n", ui.ErrorPrefix, f.Path, err)
				continue
			}
			fmt.Printf("%s %s: local edits replaced with the %s version\n", ui.NotePrefix, f.Path, version.CLIVersion)
			continue
		}
		promptEditedSkill(f, true)
	}
}

func promptEditedSkill(f exponential.EditedSkillFile, showHeader bool) {
	if showHeader {
		fmt.Printf("\n%s %s: the xpo skill has local edits\n", ui.NotePrefix, f.Path)
	}

	choice := promptSelect("How should xpo handle it?", []string{
		fmt.Sprintf("Replace with the %s version", version.CLIVersion),
		"Keep your version",
		"Show diff",
	})

	switch choice {
	case 0: // replace
		if err := exponential.ReplaceSkillFile(f); err != nil {
			fmt.Printf("%s %s: %v\n", ui.ErrorPrefix, f.Path, err)
			return
		}
		fmt.Printf("%s %s: replaced with %s version\n", ui.OKPrefix, f.Path, version.CLIVersion)
	case 1: // keep
		fmt.Printf("%s %s: kept your version\n", ui.OKPrefix, f.Path)
	case 2: // diff
		fmt.Printf("\n--- current\n+++ %s\n", version.CLIVersion)
		showSimpleDiff(f.OldContent, f.NewContent)
		fmt.Println()
		promptEditedSkill(f, false)
	}
}

// blockNeedsPrompt reports whether an agent's interactive problems include
// its instruction file (edited or legacy block), not just edited skill files.
func blockNeedsPrompt(h exponential.AgentHealth) bool {
	return len(h.Interactive) > len(h.EditedSkills)
}

func handleEditedBlock(agent exponential.AgentConfig, prefix string) {
	if !isInteractive() || initYes {
		exponential.WriteAgentInstructions(agent, prefix, true)
		return
	}

	promptEditedBlock(agent, prefix, true)
}

func promptEditedBlock(agent exponential.AgentConfig, prefix string, showHeader bool) {
	if showHeader {
		fmt.Printf("\n%s %s: the xpo section has local edits\n", ui.NotePrefix, agent.File)
	}

	choice := promptSelect("How should xpo handle it?", []string{
		fmt.Sprintf("Replace with the %s version", version.CLIVersion),
		"Keep your version",
		"Show diff",
	})

	switch choice {
	case 0: // replace
		exponential.WriteAgentInstructions(agent, prefix, true)
		fmt.Printf("%s %s: replaced with %s version\n", ui.OKPrefix, agent.File, version.CLIVersion)
	case 1: // keep
		fmt.Printf("%s %s: kept your version\n", ui.OKPrefix, agent.File)
	case 2: // diff
		result, _ := exponential.WriteAgentInstructions(agent, prefix, false)
		fmt.Printf("\n--- current\n+++ %s\n", version.CLIVersion)
		showSimpleDiff(result.OldContent, result.NewContent)
		fmt.Println()
		promptEditedBlock(agent, prefix, false)
	}
}

func showSimpleDiff(old, new string) {
	oldLines := strings.Split(old, "\n")
	newLines := strings.Split(new, "\n")

	maxLen := len(oldLines)
	if len(newLines) > maxLen {
		maxLen = len(newLines)
	}

	for i := 0; i < maxLen; i++ {
		var oldLine, newLine string
		if i < len(oldLines) {
			oldLine = oldLines[i]
		}
		if i < len(newLines) {
			newLine = newLines[i]
		}
		if oldLine != newLine {
			if oldLine != "" {
				fmt.Printf("- %s\n", oldLine)
			}
			if newLine != "" {
				fmt.Printf("+ %s\n", newLine)
			}
		}
	}
}

func runReinit(interactive bool) {
	freshCfg, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("%s Could not load config: %v\n", ui.ErrorPrefix, err)
		os.Exit(1)
	}

	prefix := freshCfg.Prefix
	integrationVer := exponential.DetectIntegrationVersion()

	// Case C: Older binary — never downgrade
	if integrationVer != "" && version.CompareVersions(integrationVer, version.CLIVersion) > 0 {
		fmt.Printf("\n%s This project's integrations are from xpo %s. You have %s.\n", ui.ErrorPrefix, integrationVer, version.CLIVersion)
		fmt.Printf("  No files were changed.\n\n")
		fmt.Printf("  Upgrade xpo, then run xpo init again: brew upgrade xpo\n\n")
		os.Exit(1)
	}

	// Case A: Up-to-date
	if integrationVer == version.CLIVersion && !initForce {
		// Check for edited blocks and skills that need the replace/keep/diff
		// prompt, and for skills that differ from the current template even
		// though the version matches.
		var blockAgents []exponential.AgentConfig
		var staleBlocks []exponential.AgentConfig
		var skillHealth []exponential.AgentHealth
		for _, agent := range exponential.DetectInstalledAgents() {
			h := exponential.CheckAgentHealth(agent, integrationVer, prefix)
			if blockNeedsPrompt(h) {
				blockAgents = append(blockAgents, agent)
			} else if h.BlockStale {
				staleBlocks = append(staleBlocks, agent)
			}
			if h.SkillStale || h.SkillEdited() {
				skillHealth = append(skillHealth, h)
			}
		}

		fmt.Printf("\nExponential is already set up in this project (prefix %s)\n", prefix)
		if len(blockAgents) == 0 && len(staleBlocks) == 0 && len(skillHealth) == 0 {
			fmt.Printf("%s Integrations are up to date (%s)\n\n", ui.OKPrefix, version.CLIVersion)
			return
		}

		// Handle only what changed — not a full re-init
		fmt.Printf("%s Integrations are up to date (%s)\n", ui.OKPrefix, version.CLIVersion)
		for _, agent := range blockAgents {
			handleEditedBlock(agent, prefix)
		}
		for _, agent := range staleBlocks {
			if _, err := exponential.WriteAgentInstructions(agent, prefix, false); err != nil {
				fmt.Printf("%s %s: %v\n", ui.ErrorPrefix, agent.File, err)
				continue
			}
			fmt.Printf("%s %s: instructions updated to the %s template\n", ui.OKPrefix, agent.Name, version.CLIVersion)
		}
		for _, h := range skillHealth {
			handleEditedSkills(h.Agent)
			if h.SkillStale {
				fmt.Printf("%s %s: skill updated to the %s template\n", ui.OKPrefix, h.Agent.Name, version.CLIVersion)
			}
		}
		fmt.Println()
		return
	}

	// Case B: Stale or first-time version stamp, or --force
	cleanup := withCtrlC("")
	defer cleanup()
	fmt.Printf("\n%s\n", ui.AccentStyle.Render("Exponential is already set up in this project"))

	cwd, _ := os.Getwd()
	home, _ := os.UserHomeDir()
	displayPath := cwd
	if home != "" && strings.HasPrefix(cwd, home) {
		displayPath = "~" + cwd[len(home):]
	}
	fmt.Printf("\nProject   %s\n", displayPath)
	fmt.Printf("Prefix    %s\n", prefix)

	detected := exponential.DetectInstalledAgents()
	var agents []exponential.AgentConfig

	fmt.Printf("\nIntegrations\n")
	for _, agent := range detected {
		if integrationVer != "" {
			fmt.Printf("  %s   %s → %s\n", agent.Name, integrationVer, version.CLIVersion)
		} else {
			fmt.Printf("  %s   Installing\n", agent.Name)
		}
		agents = append(agents, agent)
	}

	// Check for newly detected agents not yet configured
	allAgents := exponential.AgentRegistry
	for _, agent := range allAgents {
		if agent.Name == "Generic Agent" || agent.Binary == "" {
			continue
		}
		found := false
		for _, d := range detected {
			if d.Name == agent.Name {
				found = true
				break
			}
		}
		if found {
			continue
		}
		if _, err := exec.LookPath(agent.Binary); err == nil {
			fmt.Printf("  %s   Found, not set up\n", agent.Name)
			if interactive {
				if promptConfirm(fmt.Sprintf("Also set up %s?", agent.Name), false) {
					agents = append(agents, agent)
				}
			}
		}
	}

	// Change plan
	plan := exponential.ComputeInitPlan(agents, nil)
	fmt.Printf("\nChanges\n")
	for _, c := range plan.Changes {
		marker := "~"
		if c.Action == "create" {
			marker = "+"
		}
		fmt.Printf("  %s %s\n", marker, c.Path)
	}

	if interactive {
		if !promptConfirm("Apply changes?", true) {
			abortInit("xpo init")
		}
	} else if !initYes {
		fmt.Printf("\nRun with --yes to apply, or run interactively.\n")
		os.Exit(1)
	}

	cleanup()
	applyInit(prefix, agents, nil)

	fmt.Printf("\n%s Updated integrations to %s\n", ui.OKPrefix, version.CLIVersion)
	fmt.Printf("\nNext\n")
	fmt.Printf("  Commit the update:  git commit -am \"Update xpo integrations to %s\"\n\n", version.CLIVersion)
}

func contains(slice []string, val string) bool {
	for _, s := range slice {
		if s == val {
			return true
		}
	}
	return false
}

func init() {
	initCmd.Flags().BoolVarP(&initForce, "force", "f", false, "Re-initialize even if .xpo already exists")
	initCmd.Flags().BoolVar(&initYes, "yes", false, "Skip confirmation prompts")
	initCmd.Flags().StringVar(&initPrefix, "prefix", "", "Issue ID prefix (e.g. pay)")
	initCmd.Flags().StringVar(&initAgents, "agents", "", "Comma-separated list of agent harnesses to set up")
	initCmd.Flags().StringVar(&initWorkflow, "workflow", "", "Write the workflow skill from a template ("+strings.Join(exponential.WorkflowNames(), ", ")+"); without a value, ask")
	initCmd.Flags().Lookup("workflow").NoOptDefVal = workflowAsk
	rootCmd.AddCommand(initCmd)
}

// workflowAsk is the --workflow value when the flag is given without one.
const workflowAsk = "ask"

var workflowHintShown bool

// printWorkflowHint tells the user, once per run, how to set up the workflow
// after the update turned their old combined skill into the reference.
func printWorkflowHint() {
	if workflowHintShown || initWorkflow != "" {
		return
	}
	workflowHintShown = true
	fmt.Printf("%s The workflow is now a separate skill. Run %s to set it up.\n", ui.NotePrefix, ui.AccentStyle.Render("xpo init --workflow"))
}

// runWorkflowSetup writes the workflow skill in an initialized project
// (xpo init --workflow). Existing workflow skills are replaced only after the
// user confirms, or with --yes.
func runWorkflowSetup(interactive bool) {
	var tpl *exponential.WorkflowTemplate
	if initWorkflow == workflowAsk {
		tpl = promptWorkflowTemplate(interactive)
	} else {
		tpl, _ = workflowFromFlag(initWorkflow) // validated in runInit
	}

	agents := exponential.WorkflowTargets()
	if len(agents) == 0 {
		fmt.Printf("%s No agents are set up in this project. Run xpo init first.\n", ui.ErrorPrefix)
		os.Exit(1)
	}

	if existing := exponential.ExistingWorkflowDirs(agents); len(existing) > 0 {
		fmt.Printf("\n%s These workflow skills will be replaced with the %s template:\n", ui.NotePrefix, tpl.Title)
		for _, dir := range existing {
			fmt.Printf("  ~ %s/\n", dir)
		}
		switch {
		case initYes:
		case interactive:
			if !promptConfirm("Replace them?", false) {
				fmt.Printf("\nThe workflow was not changed.\n\n")
				return
			}
		default:
			fmt.Printf("%s Run with --yes to replace them, or run interactively.\n", ui.ErrorPrefix)
			os.Exit(1)
		}
	}

	res, err := exponential.SetupWorkflow(agents, tpl)
	for _, f := range res.Written {
		fmt.Printf("%s %s\n", ui.OKPrefix, f)
	}
	for _, f := range res.InlineAdded {
		fmt.Printf("%s %s: workflow added below the xpo section\n", ui.OKPrefix, f)
	}
	for _, f := range res.InlineKept {
		fmt.Printf("%s %s already has a Workflow section; left as it is\n", ui.NotePrefix, f)
	}
	if err != nil {
		fmt.Printf("%s %v\n", ui.ErrorPrefix, err)
		os.Exit(1)
	}
	fmt.Println()
}

// promptWorkflowTemplate asks which template to write; without a terminal it
// takes the first.
func promptWorkflowTemplate(interactive bool) *exponential.WorkflowTemplate {
	if !interactive || len(exponential.WorkflowTemplates) == 1 {
		tpl := exponential.WorkflowTemplates[0]
		return &tpl
	}
	var options []string
	for _, t := range exponential.WorkflowTemplates {
		options = append(options, fmt.Sprintf("%-10s %s", t.Title, ui.MutedStyle.Render(t.Description)))
	}
	tpl := exponential.WorkflowTemplates[promptSelect("Which workflow?", options)]
	return &tpl
}
