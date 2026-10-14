//go:build !windows

package chat

import (
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// setProcessGroup starts the agent in its own process group, so a Ctrl-C
// in the terminal reaches only xpo, and a group kill also stops the MCP
// servers the agent started.
func setProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// terminate asks the agent to exit. Claude Code and Codex stop their
// background commands on SIGTERM; those run in process groups of their own.
func terminate(pid int) {
	_ = syscall.Kill(pid, syscall.SIGTERM)
}

// killGroup kills the agent's process group.
func killGroup(pid int) {
	_ = syscall.Kill(-pid, syscall.SIGKILL)
}

// processStart returns when a process started, to one second, or false
// when it is not running.
func processStart(pid int) (time.Time, bool) {
	cmd := exec.Command("ps", "-o", "lstart=", "-p", strconv.Itoa(pid))
	cmd.Env = append(os.Environ(), "LC_ALL=C") // the date format follows the locale
	out, err := cmd.Output()
	if err != nil {
		return time.Time{}, false
	}
	// ps pads a one-digit day with a space; Fields drops it.
	t, err := time.ParseInLocation("Mon Jan 2 15:04:05 2006", strings.Join(strings.Fields(string(out)), " "), time.Local)
	return t, err == nil
}

// stopOrphan stops an agent a previous board process left running: the one
// with pid that started at started. A process that only reuses the PID
// started at another time, and stays.
func stopOrphan(pid int, started time.Time) bool {
	at, ok := processStart(pid)
	if !ok || at.Sub(started).Abs() > 3*time.Second {
		return false
	}
	_ = syscall.Kill(-pid, syscall.SIGTERM)
	return true
}
