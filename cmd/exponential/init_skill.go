package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/palarix/exponential/internal/config"
	"github.com/palarix/exponential/internal/exponential"
	"github.com/palarix/exponential/internal/ui"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	initSkillHarness string
	initSkillGlobal  bool
	initSkillLocal   bool
	initSkillForce   bool
)

var initSkillCmd = &cobra.Command{
	Use:   "skill",
	Short: "Install workflow skill and agent instructions",
	Long: `Install the xpo-workflow skill and agent instruction files for detected
(or specified) agent harnesses.

Skills can be installed locally (project-level) or globally (user-level).
Global installation writes files to ~/.config/xpo/skills/ and creates symlinks
into each harness's skill directory. Global install is available on macOS and
Linux only.

Agent instruction files (CLAUDE.md, AGENTS.md, .cursorrules, etc.) are always
written to the project root since they contain project-specific configuration.`,
	Run: func(cmd *cobra.Command, args []string) {
		if _, err := os.Stat(".xpo"); os.IsNotExist(err) {
			fmt.Print(ui.Stylize(fmt.Sprintf("%s `.xpo` directory not found. Run `xpo init` first.\n", ui.ErrorPrefix)))
			os.Exit(1)
		}

		prefix := "issue-"
		if freshCfg, err := config.LoadConfig(); err == nil && freshCfg.Prefix != "" {
			prefix = freshCfg.Prefix
		}

		agents := resolveAgents(initSkillHarness)
		if agents == nil {
			return
		}

		global := resolveScope()

		var globalBaseDir string
		if global {
			home, err := os.UserHomeDir()
			if err != nil {
				fmt.Print(ui.Stylize(fmt.Sprintf("%s Could not determine home directory: %v\n", ui.ErrorPrefix, err)))
				os.Exit(1)
			}
			globalBaseDir = filepath.Join(home, exponential.GlobalSkillCanonicalDir)
		}

		for _, agent := range agents {
			installAgentInstructions(agent, prefix)
			installSkillFiles(agent, globalBaseDir)
		}
	},
}

func resolveScope() bool {
	if initSkillGlobal {
		if runtime.GOOS == "windows" {
			fmt.Print(ui.Stylize(fmt.Sprintf("%s Global skill installation is not supported on Windows. Use --local instead.\n", ui.ErrorPrefix)))
			os.Exit(1)
		}
		return true
	}
	if initSkillLocal {
		return false
	}

	interactive := term.IsTerminal(int(os.Stdin.Fd()))
	if !interactive {
		return false
	}

	if runtime.GOOS == "windows" {
		return false
	}

	fmt.Print(ui.Stylize("\nInstall skills globally (shared across projects) or locally (this project only)?\n"))
	fmt.Print(ui.Stylize("  [g] Global (~/.config/xpo/skills/ with symlinks into harness dirs)\n"))
	fmt.Print(ui.Stylize("  [l] Local (project-level, default)\n"))
	fmt.Print(ui.Stylize("Choice [l]: "))

	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))

	return response == "g" || response == "global"
}

func installAgentInstructions(agent exponential.AgentConfig, prefix string) {
	content, err := os.ReadFile(agent.File)
	fileExists := err == nil
	hasXpoSection := fileExists && exponential.HasAgentInstructions(string(content))

	if hasXpoSection && !initSkillForce {
		interactive := term.IsTerminal(int(os.Stdin.Fd()))
		if interactive {
			fmt.Print(ui.Stylize(fmt.Sprintf("\n`%s` already has an xpo section. Overwrite? [y/N]: ", agent.File)))
			reader := bufio.NewReader(os.Stdin)
			response, _ := reader.ReadString('\n')
			response = strings.TrimSpace(strings.ToLower(response))
			if response != "y" && response != "yes" {
				fmt.Print(ui.Stylize(fmt.Sprintf("%s Skipped `%s` (%s)\n", ui.NotePrefix, agent.File, agent.Name)))
				return
			}
		} else {
			fmt.Print(ui.Stylize(fmt.Sprintf("%s `%s` already has xpo section (%s) — use --force to overwrite\n", ui.OKPrefix, agent.File, agent.Name)))
			return
		}
	}

	if fileExists && !hasXpoSection {
		interactive := term.IsTerminal(int(os.Stdin.Fd()))
		if interactive {
			fmt.Print(ui.Stylize(fmt.Sprintf("\n`%s` exists but has no xpo section. Append xpo instructions? [Y/n]: ", agent.File)))
			reader := bufio.NewReader(os.Stdin)
			response, _ := reader.ReadString('\n')
			response = strings.TrimSpace(strings.ToLower(response))
			if response == "n" || response == "no" {
				fmt.Print(ui.Stylize(fmt.Sprintf("%s Skipped `%s` (%s)\n", ui.NotePrefix, agent.File, agent.Name)))
				return
			}
		}
	}

	if err := exponential.AppendAgentInstructions(agent, prefix); err != nil {
		fmt.Print(ui.Stylize(fmt.Sprintf("%s Could not configure `%s` (%s): %v\n", ui.ErrorPrefix, agent.File, agent.Name, err)))
		return
	}

	action := "Configured"
	if hasXpoSection {
		action = "Updated"
	} else if fileExists {
		action = "Added xpo section to"
	} else {
		action = "Created"
	}
	fmt.Print(ui.Stylize(fmt.Sprintf("%s %s `%s` (%s)\n", ui.OKPrefix, action, agent.File, agent.Name)))
}

func installSkillFiles(agent exponential.AgentConfig, globalBaseDir string) {
	if agent.SkillDir == "" {
		return
	}

	existingStatus := exponential.DetectSkillInstall(agent)
	if (existingStatus.Local || existingStatus.Global) && !initSkillForce {
		where := "locally"
		if existingStatus.Global {
			where = "globally"
		}
		fmt.Print(ui.Stylize(fmt.Sprintf("%s Skill already installed %s for %s — use --force to overwrite\n", ui.OKPrefix, where, agent.Name)))
		return
	}

	skillDir, err := exponential.WriteAgentSkill(agent, globalBaseDir)
	if err != nil {
		fmt.Print(ui.Stylize(fmt.Sprintf("%s Could not write skill for %s: %v\n", ui.ErrorPrefix, agent.Name, err)))
		return
	}

	where := "locally"
	if globalBaseDir != "" {
		where = "globally"
	}
	fmt.Print(ui.Stylize(fmt.Sprintf("%s Installed `%s/` workflow skill %s (%s)\n", ui.OKPrefix, skillDir, where, agent.Name)))
}

func init() {
	initSkillCmd.Flags().StringVar(&initSkillHarness, "harness", "", "Install for a specific agent harness")
	initSkillCmd.Flags().BoolVar(&initSkillGlobal, "global", false, "Install skills globally (macOS/Linux only)")
	initSkillCmd.Flags().BoolVar(&initSkillLocal, "local", false, "Install skills into the project (default)")
	initSkillCmd.Flags().BoolVar(&initSkillForce, "force", false, "Overwrite existing installations")
	initCmd.AddCommand(initSkillCmd)
}
