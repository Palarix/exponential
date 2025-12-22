package beats

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CompletionResult holds the status of shell completion configuration
type CompletionResult struct {
	Shell      string
	ConfigFile string
	Configured bool
}

// CheckCompletionConfig detects the current shell and checks if completion is configured
func CheckCompletionConfig() CompletionResult {
	shell := detectShell()
	configFile := getRCFile(shell)

	configured := false
	if configFile != "" {
		content, err := os.ReadFile(configFile)
		if err == nil {
			// Check for typical completion setup lines
			// "beats completion <shell>" is the core indicator
			if strings.Contains(string(content), "beats completion "+shell) {
				configured = true
			}
		}
	}

	return CompletionResult{
		Shell:      shell,
		ConfigFile: configFile,
		Configured: configured,
	}
}

// GetCompletionInstallCmd returns the command to install completion for the given shell
func GetCompletionInstallCmd(shell string) string {
	rcFile := getRCFile(shell)
	if rcFile == "" {
		return fmt.Sprintf("# Could not determine config file, please see `beats completion %s --help`", shell)
	}

	// Simplify path for display if it's in home directory
	home, _ := os.UserHomeDir()
	displayRC := rcFile
	if strings.HasPrefix(rcFile, home) {
		displayRC = "~" + strings.TrimPrefix(rcFile, home)
	}

	switch shell {
	case "zsh":
		return fmt.Sprintf("echo 'source <(beats completion zsh)' >> %s", displayRC)
	case "bash":
		return fmt.Sprintf("echo 'source <(beats completion bash)' >> %s", displayRC)
	case "fish":
		return "beats completion fish > ~/.config/fish/completions/beats.fish"
	default:
		return fmt.Sprintf("# Please see `beats completion %s --help`", shell)
	}
}

func detectShell() string {
	shellPath := os.Getenv("SHELL")
	if shellPath == "" {
		return "unknown"
	}
	base := filepath.Base(shellPath)
	return base
}

func getRCFile(shell string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	switch shell {
	case "zsh":
		return filepath.Join(home, ".zshrc")
	case "bash":
		// Bash can be tricky, check .bashrc then .bash_profile
		rc := filepath.Join(home, ".bashrc")
		if _, err := os.Stat(rc); err == nil {
			return rc
		}
		return filepath.Join(home, ".bash_profile")
	case "fish":
		return filepath.Join(home, ".config/fish/config.fish")
	default:
		return ""
	}
}
