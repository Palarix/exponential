package beats

import (
	"fmt"
	"os/exec"
	"strings"
)

// GetUser returns the recorded author for new events. Resolution order:
// caller-provided override (used by MCP for agent identity), configured
// user, then git config as a last resort.
func (t *LocalTransport) GetUser() string {
	// 1. Caller-provided override (e.g. MCP agent identity)
	if t.UserOverride != "" {
		return t.UserOverride
	}

	if t.Config.User != "" {
		return t.Config.User
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
