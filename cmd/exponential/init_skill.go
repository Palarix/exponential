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
	Long: `Install the xpo skill and agent instruction files for detected
(or specified) agent harnesses.

Skills can be installed locally (project-level) or globally (user-level).
Global installation writes files to ~/.config/xpo/skills/ and creates symlinks
into each harness's skill directory. Global install is available on macOS and
Linux only.

Agent instruction files (CLAUDE.md, AGENTS.md, .cursorrules, etc.) are always
written to the project root since they contain project-specific configuration.`,
	Run: func(cmd *cobra.Command, args []string) {
		if _, err := os.Stat(".xpo"); os.IsNotExist(err) {
			fmt.Printf("%s .xpo directory not found — run xpo init first\n", ui.ErrorPrefix)
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
				fmt.Printf("%s Could not determine home directory: %v\n", ui.ErrorPrefix, err)
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
			fmt.Printf("%s Global skill installation is not supported on Windows. Use --local instead.\n", ui.ErrorPrefix)
			os.Exit(1)
		}
		return true
	}
	if initSkillLocal {
		return false
	}

	interactive := term.IsTerminal(int(os.Stdin.Fd()))
	if !interactive || runtime.GOOS == "windows" {
		return false
	}

	fmt.Println("\nInstall skills globally (shared across projects) or locally (this project only)?")
	fmt.Println("  [g] Global (~/.config/xpo/skills/ with symlinks into harness dirs)")
	fmt.Println("  [l] Local (project-level, default)")
	fmt.Print("Choice [l]: ")

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
			fmt.Printf("\n%s already has an xpo section. Overwrite? [y/N]: ", agent.File)
			reader := bufio.NewReader(os.Stdin)
			response, _ := reader.ReadString('\n')
			response = strings.TrimSpace(strings.ToLower(response))
			if response != "y" && response != "yes" {
				fmt.Printf("%s Skipped %s (%s)\n", ui.NotePrefix, agent.File, agent.Name)
				return
			}
		} else {
			fmt.Printf("%s %s already configured (%s) — use --force to overwrite\n", ui.OKPrefix, agent.File, agent.Name)
			return
		}
	}

	if fileExists && !hasXpoSection {
		interactive := term.IsTerminal(int(os.Stdin.Fd()))
		if interactive {
			fmt.Printf("\n%s exists but has no xpo section. Append xpo instructions? [Y/n]: ", agent.File)
			reader := bufio.NewReader(os.Stdin)
			response, _ := reader.ReadString('\n')
			response = strings.TrimSpace(strings.ToLower(response))
			if response == "n" || response == "no" {
				fmt.Printf("%s Skipped %s (%s)\n", ui.NotePrefix, agent.File, agent.Name)
				return
			}
		}
	}

	if err := exponential.AppendAgentInstructions(agent, prefix); err != nil {
		fmt.Printf("%s %s (%s): %v\n", ui.ErrorPrefix, agent.File, agent.Name, err)
		return
	}

	action := "Configured"
	if hasXpoSection {
		action = "Updated"
	} else if fileExists {
		action = "Appended to"
	} else {
		action = "Created"
	}
	fmt.Printf("%s %s %s (%s)\n", ui.OKPrefix, action, agent.File, agent.Name)
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
		fmt.Printf("%s %s skill already installed %s — use --force to overwrite\n", ui.OKPrefix, agent.Name, where)
		return
	}

	skillDir, err := exponential.WriteAgentSkill(agent, globalBaseDir)
	if err != nil {
		fmt.Printf("%s %s skill: %v\n", ui.ErrorPrefix, agent.Name, err)
		return
	}

	where := "locally"
	if globalBaseDir != "" {
		where = "globally"
	}
	fmt.Printf("%s Installed %s skill %s (%s)\n", ui.OKPrefix, agent.Name, where, skillDir)
}

func init() {
	initSkillCmd.Flags().StringVar(&initSkillHarness, "harness", "", "Install for a specific agent harness")
	initSkillCmd.Flags().BoolVar(&initSkillGlobal, "global", false, "Install skills globally (macOS/Linux only)")
	initSkillCmd.Flags().BoolVar(&initSkillLocal, "local", false, "Install skills into the project (default)")
	initSkillCmd.Flags().BoolVar(&initSkillForce, "force", false, "Overwrite existing installations")
	initCmd.AddCommand(initSkillCmd)
}
