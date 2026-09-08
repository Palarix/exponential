package exponential

import (
	"bufio"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/palarix/exponential/internal/config"
	"github.com/palarix/exponential/internal/storage"
)

// DefaultBranch returns the name of the default branch. It checks the
// global config first, then falls back to git detection.
func DefaultBranch() string {
	if cfg := config.Get(); cfg != nil && cfg.DefaultBranch != "" {
		return cfg.DefaultBranch
	}
	hub := storage.HubRoot()
	out, err := exec.Command("git", "-C", hub, "symbolic-ref", "refs/remotes/origin/HEAD").Output()
	if err == nil {
		ref := strings.TrimSpace(string(out))
		if parts := strings.SplitN(ref, "/", 4); len(parts) == 4 {
			return parts[3]
		}
	}
	// No remote — check init.defaultBranch config, then probe well-known
	// names. HEAD is unreliable here because the hub may be checked out
	// to a feature branch.
	out, err = exec.Command("git", "-C", hub, "config", "init.defaultBranch").Output()
	if err == nil {
		if name := strings.TrimSpace(string(out)); name != "" {
			return name
		}
	}
	for _, candidate := range []string{"main", "master"} {
		if err := exec.Command("git", "-C", hub, "rev-parse", "--verify", candidate).Run(); err == nil {
			return candidate
		}
	}
	return "main"
}

// BranchExists checks whether a local branch with the given name exists.
func BranchExists(name string) bool {
	err := exec.Command("git", "rev-parse", "--verify", name).Run()
	return err == nil
}

// CreateAndCheckoutBranch creates a new branch from base and checks it out.
func CreateAndCheckoutBranch(name, base string) error {
	return exec.Command("git", "checkout", "-b", name, base).Run()
}

// CheckoutBranch switches to an existing branch.
func CheckoutBranch(name string) error {
	return exec.Command("git", "checkout", name).Run()
}

// RemoteBranchExists checks whether a remote branch with the given name exists.
func RemoteBranchExists(name string) bool {
	err := exec.Command("git", "ls-remote", "--exit-code", "--heads", "origin", name).Run()
	return err == nil
}

// WorktreeDir returns the canonical worktree path for a branch inside
// the hub's .xpo/worktrees/ directory.
func WorktreeDir(branch string) string {
	return filepath.Join(storage.XpoDir(), "worktrees", branch)
}

// WorktreeAdd creates a new worktree at path on a new branch from base.
func WorktreeAdd(path, branch, base string) error {
	return exec.Command("git", "worktree", "add", "-b", branch, path, base).Run()
}

// WorktreeAddExisting creates a worktree for an already-existing local branch.
func WorktreeAddExisting(path, branch string) error {
	return exec.Command("git", "worktree", "add", path, branch).Run()
}

// WorktreeRemove removes a worktree directory and its administrative files.
func WorktreeRemove(path string) error {
	if err := exec.Command("git", "worktree", "remove", "--force", path).Run(); err != nil {
		return err
	}
	return exec.Command("git", "worktree", "prune").Run()
}

// WorktreeEntry represents one entry from `git worktree list --porcelain`.
type WorktreeEntry struct {
	Path   string
	Branch string
}

// WorktreeList returns all worktrees known to git.
func WorktreeList() ([]WorktreeEntry, error) {
	out, err := exec.Command("git", "worktree", "list", "--porcelain").Output()
	if err != nil {
		return nil, err
	}

	var entries []WorktreeEntry
	var current WorktreeEntry
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, "worktree "):
			if current.Path != "" {
				entries = append(entries, current)
			}
			current = WorktreeEntry{Path: strings.TrimPrefix(line, "worktree ")}
		case strings.HasPrefix(line, "branch "):
			ref := strings.TrimPrefix(line, "branch ")
			current.Branch = strings.TrimPrefix(ref, "refs/heads/")
		}
	}
	if current.Path != "" {
		entries = append(entries, current)
	}
	return entries, scanner.Err()
}

// FindWorktreeForBranch returns the worktree path for a branch, if one exists.
func FindWorktreeForBranch(branch string) (string, bool) {
	entries, err := WorktreeList()
	if err != nil {
		return "", false
	}
	for _, e := range entries {
		if e.Branch == branch {
			return e.Path, true
		}
	}
	return "", false
}
