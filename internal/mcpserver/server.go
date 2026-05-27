// Package mcpserver exposes beats over the Model Context Protocol so
// AI agents can manage issues with structured tool calls instead of
// shelling out to the CLI. Tool handlers are thin adapters: every
// mutation goes through the same internal/beats Client methods the CLI
// uses, so business logic is not duplicated across transports.
package mcpserver

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/kuyio/beats/internal/beats"
	"github.com/kuyio/beats/internal/config"
	"github.com/kuyio/beats/internal/version"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Run starts a stdio MCP server backed by the given config. It blocks
// until stdin is closed (the typical MCP client-disconnect signal),
// which counts as a clean shutdown.
func Run(ctx context.Context, cfg *config.Config) error {
	srv := mcp.NewServer(&mcp.Implementation{
		Name:    "beats",
		Version: version.CLIVersion,
	}, nil)

	ts := newToolset(cfg)
	ts.register(srv)

	err := srv.Run(ctx, &mcp.StdioTransport{})
	if isCleanShutdown(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("mcp server: %w", err)
	}
	return nil
}

// isCleanShutdown reports whether err represents the MCP client closing
// the connection (stdin EOF) rather than a real failure. The SDK currently
// surfaces this as a wrapped "server is closing: EOF" error.
func isCleanShutdown(err error) bool {
	if err == nil {
		return true
	}
	if errors.Is(err, io.EOF) || errors.Is(err, context.Canceled) {
		return true
	}
	return strings.Contains(err.Error(), "EOF")
}

// toolset holds dependencies shared by all tool handlers.
type toolset struct {
	cfg *config.Config
}

func newToolset(cfg *config.Config) *toolset {
	return &toolset{cfg: cfg}
}

// clientFor returns a beats.Client with UserOverride set to the resolved
// agent identity for this request. Each call gets a fresh Client so
// concurrent tool invocations don't race on shared state. A nil request
// (used in unit tests) skips the clientInfo path and falls through to
// env var / config default.
func (t *toolset) clientFor(req *mcp.CallToolRequest) *beats.Client {
	c := beats.NewClient(t.cfg)
	var session mcp.Session
	if req != nil {
		session = req.GetSession()
	}
	c.UserOverride = resolveAgentIdentity(session, t.cfg.User)
	return c
}
