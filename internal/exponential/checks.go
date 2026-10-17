package exponential

import (
	"os"
	"path/filepath"
	"strings"
)

// CheckGitRepo checks if the current directory is a git repository.
func CheckGitRepo() bool {
	return IsGitRepo(".")
}

// IsGitRepo checks if dir is a git repository: it has a .git directory, or
// a .git file as a worktree does.
func IsGitRepo(dir string) bool {
	if _, err := os.Stat(filepath.Join(dir, ".git")); os.IsNotExist(err) {
		return false
	}
	return true
}

// CheckGithubWorkflows checks for existence of GitHub workflow files.
func CheckGithubWorkflows() bool {
	matches, err := filepath.Glob(".github/workflows/*.y*ml")
	return err == nil && len(matches) > 0
}

// CheckGitHooks returns a list of active git hooks found.
func CheckGitHooks() []string {
	hookFiles, _ := filepath.Glob(".git/hooks/*")
	var activeHooks []string
	for _, h := range hookFiles {
		if !strings.HasSuffix(h, ".sample") {
			activeHooks = append(activeHooks, filepath.Base(h))
		}
	}
	return activeHooks
}
