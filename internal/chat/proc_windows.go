//go:build windows

package chat

import (
	"os"
	"os/exec"
	"time"
)

func setProcessGroup(cmd *exec.Cmd) {}

func terminate(pid int) { killGroup(pid) }

func killGroup(pid int) {
	if p, err := os.FindProcess(pid); err == nil {
		_ = p.Kill()
	}
}

func stopOrphan(pid int, started time.Time) bool { return false }
