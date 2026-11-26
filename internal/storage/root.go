package storage

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

var (
	hubRoot     string
	bareCommon  string
	hubRootOnce sync.Once
)

// ErrBareRepository: the git common directory is a bare repository, so
// there is no primary checkout to use as the hub.
var ErrBareRepository = errors.New("xpo needs a primary checkout: this is a worktree of a bare repository, which xpo does not support")

// ErrNoXpoDir: the hub has no .xpo directory.
var ErrNoXpoDir = errors.New("no .xpo directory")

// HubRoot returns the absolute path of the primary checkout ("hub").
// In a worktree it walks back through git-common-dir to find the main
// working tree; in a normal checkout or non-git context it returns the
// current working directory. For a worktree of a bare repository it
// returns the bare repository's parent, which is not a checkout:
// CheckHub reports that case.
func HubRoot() string {
	hubRootOnce.Do(func() {
		out, err := exec.Command("git", "rev-parse", "--path-format=absolute", "--git-common-dir").Output()
		if err != nil {
			hubRoot = "."
			return
		}
		gitCommonDir := strings.TrimSpace(string(out))
		hubRoot = filepath.Dir(gitCommonDir)
		bare, err := exec.Command("git", "--git-dir="+gitCommonDir, "rev-parse", "--is-bare-repository").Output()
		if err == nil && strings.TrimSpace(string(bare)) == "true" {
			bareCommon = gitCommonDir
		}
	})
	return hubRoot
}

// CheckHub reports why the hub cannot hold issue data: the common
// directory is a bare repository, or the hub has no .xpo directory.
// A .xpo directory without issues.db is a valid, empty project.
func CheckHub() error {
	hub := HubRoot()
	if bareCommon != "" {
		return fmt.Errorf("%w (%s)", ErrBareRepository, bareCommon)
	}
	info, err := os.Stat(XpoDir())
	if err != nil || !info.IsDir() {
		return fmt.Errorf("%w at %s: run xpo from an Exponential project, or run 'xpo init'", ErrNoXpoDir, hub)
	}
	return nil
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
	bareCommon = ""
}
