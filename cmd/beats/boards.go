package main

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/kuyio/beats/internal/server"
	"github.com/spf13/cobra"
)

var boardsPort int
var boardsNoOpen bool

var boardsCmd = &cobra.Command{
	Use:   "boards",
	Short: "Start the web-based board UI",
	Long: `Starts a local web server hosting the Beats board interface.

The board provides Dashboard, Backlog, Board, and Dependencies views
for visual project management.

Changes made in the UI are buffered locally until you click "Save & Sync",
which commits all changes to the repository in a single commit.`,
	RunE: runBoards,
}

func init() {
	boardsCmd.Flags().IntVarP(&boardsPort, "port", "p", 8080, "port to run the server on")
	boardsCmd.Flags().BoolVar(&boardsNoOpen, "no-open", false, "don't open browser automatically")
	rootCmd.AddCommand(boardsCmd)
}

func runBoards(cmd *cobra.Command, args []string) error {
	srv := server.NewServer(cfg, boardsPort)

	// Open browser if not disabled
	if !boardsNoOpen {
		url := fmt.Sprintf("http://localhost:%d", boardsPort)
		go openBrowser(url)
	}

	// Start server (blocks)
	return srv.Start()
}

// openBrowser opens the default browser to the specified URL.
func openBrowser(url string) {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return
	}

	_ = cmd.Start()
}
