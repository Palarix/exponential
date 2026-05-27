package beats

import (
	"fmt"
	"os/exec"
	"strings"
)

// GetUser returns the recorded author for new events. Resolution order:
// caller-provided override (used by MCP for agent identity), configured
// user, then git config as a last resort.
func (c *Client) GetUser() string {
	// 1. Caller-provided override (e.g. MCP agent identity)
	if c.UserOverride != "" {
		return c.UserOverride
	}

	// 2. Check config
	if c.Config.User != "" {
		return c.Config.User
	}

	// 2. Fallback to git config
	nameBytes, _ := exec.Command("git", "config", "user.name").Output()
	emailBytes, _ := exec.Command("git", "config", "user.email").Output()

	name := strings.TrimSpace(string(nameBytes))
	email := strings.TrimSpace(string(emailBytes))

	if name == "" {
		name = "Unknown"
	}
	if email == "" {
		email = "unknown@example.com"
	}

	return fmt.Sprintf("%s <%s>", name, email)
}
