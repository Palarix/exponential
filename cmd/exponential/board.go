package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/palarix/exponential/internal/chat"
	"github.com/palarix/exponential/internal/exponential"
	"github.com/palarix/exponential/internal/registry"
	"github.com/palarix/exponential/internal/server"
	"github.com/palarix/exponential/internal/storage"
	"github.com/spf13/cobra"
)

var boardHost string
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

The server listens on 127.0.0.1 by default. To view a board running on
another machine, prefer forwarding the port over SSH:

  ssh -L 8080:127.0.0.1:8080 user@host

When --host names a non-loopback address, the board requires a per-run
access token. It is printed on startup as a URL with ?token=…; open that
URL, or paste the token on the /auth page. Traffic is plain HTTP.

In development mode (--dev), assets are proxied from the Vite dev server,
enabling hot module replacement for faster iteration.`,
	RunE: runBoard,
}

func init() {
	boardCmd.Flags().StringVar(&boardHost, "host", server.DefaultHost, "address to listen on (non-loopback requires a per-run access token; prefer ssh -L for remote access)")
	boardCmd.Flags().IntVarP(&boardPort, "port", "p", 8080, "port to run the server on")
	boardCmd.Flags().BoolVar(&boardNoOpen, "no-open", false, "don't open browser automatically")
	boardCmd.Flags().BoolVar(&boardDev, "dev", false, "enable dev mode (proxy assets from Vite dev server)")
	boardCmd.Flags().IntVar(&boardDevPort, "dev-port", 5173, "Vite dev server port (used with --dev)")
	rootCmd.AddCommand(boardCmd)
}

func runBoard(cmd *cobra.Command, args []string) error {
	srv := server.NewServer(cfg, boardPort, boardDev, boardDevPort)
	srv.Host = boardHost
	srv.SSEHub = server.NewSSEHub()
	if !server.IsLoopbackHost(boardHost) {
		token, err := server.GenerateAccessToken()
		if err != nil {
			return err
		}
		srv.AccessToken = token
		log.Printf("WARNING: listening on %q over plain HTTP — anyone who can reach this address and holds the access token can read and modify issues. Prefer: ssh -L %d:127.0.0.1:%d <host>", boardHost, boardPort, boardPort)
	}

	if cfg.Remote.URL != "" {
		srv.ProxyURL = cfg.Remote.URL
		srv.ProxyToken = cfg.Remote.Token
		log.Printf("Remote mode: proxying API to %s", cfg.Remote.URL)
	}

	srv.Chat = newChatService()
	defer srv.Chat.Runner.Shutdown()
	if srv.AccessToken != "" && srv.Chat.Catalog.Status().Available {
		log.Printf("WARNING: agent chat is on — anyone with the access token can use this machine's agent logins (Claude Code, Codex) and read project code through the agent.")
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
		srv.Chat.Runner.Shutdown()
		os.Exit(0)
	}()

	if srv.AccessToken != "" {
		log.Printf("Access token required. Open: %s", srv.AccessURL())
	}

	if !boardNoOpen {
		go openBrowser(srv.AccessURL())
	}

	return srv.ServeOn(listener)
}

// newChatService sets up the agent chat: chats in .xpo/chats/, and an agent
// that runs in the project root and reaches the xpo tools through this
// binary's stdio MCP server, or the remote server's /mcp in remote mode.
// Work Mode runs the agent in an issue's worktree; remote mode has none
// until Team Mode (xpo-a60065).
func newChatService() *server.ChatService {
	root := storage.HubRoot()
	store := chat.NewStore(filepath.Join(storage.XpoDir(), "chats"))
	store.OnCreateDir = func() {
		exponential.EnsureGitignoreEntryAt(filepath.Join(root, ".gitignore"), ".xpo/chats/")
	}
	// A board that crashed left its agents running; stop them before any
	// chat resumes their sessions.
	if n, err := store.StopOrphans(); err != nil {
		log.Printf("warning: chat store: %v", err)
	} else if n > 0 {
		log.Printf("Stopped %d agent(s) a previous board left running", n)
	}
	if err := store.MarkInterrupted(); err != nil {
		log.Printf("warning: chat store: %v", err)
	}
	if n, err := store.AutoArchive(chat.AutoArchiveAfter, time.Now()); err != nil {
		log.Printf("warning: archiving chats: %v", err)
	} else if n > 0 {
		log.Printf("Archived %d chat(s) idle for 30 days", n)
	}

	target := chat.MCPTarget{RemoteURL: cfg.Remote.URL, RemoteToken: cfg.Remote.Token}
	if target.RemoteURL == "" {
		exe, err := os.Executable()
		if err != nil {
			exe = "xpo"
		}
		target.Executable = exe
	}

	catalog := chat.NewCatalog(cfg.Chat.Agent, exec.LookPath)
	st := catalog.Status()
	if agent := cfg.Chat.Agent; agent != "" && agent != st.DefaultProvider {
		log.Printf("warning: chat.agent %q is not installed or not supported; chat defaults to %q", agent, st.DefaultProvider)
	}
	// Model names and context windows come from earlier turns.
	if metas, err := store.List(); err == nil {
		catalog.LearnFrom(metas)
	}
	for _, p := range st.Providers {
		if p.ID == chat.ProviderCodex && p.Available {
			go loadCodexModels(catalog)
		}
	}

	return &server.ChatService{
		Store: store,
		Runner: chat.NewRunner(chat.RunnerConfig{
			Store:           store,
			Dir:             root,
			MCP:             target,
			DefaultProvider: st.DefaultProvider,
			DefaultModel:    cfg.Chat.Model,
			Learner:         catalog,
			CanWork:         target.RemoteURL == "",
			ResolveWork:     resolveWork,
			Titlers: map[string]chat.Titler{
				chat.ProviderClaude: &chat.ClaudeTitler{},
				chat.ProviderCodex:  &chat.CodexTitler{},
			},
			TitleModel: catalog.TitleModel,
			IssueTitle: issueTitle,
		}),
		Catalog: catalog,
	}
}

// issueTitle returns an issue's title for a work chat's title hint, or ""
// when the issue cannot be read.
func issueTitle(id string) string {
	issue, err := exponential.NewClient(cfg).Transport.GetIssue(id)
	if err != nil {
		return ""
	}
	return issue.Title
}

// resolveWork finds an issue's full ID and worktree, after the chat agent
// started it with its own MCP server process.
func resolveWork(ref string) (chat.WorkBinding, error) {
	issue, err := exponential.NewClient(cfg).Transport.GetIssue(ref)
	if err != nil {
		return chat.WorkBinding{}, err
	}
	wt, ok := exponential.FindWorktreeForIssue(issue.ID)
	if !ok {
		return chat.WorkBinding{}, fmt.Errorf("%s has no worktree", issue.ID)
	}
	path, err := filepath.Abs(wt.Path)
	if err != nil {
		return chat.WorkBinding{}, err
	}
	return chat.WorkBinding{IssueID: issue.ID, Worktree: path}, nil
}

// loadCodexModels fills in Codex's model list. Until it answers, or when it
// fails, Codex offers only its default model.
func loadCodexModels(catalog *chat.Catalog) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	models, err := chat.CodexModels(ctx, "codex")
	if err != nil {
		log.Printf("warning: listing Codex models: %v", err)
		return
	}
	catalog.SetModels(chat.ProviderCodex, models)
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
