package main

import (
	"fmt"
	"os"

	"github.com/kuyio/beats/internal/mcpserver"
	"github.com/spf13/cobra"
)

var mcpCmd = &cobra.Command{
	Use:           "mcp",
	SilenceUsage:  true,
	SilenceErrors: false,
	Short:         "Run beats as an MCP (Model Context Protocol) server",
	Long: `Start a Model Context Protocol server on stdio so AI agents can manage
issues via structured tool calls instead of shell commands.

Wire it into Claude Code with a project-local .mcp.json such as:

  {
    "mcpServers": {
      "beats": {
        "command": "beats",
        "args": ["mcp"]
      }
    }
  }

Agent identity recorded on writes is resolved in this order:

  1. $BEATS_AGENT_IDENTITY (e.g. "Claude Code <agent@nicspc.local>")
  2. clientInfo from the MCP initialize handshake
  3. the configured beats user (same default the CLI uses)
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := mcpserver.Run(cmd.Context(), cfg); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return err
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(mcpCmd)
}
