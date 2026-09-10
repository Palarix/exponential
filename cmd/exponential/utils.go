package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/term"
)

// getUser returns the configured user or falls back to git config.
// Unexported because it relies on the global cfg variable.
func getUser() string {
	// 1. Check config (includes XPO_USER env via Viper)
	if cfg != nil && cfg.User != "" {
		return cfg.User // Already validated at config load
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

func isInteractive() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}
