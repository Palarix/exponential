package beats

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/palarix/beats/internal/config"
)

// InitResult contains the outcome of the initialization.
type InitResult struct {
	Created bool
	Notes   []string
}

// InitBeats initializes the .beats directory structure.
func InitBeats(force bool) (*InitResult, error) {
	result := &InitResult{}
	beatsDir := ".beats"

	// 1. Create .beats directory
	if _, err := os.Stat(beatsDir); err == nil {
		if !force {
			return nil, fmt.Errorf("directory already exists") // Caller handles UI message
		}
		result.Created = false
		result.Notes = append(result.Notes, "Re-initializing existing .beats directory")
	} else {
		if err := os.MkdirAll(beatsDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create .beats directory: %w", err)
		}
		result.Created = true
	}

	// 2. Derive prefix and write config.yaml
	cwd, _ := os.Getwd()
	folderName := filepath.Base(cwd)
	prefix := sanitizePrefix(folderName) + "-"

	configPath := filepath.Join(beatsDir, "config.yaml")
	var defaultLabelLines strings.Builder
	for _, name := range config.BuiltinLabelOrder {
		defaultLabelLines.WriteString(fmt.Sprintf("  - %s\n", name))
	}
	var labelLines strings.Builder
	for _, name := range config.BuiltinLabelOrder {
		color := config.BuiltinLabels[name]
		labelLines.WriteString(fmt.Sprintf("  %s: \"%s\"\n", name, color))
	}
	configContent := fmt.Sprintf("prefix: %s\nversion: 3\nestimation_system: fibonacci\ncount_unestimated: true\nautomations:\n  auto_complete_parent: false\n  auto_close_sub_issues: false\n  auto_progress_sub_issues: false\n  auto_progress_parent: false\ndefault_labels:\n%slabels:\n%s", prefix, defaultLabelLines.String(), labelLines.String())
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		result.Notes = append(result.Notes, fmt.Sprintf("Could not write config.yaml: %v", err))
	} else {
		result.Notes = append(result.Notes, fmt.Sprintf("Configured issue prefix: %s", prefix))
	}

	// 3. Create .beats/issues.db
	issuesFile := filepath.Join(beatsDir, "issues.db")
	f, err := os.OpenFile(issuesFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to create issues.db: %w", err)
	}
	f.Close()

	// 4. Update .gitignore
	gitignorePath := ".gitignore"
	content, err := os.ReadFile(gitignorePath)
	var contentStr string
	if err == nil {
		contentStr = string(content)
	}

	ignoreEntry := ".beats/issues.snapshot.json"
	if !strings.Contains(contentStr, ignoreEntry) {
		f, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err == nil {
			defer f.Close()
			if len(contentStr) > 0 && !strings.HasSuffix(contentStr, "\n") {
				f.WriteString("\n")
			}
			f.WriteString(ignoreEntry + "\n")
		} else {
			result.Notes = append(result.Notes, fmt.Sprintf("Could not write to .gitignore: %v", err))
		}
	}

	return result, nil
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
		s = "beats" // Fallback if folder name has no valid chars
	}
	return s
}

// UpdateConfigVersion updates the version field in config.yaml
func UpdateConfigVersion(newVersion int) error {
	path := filepath.Join(".beats", "config.yaml")
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
