package main

import (
	"context"
	"crypto/ed25519"
	"log"
	"net"
	"net/http"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/palarix/beats/internal/auth"
	"github.com/palarix/beats/internal/mcpserver"
	"github.com/palarix/beats/internal/server"
	"github.com/palarix/beats/internal/version"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
)

var serveAddr string

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start a headless beats server with auth",
	Long: `Starts a headless HTTP server exposing the beats REST API and MCP
endpoint, both behind SSH key + JWT bearer token authentication.

The server reads .beats/authorized_keys for user verification and
auto-generates a signing key at .beats/server.key on first run.

Endpoints:
  POST /auth/challenge   — get a nonce for SSH key auth
  POST /auth/verify      — exchange signed nonce for a JWT
  GET  /healthz          — health check (no auth)
  /api/*                 — REST API (auth required)
  /mcp                   — MCP endpoint (auth required)`,
	RunE: runServe,
}

func init() {
	serveCmd.Flags().StringVar(&serveAddr, "addr", ":8080", "address to listen on")
	rootCmd.AddCommand(serveCmd)
}

func runServe(cmd *cobra.Command, args []string) error {
	beatsDir := ".beats"

	// Load or generate server signing key
	serverKeyPath := filepath.Join(beatsDir, "server.key")
	signingKey, err := auth.LoadOrGenerateServerKey(serverKeyPath)
	if err != nil {
		return err
	}
	verifyKey := signingKey.Public().(ed25519.PublicKey)
	log.Printf("Server key loaded from %s", serverKeyPath)

	// Load or create authorized keys
	authKeysPath := filepath.Join(beatsDir, "authorized_keys")
	authorizedKeys, err := auth.LoadOrCreateAuthorizedKeys(authKeysPath)
	if err != nil {
		return err
	}
	if authorizedKeys.Len() == 0 {
		log.Printf("Warning: %s has no keys — no one can authenticate. Add SSH public keys to enable login.", authKeysPath)
	} else {
		log.Printf("Loaded %d authorized key(s) from %s", authorizedKeys.Len(), authKeysPath)
	}

	// Create server
	srv := server.NewServer(cfg, 0, false, 0)
	srv.Headless = true
	srv.SigningKey = signingKey
	srv.VerifyKey = verifyKey
	srv.NonceStore = auth.NewNonceStore(5 * time.Minute)
	srv.AuthorizedKeys = authorizedKeys
	srv.SSEHub = server.NewSSEHub()

	// Set up MCP-over-HTTP handler
	mcpHandler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		mcpSrv := mcp.NewServer(&mcp.Implementation{
			Name:    "beats",
			Version: version.CLIVersion,
		}, nil)
		mcpserver.RegisterTools(mcpSrv, cfg, r)
		return mcpSrv
	}, nil)
	srv.MCPHandler = mcpHandler

	// Start listening
	listener, err := net.Listen("tcp", serveAddr)
	if err != nil {
		return err
	}
	srv.Port = listener.Addr().(*net.TCPAddr).Port

	log.Printf("beats serve listening on %s", listener.Addr())

	// Graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	httpServer := &http.Server{
		Handler:      srv.SetupRoutes(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() { errCh <- httpServer.Serve(listener) }()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		log.Println("Shutting down...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return httpServer.Shutdown(shutdownCtx)
	}
}
