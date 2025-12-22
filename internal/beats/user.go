package beats

import (
	"fmt"
	"os/exec"
	"strings"
)

// GetUser returns the configured user or falls back to git config.
func (c *Client) GetUser() string {
	// 1. Check config
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
