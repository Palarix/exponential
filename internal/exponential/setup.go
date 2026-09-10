package exponential

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/palarix/exponential/internal/config"
	"github.com/palarix/exponential/internal/storage"
)

// InitResult contains the outcome of the initialization.
type InitResult struct {
	Created bool
	Notes   []string
}

// DefaultPrefix derives an issue ID prefix from the current directory name.
// Returns the bare prefix without a trailing separator.
func DefaultPrefix() string {
	cwd, _ := os.Getwd()
	return sanitizePrefix(filepath.Base(cwd))
}

// InitProject initializes the .xpo directory structure.
// The prefix parameter sets the issue ID prefix (e.g. "myproject-").
func InitProject(force bool, prefix string) (*InitResult, error) {
	result := &InitResult{}
	xpoDir := ".xpo"

	// 1. Create .xpo directory
	if _, err := os.Stat(xpoDir); err == nil {
		if force {
			result.Notes = append(result.Notes, "Re-initializing existing .xpo directory")
		}
	} else {
		if err := os.MkdirAll(xpoDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create .xpo directory: %w", err)
		}
		result.Created = true
	}

	// 2. Write config.yaml (only on fresh init or --force)
	if result.Created || force {
		cwd, _ := os.Getwd()
		folderName := filepath.Base(cwd)

		configPath := filepath.Join(xpoDir, "config.yaml")
		var defaultLabelLines strings.Builder
		for _, name := range config.BuiltinLabelOrder {
			defaultLabelLines.WriteString(fmt.Sprintf("  - %s\n", name))
		}
		var labelLines strings.Builder
		for _, name := range config.BuiltinLabelOrder {
			color := config.BuiltinLabels[name]
			labelLines.WriteString(fmt.Sprintf("  %s: \"%s\"\n", name, color))
		}
		detectedBranch := DefaultBranch()
		configContent := fmt.Sprintf("name: %s\nprefix: %s\ndefault_branch: %s\nversion: 3\nworktrees: true\nestimation_system: fibonacci\ncount_unestimated: true\nautomations:\n  first_start: true\n  last_completed: true\ndefault_labels:\n%slabels:\n%sdrive:\n  supervisor:\n    agent: claude\n    model: sonnet\n  coder:\n    agent: claude\n  max_retries: 3\n  timeout: 30m\n  # test_cmd: make test\n", folderName, prefix, detectedBranch, defaultLabelLines.String(), labelLines.String())
		if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
			result.Notes = append(result.Notes, fmt.Sprintf("Could not write config.yaml: %v", err))
		} else {
			result.Notes = append(result.Notes, fmt.Sprintf("Configured issue prefix: %s", prefix))
		}
	}

	// 3. Initialize ref-based storage (refs/xpo/data)
	if CheckGitRepo() && !storage.RefStoreReady() {
		if err := storage.InitRefStore(); err != nil {
			return nil, fmt.Errorf("failed to initialize ref store: %w", err)
		}
		result.Notes = append(result.Notes, "Initialized refs/xpo/data for issue and artifact storage")
	}

	// 4. Update .gitignore
	gitignorePath := ".gitignore"
	content, err := os.ReadFile(gitignorePath)
	var contentStr string
	if err == nil {
		contentStr = string(content)
	}

	ignoreEntries := []string{".xpo/git.lock", ".xpo/worktrees/"}
	var toAdd []string
	for _, entry := range ignoreEntries {
		if !strings.Contains(contentStr, entry) {
			toAdd = append(toAdd, entry)
		}
	}
	if len(toAdd) > 0 {
		f, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err == nil {
			defer f.Close()
			if len(contentStr) > 0 && !strings.HasSuffix(contentStr, "\n") {
				f.WriteString("\n")
			}
			for _, entry := range toAdd {
				f.WriteString(entry + "\n")
			}
		} else {
			result.Notes = append(result.Notes, fmt.Sprintf("Could not write to .gitignore: %v", err))
		}
	}

	return result, nil
}

// EnsureGitattributesEntry adds an entry to .gitattributes if it's not already present.
func EnsureGitattributesEntry(entry string) {
	path := ".gitattributes"
	content, err := os.ReadFile(path)
	var contentStr string
	if err == nil {
		contentStr = string(content)
	}
	if strings.Contains(contentStr, entry) {
		return
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	if len(contentStr) > 0 && !strings.HasSuffix(contentStr, "\n") {
		f.WriteString("\n")
	}
	f.WriteString(entry + "\n")
}

// EnsureGitignoreEntry adds an entry to .gitignore if it's not already present.
func EnsureGitignoreEntry(entry string) {
	gitignorePath := ".gitignore"
	content, err := os.ReadFile(gitignorePath)
	var contentStr string
	if err == nil {
		contentStr = string(content)
	}
	if strings.Contains(contentStr, entry) {
		return
	}
	f, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	if len(contentStr) > 0 && !strings.HasSuffix(contentStr, "\n") {
		f.WriteString("\n")
	}
	f.WriteString(entry + "\n")
}

// sanitizePrefix converts a folder name to a valid issue ID prefix
func sanitizePrefix(name string) string {
	name = strings.ToLower(name)
	var result strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			result.WriteRune(r)
		}
	}
	s := result.String()
	if s == "" {
		s = "issue" // Fallback if folder name has no valid chars
	}
	return s
}

// UpdateConfigVersion updates the version field in config.yaml
func UpdateConfigVersion(newVersion int) error {
	path := filepath.Join(".xpo", "config.yaml")
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	found := false
	for i, line := range lines {
		if strings.HasPrefix(line, "version:") {
			lines[i] = fmt.Sprintf("version: %d", newVersion)
			found = true
			break
		}
	}

	if !found {
		lines = append(lines, fmt.Sprintf("version: %d", newVersion))
	}

	output := strings.Join(lines, "\n")
	return os.WriteFile(path, []byte(output), 0644)
}
