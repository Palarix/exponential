package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"

	"github.com/palarix/exponential/internal/registry"
	"github.com/palarix/exponential/internal/server"
	"github.com/spf13/cobra"
)

var boardPort int
var boardNoOpen bool
var boardDev bool
var boardDevPort int

var boardCmd = &cobra.Command{
	Use:   "board",
	Short: "Start the web-based board UI",
	Long: `Starts a local web server hosting the Exponential board interface.

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
	srv.SSEHub = server.NewSSEHub()

	if cfg.Remote.URL != "" {
		srv.ProxyURL = cfg.Remote.URL
		srv.ProxyToken = cfg.Remote.Token
		log.Printf("Remote mode: proxying API to %s", cfg.Remote.URL)
	}

	listener, err := srv.Bind()
	if err != nil {
		return err
	}

	name := cfg.Name
	cwd, _ := os.Getwd()
	if name == "" {
		name = filepath.Base(cwd)
	}
	if err := registry.Register(name, srv.Port, cwd); err != nil {
		log.Printf("warning: failed to register instance: %v", err)
	}
	defer func() { _ = registry.UnregisterByPID(os.Getpid()) }()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		_ = registry.UnregisterByPID(os.Getpid())
		os.Exit(0)
	}()

	if !boardNoOpen {
		url := fmt.Sprintf("http://localhost:%d", srv.Port)
		go openBrowser(url)
	}

	return srv.ServeOn(listener)
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
