//go:build !windows

package chat

import (
	"os/exec"
	"syscall"
)

// setProcessGroup starts the agent in its own process group, so a cancel
// also stops the MCP servers and tools it spawned.
func setProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func killProcessGroup(cmd *exec.Cmd) {
	_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
}
