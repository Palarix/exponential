// Package mcpserver exposes exponential over the Model Context Protocol so
// AI agents can manage issues with structured tool calls instead of
// shelling out to the CLI. Tool handlers are thin adapters: every
// mutation goes through the same internal/exponential Client methods the CLI
// uses, so business logic is not duplicated across transports.
package mcpserver

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/palarix/exponential/internal/auth"
	"github.com/palarix/exponential/internal/exponential"
	"github.com/palarix/exponential/internal/config"
	"github.com/palarix/exponential/internal/version"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Run starts a stdio MCP server backed by the given config. It blocks
// until stdin is closed (the typical MCP client-disconnect signal),
// which counts as a clean shutdown.
func Run(ctx context.Context, cfg *config.Config) error {
	srv := mcp.NewServer(&mcp.Implementation{
		Name:    "xpo",
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
	cfg              *config.Config
	httpUserOverride string
	onEvent          func(eventType, issueID string)
}

func newToolset(cfg *config.Config) *toolset {
	return &toolset{cfg: cfg}
}

func (t *toolset) broadcast(eventType, issueID string) {
	if t.onEvent != nil {
		t.onEvent(eventType, issueID)
	}
}

// RegisterTools registers all xpo MCP tools on the given server.
// When httpReq is non-nil (HTTP transport), the authenticated user identity
// from the request context is used for event attribution.
func RegisterTools(srv *mcp.Server, cfg *config.Config, httpReq *http.Request, onEvent func(eventType, issueID string)) {
	ts := newToolset(cfg)
	ts.onEvent = onEvent
	if httpReq != nil {
		if user, ok := auth.UserFromContext(httpReq.Context()); ok {
			ts.httpUserOverride = user.Raw
		}
	}
	ts.register(srv)
}

// clientFor returns a exponential.Client with UserOverride set to the resolved
// agent identity for this request. Each call gets a fresh Client so
// concurrent tool invocations don't race on shared state. A nil request
// (used in unit tests) skips the clientInfo path and falls through to
// env var / config default.
func (t *toolset) clientFor(req *mcp.CallToolRequest) *exponential.Client {
	c := exponential.NewClient(t.cfg)
	c.Source = "mcp"
	if t.httpUserOverride != "" {
		c.UserOverride = t.httpUserOverride
	} else {
		var session mcp.Session
		if req != nil {
			session = req.GetSession()
		}
		c.UserOverride = resolveAgentIdentity(session, t.cfg.User)
	}
	return c
}
