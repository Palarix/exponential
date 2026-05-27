package mcpserver

import (
	"fmt"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// EnvAgentIdentity is the env var consulted first to determine the calling
// agent's identity. Format: "Name <email>" or any free-form string.
const EnvAgentIdentity = "BEATS_AGENT_IDENTITY"

// resolveAgentIdentity returns the author string to record on events
// originating from MCP tool calls. Precedence (highest first):
//
//  1. BEATS_AGENT_IDENTITY env var — authoritative; the operator chose this
//  2. MCP clientInfo from the initialize handshake — best-effort attribution
//  3. configDefault — typically the user's git/config identity
//
// Returning the configDefault means MCP writes look the same as a hand-run
// CLI invocation, which is the right behaviour when no agent-specific
// identity is configured.
func resolveAgentIdentity(session mcp.Session, configDefault string) string {
	if v := os.Getenv(EnvAgentIdentity); v != "" {
		return v
	}
	if ss, ok := session.(*mcp.ServerSession); ok && ss != nil {
		if p := ss.InitializeParams(); p != nil && p.ClientInfo != nil {
			name := p.ClientInfo.Name
			if name == "" {
				name = "mcp-client"
			}
			if p.ClientInfo.Version != "" {
				return fmt.Sprintf("%s/%s <agent@mcp>", name, p.ClientInfo.Version)
			}
			return fmt.Sprintf("%s <agent@mcp>", name)
		}
	}
	return configDefault
}
