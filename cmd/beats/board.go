package main

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/kuyio/beats/internal/server"
	"github.com/spf13/cobra"
)

var boardPort int
var boardNoOpen bool
var boardDev bool
var boardDevPort int

var boardCmd = &cobra.Command{
	Use:   "board",
	Short: "Start the web-based board UI",
	Long: `Starts a local web server hosting the Beats board interface.

The board provides Dashboard, Backlog, Board, and Dependencies views
for visual project management.

Changes made in the UI are buffered locally until you click "Save & Sync",
which commits all changes to the repository in a single commit.

In development mode (--dev), assets are proxied from the Vite dev server,
enabling hot module replacement for faster iteration.`,
	RunE: runBoard,
}

func init() {
	boardCmd.Flags().IntVarP(&boardPort, "port", "p", 8080, "port to run the server on")
	boardCmd.Flags().BoolVar(&boardNoOpen, "no-open", false, "don't open browser automatically")
	boardCmd.Flags().BoolVar(&boardDev, "dev", false, "enable dev mode (proxy assets from Vite dev server)")
	boardCmd.Flags().IntVar(&boardDevPort, "dev-port", 5173, "Vite dev server port (used with --dev)")
	rootCmd.AddCommand(boardCmd)
}

func runBoard(cmd *cobra.Command, args []string) error {
	srv := server.NewServer(cfg, boardPort, boardDev, boardDevPort)

	if !boardNoOpen {
		url := fmt.Sprintf("http://localhost:%d", boardPort)
		go openBrowser(url)
	}

	return srv.Start()
}

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
