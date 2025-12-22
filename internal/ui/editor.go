package ui

import (
	"os"
	"os/exec"
	"strings"
)

// EditInteractive opens the user's EDITOR to edit the given template.
// Returns the content or an empty string if aborted/unchanged.
func EditInteractive(template string) (string, error) {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}

	tmpFile, err := os.CreateTemp("", "beats-*.txt")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(template); err != nil {
		return "", err
	}
	if err := tmpFile.Close(); err != nil {
		return "", err
	}

	cmd := exec.Command(editor, tmpFile.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", err
	}

	// Read content
	contentBytes, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		return "", err
	}
	content := string(contentBytes)

	// Check if content is unchanged or empty
	if strings.TrimSpace(content) == "" || strings.TrimSpace(content) == strings.TrimSpace(template) {
		return "", nil // Treated as empty -> abort
	}

	return content, nil
}
