//go:build !windows

package registry

import (
	"errors"
	"os"
	"syscall"
)

// isAlive reports whether a process with the given PID exists by sending it
// signal 0. EPERM means the process exists but belongs to another user, so it
// counts as alive. Non-positive PIDs are rejected because kill(0) and kill(-1)
// address process groups rather than a single process.
func isAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = p.Signal(syscall.Signal(0))
	return err == nil || errors.Is(err, syscall.EPERM)
}
