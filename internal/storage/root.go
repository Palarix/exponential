package storage

import (
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

var (
	hubRoot     string
	hubRootOnce sync.Once
)

// HubRoot returns the absolute path of the primary checkout ("hub").
// In a worktree it walks back through git-common-dir to find the main
// working tree; in a normal checkout or non-git context it returns the
// current working directory.
func HubRoot() string {
	hubRootOnce.Do(func() {
		out, err := exec.Command("git", "rev-parse", "--path-format=absolute", "--git-common-dir").Output()
		if err != nil {
			hubRoot = "."
			return
		}
		gitCommonDir := strings.TrimSpace(string(out))
		hubRoot = filepath.Dir(gitCommonDir)
	})
	return hubRoot
}

// XpoDir returns the absolute path to the .xpo directory on the hub.
func XpoDir() string {
	return filepath.Join(HubRoot(), ".xpo")
}

// ResetHubRoot clears the cached hub root so it is re-discovered on
// the next call. Intended for tests only.
func ResetHubRoot() {
	hubRootOnce = sync.Once{}
	hubRoot = ""
}
