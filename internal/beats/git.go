package beats

import (
	"os/exec"
	"strings"
)

// DefaultBranch returns the name of the default branch by inspecting
// the remote HEAD ref. Falls back to "main" if detection fails.
func DefaultBranch() string {
	out, err := exec.Command("git", "symbolic-ref", "refs/remotes/origin/HEAD").Output()
	if err == nil {
		ref := strings.TrimSpace(string(out))
		if parts := strings.SplitN(ref, "/", 4); len(parts) == 4 {
			return parts[3]
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
