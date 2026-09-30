package registry

import (
	"os"
	"os/exec"
	"testing"
)

func TestIsAliveCurrentProcess(t *testing.T) {
	if !isAlive(os.Getpid()) {
		t.Fatalf("isAlive(%d) = false for the current process, want true", os.Getpid())
	}
}

func TestIsAliveExitedChild(t *testing.T) {
	// Re-exec the test binary with a filter that matches no tests so the
	// child exits immediately on every platform.
	cmd := exec.Command(os.Args[0], "-test.run=^$")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start child: %v", err)
	}
	pid := cmd.Process.Pid
	if err := cmd.Wait(); err != nil {
		t.Fatalf("wait child: %v", err)
	}
	if isAlive(pid) {
		t.Fatalf("isAlive(%d) = true for an exited child, want false", pid)
	}
}

func TestIsAliveNonPositivePID(t *testing.T) {
	for _, pid := range []int{0, -1} {
		if isAlive(pid) {
			t.Errorf("isAlive(%d) = true, want false", pid)
		}
	}
}
