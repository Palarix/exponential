package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/palarix/exponential/internal/exponential"
	"github.com/palarix/exponential/internal/ui"
	"github.com/spf13/cobra"
)

var (
	initMCPHarness string
	initMCPForce   bool
)

var initMCPCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Configure MCP server for agent harnesses",
	Long: `Write MCP server configuration for detected (or specified) agent harnesses.

Each harness has its own MCP config format and location:
  Claude Code   .mcp.json           (JSON, mcpServers key)
  Cursor        .cursor/mcp.json    (JSON, mcpServers key)
  Codex         .codex/config.toml  (TOML, mcp_servers section)
  OpenCode      opencode.json       (JSON, mcp key)`,
	Run: func(cmd *cobra.Command, args []string) {
		if _, err := os.Stat(".xpo"); os.IsNotExist(err) {
			fmt.Print(ui.Stylize(fmt.Sprintf("%s `.xpo` directory not found. Run `xpo init` first.\n", ui.ErrorPrefix)))
			os.Exit(1)
		}

		agents := resolveAgents(initMCPHarness)
		if agents == nil {
			return
		}

		var configured int
		for _, agent := range agents {
			if !agent.MCPConfig.HasMCPConfig() {
				continue
			}

			status := exponential.DetectMCPConfigFor(agent.MCPConfig)
			if status.HasExponential && !initMCPForce {
				fmt.Print(ui.Stylize(fmt.Sprintf("%s `%s` already has xpo entry (%s) — use --force to overwrite\n", ui.OKPrefix, agent.MCPConfig.File, agent.Name)))
				continue
			}

			if err := exponential.EnsureMCPConfigFor(agent.MCPConfig); err != nil {
				fmt.Print(ui.Stylize(fmt.Sprintf("%s Could not configure `%s` (%s): %v\n", ui.ErrorPrefix, agent.MCPConfig.File, agent.Name, err)))
				continue
			}

			action := "Configured"
			if status.HasExponential {
				action = "Updated"
			}
			fmt.Print(ui.Stylize(fmt.Sprintf("%s %s `%s` with xpo MCP server (%s)\n", ui.OKPrefix, action, agent.MCPConfig.File, agent.Name)))
			configured++
		}

		if configured == 0 {
			fmt.Print(ui.Stylize(fmt.Sprintf("\n%s No MCP configuration was written. Use --force to overwrite existing entries.\n", ui.NotePrefix)))
		}
	},
}

func resolveAgents(harness string) []exponential.AgentConfig {
	if harness != "" {
		agent, ok := exponential.LookupAgent(harness)
		if !ok {
			fmt.Print(ui.Stylize(fmt.Sprintf("%s Unknown harness: %s\n", ui.ErrorPrefix, harness)))
			fmt.Print(ui.Stylize(fmt.Sprintf("    Available: %s\n", strings.Join(exponential.AgentRegistryNames(), ", "))))
			os.Exit(1)
		}
		return []exponential.AgentConfig{agent}
	}
	return exponential.DetectInstalledAgents()
}

func init() {
	initMCPCmd.Flags().StringVar(&initMCPHarness, "harness", "", "Install for a specific agent harness")
	initMCPCmd.Flags().BoolVar(&initMCPForce, "force", false, "Overwrite existing MCP config entries")
	initCmd.AddCommand(initMCPCmd)
}
