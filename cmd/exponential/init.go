package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/palarix/exponential/internal/config"
	"github.com/palarix/exponential/internal/exponential"
	"github.com/palarix/exponential/internal/ui"
	"github.com/palarix/exponential/internal/version"
	"github.com/spf13/cobra"
)

var (
	initForce  bool
	initYes    bool
	initPrefix string
	initAgents string
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Set up Exponential in this project",
	Long: `Initialize Exponential: create the .xpo directory, configure MCP servers,
and install agent skills — all in one step.

Re-running in an initialized project refreshes integration files to the
current CLI version.`,
	Run: runInit,
}

func runInit(cmd *cobra.Command, args []string) {
	interactive := isInteractive() && !initYes

	// Pre-flight
	alreadyInit := false
	if _, err := os.Stat(".xpo"); err == nil {
		alreadyInit = true
	}

	if alreadyInit {
		runReinit(interactive)
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

	// Change plan
	plan := exponential.ComputeInitPlan(agents)
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

	applyInit(prefix, agents)

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

func applyInit(prefix string, agents []exponential.AgentConfig) {
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

	for _, agent := range agents {
		if agent.SkillDir == "" {
			continue
		}
		if _, err := exponential.WriteAgentSkill(agent, ""); err != nil {
			fmt.Printf("  %s %s skill: %v\n", ui.ErrorPrefix, agent.Name, err)
		}
	}

	updateIntegrationVersion()
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
	integrationVer := freshCfg.IntegrationVersion

	// Case C: Older binary — never downgrade
	if integrationVer != "" && version.CompareVersions(integrationVer, version.CLIVersion) > 0 {
		fmt.Printf("\n%s This project's integrations are from xpo %s. You have %s.\n", ui.ErrorPrefix, integrationVer, version.CLIVersion)
		fmt.Printf("  No files were changed.\n\n")
		fmt.Printf("  Upgrade xpo, then run xpo init again: brew upgrade xpo\n\n")
		os.Exit(1)
	}

	// Case A: Up-to-date
	if integrationVer == version.CLIVersion && !initForce {
		// Check for edited blocks that need the replace/keep/diff prompt
		var editedAgents []exponential.AgentConfig
		for _, agent := range exponential.DetectInstalledAgents() {
			h := exponential.CheckAgentHealth(agent, integrationVer)
			if len(h.Interactive) > 0 {
				editedAgents = append(editedAgents, agent)
			}
		}

		if len(editedAgents) == 0 {
			fmt.Printf("\nExponential is already set up in this project (prefix %s)\n", prefix)
			fmt.Printf("%s Integrations are up to date (%s)\n\n", ui.OKPrefix, version.CLIVersion)
			return
		}

		// Handle only the edited blocks — not a full re-init
		fmt.Printf("\nExponential is already set up in this project (prefix %s)\n", prefix)
		fmt.Printf("%s Integrations are up to date (%s)\n", ui.OKPrefix, version.CLIVersion)
		for _, agent := range editedAgents {
			handleEditedBlock(agent, prefix)
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
	plan := exponential.ComputeInitPlan(agents)
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
	applyInit(prefix, agents)

	fmt.Printf("\n%s Updated integrations to %s\n", ui.OKPrefix, version.CLIVersion)
	fmt.Printf("\nNext\n")
	fmt.Printf("  Commit the update:  git commit -am \"Update xpo integrations to %s\"\n\n", version.CLIVersion)
}

func updateIntegrationVersion() {
	path := ".xpo/config.yaml"
	content, err := os.ReadFile(path)
	if err != nil {
		return
	}

	lines := strings.Split(string(content), "\n")
	found := false
	for i, line := range lines {
		if strings.HasPrefix(line, "integration_version:") {
			lines[i] = fmt.Sprintf("integration_version: \"%s\"", version.CLIVersion)
			found = true
			break
		}
	}
	if !found {
		for i, line := range lines {
			if strings.HasPrefix(line, "version:") {
				rest := append([]string{fmt.Sprintf("integration_version: \"%s\"", version.CLIVersion)}, lines[i+1:]...)
				lines = append(lines[:i+1], rest...)
				break
			}
		}
	}
	os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0644)
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
	rootCmd.AddCommand(initCmd)
}
