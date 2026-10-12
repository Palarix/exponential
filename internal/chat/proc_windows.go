//go:build windows

package chat

import "os/exec"

func setProcessGroup(cmd *exec.Cmd) {}

func killProcessGroup(cmd *exec.Cmd) {
	_ = cmd.Process.Kill()
}
